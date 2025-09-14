package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestProcessStatusReporting_Integration(t *testing.T) {
	// Create a buffer to capture console output
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	// Create executor with the console reporter
	executor := NewExecutor(ExecutorOptions{
		Verbose:  true,
		Reporter: reporter,
	})
	ctx := context.Background()

	// Start health monitoring
	err := executor.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer executor.StopHealthMonitoring()

	// Create a configuration with mixed command types
	cfg := &config.Config{
		Commands: []config.Command{
			{
				Name:    "setup-task",
				Command: "echo",
				Args:    []string{"Setting up environment"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "web-server",
				Command: "sleep",
				Args:    []string{"0.3"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "database",
				Command: "sleep",
				Args:    []string{"0.4"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "config-check",
				Command: "echo",
				Args:    []string{"Configuration validated"},
				Mode:    config.ModeOnce,
			},
		},
	}

	// Execute the configuration
	err = executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	// Report current process status
	executor.ReportProcessStatus()

	// Report health status
	executor.ReportHealthStatus()

	// Wait for keepAlive processes to complete
	time.Sleep(500 * time.Millisecond)

	// Report final status
	executor.ReportProcessStatus()
	executor.ReportHealthStatus()

	// Verify output contains expected elements
	output := buf.String()

	// Check for execution output
	if !strings.Contains(output, "Starting execution of 4 command(s)") {
		t.Error("Expected execution start message")
	}

	// Check for command execution
	if !strings.Contains(output, "setup-task") {
		t.Error("Expected setup-task command to be mentioned")
	}
	if !strings.Contains(output, "web-server") {
		t.Error("Expected web-server command to be mentioned")
	}

	// Check for process status reporting
	if !strings.Contains(output, "Active Processes:") && !strings.Contains(output, "No active processes") {
		t.Error("Expected process status reporting")
	}

	// Check for health status reporting
	if !strings.Contains(output, "Process Health Status:") {
		t.Error("Expected health status reporting")
	}

	// Check for lifecycle events (process started/exited)
	if !strings.Contains(output, "🚀") || !strings.Contains(output, "🏁") {
		t.Error("Expected lifecycle event icons in output")
	}

	// Verify execution summary
	if !strings.Contains(output, "All commands completed successfully") {
		t.Error("Expected successful completion message")
	}

	t.Logf("Integration test output:\n%s", output)
}

func TestProcessStatusReporting_RealTimeUpdates(t *testing.T) {
	// Create a buffer to capture console output
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	// Create process manager with the console reporter
	pm := NewProcessManager(ProcessManagerOptions{
		Verbose:  true,
		Reporter: reporter,
	})
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Execute a long-running keepAlive command
	cmd := config.Command{
		Name:    "long-service",
		Command: "sleep",
		Args:    []string{"0.5"},
		Mode:    config.ModeKeepAlive,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}

	// Report status while process is running
	pm.ReportCurrentStatus()
	pm.ReportHealthStatus()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Report status again
	pm.ReportCurrentStatus()
	pm.ReportHealthStatus()

	// Wait for process to complete
	time.Sleep(500 * time.Millisecond)

	// Report final status
	pm.ReportCurrentStatus()
	pm.ReportHealthStatus()

	// Get health summary
	summary := pm.GetHealthSummary()
	reporter.ReportHealthSummary(summary)

	// Verify output
	output := buf.String()

	// Check for process information
	if !strings.Contains(output, "long-service") {
		t.Error("Expected process name in output")
	}

	// Check for PID information
	if !strings.Contains(output, "PID:") {
		t.Error("Expected PID information in output")
	}

	// Check for uptime information
	if !strings.Contains(output, "Uptime:") {
		t.Error("Expected uptime information in output")
	}

	// Check for health summary
	if !strings.Contains(output, "Health Summary:") {
		t.Error("Expected health summary in output")
	}

	t.Logf("Real-time updates test output:\n%s", output)
}

func TestProcessStatusReporting_LifecycleEvents(t *testing.T) {
	// Create a buffer to capture console output
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	// Create process manager with lifecycle reporting enabled
	pm := NewProcessManager(ProcessManagerOptions{
		Verbose:  true,
		Reporter: reporter,
	})
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Ensure lifecycle reporting is enabled
	pm.EnableLifecycleReporting(true)

	// Execute multiple short-lived processes
	commands := []config.Command{
		{
			Name:    "service-1",
			Command: "sleep",
			Args:    []string{"0.1"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "service-2",
			Command: "sleep",
			Args:    []string{"0.15"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "service-3",
			Command: "sleep",
			Args:    []string{"0.2"},
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

	// Wait for all processes to complete and events to be generated
	time.Sleep(300 * time.Millisecond)

	// Verify lifecycle events in output
	output := buf.String()

	// Check for process started events
	if !strings.Contains(output, "🚀") {
		t.Error("Expected process started events (🚀)")
	}

	// Check for process exited events
	if !strings.Contains(output, "🏁") {
		t.Error("Expected process exited events (🏁)")
	}

	// Check for all service names in lifecycle events
	for _, cmd := range commands {
		if !strings.Contains(output, cmd.Name) {
			t.Errorf("Expected lifecycle events for %s", cmd.Name)
		}
	}

	// Check for timestamp format in events
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Error("Expected timestamp format in lifecycle events")
	}

	t.Logf("Lifecycle events test output:\n%s", output)
}

func TestProcessStatusReporting_ErrorScenarios(t *testing.T) {
	// Create a buffer to capture console output
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	// Create process manager
	pm := NewProcessManager(ProcessManagerOptions{
		Verbose:  true,
		Reporter: reporter,
	})
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	// Try to execute a command that will fail to start
	cmd := config.Command{
		Name:    "failing-service",
		Command: "nonexistent_command_xyz",
		Mode:    config.ModeKeepAlive,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err == nil {
		t.Fatal("Expected error for nonexistent command")
	}

	if result.Success {
		t.Error("Expected command to fail")
	}

	// Report status (should show no active processes)
	pm.ReportCurrentStatus()
	pm.ReportHealthStatus()

	// Verify error handling in output
	output := buf.String()

	// Should show no active processes
	if !strings.Contains(output, "No active processes") {
		t.Error("Expected 'No active processes' message")
	}

	// Should show no monitored processes
	if !strings.Contains(output, "No processes being monitored") {
		t.Error("Expected 'No processes being monitored' message")
	}

	t.Logf("Error scenarios test output:\n%s", output)
}
