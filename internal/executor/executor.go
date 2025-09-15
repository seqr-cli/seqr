package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

type Executor struct {
	mu              sync.RWMutex
	status          ExecutionStatus
	verbose         bool
	stopped         bool
	processes       map[string]*exec.Cmd
	reporter        Reporter
	tracker         *ProcessTracker
	monitor         *ProcessMonitor
	streamingActive map[string]context.CancelFunc // Track active streaming sessions
}

func NewExecutor(verbose bool) *Executor {
	tracker := NewProcessTracker()
	monitor := NewProcessMonitor(verbose, tracker)

	return &Executor{
		verbose:         verbose,
		processes:       make(map[string]*exec.Cmd),
		reporter:        NewConsoleReporter(os.Stdout, verbose),
		tracker:         tracker,
		monitor:         monitor,
		streamingActive: make(map[string]context.CancelFunc),
		status: ExecutionStatus{
			State:   StateReady,
			Results: make([]ExecutionResult, 0),
		},
	}
}

func (e *Executor) Execute(ctx context.Context, cfg *config.Config) error {
	if len(cfg.Commands) == 0 {
		return fmt.Errorf("no commands to execute")
	}

	e.mu.Lock()
	e.status = ExecutionStatus{
		State:      StateReady,
		TotalCount: len(cfg.Commands),
		Results:    make([]ExecutionResult, 0, len(cfg.Commands)),
	}
	e.stopped = false
	e.mu.Unlock()

	// Start process monitoring
	e.monitor.StartMonitoring(ctx)
	defer e.monitor.StopMonitoring()

	// Start monitoring status changes in a separate goroutine
	go e.handleStatusChanges(ctx)

	e.reporter.ReportStart(len(cfg.Commands))

	// Group commands by concurrent execution
	commandGroups := e.groupCommandsByConcurrency(cfg.Commands)

	commandIndex := 0
	for _, group := range commandGroups {
		if e.isStopped() {
			return fmt.Errorf("execution stopped")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if len(group) == 1 {
			// Single command - execute sequentially
			cmd := group[0]
			e.updateCurrentCommand(&cmd)
			e.reporter.ReportCommandStart(cmd.Name, commandIndex)

			result, err := e.executeCommand(ctx, cmd)
			e.addResult(result)

			if err != nil {
				e.reporter.ReportCommandFailure(result, commandIndex)
				e.updateState(StateFailed, err.Error())
				return err
			}

			e.reporter.ReportCommandSuccess(result, commandIndex)
			e.updateCompletedCount(commandIndex + 1)
			commandIndex++
		} else {
			// Multiple concurrent commands - execute in parallel
			if err := e.executeConcurrentCommands(ctx, group, &commandIndex); err != nil {
				return err
			}
		}
	}

	e.updateState(StateSuccess, "")
	status := e.GetStatus()
	e.reporter.ReportExecutionComplete(status)
	return nil
}

func (e *Executor) executeCommand(ctx context.Context, cmd config.Command) (ExecutionResult, error) {
	result := ExecutionResult{
		Command:   cmd,
		StartTime: time.Now(),
	}

	execCmd := exec.CommandContext(ctx, cmd.Command, cmd.Args...)

	if cmd.WorkDir != "" {
		execCmd.Dir = cmd.WorkDir
	}

	if len(cmd.Env) > 0 {
		execCmd.Env = os.Environ()
		for key, value := range cmd.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Configure process group for proper child process cleanup
	e.configureProcessGroup(execCmd)

	switch cmd.Mode {
	case config.ModeOnce:
		return e.executeOnce(execCmd, result)
	case config.ModeKeepAlive:
		return e.executeKeepAlive(execCmd, result, cmd.Name)
	default:
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = fmt.Sprintf("unsupported mode: %s", cmd.Mode)
		return result, fmt.Errorf("unsupported mode: %s", cmd.Mode)
	}
}

func (e *Executor) executeOnce(execCmd *exec.Cmd, result ExecutionResult) (ExecutionResult, error) {
	if e.verbose {
		return e.executeOnceWithRealTimeOutput(execCmd, result)
	}

	// Non-verbose mode: use existing behavior
	output, err := execCmd.CombinedOutput()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Output = strings.TrimSpace(string(output))

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result, err
	}

	result.Success = true
	result.ExitCode = 0
	return result, nil
}

func (e *Executor) executeOnceWithRealTimeOutput(execCmd *exec.Cmd, result ExecutionResult) (ExecutionResult, error) {
	// Create pipes for stdout and stderr
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = fmt.Sprintf("failed to create stdout pipe: %v", err)
		result.ExitCode = -1
		return result, err
	}

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = fmt.Sprintf("failed to create stderr pipe: %v", err)
		result.ExitCode = -1
		return result, err
	}

	// Start the command
	if err := execCmd.Start(); err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = err.Error()
		result.ExitCode = -1
		return result, err
	}

	// Capture output in real-time
	var outputBuilder strings.Builder
	var wg sync.WaitGroup

	// Stream stdout with proper error handling
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[%s] [%s] ❌ Streaming panic recovered: %v\n",
					time.Now().Format("15:04:05.000"), result.Command.Name, r)
				os.Stdout.Sync()
			}
		}()
		e.streamOutput(stdoutPipe, &outputBuilder, result.Command.Name, "stdout", result.Command.Command)
	}()

	// Stream stderr with proper error handling
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[%s] [%s] ❌ Streaming panic recovered: %v\n",
					time.Now().Format("15:04:05.000"), result.Command.Name, r)
				os.Stdout.Sync()
			}
		}()
		e.streamOutput(stderrPipe, &outputBuilder, result.Command.Name, "stderr", result.Command.Command)
	}()

	// Wait for command to complete
	err = execCmd.Wait()

	// Wait for all output streaming to complete
	wg.Wait()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Output = strings.TrimSpace(outputBuilder.String())

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result, err
	}

	result.Success = true
	result.ExitCode = 0
	return result, nil
}

