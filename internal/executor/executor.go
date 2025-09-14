package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// sequentialExecutor implements the Executor interface with sequential command execution
type sequentialExecutor struct {
	mu             sync.RWMutex
	status         ExecutionStatus
	options        ExecutorOptions
	stopped        bool
	keepAliveProcs map[string]*exec.Cmd // Track keepAlive processes by command name
	reporter       Reporter             // Reporter for execution output
	processManager ProcessManager       // Process manager for advanced process handling
}

// NewExecutor creates a new sequential executor with the provided options
func NewExecutor(opts ExecutorOptions) Executor {
	// Use provided reporter or default to console reporter
	reporter := opts.Reporter
	if reporter == nil {
		reporter = NewConsoleReporter(os.Stdout, opts.Verbose)
	}

	// Create process manager with the same options
	pmOpts := ProcessManagerOptions{
		Verbose:    opts.Verbose,
		WorkingDir: opts.WorkingDir,
		Timeout:    opts.Timeout,
		Reporter:   reporter,
	}
	processManager := NewProcessManager(pmOpts)

	return &sequentialExecutor{
		options:        opts,
		keepAliveProcs: make(map[string]*exec.Cmd),
		reporter:       reporter,
		processManager: processManager,
		status: ExecutionStatus{
			State:   StateReady,
			Results: make([]ExecutionResult, 0),
		},
	}
}

func (e *sequentialExecutor) Execute(ctx context.Context, cfg *config.Config) error {
	if err := e.validateExecutionInput(cfg); err != nil {
		return err
	}

	e.initializeExecution(cfg)
	e.reporter.ReportStart(len(cfg.Commands))

	for i, cmd := range cfg.Commands {
		if err := e.checkExecutionPreconditions(ctx); err != nil {
			return err
		}

		if err := e.executeAndReportCommand(ctx, cmd, i); err != nil {
			e.handleExecutionFailure(err, i+1, len(cfg.Commands))
			return err
		}

		e.updateCompletedCount(i + 1)
	}

	e.completeExecution()
	return nil
}

// GetStatus returns the current execution status
func (e *sequentialExecutor) GetStatus() ExecutionStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a copy to avoid race conditions
	status := e.status
	status.Results = make([]ExecutionResult, len(e.status.Results))
	copy(status.Results, e.status.Results)

	return status
}

func (e *sequentialExecutor) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopped = true

	// Use process manager to terminate processes if available
	if e.processManager != nil {
		e.processManager.TerminateAll()
		e.processManager.StopHealthMonitoring()
	} else {
		// Fallback to legacy termination
		e.terminateAllKeepAliveProcesses()
	}

	e.keepAliveProcs = make(map[string]*exec.Cmd)
}

func (e *sequentialExecutor) terminateAllKeepAliveProcesses() {
	for name, cmd := range e.keepAliveProcs {
		e.terminateKeepAliveProcess(name, cmd)
	}
}

func (e *sequentialExecutor) terminateKeepAliveProcess(name string, cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	if e.options.Verbose {
		fmt.Fprintf(os.Stderr, "Terminating keepAlive process '%s' (PID %d)\n", name, cmd.Process.Pid)
	}
	cmd.Process.Kill()
}

func (e *sequentialExecutor) executeCommand(ctx context.Context, cmd config.Command) (ExecutionResult, error) {
	// Use the process manager for command execution if available
	if e.processManager != nil {
		return e.processManager.ExecuteCommand(ctx, cmd)
	}

	// Fallback to legacy implementation
	result := ExecutionResult{
		Command:   cmd,
		StartTime: time.Now(),
	}

	switch cmd.Mode {
	case config.ModeOnce:
		return e.executeOnceCommand(ctx, cmd, result)
	case config.ModeKeepAlive:
		return e.executeKeepAliveCommand(ctx, cmd, result)
	default:
		return e.handleUnsupportedMode(cmd, result)
	}
}

func (e *sequentialExecutor) executeOnceCommand(ctx context.Context, cmd config.Command, result ExecutionResult) (ExecutionResult, error) {
	execCmd := e.prepareCommand(ctx, cmd)
	output, err := execCmd.CombinedOutput()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Output = strings.TrimSpace(string(output))

	if err != nil {
		return e.handleCommandFailure(cmd, execCmd, result, err, string(output))
	}

	result.Success = true
	result.ExitCode = 0
	return result, nil
}

func (e *sequentialExecutor) executeKeepAliveCommand(ctx context.Context, cmd config.Command, result ExecutionResult) (ExecutionResult, error) {
	execCmd := e.prepareCommand(ctx, cmd)
	err := execCmd.Start()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		return e.handleKeepAliveStartupFailure(cmd, execCmd, result, err)
	}

	e.trackKeepAliveProcess(cmd.Name, execCmd)

	result.Success = true
	result.ExitCode = 0
	result.Output = fmt.Sprintf("keepAlive process started with PID %d", execCmd.Process.Pid)

	return result, nil
}

