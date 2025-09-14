package executor

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

// healthMonitor implements the background health monitoring system
type healthMonitor struct {
	mu             sync.RWMutex
	processMap     map[string]*ProcessHealth
	options        HealthMonitorOptions
	processManager *processManager
}

// newHealthMonitor creates a new health monitor instance
func newHealthMonitor(pm *processManager, opts HealthMonitorOptions) *healthMonitor {
	return &healthMonitor{
		processMap:     make(map[string]*ProcessHealth),
		options:        opts,
		processManager: pm,
	}
}

// Start begins the health monitoring loop
func (hm *healthMonitor) Start(ctx context.Context, eventChan chan<- HealthMonitorEvent) {
	ticker := time.NewTicker(hm.options.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.performHealthCheck(eventChan)
		}
	}
}

// NotifyProcessStarted immediately notifies the health monitor about a new process
func (hm *healthMonitor) NotifyProcessStarted(name string, procInfo ProcessInfo) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now()
	health := &ProcessHealth{
		Name:           name,
		PID:            procInfo.PID,
		Status:         ProcessStatusRunning,
		StartTime:      procInfo.StartTime,
		LastCheckTime:  now,
		RestartCount:   0,
		UptimeDuration: now.Sub(procInfo.StartTime),
	}

	hm.processMap[name] = health
}

// NotifyProcessExited immediately notifies the health monitor about a process exit
func (hm *healthMonitor) NotifyProcessExited(name string, exitTime time.Time, exitReason string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if health, exists := hm.processMap[name]; exists {
		health.Status = ProcessStatusExited
		health.ExitTime = &exitTime
		health.ExitReason = exitReason
		health.LastCheckTime = exitTime
		health.UptimeDuration = exitTime.Sub(health.StartTime)
	}
}

// performHealthCheck checks the health of all active processes
func (hm *healthMonitor) performHealthCheck(eventChan chan<- HealthMonitorEvent) {
	// Get current active processes from the process manager
	activeProcs := hm.processManager.GetActiveProcesses()

	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Update health status for active processes
	for name, procInfo := range activeProcs {
		health := hm.updateProcessHealth(name, procInfo, eventChan)
		hm.processMap[name] = health
	}

	// Check for processes that are no longer active
	for name, health := range hm.processMap {
		if _, exists := activeProcs[name]; !exists {
			// Process is no longer active, mark as exited
			if health.Status == ProcessStatusRunning {
				health.Status = ProcessStatusExited
				health.ExitTime = &[]time.Time{time.Now()}[0]
				health.LastCheckTime = time.Now()

				event := HealthMonitorEvent{
					Type:      HealthEventProcessExited,
					ProcessID: name,
					Timestamp: time.Now(),
					Message:   fmt.Sprintf("Process no longer active (PID %d)", health.PID),
					Health:    health,
				}

				hm.sendEvent(eventChan, event)
			}
		}
	}
}

// updateProcessHealth updates the health status for a specific process
func (hm *healthMonitor) updateProcessHealth(name string, procInfo ProcessInfo, eventChan chan<- HealthMonitorEvent) *ProcessHealth {
	now := time.Now()

	// Get existing health record or create new one
	health, exists := hm.processMap[name]
	if !exists {
		health = &ProcessHealth{
			Name:           name,
			PID:            procInfo.PID,
			Status:         ProcessStatusRunning,
			StartTime:      procInfo.StartTime,
			LastCheckTime:  now,
			RestartCount:   0,
			UptimeDuration: now.Sub(procInfo.StartTime),
		}

		// Send process started event
		event := HealthMonitorEvent{
			Type:      HealthEventProcessStarted,
			ProcessID: name,
			Timestamp: now,
			Message:   fmt.Sprintf("Process monitoring started (PID %d)", procInfo.PID),
			Health:    health,
		}
		hm.sendEvent(eventChan, event)
	} else {
		// Update existing health record
		health.LastCheckTime = now
		health.UptimeDuration = now.Sub(health.StartTime)
	}

	// Check if process is still running
	if !hm.isProcessRunning(procInfo.PID) {
		if health.Status == ProcessStatusRunning {
			health.Status = ProcessStatusExited
			health.ExitTime = &now

			event := HealthMonitorEvent{
				Type:      HealthEventProcessExited,
				ProcessID: name,
				Timestamp: now,
				Message:   fmt.Sprintf("Process exited (PID %d)", procInfo.PID),
				Health:    health,
			}
			hm.sendEvent(eventChan, event)
		}
		return health
	}

	// Collect metrics if enabled
	if hm.options.EnableMetrics {
		hm.collectProcessMetrics(health, eventChan)
	}

	return health
}