func (e *Executor) detectCommandType(command string) string {
	if strings.Contains(command, "docker") {
		return "docker"
	}
	if strings.Contains(command, "vite") {
		return "vite"
	}
	if strings.Contains(command, "node") {
		return "node"
	}
	if strings.Contains(command, "bun") {
		return "bun"
	}
	if strings.Contains(command, "npm") {
		return "npm"
	}
	if strings.Contains(command, "yarn") {
		return "yarn"
	}
	if strings.Contains(command, "pnpm") {
		return "pnpm"
	}
	return "exec"
}

func (e *Executor) streamOutput(pipe io.ReadCloser, outputBuilder *strings.Builder, commandName, streamType, command string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[%s] [%s] ❌ Streaming panic recovered: %v\n",
				time.Now().Format("15:04:05.000"), commandName, r)
			os.Stdout.Sync()
		}
	}()

	cmdType := e.detectCommandType(command)
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		timestamp := time.Now().Format("15:04:05.000")

		// Write to console with timestamp, type, command identification
		// Use different visual indicators for stdout vs stderr
		if streamType == "stderr" {
			fmt.Printf("[%s] [%s] [%s] ❌ %s\n", timestamp, cmdType, commandName, line)
		} else {
			fmt.Printf("[%s] [%s] [%s] ✓  %s\n", timestamp, cmdType, commandName, line)
		}

		// Ensure immediate output by flushing stdout
		os.Stdout.Sync()

		// Also capture for the result output
		outputBuilder.WriteString(line)
		outputBuilder.WriteString("\n")
	}

	if err := scanner.Err(); err != nil && !strings.Contains(err.Error(), "file already closed") {
		fmt.Printf("[%s] [%s] [%s] ❌ Error reading %s: %v\n",
			time.Now().Format("15:04:05.000"), cmdType, commandName, streamType, err)
		os.Stdout.Sync()
	}
}

