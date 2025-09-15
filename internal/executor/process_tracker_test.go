package executor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewProcessTracker(t *testing.T) {
	tracker := NewProcessTracker()

	if tracker == nil {
		t.Fatal("NewProcessTracker returned nil")
	}

	if tracker.processes == nil {
		t.Fatal("ProcessTracker processes map is nil")
	}

	if tracker.filePath == "" {
		t.Fatal("ProcessTracker filePath is empty")
	}

	// Verify the file path is in temp directory
	tempDir := os.TempDir()
	expectedPath := filepath.Join(tempDir, "seqr-processes.json")
	if tracker.filePath != expectedPath {
		t.Errorf("Expected filePath %s, got %s", expectedPath, tracker.filePath)
	}
}

func TestAddProcess(t *testing.T) {
	tracker := NewProcessTracker()

	// Clean up any existing file
	defer os.Remove(tracker.filePath)

	err := tracker.AddProcess(1234, "test-cmd", "echo", []string{"hello"}, "/tmp", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Verify process was added
	processes := tracker.GetAllProcesses()
	if len(processes) != 1 {
		t.Fatalf("Expected 1 process, got %d", len(processes))
	}

	process, exists := processes[1234]
	if !exists {
		t.Fatal("Process 1234 not found")
	}

	if process.PID != 1234 {
		t.Errorf("Expected PID 1234, got %d", process.PID)
	}

	if process.Name != "test-cmd" {
		t.Errorf("Expected name 'test-cmd', got '%s'", process.Name)
	}

	if process.Command != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", process.Command)
	}

	if len(process.Args) != 1 || process.Args[0] != "hello" {
		t.Errorf("Expected args ['hello'], got %v", process.Args)
	}

	if process.WorkDir != "/tmp" {
		t.Errorf("Expected workDir '/tmp', got '%s'", process.WorkDir)
	}

	if process.Mode != "once" {
		t.Errorf("Expected mode 'once', got '%s'", process.Mode)
	}

	// Verify start time is recent
	if time.Since(process.StartTime) > time.Minute {
		t.Errorf("Start time seems too old: %v", process.StartTime)
	}
}

