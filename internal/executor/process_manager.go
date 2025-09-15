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
	// Try to kill the entire process group first
	if err := pm.killProcessGroup(process.Pid, true); err != nil {
		// Fall back to single process termination
		return pm.terminateGracefullyFallback(process, info)
	}

	// Wait up to 10 seconds for graceful shutdown by checking if process is removed from tracking
	return pm.waitForProcessRemoval(process.Pid, 10*time.Second, func() error {
		return pm.forceKillProcessGroup(process, info)
	})
}

// forceKill forcefully terminates a process with SIGKILL
func (pm *ProcessManager) forceKill(process *os.Process, info *ProcessInfo) error {
	// Try to force kill the entire process group first
	if err := pm.killProcessGroup(process.Pid, false); err != nil {
		// Fall back to single process force kill
		return pm.forceKillFallback(process, info)
	}

	// Wait for process to be removed from tracking
	return pm.waitForProcessRemoval(process.Pid, 5*time.Second, func() error {
		return fmt.Errorf("timeout waiting for process %d to terminate", process.Pid)
	})
}

// GetProcessCount returns the number of currently tracked processes
func (pm *ProcessManager) GetProcessCount() (int, error) {
	// Clean up dead processes first
	if err := pm.tracker.CleanupDeadProcesses(); err != nil {
		return 0, fmt.Errorf("failed to cleanup dead processes: %w", err)
	}

	return pm.tracker.GetRunningProcessCount(), nil
}

// killProcessGroup kills an entire process group using platform-specific methods
func (pm *ProcessManager) killProcessGroup(pid int, graceful bool) error {
	// The actual implementation is in platform-specific files
	return pm.killProcessGroupPlatform(pid, graceful)
}

// terminateGracefullyFallback falls back to single process termination when process group termination fails
func (pm *ProcessManager) terminateGracefullyFallback(process *os.Process, info *ProcessInfo) error {
	if runtime.GOOS == "windows" {
		// On Windows, we don't have SIGTERM, so we'll just force kill
		return pm.forceKillFallback(process, info)
	}

	// Send SIGTERM for graceful shutdown on Unix-like systems
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait up to 10 seconds for graceful shutdown by checking if process is removed from tracking
	return pm.waitForProcessRemoval(process.Pid, 10*time.Second, func() error {
		return pm.forceKillFallback(process, info)
	})
}

// forceKillFallback forcefully terminates a single process with SIGKILL
func (pm *ProcessManager) forceKillFallback(process *os.Process, info *ProcessInfo) error {
	// Send SIGKILL for immediate termination
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to send SIGKILL: %w", err)
	}

	// Wait for process to be removed from tracking
	return pm.waitForProcessRemoval(process.Pid, 5*time.Second, func() error {
		return fmt.Errorf("timeout waiting for process %d to terminate", process.Pid)
	})
}

// forceKillProcessGroup forcefully terminates a process group
func (pm *ProcessManager) forceKillProcessGroup(process *os.Process, info *ProcessInfo) error {
	// Try to force kill the entire process group first
	if err := pm.killProcessGroup(process.Pid, false); err != nil {
		// Fall back to single process force kill
		return pm.forceKillFallback(process, info)
	}

	// Wait for process to be removed from tracking
	return pm.waitForProcessRemoval(process.Pid, 5*time.Second, func() error {
		return fmt.Errorf("timeout waiting for process %d to terminate", process.Pid)
	})
}

// waitForProcessRemoval waits for a process to be removed from tracking
func (pm *ProcessManager) waitForProcessRemoval(pid int, timeout time.Duration, onTimeout func() error) error {
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			return onTimeout()
		case <-ticker.C:
			if _, exists := pm.tracker.GetProcess(pid); !exists {
				return nil
			}
		}
	}
}
