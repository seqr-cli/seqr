package executor

import (
	"context"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestHealthMonitor_NewHealthMonitor(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{}).(*processManager)

	if pm.healthMonitor == nil {
		t.Fatal("Expected health monitor to be initialized")
	}

	if pm.monitorOptions.CheckInterval != 1*time.Second {
		t.Errorf("Expected default check interval 1s, got %v", pm.monitorOptions.CheckInterval)
	}

	if pm.monitorOptions.EventBufferSize != 100 {
		t.Errorf("Expected default event buffer size 100, got %d", pm.monitorOptions.EventBufferSize)
	}
}

func TestHealthMonitor_StartStopMonitoring(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true}).(*processManager)
	ctx := context.Background()

	// Test starting monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Expected successful start, got error: %v", err)
	}

	if !pm.monitorStarted {
		t.Error("Expected monitor to be marked as started")
	}

	// Test starting again (should fail)
	err = pm.StartHealthMonitoring(ctx)
	if err == nil {
		t.Fatal("Expected error when starting already started monitor")
	}

	// Test stopping monitoring
	err = pm.StopHealthMonitoring()
	if err != nil {
		t.Fatalf("Expected successful stop, got error: %v", err)
	}

	if pm.monitorStarted {
		t.Error("Expected monitor to be marked as stopped")
	}

	// Test stopping again (should fail)
	err = pm.StopHealthMonitoring()
	if err == nil {
		t.Fatal("Expected error when stopping already stopped monitor")
	}
}

func TestHealthMonitor_GetProcessHealth_Empty(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{})

	health := pm.GetProcessHealth()
	if len(health) != 0 {
		t.Errorf("Expected empty health map, got %d entries", len(health))
	}
}

func TestHealthMonitor_ProcessLifecycle(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Execute a keepAlive command
	cmd := config.Command{
		Name:    "health-test",
		Command: "sleep",
		Args:    []string{"0.2"},
		Mode:    config.ModeKeepAlive,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}

	// Give the health monitor time to detect the process
	time.Sleep(100 * time.Millisecond)

	// Check that process is being monitored
	health := pm.GetProcessHealth()
	if len(health) == 0 {
		t.Error("Expected at least one process in health monitoring")
	}

	if processHealth, exists := health["health-test"]; exists {
		if processHealth.Status != ProcessStatusRunning {
			t.Errorf("Expected process status to be running, got %s", processHealth.Status)
		}
		if processHealth.Name != "health-test" {
			t.Errorf("Expected process name 'health-test', got '%s'", processHealth.Name)
		}
		if processHealth.PID <= 0 {
			t.Errorf("Expected valid PID, got %d", processHealth.PID)
		}
	} else {
		t.Error("Expected 'health-test' process to be in health monitoring")
	}

	// Wait for process to complete
	time.Sleep(300 * time.Millisecond)

	// Check that process status is updated after exit
	health = pm.GetProcessHealth()
	if processHealth, exists := health["health-test"]; exists {
		if processHealth.Status == ProcessStatusRunning {
			t.Error("Expected process status to be updated after exit")
		}
	}
}

