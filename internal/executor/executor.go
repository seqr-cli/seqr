package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
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
}

func NewExecutor(verbose bool) *Executor {
	return &Executor{
		verbose:   verbose,
		processes: make(map[string]*exec.Cmd),
		reporter:  NewConsoleReporter(os.Stdout, verbose),
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

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutput(stdoutPipe, &outputBuilder, result.Command.Name, "stdout")
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
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

		// Also capture for the result output
		outputBuilder.WriteString(line)
		outputBuilder.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("[%s] [%s] ❌ Error reading %s: %v\n",
			time.Now().Format("15:04:05.000"), commandName, streamType, err)
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

	// Start streaming output in background goroutines
	go e.streamOutputContinuous(stdoutPipe, name, "stdout")
	go e.streamOutputContinuous(stderrPipe, name, "stderr")

	// Monitor the process
	go e.monitorProcess(name, execCmd)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true
	result.ExitCode = 0
	result.Output = fmt.Sprintf("started with PID %d", execCmd.Process.Pid)
	return result, nil
}

func (e *Executor) streamOutputContinuous(pipe io.ReadCloser, commandName, streamType string) {
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
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("[%s] [%s] ❌ Error reading %s: %v\n",
			time.Now().Format("15:04:05.000"), commandName, streamType, err)
	}
}

func (e *Executor) monitorProcess(name string, cmd *exec.Cmd) {
	err := cmd.Wait()

	e.mu.Lock()
	delete(e.processes, name)
	e.mu.Unlock()

	if e.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		if err != nil {
			fmt.Printf("[%s] [%s] [process] Process exited with error: %v\n", timestamp, name, err)
		} else {
			fmt.Printf("[%s] [%s] [process] Process exited cleanly\n", timestamp, name)
		}
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
				fmt.Printf("[%s] [%s] [process] Terminating process (PID %d)\n", timestamp, name, cmd.Process.Pid)
			}
			cmd.Process.Kill()
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