func (e *Executor) executeKeepAlive(execCmd *exec.Cmd, result ExecutionResult, name string) (ExecutionResult, error) {
	if e.verbose {
		return e.executeKeepAliveWithRealTimeOutput(execCmd, result, name)
	}

	// Non-verbose mode: use existing behavior
	err := execCmd.Start()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.ExitCode = -1
		return result, err
	}

	e.mu.Lock()
	e.processes[name] = execCmd
	e.mu.Unlock()

	// Track the process for kill functionality
	if err := e.tracker.AddProcess(
		execCmd.Process.Pid,
		name,
		result.Command.Command,
		result.Command.Args,
		result.Command.WorkDir,
		string(result.Command.Mode),
	); err != nil && e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [%s] [process] Warning: Failed to track process: %v\n", timestamp, name, err)
	}

	// Add process to monitoring
	e.monitor.AddProcess(execCmd.Process.Pid, name)

	go e.monitorProcess(name, execCmd)

	result.Success = true
	result.ExitCode = 0
	result.Output = fmt.Sprintf("started with PID %d", execCmd.Process.Pid)
	return result, nil
}

func (e *Executor) executeKeepAliveWithRealTimeOutput(execCmd *exec.Cmd, result ExecutionResult, name string) (ExecutionResult, error) {
	// Create pipes for stdout and stderr
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = fmt.Sprintf("failed to create stdout pipe: %v", err)
		result.ExitCode = -1
		return result, err
	}

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = fmt.Sprintf("failed to create stderr pipe: %v", err)
		result.ExitCode = -1
		return result, err
	}

	// Start the command
	if err := execCmd.Start(); err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = err.Error()
		result.ExitCode = -1
		return result, err
	}

	e.mu.Lock()
	e.processes[name] = execCmd
	e.mu.Unlock()

	// Track the process for kill functionality
	if err := e.tracker.AddProcess(
		execCmd.Process.Pid,
		name,
		result.Command.Command,
		result.Command.Args,
		result.Command.WorkDir,
		string(result.Command.Mode),
	); err != nil && e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [%s] [process] Warning: Failed to track process: %v\n", timestamp, name, err)
	}

	// Add process to monitoring
	e.monitor.AddProcess(execCmd.Process.Pid, name)

	// Create streaming context that can be cancelled independently of process lifecycle
	streamCtx, streamCancel := context.WithCancel(context.Background())

	// Track the streaming session
	e.mu.Lock()
	e.streamingActive[name] = streamCancel
	e.mu.Unlock()

	// Start streaming output in background goroutines with proper lifecycle management
	var streamWg sync.WaitGroup

	streamWg.Add(2)
	go func() {
		defer streamWg.Done()
		e.streamOutputContinuousWithContext(streamCtx, stdoutPipe, name, "stdout", result.Command.Command)
	}()

	go func() {
		defer streamWg.Done()
		e.streamOutputContinuousWithContext(streamCtx, stderrPipe, name, "stderr", result.Command.Command)
	}()

	// Monitor the process and streaming lifecycle
	go func() {
		e.monitorProcessWithStreaming(name, execCmd, streamCancel, &streamWg)
	}()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true
	result.ExitCode = 0
	result.Output = fmt.Sprintf("started with PID %d", execCmd.Process.Pid)
	return result, nil
}

func (e *Executor) streamOutputContinuous(pipe io.ReadCloser, commandName, streamType, command string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[%s] [%s] ❌ Streaming panic recovered: %v\n",
				time.Now().Format("15:04:05.000"), commandName, r)
			os.Stdout.Sync()
		}
		// Ensure pipe is closed
		pipe.Close()
	}()

	cmdType := e.detectCommandType(command)
	scanner := bufio.NewScanner(pipe)

	// Set a smaller buffer size to reduce latency for real-time streaming
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		// Check if executor has been stopped
		if e.isStopped() {
			break
		}

		line := scanner.Text()
		timestamp := time.Now().Format("15:04:05.000")

		// Write to console with timestamp, type, and command identification
		// Use different visual indicators for stdout vs stderr
		if streamType == "stderr" {
			fmt.Printf("[%s] [%s] [%s] ❌ %s\n", timestamp, cmdType, commandName, line)
		} else {
			fmt.Printf("[%s] [%s] [%s] ✓  %s\n", timestamp, cmdType, commandName, line)
		}

		// Ensure immediate output by flushing stdout for real-time streaming
		os.Stdout.Sync()
	}

	if err := scanner.Err(); err != nil && !e.isStopped() && !strings.Contains(err.Error(), "file already closed") {
		fmt.Printf("[%s] [%s] [%s] ❌ Error reading %s: %v\n",
			time.Now().Format("15:04:05.000"), cmdType, commandName, streamType, err)
		os.Stdout.Sync()
	}
}

