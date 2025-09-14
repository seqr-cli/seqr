package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestProcessStatusReporting(t *testing.T) {
	// Create a test reporter to capture output
	testReporter := &TestReporter{
		events: make([]string, 0),
	}

	opts := ProcessManagerOptions{
		Verbose:  true,
		Reporter: testReporter,
	}

	pm := NewProcessManager(opts)
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Execute a keepAlive command
	cmd := config.Command{
		Name:    "status-test",
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

	// Test ReportCurrentStatus
	pm.ReportCurrentStatus()

	// Verify that status reporting was called
	if !testReporter.HasEvent("ReportProcessStatus") {
		t.Error("Expected ReportProcessStatus to be called")
	}

	// Test ReportHealthStatus
	pm.ReportHealthStatus()

	// Verify that health reporting was called
	if !testReporter.HasEvent("ReportProcessHealth") {
		t.Error("Expected ReportProcessHealth to be called")
	}

	// Wait for process to complete and generate lifecycle events
	time.Sleep(300 * time.Millisecond)

	// Verify lifecycle events were reported
	if !testReporter.HasEvent("ReportProcessLifecycleEvent") {
		t.Error("Expected lifecycle events to be reported")
	}
}

func TestExecutorProcessStatusIntegration(t *testing.T) {
	// Create a test reporter to capture output
	testReporter := &TestReporter{
		events: make([]string, 0),
	}

	opts := ExecutorOptions{
		Verbose:  true,
		Reporter: testReporter,
	}

	executor := NewExecutor(opts)
	ctx := context.Background()

	// Start health monitoring
	err := executor.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer executor.StopHealthMonitoring()

	// Create a configuration with keepAlive commands
	cfg := &config.Config{
		Commands: []config.Command{
			{
				Name:    "test-process-1",
				Command: "sleep",
				Args:    []string{"0.2"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "test-process-2",
				Command: "sleep",
				Args:    []string{"0.2"},
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	// Execute the configuration
	err = executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	// Test process status reporting
	executor.ReportProcessStatus()
	if !testReporter.HasEvent("ReportProcessStatus") {
		t.Error("Expected ReportProcessStatus to be called")
	}

	// Test health status reporting
	executor.ReportHealthStatus()
	if !testReporter.HasEvent("ReportProcessHealth") {
		t.Error("Expected ReportProcessHealth to be called")
	}

	// Test getting active processes
	activeProcs := executor.GetActiveProcesses()
	if len(activeProcs) == 0 {
		t.Error("Expected active processes to be tracked")
	}

	// Test getting process health
	health := executor.GetProcessHealth()
	if len(health) == 0 {
		t.Error("Expected process health to be tracked")
	}

	// Wait for processes to complete
	time.Sleep(300 * time.Millisecond)
}

func TestLifecycleEventHandling(t *testing.T) {
	// Create a test reporter to capture lifecycle events
	testReporter := &TestReporter{
		events:          make([]string, 0),
		lifecycleEvents: make([]HealthMonitorEvent, 0),
	}

	opts := ProcessManagerOptions{
		Verbose:  true,
		Reporter: testReporter,
	}

	pm := NewProcessManager(opts)
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Enable lifecycle reporting
	pm.EnableLifecycleReporting(true)

	// Execute a short-lived keepAlive command
	cmd := config.Command{
		Name:    "lifecycle-test",
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

	// Wait for process to complete and events to be generated
	time.Sleep(200 * time.Millisecond)

	// Verify lifecycle events were captured
	if len(testReporter.lifecycleEvents) == 0 {
		t.Error("Expected lifecycle events to be captured")
	}

	// Check for process started event
	hasStartEvent := false
	hasExitEvent := false
	for _, event := range testReporter.lifecycleEvents {
		if event.Type == HealthEventProcessStarted && event.ProcessID == "lifecycle-test" {
			hasStartEvent = true
		}
		if event.Type == HealthEventProcessExited && event.ProcessID == "lifecycle-test" {
			hasExitEvent = true
		}
	}

	if !hasStartEvent {
		t.Error("Expected process started event")
	}
	if !hasExitEvent {
		t.Error("Expected process exited event")
	}
}

func TestHealthSummaryReporting(t *testing.T) {
	// Create a test reporter
	testReporter := &TestReporter{
		events: make([]string, 0),
	}

	opts := ProcessManagerOptions{
		Verbose:  true,
		Reporter: testReporter,
	}

	pm := NewProcessManager(opts)

	// Test getting health summary
	summary := pm.GetHealthSummary()
	if summary.TotalProcesses != 0 {
		t.Errorf("Expected 0 total processes initially, got %d", summary.TotalProcesses)
	}

	// Test reporting health summary
	testReporter.ReportHealthSummary(summary)
	if !testReporter.HasEvent("ReportHealthSummary") {
		t.Error("Expected ReportHealthSummary to be called")
	}
}

func TestProcessStatusReportingDisabled(t *testing.T) {
	// Create process manager without verbose mode (lifecycle reporting disabled)
	opts := ProcessManagerOptions{
		Verbose: false,
	}

	pm := NewProcessManager(opts)
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Verify lifecycle reporting is disabled by default in non-verbose mode
	pmImpl := pm.(*processManager)
	if pmImpl.lifecycleReportingEnabled {
		t.Error("Expected lifecycle reporting to be disabled in non-verbose mode")
	}

	// Enable lifecycle reporting manually
	pm.EnableLifecycleReporting(true)
	if !pmImpl.lifecycleReportingEnabled {
		t.Error("Expected lifecycle reporting to be enabled after manual enable")
	}

	// Disable lifecycle reporting
	pm.EnableLifecycleReporting(false)
	if pmImpl.lifecycleReportingEnabled {
		t.Error("Expected lifecycle reporting to be disabled after manual disable")
	}
}

// TestReporter is a test implementation of the Reporter interface
type TestReporter struct {
	events          []string
	lifecycleEvents []HealthMonitorEvent
}

func (tr *TestReporter) ReportStart(totalCommands int) {
	tr.events = append(tr.events, "ReportStart")
}

func (tr *TestReporter) ReportExecutionComplete(status ExecutionStatus) {
	tr.events = append(tr.events, "ReportExecutionComplete")
}

func (tr *TestReporter) ReportExecutionSummary(status ExecutionStatus) {
	tr.events = append(tr.events, "ReportExecutionSummary")
}

func (tr *TestReporter) ReportCommandStart(commandName string, commandIndex int) {
	tr.events = append(tr.events, "ReportCommandStart")
}

func (tr *TestReporter) ReportCommandSuccess(result ExecutionResult, commandIndex int) {
	tr.events = append(tr.events, "ReportCommandSuccess")
}

func (tr *TestReporter) ReportCommandFailure(result ExecutionResult, commandIndex int) {
	tr.events = append(tr.events, "ReportCommandFailure")
}

func (tr *TestReporter) ReportProcessStatus(processes map[string]ProcessInfo) {
	tr.events = append(tr.events, "ReportProcessStatus")
}

func (tr *TestReporter) ReportProcessHealth(health map[string]ProcessHealth) {
	tr.events = append(tr.events, "ReportProcessHealth")
}

func (tr *TestReporter) ReportProcessLifecycleEvent(event HealthMonitorEvent) {
	tr.events = append(tr.events, "ReportProcessLifecycleEvent")
	tr.lifecycleEvents = append(tr.lifecycleEvents, event)
}

func (tr *TestReporter) ReportHealthSummary(summary HealthSummary) {
	tr.events = append(tr.events, "ReportHealthSummary")
}

func (tr *TestReporter) HasEvent(eventName string) bool {
	for _, event := range tr.events {
		if strings.Contains(event, eventName) {
			return true
		}
	}
	return false
}
