package executor

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// ProcessStatus represents the current status of a process
type ProcessStatus int

const (
	ProcessStatusRunning ProcessStatus = iota
	ProcessStatusExited
	ProcessStatusCrashed
	ProcessStatusKilled
)

func (s ProcessStatus) String() string {
	switch s {
	case ProcessStatusRunning:
		return "running"
	case ProcessStatusExited:
		return "exited"
	case ProcessStatusCrashed:
		return "crashed"
	case ProcessStatusKilled:
		return "killed"
	default:
		return "unknown"
	}
}

// ProcessStatusChange represents a change in process status
type ProcessStatusChange struct {
	PID        int           `json:"pid"`
	Name       string        `json:"name"`
	OldStatus  ProcessStatus `json:"oldStatus"`
	NewStatus  ProcessStatus `json:"newStatus"`
	ExitCode   int           `json:"exitCode,omitempty"`
	Error      string        `json:"error,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
	Unexpected bool          `json:"unexpected"` // True if the process terminated unexpectedly
}

// ProcessMonitor monitors running processes and provides status notifications
type ProcessMonitor struct {
	mu               sync.RWMutex
	processStatuses  map[int]ProcessStatus
	statusChanges    chan ProcessStatusChange
	stopChan         chan struct{}
	verbose          bool
	tracker          *ProcessTracker
	expectedExits    map[int]bool // Track processes that are expected to exit
	monitoringActive bool
}

// NewProcessMonitor creates a new process monitor
func NewProcessMonitor(verbose bool, tracker *ProcessTracker) *ProcessMonitor {
	return &ProcessMonitor{
		processStatuses: make(map[int]ProcessStatus),
		statusChanges:   make(chan ProcessStatusChange, 100), // Buffered channel for status changes
		stopChan:        make(chan struct{}),
		verbose:         verbose,
		tracker:         tracker,
		expectedExits:   make(map[int]bool),
	}
}

// StartMonitoring begins monitoring all tracked processes
func (pm *ProcessMonitor) StartMonitoring(ctx context.Context) {
	pm.mu.Lock()
	if pm.monitoringActive {
		pm.mu.Unlock()
		return
	}
	pm.monitoringActive = true
	pm.mu.Unlock()

	go pm.monitorLoop(ctx)
}

// StopMonitoring stops the process monitoring
func (pm *ProcessMonitor) StopMonitoring() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.monitoringActive {
		return
	}

	pm.monitoringActive = false
	close(pm.stopChan)
}

// AddProcess adds a process to monitoring
func (pm *ProcessMonitor) AddProcess(pid int, name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.processStatuses[pid] = ProcessStatusRunning
	pm.expectedExits[pid] = false

	if pm.verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] [%s] [monitor] Started monitoring process (PID %d)\n", timestamp, name, pid)
		os.Stdout.Sync()
	}
}

// MarkExpectedExit marks a process as expected to exit (for graceful shutdowns)
func (pm *ProcessMonitor) MarkExpectedExit(pid int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.expectedExits[pid] = true
}

// RemoveProcess removes a process from monitoring
func (pm *ProcessMonitor) RemoveProcess(pid int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.processStatuses, pid)
	delete(pm.expectedExits, pid)
}

// GetStatusChanges returns a channel for receiving process status changes
func (pm *ProcessMonitor) GetStatusChanges() <-chan ProcessStatusChange {
	return pm.statusChanges
}

// GetProcessStatus returns the current status of a process
func (pm *ProcessMonitor) GetProcessStatus(pid int) (ProcessStatus, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	status, exists := pm.processStatuses[pid]
	return status, exists
}

// GetAllProcessStatuses returns all current process statuses
func (pm *ProcessMonitor) GetAllProcessStatuses() map[int]ProcessStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[int]ProcessStatus)
	for pid, status := range pm.processStatuses {
		result[pid] = status
	}
	return result
}

// monitorLoop is the main monitoring loop
func (pm *ProcessMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.checkProcessStatuses()
		}
	}
}

// checkProcessStatuses checks the status of all monitored processes
func (pm *ProcessMonitor) checkProcessStatuses() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for pid, currentStatus := range pm.processStatuses {
		if currentStatus != ProcessStatusRunning {
			continue // Skip processes that are already marked as not running
		}

		// Check if process is still running
		if !isProcessRunning(pid) {
			// Process has stopped, determine the new status
			newStatus := pm.determineExitStatus(pid)

			// Get process info for the status change
			processInfo, exists := pm.tracker.GetProcess(pid)
			name := fmt.Sprintf("PID-%d", pid)
			if exists {
				name = processInfo.Name
			}

			// Check if this was an expected exit
			unexpected := !pm.expectedExits[pid]

			// Create status change notification
			change := ProcessStatusChange{
				PID:        pid,
				Name:       name,
				OldStatus:  currentStatus,
				NewStatus:  newStatus,
				Timestamp:  time.Now(),
				Unexpected: unexpected,
			}

			// Update the status
			pm.processStatuses[pid] = newStatus

			// Send notification (non-blocking)
			select {
			case pm.statusChanges <- change:
			default:
				// Channel is full, skip this notification
				if pm.verbose {
					timestamp := time.Now().Format("15:04:05.000")
					fmt.Printf("[%s] [%s] [monitor] Warning: Status change notification dropped (channel full)\n", timestamp, name)
					os.Stdout.Sync()
				}
			}

			// Log the status change if verbose
			if pm.verbose {
				pm.logStatusChange(change)
			}
		}
	}
}

// determineExitStatus determines the exit status of a process
func (pm *ProcessMonitor) determineExitStatus(pid int) ProcessStatus {
	// Try to find the process to get exit information
	process, err := os.FindProcess(pid)
	if err != nil {
		return ProcessStatusExited
	}

	// On Unix systems, we can try to get the process state
	// For now, we'll use a simple heuristic: if the process was expected to exit, it's exited
	// Otherwise, it's considered crashed
	if pm.expectedExits[pid] {
		return ProcessStatusExited
	}

	// Check if the process was killed by checking if it exists in our tracker
	// but is no longer running (this is a simplified approach)
	_ = process // Use the process variable to avoid unused variable warning
	return ProcessStatusCrashed
}

// logStatusChange logs a process status change
func (pm *ProcessMonitor) logStatusChange(change ProcessStatusChange) {
	timestamp := change.Timestamp.Format("15:04:05.000")

	switch change.NewStatus {
	case ProcessStatusExited:
		if change.Unexpected {
			fmt.Printf("[%s] [%s] [monitor] ⚠️  Process exited unexpectedly (PID %d)\n",
				timestamp, change.Name, change.PID)
		} else {
			fmt.Printf("[%s] [%s] [monitor] ✓  Process exited gracefully (PID %d)\n",
				timestamp, change.Name, change.PID)
		}
	case ProcessStatusCrashed:
		fmt.Printf("[%s] [%s] [monitor] ❌ Process crashed (PID %d)\n",
			timestamp, change.Name, change.PID)
	case ProcessStatusKilled:
		fmt.Printf("[%s] [%s] [monitor] 🔪 Process was killed (PID %d)\n",
			timestamp, change.Name, change.PID)
	}

	if change.Error != "" {
		fmt.Printf("[%s] [%s] [monitor] Error: %s\n", timestamp, change.Name, change.Error)
	}

	os.Stdout.Sync()
}

// NotifyUnexpectedTermination sends a notification for unexpected process termination
func (pm *ProcessMonitor) NotifyUnexpectedTermination(pid int, name string, exitCode int, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	currentStatus, exists := pm.processStatuses[pid]
	if !exists {
		currentStatus = ProcessStatusRunning
	}

	newStatus := ProcessStatusCrashed
	if exitCode == 0 {
		newStatus = ProcessStatusExited
	}

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	change := ProcessStatusChange{
		PID:        pid,
		Name:       name,
		OldStatus:  currentStatus,
		NewStatus:  newStatus,
		ExitCode:   exitCode,
		Error:      errorMsg,
		Timestamp:  time.Now(),
		Unexpected: true,
	}

	// Update the status
	pm.processStatuses[pid] = newStatus

	// Send notification (non-blocking)
	select {
	case pm.statusChanges <- change:
	default:
		// Channel is full, skip this notification
		if pm.verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [%s] [monitor] Warning: Unexpected termination notification dropped (channel full)\n", timestamp, name)
			os.Stdout.Sync()
		}
	}

	// Always log unexpected terminations, even in non-verbose mode
	timestamp := change.Timestamp.Format("15:04:05.000")
	if exitCode == 0 {
		fmt.Printf("[%s] [%s] [monitor] ⚠️  Process terminated unexpectedly with exit code 0 (PID %d)\n",
			timestamp, name, pid)
	} else {
		fmt.Printf("[%s] [%s] [monitor] ❌ Process terminated unexpectedly with exit code %d (PID %d)\n",
			timestamp, name, exitCode, pid)
	}

	if errorMsg != "" {
		fmt.Printf("[%s] [%s] [monitor] Error: %s\n", timestamp, name, errorMsg)
	}

	os.Stdout.Sync()
}

// GetMonitoringStats returns statistics about the monitoring
func (pm *ProcessMonitor) GetMonitoringStats() (int, int, int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	running := 0
	exited := 0
	crashed := 0

	for _, status := range pm.processStatuses {
		switch status {
		case ProcessStatusRunning:
			running++
		case ProcessStatusExited:
			exited++
		case ProcessStatusCrashed:
			crashed++
		case ProcessStatusKilled:
			exited++ // Count killed processes as exited
		}
	}

	return running, exited, crashed
}
