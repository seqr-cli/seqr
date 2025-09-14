package executor

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// onceExecutor implements synchronous command execution
type onceExecutor struct{}

// NewOnceExecutor creates a new once executor
func NewOnceExecutor() OnceExecutor {
	return &onceExecutor{}
}

// Execute runs a command synchronously and waits for completion
func (oe *onceExecutor) Execute(ctx context.Context, cmd config.Command, opts ProcessManagerOptions) (ExecutionResult, error) {
	result := ExecutionResult{
		Command:   cmd,
		StartTime: time.Now(),
	}

	// Prepare the command
	execCmd := oe.prepareCommand(ctx, cmd, opts)

	// Execute and wait for completion
	output, err := execCmd.CombinedOutput()

	// Record completion time and duration
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Output = strings.TrimSpace(string(output))

	if err != nil {
		return oe.handleCommandFailure(cmd, execCmd, result, err, string(output))
	}

	// Command succeeded
	result.Success = true
	result.ExitCode = 0

	return result, nil
}

// prepareCommand creates and configures an exec.Cmd for the given command
func (oe *onceExecutor) prepareCommand(ctx context.Context, cmd config.Command, opts ProcessManagerOptions) *exec.Cmd {
	execCmd := exec.CommandContext(ctx, cmd.Command, cmd.Args...)

	// Set working directory
	oe.setWorkingDirectory(execCmd, cmd, opts)

	// Set environment variables
	oe.setEnvironmentVariables(execCmd, cmd)

	return execCmd
}

// setWorkingDirectory sets the working directory for the command
func (oe *onceExecutor) setWorkingDirectory(execCmd *exec.Cmd, cmd config.Command, opts ProcessManagerOptions) {
	if cmd.WorkDir != "" {
		execCmd.Dir = cmd.WorkDir
	} else if opts.WorkingDir != "" {
		execCmd.Dir = opts.WorkingDir
	}
}

// setEnvironmentVariables sets environment variables for the command
func (oe *onceExecutor) setEnvironmentVariables(execCmd *exec.Cmd, cmd config.Command) {
	if len(cmd.Env) > 0 {
		execCmd.Env = os.Environ()
		for key, value := range cmd.Env {
			execCmd.Env = append(execCmd.Env, key+"="+value)
		}
	}
}

// handleCommandFailure creates a failure result with detailed error information
func (oe *onceExecutor) handleCommandFailure(cmd config.Command, execCmd *exec.Cmd, result ExecutionResult, err error, output string) (ExecutionResult, error) {
	result.Success = false
	result.Error = err.Error()
	result.ExitCode = oe.extractExitCode(err)
	result.ErrorDetail = createErrorDetail(cmd, execCmd, err, output, output)

	enhancedErr := oe.createEnhancedError(cmd, execCmd, err, result.ExitCode)

	return result, enhancedErr
}

// extractExitCode extracts the exit code from an error
func (oe *onceExecutor) extractExitCode(err error) int {
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	return -1
}

// createEnhancedError creates a detailed error message for command failures
func (oe *onceExecutor) createEnhancedError(cmd config.Command, execCmd *exec.Cmd, err error, exitCode int) error {
	cmdLine := buildCommandLine(cmd)
	workDir := execCmd.Dir
	if workDir == "" {
		workDir = "."
	}

	return &CommandExecutionError{
		CommandName:   cmd.Name,
		CommandLine:   cmdLine,
		WorkingDir:    workDir,
		ExitCode:      exitCode,
		OriginalError: err,
	}
}
