package executor

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// ExecutionState represents the current state of command execution
type ExecutionState int

const (
	// StateReady indicates the executor is ready to execute the next command
	StateReady ExecutionState = iota
	// StateRunning indicates a command is currently executing
	StateRunning
	// StateSuccess indicates the last command completed successfully
	StateSuccess
	// StateFailed indicates a command failed and execution has stopped
	StateFailed
)

// String returns a human-readable representation of the execution state
func (s ExecutionState) String() string {
	switch s {
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StateSuccess:
		return "success"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ExecutionResult contains the result of a command execution
type ExecutionResult struct {
	Command     config.Command `json:"command"`
	Success     bool           `json:"success"`
	ExitCode    int            `json:"exitCode"`
	Output      string         `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	ErrorDetail *ErrorDetail   `json:"errorDetail,omitempty"`
	StartTime   time.Time      `json:"startTime"`
	EndTime     time.Time      `json:"endTime"`
	Duration    time.Duration  `json:"duration"`
}

// ErrorDetail provides detailed context about command execution failures
type ErrorDetail struct {
	Type        ErrorType `json:"type"`
	Message     string    `json:"message"`
	CommandLine string    `json:"commandLine"`
	WorkingDir  string    `json:"workingDir,omitempty"`
	Environment []string  `json:"environment,omitempty"`
	Stderr      string    `json:"stderr,omitempty"`
	Stdout      string    `json:"stdout,omitempty"`
	SystemError string    `json:"systemError,omitempty"`
}

// ErrorType categorizes different types of execution errors
type ErrorType string

const (
	ErrorTypeCommandNotFound  ErrorType = "command_not_found"
	ErrorTypePermissionDenied ErrorType = "permission_denied"
	ErrorTypeNonZeroExit      ErrorType = "non_zero_exit"
	ErrorTypeTimeout          ErrorType = "timeout"
	ErrorTypeContextCancelled ErrorType = "context_cancelled"
	ErrorTypeStartupFailure   ErrorType = "startup_failure"
	ErrorTypeSystemError      ErrorType = "system_error"
	ErrorTypeUnsupportedMode  ErrorType = "unsupported_mode"
)

// ExecutionStatus provides current status information about the execution
type ExecutionStatus struct {
	State          ExecutionState    `json:"state"`
	CurrentCommand *config.Command   `json:"currentCommand,omitempty"`
	CompletedCount int               `json:"completedCount"`
	TotalCount     int               `json:"totalCount"`
	Results        []ExecutionResult `json:"results"`
	LastError      string            `json:"lastError,omitempty"`
}

// CommandExecutor defines the core interface for command execution
type CommandExecutor interface {
	// Execute runs all commands in the provided configuration sequentially
	// Returns when all commands complete successfully or when any command fails
	Execute(ctx context.Context, cfg *config.Config) error

	// GetStatus returns the current execution status
	GetStatus() ExecutionStatus

	// Stop gracefully stops execution after the current command completes
	Stop()
}

// ProcessStatusProvider defines the interface for process status reporting
type ProcessStatusProvider interface {
	// ReportProcessStatus reports the current status of all active processes
	ReportProcessStatus()

	// ReportHealthStatus reports the health status of all monitored processes
	ReportHealthStatus()

	// GetActiveProcesses returns information about currently running processes
	GetActiveProcesses() map[string]ProcessInfo

	// GetProcessHealth returns health status for all monitored processes
	GetProcessHealth() map[string]ProcessHealth
}

// HealthMonitorController defines the interface for health monitoring control
type HealthMonitorController interface {
	// StartHealthMonitoring starts background health monitoring
	StartHealthMonitoring(ctx context.Context) error

	// StopHealthMonitoring stops background health monitoring
	StopHealthMonitoring() error
}

// Executor combines all executor interfaces for backward compatibility
type Executor interface {
	CommandExecutor
	ProcessStatusProvider
	HealthMonitorController
}

// ExecutorOptions contains configuration options for the executor
type ExecutorOptions struct {
	// Verbose enables detailed output logging
	Verbose bool
	// WorkingDir sets the default working directory for commands
	WorkingDir string
	// Timeout sets the maximum time to wait for a command to complete
	Timeout time.Duration
	// Reporter handles execution output (optional, defaults to console reporter)
	Reporter Reporter
}

// createErrorDetail creates detailed error context for failed commands
func createErrorDetail(cmd config.Command, execCmd *exec.Cmd, err error, stderr, stdout string) *ErrorDetail {
	detail := &ErrorDetail{
		Message:     err.Error(),
		CommandLine: buildCommandLine(cmd),
		Stderr:      stderr,
		Stdout:      stdout,
		SystemError: err.Error(),
	}

	// Set working directory if available
	if execCmd != nil && execCmd.Dir != "" {
		detail.WorkingDir = execCmd.Dir
	} else if cmd.WorkDir != "" {
		detail.WorkingDir = cmd.WorkDir
	}

	// Set environment variables if available
	if execCmd != nil && len(execCmd.Env) > 0 {
		detail.Environment = execCmd.Env
	}

	// Categorize error type
	detail.Type = categorizeError(err)

	return detail
}

// buildCommandLine reconstructs the full command line for debugging
func buildCommandLine(cmd config.Command) string {
	if len(cmd.Args) == 0 {
		return cmd.Command
	}

	cmdLine := cmd.Command
	for _, arg := range cmd.Args {
		// Simple quoting for arguments with spaces
		if strings.Contains(arg, " ") {
			cmdLine += " \"" + arg + "\""
		} else {
			cmdLine += " " + arg
		}
	}
	return cmdLine
}

// categorizeError determines the error type based on the error details
func categorizeError(err error) ErrorType {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Check for context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return ErrorTypeContextCancelled
	}

	// Check for exec.ExitError (command ran but failed)
	if exitError, ok := err.(*exec.ExitError); ok {
		_ = exitError // Use the variable to avoid unused warning
		return ErrorTypeNonZeroExit
	}

	// Check for common system errors
	if strings.Contains(errStr, "executable file not found") ||
		strings.Contains(errStr, "no such file or directory") {
		return ErrorTypeCommandNotFound
	}

	if strings.Contains(errStr, "permission denied") {
		return ErrorTypePermissionDenied
	}

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return ErrorTypeTimeout
	}

	// Default to system error
	return ErrorTypeSystemError
}
