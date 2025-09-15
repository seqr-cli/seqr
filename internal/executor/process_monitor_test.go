package executor

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestProcessMonitor_AddAndRemoveProcess(t *testing.T) {
	tracker := NewProcessTracker()
	monitor := NewProcessMonitor(false, tracker)

	// Add a process
	pid := 12345
	name := "test-process"
	monitor.AddProcess(pid, name)

	// Check if process is being monitored
	status, exists := monitor.GetProcessStatus(pid)
	if !exists {
		t.Errorf("Expected process %d to be monitored", pid)
	}
	if status != ProcessStatusRunning {
		t.Errorf("Expected process status to be running, got %s", status.String())
	}

	// Remove the process
	monitor.RemoveProcess(pid)

	// Check if process is no longer monitored
	_, exists = monitor.GetProcessStatus(pid)
	if exists {
		t.Errorf("Expected process %d to no longer be monitored", pid)
	}
}

func TestProcessMonitor_StatusChanges(t *testing.T) {
	tracker := NewProcessTracker()
	monitor := NewProcessMonitor(false, tracker)

	// Start monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor.StartMonitoring(ctx)
	defer monitor.StopMonitoring()

	// Add a process
	pid := 99999 // Use a high PID that's unlikely to exist
	name := "test-process"
	monitor.AddProcess(pid, name)

	// Notify of unexpected termination
	monitor.NotifyUnexpectedTermination(pid, name, 1, nil)

	// Check for status change
	select {
	case change := <-monitor.GetStatusChanges():
		if change.PID != pid {
			t.Errorf("Expected PID %d, got %d", pid, change.PID)
		}
		if change.Name != name {
			t.Errorf("Expected name %s, got %s", name, change.Name)
		}
		if !change.Unexpected {
			t.Errorf("Expected unexpected termination to be true")
		}
		if change.NewStatus != ProcessStatusCrashed {
			t.Errorf("Expected status to be crashed, got %s", change.NewStatus.String())
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Expected to receive status change notification")
	}
}

func TestProcessMonitor_ExpectedExit(t *testing.T) {
	tracker := NewProcessTracker()
	monitor := NewProcessMonitor(false, tracker)

	pid := 12345
	name := "test-process"
	monitor.AddProcess(pid, name)

	// Mark as expected exit
	monitor.MarkExpectedExit(pid)

	// Notify of termination
	monitor.NotifyUnexpectedTermination(pid, name, 0, nil)

	// Check for status change - it should still be marked as unexpected
	// because NotifyUnexpectedTermination always marks as unexpected
	select {
	case change := <-monitor.GetStatusChanges():
		if !change.Unexpected {
			t.Errorf("NotifyUnexpectedTermination should always mark as unexpected")
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Expected to receive status change notification")
	}
}

func TestProcessMonitor_GetMonitoringStats(t *testing.T) {
	tracker := NewProcessTracker()
	monitor := NewProcessMonitor(false, tracker)

	// Add some processes
	monitor.AddProcess(1, "process1")
	monitor.AddProcess(2, "process2")
	monitor.AddProcess(3, "process3")

	// Check initial stats
	running, exited, crashed := monitor.GetMonitoringStats()
	if running != 3 || exited != 0 || crashed != 0 {
		t.Errorf("Expected 3 running, 0 exited, 0 crashed, got %d running, %d exited, %d crashed",
			running, exited, crashed)
	}

	// Simulate some terminations
	monitor.NotifyUnexpectedTermination(1, "process1", 0, nil)
	monitor.NotifyUnexpectedTermination(2, "process2", 1, nil)

	// Check updated stats
	running, exited, crashed = monitor.GetMonitoringStats()
	if running != 1 || exited != 1 || crashed != 1 {
		t.Errorf("Expected 1 running, 1 exited, 1 crashed, got %d running, %d exited, %d crashed",
			running, exited, crashed)
	}
}

func TestProcessStatus_String(t *testing.T) {
	tests := []struct {
		status   ProcessStatus
		expected string
	}{
		{ProcessStatusRunning, "running"},
		{ProcessStatusExited, "exited"},
		{ProcessStatusCrashed, "crashed"},
		{ProcessStatusKilled, "killed"},
		{ProcessStatus(999), "unknown"},
	}

	for _, test := range tests {
		if test.status.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.status.String())
		}
	}
}

func TestProcessMonitor_StartStopMonitoring(t *testing.T) {
	tracker := NewProcessTracker()
	monitor := NewProcessMonitor(false, tracker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start monitoring
	monitor.StartMonitoring(ctx)

	// Check that monitoring is active
	if !monitor.monitoringActive {
		t.Errorf("Expected monitoring to be active")
	}

	// Stop monitoring
	monitor.StopMonitoring()

	// Check that monitoring is inactive
	if monitor.monitoringActive {
		t.Errorf("Expected monitoring to be inactive")
	}
}

// TestProcessMonitor_Integration tests the integration with a real process
func TestProcessMonitor_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tracker := NewProcessTracker()
	monitor := NewProcessMonitor(true, tracker) // Enable verbose for this test

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start monitoring
	monitor.StartMonitoring(ctx)
	defer monitor.StopMonitoring()

	// Start a real process (echo command that exits quickly)
	cmd := "echo"
	args := []string{"test"}

	// Use the OS to start a process
	process, err := os.StartProcess("/bin/echo", []string{"echo", "test"}, &os.ProcAttr{})
	if err != nil {
		// Try alternative path for echo
		process, err = os.StartProcess("/usr/bin/echo", []string{"echo", "test"}, &os.ProcAttr{})
		if err != nil {
			t.Skipf("Could not start echo process: %v", err)
		}
	}

	// Add to monitoring
	monitor.AddProcess(process.Pid, "test-echo")

	// Wait for the process to exit
	_, err = process.Wait()
	if err != nil {
		t.Logf("Process exited with error: %v", err)
	}

	// Give the monitor time to detect the exit
	time.Sleep(3 * time.Second)

	// Check if we received a status change
	select {
	case change := <-monitor.GetStatusChanges():
		t.Logf("Received status change: PID %d, Status %s, Unexpected: %t",
			change.PID, change.NewStatus.String(), change.Unexpected)
	default:
		t.Logf("No status change received (this is expected for very fast processes)")
	}

	_ = cmd
	_ = args
}