// isProcessRunning checks if a process with the given PID is still running
func (hm *healthMonitor) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// collectProcessMetrics collects CPU and memory metrics for a process
func (hm *healthMonitor) collectProcessMetrics(health *ProcessHealth, eventChan chan<- HealthMonitorEvent) {
	// Note: This is a simplified implementation
	// In a production system, you might want to use more sophisticated
	// process monitoring libraries or system calls

	// For now, we'll simulate basic metrics collection
	// In a real implementation, you would read from /proc/[pid]/stat and /proc/[pid]/status
	// or use platform-specific APIs

	// Check memory usage (simplified - would need platform-specific implementation)
	if hm.options.MemoryThreshold > 0 {
		// Simulate memory check - in real implementation, read from /proc/[pid]/status
		// For now, we'll skip actual memory collection to keep the implementation simple
		// but the structure is here for future enhancement
		_ = health    // Use health parameter to avoid unused warning
		_ = eventChan // Use eventChan parameter to avoid unused warning
	}

	// Check CPU usage (simplified - would need platform-specific implementation)
	if hm.options.CPUThreshold > 0 {
		// Simulate CPU check - in real implementation, calculate from /proc/[pid]/stat
		// For now, we'll skip actual CPU collection to keep the implementation simple
		// but the structure is here for future enhancement
	}
}

// sendEvent safely sends an event to the event channel
func (hm *healthMonitor) sendEvent(eventChan chan<- HealthMonitorEvent, event HealthMonitorEvent) {
	select {
	case eventChan <- event:
		// Event sent successfully
	default:
		// Channel is full, drop the event to avoid blocking
		// In a production system, you might want to log this
	}
}

// GetAllHealth returns a copy of all current health records
func (hm *healthMonitor) GetAllHealth() map[string]ProcessHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]ProcessHealth)
	for name, health := range hm.processMap {
		// Create a copy to avoid race conditions
		healthCopy := *health
		result[name] = healthCopy
	}

	return result
}

// GetProcessHealth returns the health status for a specific process
func (hm *healthMonitor) GetProcessHealth(name string) (*ProcessHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.processMap[name]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	healthCopy := *health
	return &healthCopy, true
}

// RemoveProcess removes a process from health monitoring
func (hm *healthMonitor) RemoveProcess(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.processMap, name)
}

// GetHealthSummary returns a summary of all monitored processes
func (hm *healthMonitor) GetHealthSummary() HealthSummary {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	summary := HealthSummary{
		TotalProcesses: len(hm.processMap),
		Timestamp:      time.Now(),
	}

	for _, health := range hm.processMap {
		switch health.Status {
		case ProcessStatusRunning:
			summary.RunningProcesses++
		case ProcessStatusExited:
			summary.ExitedProcesses++
		case ProcessStatusFailed:
			summary.FailedProcesses++
		}
	}

	return summary
}

// HealthSummary provides an overview of all monitored processes
type HealthSummary struct {
	TotalProcesses   int       `json:"totalProcesses"`
	RunningProcesses int       `json:"runningProcesses"`
	ExitedProcesses  int       `json:"exitedProcesses"`
	FailedProcesses  int       `json:"failedProcesses"`
	Timestamp        time.Time `json:"timestamp"`
}