func (e *Executor) streamOutputContinuousWithContext(ctx context.Context, pipe io.ReadCloser, commandName, streamType, command string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[%s] [%s] ❌ Streaming panic recovered: %v\n",
				time.Now().Format("15:04:05.000"), commandName, r)
			os.Stdout.Sync()
		}
		// Ensure pipe is closed
		pipe.Close()
	}()

	cmdType := e.detectCommandType(command)
	scanner := bufio.NewScanner(pipe)

	// Set a smaller buffer size to reduce latency for real-time streaming
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		// Check if streaming context has been cancelled or executor has been stopped
		select {
		case <-ctx.Done():
			// Streaming has been cancelled, but process continues running
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [%s] [streaming] Detached from output streaming (process continues in background)\n", timestamp, cmdType, commandName)
			os.Stdout.Sync()
			return
		default:
		}

		if e.isStopped() {
			break
		}

		line := scanner.Text()
		timestamp := time.Now().Format("15:04:05.000")

		// Write to console with timestamp, type, and command identification
		// Use different visual indicators for stdout vs stderr
		if streamType == "stderr" {
			fmt.Printf("[%s] [%s] [%s] ❌ %s\n", timestamp, cmdType, commandName, line)
		} else {
			fmt.Printf("[%s] [%s] [%s] ✓  %s\n", timestamp, cmdType, commandName, line)
		}

		// Ensure immediate output by flushing stdout for real-time streaming
		os.Stdout.Sync()
	}

	if err := scanner.Err(); err != nil && !e.isStopped() && !strings.Contains(err.Error(), "file already closed") {
		// Only log errors if context hasn't been cancelled (streaming wasn't intentionally stopped)
		select {
		case <-ctx.Done():
			// Context was cancelled, this is expected
		default:
			fmt.Printf("[%s] [%s] [%s] ❌ Error reading %s: %v\n",
				time.Now().Format("15:04:05.000"), cmdType, commandName, streamType, err)
			os.Stdout.Sync()
		}
	}
}

func (e *Executor) monitorProcess(name string, cmd *exec.Cmd) {
	err := cmd.Wait()

	e.mu.Lock()
	delete(e.processes, name)
	e.mu.Unlock()

	// Remove from process tracker and monitoring
	if cmd.Process != nil {
		pid := cmd.Process.Pid

		// Check if this was an unexpected termination
		if err != nil {
			exitCode := -1
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			}
			// Notify monitor of unexpected termination
			e.monitor.NotifyUnexpectedTermination(pid, name, exitCode, err)
		}

		// Remove from tracking
		if trackErr := e.tracker.RemoveProcess(pid); trackErr != nil && e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Warning: Failed to untrack process: %v\n", timestamp, name, trackErr)
		}

		// Remove from monitoring
		e.monitor.RemoveProcess(pid)
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		if err != nil {
			fmt.Printf("[%s] [%s] [process] Process exited with error: %v\n", timestamp, name, err)
		} else {
			fmt.Printf("[%s] [%s] [process] Process exited cleanly\n", timestamp, name)
		}
		// Ensure immediate output for process status updates
		os.Stdout.Sync()
	}
}

