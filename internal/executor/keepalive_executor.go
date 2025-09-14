package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// keepAliveExecutor implements asynchronous command execution for long-running processes
type keepAliveExecutor struct{}

// NewKeepAliveExecutor creates a new keepAlive executor
func NewKeepAliveExecutor() KeepAliveExecutor {
	return &keepAliveExecutor{}
}

// Execute starts a command asynchronously and returns immediately
func (kae *keepAliveExecutor) Execute(ctx context.Context, cmd config.Command, opts ProcessManagerOptions) (ExecutionResult, *managedProcess, error) {
	result := ExecutionResult{
		Command:   cmd,
		StartTime: time.Now(),
	}

	// Prepare the command
	execCmd := kae.prepareCommand(ctx, cmd, opts)

	// Start the process (non-blocking)
	err := execCmd.Start()

	// Record completion time and duration (for the start operation)
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		return kae.handleStartupFailure(cmd, execCmd, result, err)
	}

	// Create managed process for tracking
	managedProc := &managedProcess{
		cmd:       execCmd,
		name:      cmd.Name,
		startTime: result.StartTime,
		command:   cmd,
		done:      make(chan error, 1),
	}

	// Command started successfully
	result.Success = true
	result.ExitCode = 0
	result.Output = fmt.Sprintf("keepAlive process started with PID %d", execCmd.Process.Pid)

	return result, managedProc, nil
}

// prepareCommand creates and configures an exec.Cmd for the given command
func (kae *keepAliveExecutor) prepareCommand(ctx context.Context, cmd config.Command, opts ProcessManagerOptions) *exec.Cmd {
	execCmd := exec.CommandContext(ctx, cmd.Command, cmd.Args...)

	// Set working directory
	kae.setWorkingDirectory(execCmd, cmd, opts)

	// Set environment variables
	kae.setEnvironmentVariables(execCmd, cmd)

	return execCmd
}

// setWorkingDirectory sets the working directory for the command
func (kae *keepAliveExecutor) setWorkingDirectory(execCmd *exec.Cmd, cmd config.Command, opts ProcessManagerOptions) {
	if cmd.WorkDir != "" {
		execCmd.Dir = cmd.WorkDir
	} else if opts.WorkingDir != "" {
		execCmd.Dir = opts.WorkingDir
	}
}

// setEnvironmentVariables sets environment variables for the command
func (kae *keepAliveExecutor) setEnvironmentVariables(execCmd *exec.Cmd, cmd config.Command) {
	if len(cmd.Env) > 0 {
		execCmd.Env = os.Environ()
		for key, value := range cmd.Env {
			execCmd.Env = append(execCmd.Env, key+"="+value)
		}
	}
}

// handleStartupFailure creates a failure result for commands that fail to start
func (kae *keepAliveExecutor) handleStartupFailure(cmd config.Command, execCmd *exec.Cmd, result ExecutionResult, err error) (ExecutionResult, *managedProcess, error) {
	result.Success = false
	result.Error = err.Error()
	result.ExitCode = -1
	result.ErrorDetail = createErrorDetail(cmd, execCmd, err, "", "")
	result.ErrorDetail.Type = ErrorTypeStartupFailure

	enhancedErr := kae.createEnhancedStartupError(cmd, execCmd, err)

	return result, nil, enhancedErr
}

// createEnhancedStartupError creates a detailed error message for startup failures
func (kae *keepAliveExecutor) createEnhancedStartupError(cmd config.Command, execCmd *exec.Cmd, err error) error {
	cmdLine := buildCommandLine(cmd)
	workDir := execCmd.Dir
	if workDir == "" {
		workDir = "."
	}

	return &KeepAliveStartupError{
		CommandName:   cmd.Name,
		CommandLine:   cmdLine,
		WorkingDir:    workDir,
		OriginalError: err,
	}
}
