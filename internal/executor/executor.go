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

func (e *Executor) executeKeepAlive(execCmd *exec.Cmd, result ExecutionResult, name string) (ExecutionResult, error) {
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

func (e *Executor) monitorProcess(name string, cmd *exec.Cmd) {
	err := cmd.Wait()

	e.mu.Lock()
	delete(e.processes, name)
	e.mu.Unlock()

	if e.verbose {
		if err != nil {
			fmt.Printf("Process '%s' exited with error: %v\n", name, err)
		} else {
			fmt.Printf("Process '%s' exited cleanly\n", name)
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
				fmt.Printf("Terminating process '%s' (PID %d)\n", name, cmd.Process.Pid)
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