func (e *sequentialExecutor) monitorKeepAliveProcess(name string, cmd *exec.Cmd) {
	err := cmd.Wait()
	e.untrackKeepAliveProcess(name)
	e.logKeepAliveProcessExit(name, cmd, err)
}

func (e *sequentialExecutor) untrackKeepAliveProcess(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.keepAliveProcs, name)
}

func (e *sequentialExecutor) logKeepAliveProcessExit(name string, cmd *exec.Cmd, err error) {
	if !e.options.Verbose {
		return
	}

	if err != nil {
		e.logKeepAliveProcessFailure(name, cmd, err)
	} else {
		fmt.Fprintf(os.Stderr, "keepAlive process '%s' exited cleanly (PID %d)\n", name, cmd.Process.Pid)
	}
}

func (e *sequentialExecutor) logKeepAliveProcessFailure(name string, cmd *exec.Cmd, err error) {
	exitCode := e.extractExitCode(err)
	errorType := ErrorTypeSystemError
	if _, ok := err.(*exec.ExitError); ok {
		errorType = ErrorTypeNonZeroExit
	}

	fmt.Fprintf(os.Stderr, "keepAlive process '%s' exited unexpectedly:\n", name)
	fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
	fmt.Fprintf(os.Stderr, "  Exit Code: %d\n", exitCode)
	fmt.Fprintf(os.Stderr, "  Error Type: %s\n", errorType)
	fmt.Fprintf(os.Stderr, "  PID: %d\n", cmd.Process.Pid)
}

// Helper methods for thread-safe status updates

func (e *sequentialExecutor) updateCurrentCommand(cmd *config.Command) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.State = StateRunning
	e.status.CurrentCommand = cmd
}

func (e *sequentialExecutor) updateState(state ExecutionState, errorMsg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.State = state
	e.status.CurrentCommand = nil
	e.status.LastError = errorMsg
}

func (e *sequentialExecutor) addResult(result ExecutionResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.Results = append(e.status.Results, result)
}

func (e *sequentialExecutor) updateCompletedCount(count int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.CompletedCount = count
}

func (e *sequentialExecutor) isStopped() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stopped
}

func (e *sequentialExecutor) validateExecutionInput(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	if len(cfg.Commands) == 0 {
		return fmt.Errorf("no commands to execute")
	}
	return nil
}

func (e *sequentialExecutor) initializeExecution(cfg *config.Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = ExecutionStatus{
		State:      StateReady,
		TotalCount: len(cfg.Commands),
		Results:    make([]ExecutionResult, 0, len(cfg.Commands)),
	}
	e.stopped = false
}