func (e *Executor) monitorProcessWithStreaming(name string, cmd *exec.Cmd, streamCancel context.CancelFunc, streamWg *sync.WaitGroup) {
	err := cmd.Wait()

	// Cancel streaming when process ends
	streamCancel()

	// Wait for streaming goroutines to finish
	streamWg.Wait()

	e.mu.Lock()
	delete(e.processes, name)
	delete(e.streamingActive, name) // Clean up streaming tracking
	e.mu.Unlock()

	// Remove from process tracker and monitoring
	if cmd.Process != nil {
		pid := cmd.Process.Pid

		// Check if this was an unexpected termination
		if err != nil {
			exitCode := -1
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			}
			// Notify monitor of unexpected termination
			e.monitor.NotifyUnexpectedTermination(pid, name, exitCode, err)
		}

		// Remove from tracking
		if trackErr := e.tracker.RemoveProcess(pid); trackErr != nil && e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Warning: Failed to untrack process: %v\n", timestamp, name, trackErr)
		}

		// Remove from monitoring
		e.monitor.RemoveProcess(pid)
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		if err != nil {
			fmt.Printf("[%s] [%s] [process] Process exited with error: %v\n", timestamp, name, err)
		} else {
			fmt.Printf("[%s] [%s] [process] Process exited cleanly\n", timestamp, name)
		}
		// Ensure immediate output for process status updates
		os.Stdout.Sync()
	}
}

func (e *Executor) GetStatus() ExecutionStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := e.status
	status.Results = make([]ExecutionResult, len(e.status.Results))
	copy(status.Results, e.status.Results)
	return status
}

func (e *Executor) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.stopped = true

	for name, cmd := range e.processes {
		if cmd.Process != nil {
			// Mark this as an expected exit since we're stopping it
			e.monitor.MarkExpectedExit(cmd.Process.Pid)

			if e.verbose {
				timestamp := time.Now().Format("15:04:05.000")
				fmt.Printf("[%s] [%s] [process] Gracefully terminating process (PID %d)\n", timestamp, name, cmd.Process.Pid)
			}
			e.terminateProcessGracefully(cmd.Process, name)
		}
	}

	e.processes = make(map[string]*exec.Cmd)
}

func (e *Executor) isStopped() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stopped
}

func (e *Executor) updateCurrentCommand(cmd *config.Command) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.State = StateRunning
	e.status.CurrentCommand = cmd
}

func (e *Executor) updateState(state ExecutionState, errorMsg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.State = state
	e.status.CurrentCommand = nil
	e.status.LastError = errorMsg
}

func (e *Executor) addResult(result ExecutionResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.Results = append(e.status.Results, result)
}

func (e *Executor) updateCompletedCount(count int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.CompletedCount = count
}

// GetTrackedProcesses returns all currently tracked processes
func (e *Executor) GetTrackedProcesses() map[int]*ProcessInfo {
	return e.tracker.GetAllProcesses()
}

// GetTrackedProcess returns information about a specific tracked process
func (e *Executor) GetTrackedProcess(pid int) (*ProcessInfo, bool) {
	return e.tracker.GetProcess(pid)
}

// CleanupDeadProcesses removes dead processes from tracking
func (e *Executor) CleanupDeadProcesses() error {
	return e.tracker.CleanupDeadProcesses()
}

// GetTrackedProcessCount returns the number of currently tracked processes
func (e *Executor) GetTrackedProcessCount() int {
	return e.tracker.GetRunningProcessCount()
}

// HasActiveKeepAliveProcesses returns true if there are currently running keepAlive processes
func (e *Executor) HasActiveKeepAliveProcesses() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.processes) > 0
}

// HasActiveStreaming returns true if there are currently active streaming sessions
func (e *Executor) HasActiveStreaming() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.streamingActive) > 0
}

// DetachFromStreaming cancels all active streaming sessions while keeping processes running
func (e *Executor) DetachFromStreaming() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.streamingActive) == 0 {
		return
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [seqr] [streaming] Detaching from %d active streaming session(s)...\n", timestamp, len(e.streamingActive))
		os.Stdout.Sync()
	}

	// Cancel all active streaming sessions
	for name, cancelFunc := range e.streamingActive {
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [streaming] Detaching from output streaming (process continues in background)\n", timestamp, name)
		}
		cancelFunc()
	}

	// Clear the tracking map
	e.streamingActive = make(map[string]context.CancelFunc)

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [seqr] [streaming] Detached from all streaming sessions. Processes continue running in background.\n", timestamp)
		fmt.Printf("[%s] [seqr] [streaming] Use 'seqr --kill' to terminate background processes when needed.\n", timestamp)
		os.Stdout.Sync()
	}
}

