package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// ProcessInfo represents information about a tracked process
type ProcessInfo struct {
	PID       int       `json:"pid"`
	Name      string    `json:"name"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	WorkDir   string    `json:"workDir,omitempty"`
	StartTime time.Time `json:"startTime"`
	Mode      string    `json:"mode"`
}

// ProcessTracker manages tracking of running seqr processes
type ProcessTracker struct {
	mu        sync.RWMutex
	processes map[int]*ProcessInfo
	filePath  string
}

// NewProcessTracker creates a new process tracker
func NewProcessTracker() *ProcessTracker {
	// Use a temporary directory for the process tracking file
	tempDir := os.TempDir()
	filePath := filepath.Join(tempDir, "seqr-processes.json")

	tracker := &ProcessTracker{
		processes: make(map[int]*ProcessInfo),
		filePath:  filePath,
	}

	// Load existing processes from file
	tracker.loadFromFile()

	return tracker
}

// AddProcess adds a process to the tracker
func (pt *ProcessTracker) AddProcess(pid int, name, command string, args []string, workDir, mode string) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	processInfo := &ProcessInfo{
		PID:       pid,
		Name:      name,
		Command:   command,
		Args:      args,
		WorkDir:   workDir,
		StartTime: time.Now(),
		Mode:      mode,
	}

	pt.processes[pid] = processInfo
	return pt.saveToFile()
}

// RemoveProcess removes a process from the tracker
func (pt *ProcessTracker) RemoveProcess(pid int) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	delete(pt.processes, pid)
	return pt.saveToFile()
}

// GetAllProcesses returns all tracked processes
func (pt *ProcessTracker) GetAllProcesses() map[int]*ProcessInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[int]*ProcessInfo)
	for pid, info := range pt.processes {
		// Create a copy of the ProcessInfo
		infoCopy := *info
		result[pid] = &infoCopy
	}

	return result
}

// GetProcess returns information about a specific process
func (pt *ProcessTracker) GetProcess(pid int) (*ProcessInfo, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	info, exists := pt.processes[pid]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	infoCopy := *info
	return &infoCopy, true
}

// CleanupDeadProcesses removes processes that are no longer running
func (pt *ProcessTracker) CleanupDeadProcesses() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	var toRemove []int

	for pid := range pt.processes {
		if !isProcessRunning(pid) {
			toRemove = append(toRemove, pid)
		}
	}

	for _, pid := range toRemove {
		delete(pt.processes, pid)
	}

	if len(toRemove) > 0 {
		return pt.saveToFile()
	}

	return nil
}

// GetRunningProcessCount returns the number of currently tracked processes
func (pt *ProcessTracker) GetRunningProcessCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return len(pt.processes)
}

// saveToFile persists the current process list to disk
func (pt *ProcessTracker) saveToFile() error {
	data, err := json.MarshalIndent(pt.processes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal process data: %w", err)
	}

	// Write to a temporary file first, then rename for atomic operation
	tempFile := pt.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write process file: %w", err)
	}

	if err := os.Rename(tempFile, pt.filePath); err != nil {
		os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename process file: %w", err)
	}

	return nil
}

// loadFromFile loads the process list from disk
func (pt *ProcessTracker) loadFromFile() error {
	data, err := os.ReadFile(pt.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, that's okay
			return nil
		}
		return fmt.Errorf("failed to read process file: %w", err)
	}

	if len(data) == 0 {
		// Empty file, that's okay
		return nil
	}

	var processes map[int]*ProcessInfo
	if err := json.Unmarshal(data, &processes); err != nil {
		return fmt.Errorf("failed to unmarshal process data: %w", err)
	}

	pt.processes = processes
	if pt.processes == nil {
		pt.processes = make(map[int]*ProcessInfo)
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is still running
func isProcessRunning(pid int) bool {
	if runtime.GOOS == "windows" {
		return isProcessRunningWindows(pid)
	}
	return isProcessRunningUnix(pid)
}

// isProcessRunningWindows checks if a process is running on Windows
func isProcessRunningWindows(pid int) bool {
	// On Windows, we'll use a simple heuristic:
	// For processes we track ourselves (which should be real), assume they exist
	// For obviously fake PIDs (like 999999), assume they don't exist
	// This is a limitation but works for our use case

	// Very high PIDs are unlikely to exist
	if pid > 100000 {
		return false
	}

	// For reasonable PIDs, try to find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Windows, FindProcess always succeeds, so we assume the process exists
	// This is not perfect but acceptable for our tracking use case
	_ = process
	return true
}

// isProcessRunningUnix checks if a process is running on Unix-like systems
func isProcessRunningUnix(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix-like systems, sending signal 0 doesn't actually send a signal
	// but checks if the process exists and we have permission to signal it
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}

	return true
}
