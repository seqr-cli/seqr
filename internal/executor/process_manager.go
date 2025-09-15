package executor

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
)

// ProcessManager handles operations on tracked processes
type ProcessManager struct {
	tracker *ProcessTracker
}

// NewProcessManager creates a new process manager
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		tracker: NewProcessTracker(),
	}
}

// GetAllRunningProcesses returns all currently tracked processes
func (pm *ProcessManager) GetAllRunningProcesses() (map[int]*ProcessInfo, error) {
	// Clean up dead processes first
	if err := pm.tracker.CleanupDeadProcesses(); err != nil {
		return nil, fmt.Errorf("failed to cleanup dead processes: %w", err)
	}

	return pm.tracker.GetAllProcesses(), nil
}

// KillProcess terminates a specific process by PID
func (pm *ProcessManager) KillProcess(pid int, graceful bool) error {
	// Check if process is tracked
	processInfo, exists := pm.tracker.GetProcess(pid)
	if !exists {
		return fmt.Errorf("process with PID %d is not tracked by seqr", pid)
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist, remove from tracking
		pm.tracker.RemoveProcess(pid)
		return fmt.Errorf("process with PID %d not found: %w", pid, err)
	}

	if graceful {
		// Try graceful termination first
		if err := pm.terminateGracefully(process, processInfo); err != nil {
			return fmt.Errorf("failed to terminate process %d gracefully: %w", pid, err)
		}
	} else {
		// Force kill immediately
		if err := pm.forceKill(process, processInfo); err != nil {
			return fmt.Errorf("failed to force kill process %d: %w", pid, err)
		}
	}

	// Remove from tracking
	return pm.tracker.RemoveProcess(pid)
}

// KillAllProcesses terminates all tracked processes
func (pm *ProcessManager) KillAllProcesses(graceful bool) error {
	processes, err := pm.GetAllRunningProcesses()
	if err != nil {
		return fmt.Errorf("failed to get running processes: %w", err)
	}

	if len(processes) == 0 {
		return fmt.Errorf("no seqr processes are currently running")
	}

	var errors []error

	for pid := range processes {
		if err := pm.KillProcess(pid, graceful); err != nil {
			errors = append(errors, fmt.Errorf("failed to kill process %d: %w", pid, err))
		}
	}

	if len(errors) > 0 {
		// Return the first error, but log all of them
		return errors[0]
	}

	return nil
}

// terminateGracefully attempts to terminate a process gracefully with SIGTERM
func (pm *ProcessManager) terminateGracefully(process *os.Process, info *ProcessInfo) error {
	if runtime.GOOS == "windows" {
		// On Windows, we don't have SIGTERM, so we'll just force kill
		return pm.forceKill(process, info)
	}

	// Send SIGTERM for graceful shutdown on Unix-like systems
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait up to 10 seconds for graceful shutdown
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		// Process exited gracefully
		return err
	case <-time.After(10 * time.Second):
		// Timeout, force kill
		return pm.forceKill(process, info)
	}
}

// forceKill forcefully terminates a process with SIGKILL
func (pm *ProcessManager) forceKill(process *os.Process, info *ProcessInfo) error {
	// Send SIGKILL for immediate termination
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to send SIGKILL: %w", err)
	}

	// Wait for process to exit (should be immediate with SIGKILL)
	_, err := process.Wait()
	return err
}

// GetProcessCount returns the number of currently tracked processes
func (pm *ProcessManager) GetProcessCount() (int, error) {
	// Clean up dead processes first
	if err := pm.tracker.CleanupDeadProcesses(); err != nil {
		return 0, fmt.Errorf("failed to cleanup dead processes: %w", err)
	}

	return pm.tracker.GetRunningProcessCount(), nil
}