// GetActiveStreamingProcesses returns the names of processes with active streaming
func (e *Executor) GetActiveStreamingProcesses() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.streamingActive))
	for name := range e.streamingActive {
		names = append(names, name)
	}
	return names
}

// terminateProcessGracefully attempts to terminate a process gracefully with SIGTERM before falling back to SIGKILL
func (e *Executor) terminateProcessGracefully(process *os.Process, name string) {
	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [%s] [process] Terminating process group (PID %d) gracefully...\n", timestamp, name, process.Pid)
	}

	// Try to kill the entire process group first
	if err := e.killProcessGroup(process.Pid, true); err != nil {
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Failed to terminate process group (PID %d): %v, falling back to single process termination\n", timestamp, name, process.Pid, err)
		}
		// Fall back to single process termination
		e.terminateProcessGracefullyFallback(process, name)
		return
	}

	// Wait up to 5 seconds for graceful shutdown
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		// Process exited gracefully
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			if err != nil {
				fmt.Printf("[%s] [%s] [process] Process group exited gracefully with error (PID %d): %v\n", timestamp, name, process.Pid, err)
			} else {
				fmt.Printf("[%s] [%s] [process] Process group exited gracefully (PID %d)\n", timestamp, name, process.Pid)
			}
		}
	case <-time.After(5 * time.Second):
		// Timeout, force kill with SIGKILL
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Graceful shutdown timeout (PID %d), using force kill on process group\n", timestamp, name, process.Pid)
		}
		e.forceKillProcessGroupWithTimeout(process, name, done)
	}
}

// forceKillProcess immediately terminates a process with SIGKILL
func (e *Executor) forceKillProcess(process *os.Process, name string) {
	if err := process.Kill(); err != nil {
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Failed to send SIGKILL (PID %d): %v\n", timestamp, name, process.Pid, err)
		}
		return
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [%s] [process] Sent SIGKILL (PID %d)\n", timestamp, name, process.Pid)
	}

	// Wait for process to exit after SIGKILL
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			if err != nil {
				fmt.Printf("[%s] [%s] [process] Process terminated with SIGKILL (PID %d): %v\n", timestamp, name, process.Pid, err)
			} else {
				fmt.Printf("[%s] [%s] [process] Process terminated with SIGKILL (PID %d)\n", timestamp, name, process.Pid)
			}
		}
	case <-time.After(3 * time.Second):
		// Even SIGKILL timed out, log warning
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Warning: SIGKILL timeout (PID %d) - process may be in uninterruptible state\n", timestamp, name, process.Pid)
		}
	}
}

// forceKillProcessWithTimeout sends SIGKILL and waits for process termination with timeout
func (e *Executor) forceKillProcessWithTimeout(process *os.Process, name string, gracefulDone chan error) {
	if err := process.Kill(); err != nil {
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Failed to send SIGKILL (PID %d): %v\n", timestamp, name, process.Pid, err)
		}
		return
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [%s] [process] Sent SIGKILL (PID %d), waiting for termination...\n", timestamp, name, process.Pid)
	}

	// Wait for either the graceful wait to complete or SIGKILL to take effect
	select {
	case err := <-gracefulDone:
		// Process finally exited (either from SIGTERM or SIGKILL)
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			if err != nil {
				fmt.Printf("[%s] [%s] [process] Process terminated after SIGKILL (PID %d): %v\n", timestamp, name, process.Pid, err)
			} else {
				fmt.Printf("[%s] [%s] [process] Process terminated after SIGKILL (PID %d)\n", timestamp, name, process.Pid)
			}
		}
	case <-time.After(3 * time.Second):
		// Even SIGKILL timed out, log warning
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Warning: SIGKILL timeout (PID %d) - process may be in uninterruptible state\n", timestamp, name, process.Pid)
		}
	}
}