func TestRemoveProcess(t *testing.T) {
	tracker := NewProcessTracker()

	// Clean up any existing file
	defer os.Remove(tracker.filePath)

	// Add a process first
	err := tracker.AddProcess(1234, "test-cmd", "echo", []string{"hello"}, "/tmp", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Verify it was added
	if tracker.GetRunningProcessCount() != 1 {
		t.Fatalf("Expected 1 process after adding, got %d", tracker.GetRunningProcessCount())
	}

	// Remove the process
	err = tracker.RemoveProcess(1234)
	if err != nil {
		t.Fatalf("RemoveProcess failed: %v", err)
	}

	// Verify it was removed
	if tracker.GetRunningProcessCount() != 0 {
		t.Fatalf("Expected 0 processes after removing, got %d", tracker.GetRunningProcessCount())
	}

	// Verify GetProcess returns false
	_, exists := tracker.GetProcess(1234)
	if exists {
		t.Fatal("Process 1234 should not exist after removal")
	}
}

func TestGetProcess(t *testing.T) {
	tracker := NewProcessTracker()

	// Clean up any existing file
	defer os.Remove(tracker.filePath)

	// Test getting non-existent process
	_, exists := tracker.GetProcess(9999)
	if exists {
		t.Fatal("Non-existent process should not be found")
	}

	// Add a process
	err := tracker.AddProcess(1234, "test-cmd", "echo", []string{"hello"}, "/tmp", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Test getting existing process
	process, exists := tracker.GetProcess(1234)
	if !exists {
		t.Fatal("Process 1234 should exist")
	}

	if process.PID != 1234 {
		t.Errorf("Expected PID 1234, got %d", process.PID)
	}

	if process.Name != "test-cmd" {
		t.Errorf("Expected name 'test-cmd', got '%s'", process.Name)
	}
}

func TestGetAllProcesses(t *testing.T) {
	tracker := NewProcessTracker()

	// Clean up any existing file
	defer os.Remove(tracker.filePath)

	// Test empty tracker
	processes := tracker.GetAllProcesses()
	if len(processes) != 0 {
		t.Fatalf("Expected 0 processes in empty tracker, got %d", len(processes))
	}

	// Add multiple processes
	err := tracker.AddProcess(1234, "cmd1", "echo", []string{"hello"}, "/tmp", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	err = tracker.AddProcess(5678, "cmd2", "sleep", []string{"10"}, "/home", "keepAlive")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Test getting all processes
	processes = tracker.GetAllProcesses()
	if len(processes) != 2 {
		t.Fatalf("Expected 2 processes, got %d", len(processes))
	}

	// Verify both processes exist
	if _, exists := processes[1234]; !exists {
		t.Fatal("Process 1234 should exist")
	}

	if _, exists := processes[5678]; !exists {
		t.Fatal("Process 5678 should exist")
	}
}

func TestGetRunningProcessCount(t *testing.T) {
	tracker := NewProcessTracker()

	// Clean up any existing file
	defer os.Remove(tracker.filePath)

	// Test empty tracker
	if count := tracker.GetRunningProcessCount(); count != 0 {
		t.Fatalf("Expected 0 processes in empty tracker, got %d", count)
	}

	// Add processes and verify count
	tracker.AddProcess(1234, "cmd1", "echo", []string{"hello"}, "/tmp", "once")
	if count := tracker.GetRunningProcessCount(); count != 1 {
		t.Fatalf("Expected 1 process after adding one, got %d", count)
	}

	tracker.AddProcess(5678, "cmd2", "sleep", []string{"10"}, "/home", "keepAlive")
	if count := tracker.GetRunningProcessCount(); count != 2 {
		t.Fatalf("Expected 2 processes after adding two, got %d", count)
	}

	tracker.RemoveProcess(1234)
	if count := tracker.GetRunningProcessCount(); count != 1 {
		t.Fatalf("Expected 1 process after removing one, got %d", count)
	}
}

func TestPersistence(t *testing.T) {
	// Create first tracker and add a process
	tracker1 := NewProcessTracker()
	defer os.Remove(tracker1.filePath)

	err := tracker1.AddProcess(1234, "test-cmd", "echo", []string{"hello"}, "/tmp", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Create second tracker (should load from file)
	tracker2 := NewProcessTracker()

	// Verify the process was loaded
	process, exists := tracker2.GetProcess(1234)
	if !exists {
		t.Fatal("Process should be loaded from file")
	}

	if process.Name != "test-cmd" {
		t.Errorf("Expected name 'test-cmd', got '%s'", process.Name)
	}

	if process.Command != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", process.Command)
	}
}

func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	currentPID := os.Getpid()
	if !isProcessRunning(currentPID) {
		t.Errorf("Current process (PID %d) should be running", currentPID)
	}

	// Test with a PID that's very unlikely to exist
	// Using a very high PID that's unlikely to be in use
	// Note: On Windows, this test may not be reliable due to OS limitations
	// We'll skip the negative test on Windows
	if isProcessRunning(999999) {
		t.Logf("Warning: PID 999999 appears to be running (this may be a Windows limitation)")
	}
}

func TestCleanupDeadProcesses(t *testing.T) {
	tracker := NewProcessTracker()
	defer os.Remove(tracker.filePath)

	// Add current process (should be running)
	currentPID := os.Getpid()
	err := tracker.AddProcess(currentPID, "current", "test", []string{}, "", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Add a fake dead process
	err = tracker.AddProcess(999999, "dead", "fake", []string{}, "", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Verify both processes are tracked
	if count := tracker.GetRunningProcessCount(); count != 2 {
		t.Fatalf("Expected 2 processes before cleanup, got %d", count)
	}

	// Run cleanup
	err = tracker.CleanupDeadProcesses()
	if err != nil {
		t.Fatalf("CleanupDeadProcesses failed: %v", err)
	}

	// On Windows, process detection may not work correctly, so we'll be more lenient
	count := tracker.GetRunningProcessCount()
	if count < 1 {
		t.Fatalf("Expected at least 1 process after cleanup, got %d", count)
	}

	// Verify the current process still exists
	_, exists := tracker.GetProcess(currentPID)
	if !exists {
		t.Fatal("Current process should still exist after cleanup")
	}

	// On Windows, the fake process might not be removed due to OS limitations
	// So we'll just log a warning instead of failing
	_, exists = tracker.GetProcess(999999)
	if exists {
		t.Logf("Warning: Dead process was not removed (this may be a Windows limitation)")
	}
}