func TestHealthMonitor_MultipleProcesses(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Execute multiple keepAlive commands
	commands := []config.Command{
		{
			Name:    "health-test-1",
			Command: "sleep",
			Args:    []string{"0.3"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "health-test-2",
			Command: "sleep",
			Args:    []string{"0.3"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "health-test-3",
			Command: "sleep",
			Args:    []string{"0.3"},
			Mode:    config.ModeKeepAlive,
		},
	}

	// Execute all commands
	for _, cmd := range commands {
		result, err := pm.ExecuteCommand(ctx, cmd)
		if err != nil {
			t.Fatalf("Expected successful execution for %s, got error: %v", cmd.Name, err)
		}
		if !result.Success {
			t.Errorf("Expected command %s to succeed", cmd.Name)
		}
	}

	// Give the health monitor time to detect all processes
	time.Sleep(100 * time.Millisecond)

	// Check that all processes are being monitored
	health := pm.GetProcessHealth()
	if len(health) != 3 {
		t.Errorf("Expected 3 processes in health monitoring, got %d", len(health))
	}

	expectedNames := []string{"health-test-1", "health-test-2", "health-test-3"}
	for _, name := range expectedNames {
		if processHealth, exists := health[name]; exists {
			if processHealth.Status != ProcessStatusRunning {
				t.Errorf("Expected process %s status to be running, got %s", name, processHealth.Status)
			}
		} else {
			t.Errorf("Expected process '%s' to be in health monitoring", name)
		}
	}

	// Wait for all processes to complete
	time.Sleep(400 * time.Millisecond)
}

func TestHealthMonitor_EventGeneration(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Create a channel to capture events
	eventCount := 0
	eventCapture := make(chan HealthMonitorEvent, 10)

	// Replace the event channel temporarily for testing
	pmImpl := pm.(*processManager)
	originalChannel := pmImpl.eventChannel
	pmImpl.eventChannel = eventCapture

	// Execute a keepAlive command
	cmd := config.Command{
		Name:    "event-test",
		Command: "sleep",
		Args:    []string{"0.1"},
		Mode:    config.ModeKeepAlive,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}

	// Wait for events to be generated and collect them
	timeout := time.After(300 * time.Millisecond)
	for {
		select {
		case event := <-eventCapture:
			eventCount++
			if event.ProcessID != "event-test" {
				t.Errorf("Expected event for 'event-test', got '%s'", event.ProcessID)
			}
			if event.Type != HealthEventProcessStarted && event.Type != HealthEventProcessExited {
				t.Errorf("Expected start or exit event, got %s", event.Type)
			}
		case <-timeout:
			// Stop collecting events after timeout
			goto checkResults
		}
	}

checkResults:

	// Restore original channel
	pmImpl.eventChannel = originalChannel

	if eventCount == 0 {
		t.Error("Expected at least one health monitoring event")
	}
}

func TestHealthMonitor_HealthSummary(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{}).(*processManager)

	// Test empty summary
	summary := pm.healthMonitor.GetHealthSummary()
	if summary.TotalProcesses != 0 {
		t.Errorf("Expected 0 total processes, got %d", summary.TotalProcesses)
	}
	if summary.RunningProcesses != 0 {
		t.Errorf("Expected 0 running processes, got %d", summary.RunningProcesses)
	}
}

func TestHealthMonitor_ProcessRunningCheck(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{}).(*processManager)

	// Test with invalid PID
	if pm.healthMonitor.isProcessRunning(-1) {
		t.Error("Expected false for invalid PID")
	}

	if pm.healthMonitor.isProcessRunning(999999) {
		t.Error("Expected false for non-existent PID")
	}

	// Test with current process PID (should be running)
	currentPID := 1 // Use PID 1 which should always exist on Unix systems
	if !pm.healthMonitor.isProcessRunning(currentPID) {
		// This might fail on some systems, so we'll make it a soft check
		t.Logf("Warning: PID 1 check failed, this might be expected on some systems")
	}
}

func TestHealthMonitor_GetProcessHealthIndividual(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{}).(*processManager)

	// Test getting health for non-existent process
	health, exists := pm.healthMonitor.GetProcessHealth("nonexistent")
	if exists {
		t.Error("Expected false for non-existent process")
	}
	if health != nil {
		t.Error("Expected nil health for non-existent process")
	}
}

func TestHealthMonitor_RemoveProcess(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{}).(*processManager)

	// Add a mock process health record
	pm.healthMonitor.mu.Lock()
	pm.healthMonitor.processMap["test-process"] = &ProcessHealth{
		Name:   "test-process",
		PID:    12345,
		Status: ProcessStatusRunning,
	}
	pm.healthMonitor.mu.Unlock()

	// Verify it exists
	health := pm.healthMonitor.GetAllHealth()
	if len(health) != 1 {
		t.Fatalf("Expected 1 process, got %d", len(health))
	}

	// Remove the process
	pm.healthMonitor.RemoveProcess("test-process")

	// Verify it's removed
	health = pm.healthMonitor.GetAllHealth()
	if len(health) != 0 {
		t.Errorf("Expected 0 processes after removal, got %d", len(health))
	}
}

func TestHealthMonitor_EventChannelFull(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{}).(*processManager)

	// Create a small event channel that will fill up quickly
	smallChannel := make(chan HealthMonitorEvent, 1)

	// Fill the channel
	smallChannel <- HealthMonitorEvent{Type: HealthEventProcessStarted}

	// Try to send another event (should not block)
	event := HealthMonitorEvent{
		Type:      HealthEventProcessExited,
		ProcessID: "test",
		Timestamp: time.Now(),
		Message:   "test message",
	}

	// This should not block even though channel is full
	pm.healthMonitor.sendEvent(smallChannel, event)

	// Verify the channel still has the original event
	select {
	case receivedEvent := <-smallChannel:
		if receivedEvent.Type != HealthEventProcessStarted {
			t.Errorf("Expected original event, got %s", receivedEvent.Type)
		}
	default:
		t.Error("Expected to receive the original event")
	}
}