// configureProcessGroup sets up process group for proper child process cleanup
func (e *Executor) configureProcessGroup(cmd *exec.Cmd) {
	// The actual implementation is in platform-specific files
	e.configureProcessGroupPlatform(cmd)
}

// killProcessGroup kills an entire process group using platform-specific methods
func (e *Executor) killProcessGroup(pid int, graceful bool) error {
	// The actual implementation is in platform-specific files
	return e.killProcessGroupPlatform(pid, graceful)
}

// terminateProcessGracefullyFallback falls back to single process termination when process group termination fails
func (e *Executor) terminateProcessGracefullyFallback(process *os.Process, name string) {
	if runtime.GOOS == "windows" {
		// On Windows, we don't have SIGTERM, so we'll just force kill
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Windows detected, using force termination (PID %d)\n", timestamp, name, process.Pid)
		}
		e.forceKillProcess(process, name)
		return
	}

	// Send SIGTERM for graceful shutdown on Unix-like systems
	if err := process.Signal(syscall.SIGTERM); err != nil {
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Failed to send SIGTERM (PID %d): %v, using force kill\n", timestamp, name, process.Pid, err)
		}
		e.forceKillProcess(process, name)
		return
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [%s] [process] Sent SIGTERM (PID %d), waiting for graceful shutdown...\n", timestamp, name, process.Pid)
	}

	// Wait up to 5 seconds for graceful shutdown
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		// Process exited gracefully
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			if err != nil {
				fmt.Printf("[%s] [%s] [process] Process exited gracefully with error (PID %d): %v\n", timestamp, name, process.Pid, err)
			} else {
				fmt.Printf("[%s] [%s] [process] Process exited gracefully (PID %d)\n", timestamp, name, process.Pid)
			}
		}
	case <-time.After(5 * time.Second):
		// Timeout, force kill with SIGKILL
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Graceful shutdown timeout (PID %d), using force kill with SIGKILL\n", timestamp, name, process.Pid)
		}
		e.forceKillProcessWithTimeout(process, name, done)
	}
}

// forceKillProcessGroupWithTimeout sends force kill signal to process group and waits for termination with timeout
func (e *Executor) forceKillProcessGroupWithTimeout(process *os.Process, name string, gracefulDone chan error) {
	// Try to force kill the entire process group
	if err := e.killProcessGroup(process.Pid, false); err != nil {
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Failed to force kill process group (PID %d): %v, falling back to single process kill\n", timestamp, name, process.Pid, err)
		}
		// Fall back to single process force kill
		e.forceKillProcessWithTimeout(process, name, gracefulDone)
		return
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [%s] [process] Sent force kill to process group (PID %d), waiting for termination...\n", timestamp, name, process.Pid)
	}

	// Wait for either the graceful wait to complete or force kill to take effect
	select {
	case err := <-gracefulDone:
		// Process finally exited (either from graceful or force kill)
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			if err != nil {
				fmt.Printf("[%s] [%s] [process] Process group terminated after force kill (PID %d): %v\n", timestamp, name, process.Pid, err)
			} else {
				fmt.Printf("[%s] [%s] [process] Process group terminated after force kill (PID %d)\n", timestamp, name, process.Pid)
			}
		}
	case <-time.After(3 * time.Second):
		// Even force kill timed out, log warning
		if e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Warning: Force kill timeout on process group (PID %d) - processes may be in uninterruptible state\n", timestamp, name, process.Pid)
		}
	}
}

// handleStatusChanges processes status change notifications from the monitor
func (e *Executor) handleStatusChanges(ctx context.Context) {
	statusChanges := e.monitor.GetStatusChanges()

	for {
		select {
		case <-ctx.Done():
			return
		case change := <-statusChanges:
			// Handle the status change
			e.processStatusChange(change)
		}
	}
}

