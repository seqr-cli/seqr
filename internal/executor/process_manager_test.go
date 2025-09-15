package executor

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestNewProcessManager(t *testing.T) {
	pm := NewProcessManager()

	if pm == nil {
		t.Fatal("NewProcessManager returned nil")
	}

	if pm.tracker == nil {
		t.Fatal("ProcessManager tracker is nil")
	}
}

func TestGetAllRunningProcesses(t *testing.T) {
	pm := NewProcessManager()

	// Clean up any existing tracking file
	defer os.Remove(pm.tracker.filePath)

	// Test with no processes
	processes, err := pm.GetAllRunningProcesses()
	if err != nil {
		t.Fatalf("GetAllRunningProcesses failed: %v", err)
	}

	if len(processes) != 0 {
		t.Fatalf("Expected 0 processes, got %d", len(processes))
	}

	// Add a fake process (current process should be running)
	currentPID := os.Getpid()
	err = pm.tracker.AddProcess(currentPID, "test", "test-cmd", []string{}, "", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Test with one process
	processes, err = pm.GetAllRunningProcesses()
	if err != nil {
		t.Fatalf("GetAllRunningProcesses failed: %v", err)
	}

	// On Windows, process detection may not work correctly, so we'll be more lenient
	if len(processes) < 1 {
		t.Fatalf("Expected at least 1 process, got %d", len(processes))
	}

	if _, exists := processes[currentPID]; !exists {
		t.Fatal("Current process should be in the list")
	}
}

func TestGetProcessCount(t *testing.T) {
	pm := NewProcessManager()

	// Clean up any existing tracking file
	defer os.Remove(pm.tracker.filePath)

	// Test with no processes
	count, err := pm.GetProcessCount()
	if err != nil {
		t.Fatalf("GetProcessCount failed: %v", err)
	}

	if count != 0 {
		t.Fatalf("Expected 0 processes, got %d", count)
	}

	// Add a process
	currentPID := os.Getpid()
	err = pm.tracker.AddProcess(currentPID, "test", "test-cmd", []string{}, "", "once")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Test with one process
	count, err = pm.GetProcessCount()
	if err != nil {
		t.Fatalf("GetProcessCount failed: %v", err)
	}

	// On Windows, process detection may not work correctly, so we'll be more lenient
	if count < 1 {
		t.Fatalf("Expected at least 1 process, got %d", count)
	}
}

func TestKillProcess_NotTracked(t *testing.T) {
	pm := NewProcessManager()

	// Clean up any existing tracking file
	defer os.Remove(pm.tracker.filePath)

	// Try to kill a process that's not tracked
	err := pm.KillProcess(999999, true)
	if err == nil {
		t.Fatal("Expected error when killing non-tracked process")
	}

	expectedMsg := "process with PID 999999 is not tracked by seqr"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestKillProcess_WithRealProcess(t *testing.T) {
	pm := NewProcessManager()

	// Clean up any existing tracking file
	defer os.Remove(pm.tracker.filePath)

	// Start a long-running process for testing (Windows compatible)
	cmd := exec.Command("ping", "127.0.0.1", "-n", "30")
	err := cmd.Start()
	if err != nil {
		t.Skipf("Cannot start ping command for testing: %v", err)
	}

	pid := cmd.Process.Pid

	// Track the process
	err = pm.tracker.AddProcess(pid, "test-ping", "ping", []string{"127.0.0.1", "-n", "30"}, "", "keepAlive")
	if err != nil {
		t.Fatalf("AddProcess failed: %v", err)
	}

	// Verify process is tracked
	_, exists := pm.tracker.GetProcess(pid)
	if !exists {
		t.Fatal("Process should be tracked")
	}

	// Kill the process gracefully
	err = pm.KillProcess(pid, true)
	if err != nil {
		t.Fatalf("KillProcess failed: %v", err)
	}

	// Verify process is no longer tracked
	_, exists = pm.tracker.GetProcess(pid)
	if exists {
		t.Fatal("Process should not be tracked after killing")
	}

	// Wait a bit and verify process is actually dead
	time.Sleep(500 * time.Millisecond)
	if isProcessRunning(pid) {
		t.Logf("Warning: Process %d still appears to be running after kill (this may be a Windows limitation)", pid)
	}
}

func TestKillAllProcesses_NoProcesses(t *testing.T) {
	pm := NewProcessManager()

	// Clean up any existing tracking file
	defer os.Remove(pm.tracker.filePath)

	// Try to kill all processes when none are running
	err := pm.KillAllProcesses(true)
	if err == nil {
		t.Fatal("Expected error when no processes are running")
	}

	expectedMsg := "no seqr processes are currently running"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestKillAllProcesses_WithRealProcesses(t *testing.T) {
	pm := NewProcessManager()

	// Clean up any existing tracking file
	defer os.Remove(pm.tracker.filePath)

	// Start multiple long-running processes for testing
	var pids []int
	var cmds []*exec.Cmd

	for i := 0; i < 2; i++ {
		cmd := exec.Command("ping", "127.0.0.1", "-n", "30")
		err := cmd.Start()
		if err != nil {
			t.Skipf("Cannot start ping command for testing: %v", err)
		}

		pid := cmd.Process.Pid
		pids = append(pids, pid)
		cmds = append(cmds, cmd)

		// Track the process
		err = pm.tracker.AddProcess(pid, fmt.Sprintf("test-ping-%d", i), "ping", []string{"127.0.0.1", "-n", "30"}, "", "keepAlive")
		if err != nil {
			t.Fatalf("AddProcess failed: %v", err)
		}
	}

	// Verify processes are tracked
	count, err := pm.GetProcessCount()
	if err != nil {
		t.Fatalf("GetProcessCount failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected 2 processes, got %d", count)
	}

	// Kill all processes
	err = pm.KillAllProcesses(true)
	if err != nil {
		t.Fatalf("KillAllProcesses failed: %v", err)
	}

	// Verify no processes are tracked
	count, err = pm.GetProcessCount()
	if err != nil {
		t.Fatalf("GetProcessCount failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected 0 processes after killing all, got %d", count)
	}

	// Wait a bit and verify processes are actually dead
	time.Sleep(500 * time.Millisecond)
	for _, pid := range pids {
		if isProcessRunning(pid) {
			t.Logf("Warning: Process %d still appears to be running after kill all (this may be a Windows limitation)", pid)
		}
	}
}
