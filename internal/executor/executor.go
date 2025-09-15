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
	mu        sync.RWMutex
	status    ExecutionStatus
	verbose   bool
	stopped   bool
	processes map[string]*exec.Cmd
	reporter  Reporter
	tracker   *ProcessTracker
}

func NewExecutor(verbose bool) *Executor {
	return &Executor{
		verbose:   verbose,
		processes: make(map[string]*exec.Cmd),
		reporter:  NewConsoleReporter(os.Stdout, verbose),
		tracker:   NewProcessTracker(),
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

	e.reporter.ReportStart(len(cfg.Commands))

	for i, cmd := range cfg.Commands {
		if e.isStopped() {
			return fmt.Errorf("execution stopped")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		e.updateCurrentCommand(&cmd)
		e.reporter.ReportCommandStart(cmd.Name, i)

		result, err := e.executeCommand(ctx, cmd)
		e.addResult(result)

		if err != nil {
			e.reporter.ReportCommandFailure(result, i)
			e.updateState(StateFailed, err.Error())
			return err
		}

		e.reporter.ReportCommandSuccess(result, i)
		e.updateCompletedCount(i + 1)
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
		e.streamOutput(stdoutPipe, &outputBuilder, result.Command.Name, "stdout")
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
		e.streamOutput(stderrPipe, &outputBuilder, result.Command.Name, "stderr")
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

func (e *Executor) streamOutput(pipe io.ReadCloser, outputBuilder *strings.Builder, commandName, streamType string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[%s] [%s] ❌ Streaming panic recovered: %v\n",
				time.Now().Format("15:04:05.000"), commandName, r)
			os.Stdout.Sync()
		}
	}()

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		timestamp := time.Now().Format("15:04:05.000")

		// Write to console with timestamp and command identification
		// Use different visual indicators for stdout vs stderr
		if streamType == "stderr" {
			fmt.Printf("[%s] [%s] ❌ %s\n", timestamp, commandName, line)
		} else {
			fmt.Printf("[%s] [%s] ✓  %s\n", timestamp, commandName, line)
		}

		// Ensure immediate output by flushing stdout
		os.Stdout.Sync()

		// Also capture for the result output
		outputBuilder.WriteString(line)
		outputBuilder.WriteString("\n")
	}

	if err := scanner.Err(); err != nil && !strings.Contains(err.Error(), "file already closed") {
		fmt.Printf("[%s] [%s] ❌ Error reading %s: %v\n",
			time.Now().Format("15:04:05.000"), commandName, streamType, err)
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

	// Start streaming output in background goroutines with proper lifecycle management
	var streamWg sync.WaitGroup

	streamWg.Add(2)
	go func() {
		defer streamWg.Done()
		e.streamOutputContinuous(stdoutPipe, name, "stdout")
	}()

	go func() {
		defer streamWg.Done()
		e.streamOutputContinuous(stderrPipe, name, "stderr")
	}()

	// Monitor the process and streaming lifecycle
	go func() {
		e.monitorProcess(name, execCmd)
		// Wait for streaming to complete after process ends
		streamWg.Wait()
	}()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true
	result.ExitCode = 0
	result.Output = fmt.Sprintf("started with PID %d", execCmd.Process.Pid)
	return result, nil
}

func (e *Executor) streamOutputContinuous(pipe io.ReadCloser, commandName, streamType string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[%s] [%s] ❌ Streaming panic recovered: %v\n",
				time.Now().Format("15:04:05.000"), commandName, r)
			os.Stdout.Sync()
		}
		// Ensure pipe is closed
		pipe.Close()
	}()

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

		// Write to console with timestamp and command identification
		// Use different visual indicators for stdout vs stderr
		if streamType == "stderr" {
			fmt.Printf("[%s] [%s] ❌ %s\n", timestamp, commandName, line)
		} else {
			fmt.Printf("[%s] [%s] ✓  %s\n", timestamp, commandName, line)
		}

		// Ensure immediate output by flushing stdout for real-time streaming
		os.Stdout.Sync()
	}

	if err := scanner.Err(); err != nil && !e.isStopped() && !strings.Contains(err.Error(), "file already closed") {
		fmt.Printf("[%s] [%s] ❌ Error reading %s: %v\n",
			time.Now().Format("15:04:05.000"), commandName, streamType, err)
		os.Stdout.Sync()
	}
}

func (e *Executor) monitorProcess(name string, cmd *exec.Cmd) {
	err := cmd.Wait()

	e.mu.Lock()
	delete(e.processes, name)
	e.mu.Unlock()

	// Remove from process tracker
	if cmd.Process != nil {
		if trackErr := e.tracker.RemoveProcess(cmd.Process.Pid); trackErr != nil && e.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [process] Warning: Failed to untrack process: %v\n", timestamp, name, trackErr)
		}
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