// processStatusChange processes a single status change notification
func (e *Executor) processStatusChange(change ProcessStatusChange) {
	// For now, we mainly log unexpected terminations
	// The monitor already handles logging, but we could add additional logic here
	// such as restarting processes, sending alerts, etc.

	if change.Unexpected && (change.NewStatus == ProcessStatusCrashed || change.NewStatus == ProcessStatusExited) {
		// This is an unexpected termination - the monitor already logged it
		// We could add additional handling here if needed, such as:
		// - Attempting to restart the process
		// - Sending notifications to external systems
		// - Updating execution status

		// For now, we'll just ensure the process is cleaned up from our tracking
		e.mu.Lock()
		// Remove from our internal processes map if it's still there
		for name, cmd := range e.processes {
			if cmd.Process != nil && cmd.Process.Pid == change.PID {
				delete(e.processes, name)
				break
			}
		}
		e.mu.Unlock()
	}
}

// GetProcessMonitor returns the process monitor for external access
func (e *Executor) GetProcessMonitor() *ProcessMonitor {
	return e.monitor
}

// groupCommandsByConcurrency groups commands into execution groups based on concurrent flag
func (e *Executor) groupCommandsByConcurrency(commands []config.Command) [][]config.Command {
	var groups [][]config.Command
	var currentConcurrentGroup []config.Command

	for _, cmd := range commands {
		if cmd.Concurrent {
			// Add to current concurrent group
			currentConcurrentGroup = append(currentConcurrentGroup, cmd)
		} else {
			// Non-concurrent command - flush any pending concurrent group first
			if len(currentConcurrentGroup) > 0 {
				groups = append(groups, currentConcurrentGroup)
				currentConcurrentGroup = nil
			}
			// Add as single command group
			groups = append(groups, []config.Command{cmd})
		}
	}

	// Flush any remaining concurrent group
	if len(currentConcurrentGroup) > 0 {
		groups = append(groups, currentConcurrentGroup)
	}

	return groups
}

// executeConcurrentCommands executes a group of commands concurrently
func (e *Executor) executeConcurrentCommands(ctx context.Context, commands []config.Command, commandIndex *int) error {
	if len(commands) == 0 {
		return nil
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [seqr] [concurrent] Starting %d commands concurrently\n", timestamp, len(commands))
		for _, cmd := range commands {
			fmt.Printf("[%s] [seqr] [concurrent] - %s\n", timestamp, cmd.Name)
		}
		os.Stdout.Sync()
	}

	// Channel to collect results from concurrent executions
	type concurrentResult struct {
		index  int
		result ExecutionResult
		err    error
	}

	resultChan := make(chan concurrentResult, len(commands))
	var wg sync.WaitGroup

	// Start all concurrent commands
	for i, cmd := range commands {
		wg.Add(1)
		go func(cmdIndex int, command config.Command) {
			defer wg.Done()

			// Report command start
			currentIndex := *commandIndex + cmdIndex
			e.reporter.ReportCommandStart(command.Name, currentIndex)

			// Execute the command
			result, err := e.executeCommand(ctx, command)

			// Send result through channel
			resultChan <- concurrentResult{
				index:  cmdIndex,
				result: result,
				err:    err,
			}
		}(i, cmd)
	}

	// Wait for all commands to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and handle errors
	results := make([]ExecutionResult, len(commands))
	var firstError error

	for result := range resultChan {
		results[result.index] = result.result
		e.addResult(result.result)

		currentIndex := *commandIndex + result.index
		if result.err != nil {
			e.reporter.ReportCommandFailure(result.result, currentIndex)
			if firstError == nil {
				firstError = result.err
			}
		} else {
			e.reporter.ReportCommandSuccess(result.result, currentIndex)
		}
	}

	// Update command index
	*commandIndex += len(commands)
	e.updateCompletedCount(*commandIndex)

	// If any command failed, return the first error
	if firstError != nil {
		e.updateState(StateFailed, firstError.Error())
		return firstError
	}

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [seqr] [concurrent] All %d concurrent commands completed successfully\n", timestamp, len(commands))
		os.Stdout.Sync()
	}

	return nil
}
