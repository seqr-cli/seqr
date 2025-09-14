package executor

import "fmt"

// CommandExecutionError represents an error that occurred during command execution
type CommandExecutionError struct {
	CommandName   string
	CommandLine   string
	WorkingDir    string
	ExitCode      int
	OriginalError error
}

// Error implements the error interface
func (e *CommandExecutionError) Error() string {
	return fmt.Sprintf("command '%s' failed: %v\n  Command: %s\n  Working Directory: %s\n  Exit Code: %d",
		e.CommandName, e.OriginalError, e.CommandLine, e.WorkingDir, e.ExitCode)
}

// Unwrap returns the original error for error unwrapping
func (e *CommandExecutionError) Unwrap() error {
	return e.OriginalError
}

// KeepAliveStartupError represents an error that occurred when starting a keepAlive process
type KeepAliveStartupError struct {
	CommandName   string
	CommandLine   string
	WorkingDir    string
	OriginalError error
}

// Error implements the error interface
func (e *KeepAliveStartupError) Error() string {
	return fmt.Sprintf("failed to start keepAlive command '%s': %v\n  Command: %s\n  Working Directory: %s",
		e.CommandName, e.OriginalError, e.CommandLine, e.WorkingDir)
}

// Unwrap returns the original error for error unwrapping
func (e *KeepAliveStartupError) Unwrap() error {
	return e.OriginalError
}

// ProcessNotFoundError represents an error when trying to operate on a non-existent process
type ProcessNotFoundError struct {
	ProcessName string
}

// Error implements the error interface
func (e *ProcessNotFoundError) Error() string {
	return fmt.Sprintf("process '%s' not found", e.ProcessName)
}

// ProcessTerminationError represents an error that occurred during process termination
type ProcessTerminationError struct {
	ProcessName   string
	PID           int
	OriginalError error
}

// Error implements the error interface
func (e *ProcessTerminationError) Error() string {
	return fmt.Sprintf("failed to terminate process '%s' (PID %d): %v",
		e.ProcessName, e.PID, e.OriginalError)
}

// Unwrap returns the original error for error unwrapping
func (e *ProcessTerminationError) Unwrap() error {
	return e.OriginalError
}