func (e *sequentialExecutor) checkExecutionPreconditions(ctx context.Context) error {
	if e.isStopped() {
		return fmt.Errorf("execution stopped by user")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (e *sequentialExecutor) executeAndReportCommand(ctx context.Context, cmd config.Command, index int) error {
	e.updateCurrentCommand(&cmd)
	e.reporter.ReportCommandStart(cmd.Name, index)

	result, err := e.executeCommand(ctx, cmd)
	e.addResult(result)

	if err != nil {
		e.reporter.ReportCommandFailure(result, index)
		return err
	}

	e.reporter.ReportCommandSuccess(result, index)
	return nil
}

func (e *sequentialExecutor) handleExecutionFailure(err error, commandIndex, totalCommands int) {
	status := e.GetStatus()
	failureContext := e.buildFailureContext(commandIndex, totalCommands, status)
	e.updateState(StateFailed, failureContext)

	finalStatus := e.GetStatus()
	e.reporter.ReportExecutionComplete(finalStatus)
	e.reporter.ReportExecutionSummary(finalStatus)
}

func (e *sequentialExecutor) buildFailureContext(commandIndex, totalCommands int, status ExecutionStatus) string {
	context := fmt.Sprintf("Execution stopped at command %d of %d", commandIndex, totalCommands)

	if len(status.Results) == 0 {
		return context
	}

	lastResult := status.Results[len(status.Results)-1]
	if lastResult.ErrorDetail == nil {
		return context
	}

	detail := lastResult.ErrorDetail
	context += fmt.Sprintf("\nError Type: %s", detail.Type)

	if detail.CommandLine != "" {
		context += fmt.Sprintf("\nFull Command: %s", detail.CommandLine)
	}
	if detail.WorkingDir != "" {
		context += fmt.Sprintf("\nWorking Directory: %s", detail.WorkingDir)
	}
	if detail.Stderr != "" {
		context += fmt.Sprintf("\nStderr: %s", detail.Stderr)
	}

	return context
}

func (e *sequentialExecutor) completeExecution() {
	e.updateState(StateSuccess, "")
	status := e.GetStatus()
	e.reporter.ReportExecutionComplete(status)
	e.reporter.ReportExecutionSummary(status)
}

func (e *sequentialExecutor) handleUnsupportedMode(cmd config.Command, result ExecutionResult) (ExecutionResult, error) {
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = false
	result.Error = fmt.Sprintf("unsupported execution mode: %s", cmd.Mode)
	result.ExitCode = -1

	modeErr := fmt.Errorf("unsupported execution mode: %s", cmd.Mode)
	result.ErrorDetail = createErrorDetail(cmd, nil, modeErr, "", "")
	result.ErrorDetail.Type = ErrorTypeUnsupportedMode

	enhancedErr := fmt.Errorf("command '%s' has unsupported execution mode '%s': supported modes are 'once' and 'keepAlive'\n  Command: %s",
		cmd.Name, cmd.Mode, result.ErrorDetail.CommandLine)

	return result, enhancedErr
}

func (e *sequentialExecutor) prepareCommand(ctx context.Context, cmd config.Command) *exec.Cmd {
	execCmd := exec.CommandContext(ctx, cmd.Command, cmd.Args...)
	e.setWorkingDirectory(execCmd, cmd)
	e.setEnvironmentVariables(execCmd, cmd)
	return execCmd
}

func (e *sequentialExecutor) setWorkingDirectory(execCmd *exec.Cmd, cmd config.Command) {
	if cmd.WorkDir != "" {
		execCmd.Dir = cmd.WorkDir
	} else if e.options.WorkingDir != "" {
		execCmd.Dir = e.options.WorkingDir
	}
}

func (e *sequentialExecutor) setEnvironmentVariables(execCmd *exec.Cmd, cmd config.Command) {
	if len(cmd.Env) > 0 {
		execCmd.Env = os.Environ()
		for key, value := range cmd.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}
}

func (e *sequentialExecutor) handleCommandFailure(cmd config.Command, execCmd *exec.Cmd, result ExecutionResult, err error, output string) (ExecutionResult, error) {
	result.Success = false
	result.Error = err.Error()
	result.ExitCode = e.extractExitCode(err)
	result.ErrorDetail = createErrorDetail(cmd, execCmd, err, output, output)

	enhancedErr := fmt.Errorf("command '%s' failed: %w\n  Command: %s\n  Working Directory: %s\n  Exit Code: %d",
		cmd.Name, err, result.ErrorDetail.CommandLine, result.ErrorDetail.WorkingDir, result.ExitCode)

	return result, enhancedErr
}

func (e *sequentialExecutor) extractExitCode(err error) int {
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	return -1
}

func (e *sequentialExecutor) handleKeepAliveStartupFailure(cmd config.Command, execCmd *exec.Cmd, result ExecutionResult, err error) (ExecutionResult, error) {
	result.Success = false
	result.Error = err.Error()
	result.ExitCode = -1
	result.ErrorDetail = createErrorDetail(cmd, execCmd, err, "", "")
	result.ErrorDetail.Type = ErrorTypeStartupFailure

	enhancedErr := fmt.Errorf("failed to start keepAlive command '%s': %w\n  Command: %s\n  Working Directory: %s",
		cmd.Name, err, result.ErrorDetail.CommandLine, result.ErrorDetail.WorkingDir)

	return result, enhancedErr
}

func (e *sequentialExecutor) trackKeepAliveProcess(name string, execCmd *exec.Cmd) {
	e.mu.Lock()
	e.keepAliveProcs[name] = execCmd
	e.mu.Unlock()
	go e.monitorKeepAliveProcess(name, execCmd)
}

// ReportProcessStatus reports the current status of all active processes
func (e *sequentialExecutor) ReportProcessStatus() {
	if e.processManager != nil {
		e.processManager.ReportCurrentStatus()
	}
}

// ReportHealthStatus reports the health status of all monitored processes
func (e *sequentialExecutor) ReportHealthStatus() {
	if e.processManager != nil {
		e.processManager.ReportHealthStatus()
	}
}

// GetActiveProcesses returns information about currently running processes
func (e *sequentialExecutor) GetActiveProcesses() map[string]ProcessInfo {
	if e.processManager != nil {
		return e.processManager.GetActiveProcesses()
	}
	return make(map[string]ProcessInfo)
}

// GetProcessHealth returns health status for all monitored processes
func (e *sequentialExecutor) GetProcessHealth() map[string]ProcessHealth {
	if e.processManager != nil {
		return e.processManager.GetProcessHealth()
	}
	return make(map[string]ProcessHealth)
}

// StartHealthMonitoring starts background health monitoring
func (e *sequentialExecutor) StartHealthMonitoring(ctx context.Context) error {
	if e.processManager != nil {
		return e.processManager.StartHealthMonitoring(ctx)
	}
	return fmt.Errorf("process manager not available")
}

// StopHealthMonitoring stops background health monitoring
func (e *sequentialExecutor) StopHealthMonitoring() error {
	if e.processManager != nil {
		return e.processManager.StopHealthMonitoring()
	}
	return fmt.Errorf("process manager not available")
}
