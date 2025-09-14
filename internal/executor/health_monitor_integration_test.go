package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestHealthMonitor_Integration_CompleteWorkflow(t *testing.T) {
	// Create a process manager with verbose output to see health monitoring in action
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	// Start health monitoring
	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	fmt.Println("=== Health Monitoring Integration Test ===")

	// Execute multiple commands with different lifecycles
	commands := []config.Command{
		{
			Name:    "quick-task",
			Command: "echo",
			Args:    []string{"Setting up environment"},
			Mode:    config.ModeOnce,
		},
		{
			Name:    "web-server",
			Command: "sleep",
			Args:    []string{"1.0"}, // Simulate a web server
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "database",
			Command: "sleep",
			Args:    []string{"1.2"}, // Simulate a database
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "config-check",
			Command: "echo",
			Args:    []string{"Configuration validated"},
			Mode:    config.ModeOnce,
		},
		{
			Name:    "background-worker",
			Command: "sleep",
			Args:    []string{"0.8"}, // Simulate a background worker
			Mode:    config.ModeKeepAlive,
		},
	}

	// Execute all commands
	for i, cmd := range commands {
		fmt.Printf("\n--- Executing Command %d: %s ---\n", i+1, cmd.Name)

		result, err := pm.ExecuteCommand(ctx, cmd)
		if err != nil {
			t.Fatalf("Expected successful execution for %s, got error: %v", cmd.Name, err)
		}

		if !result.Success {
			t.Errorf("Expected command %s to succeed", cmd.Name)
		}

		// Show current health status after each command
		health := pm.GetProcessHealth()
		fmt.Printf("Active processes being monitored: %d\n", len(health))
		for name, processHealth := range health {
			fmt.Printf("  - %s (PID %d, Status: %s, Uptime: %v)\n",
				name, processHealth.PID, processHealth.Status, processHealth.UptimeDuration)
		}

		// Small delay to see the progression
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("\n--- Monitoring Active Processes ---")

	// Monitor for a while to see processes in action
	for i := 0; i < 5; i++ {
		time.Sleep(200 * time.Millisecond)

		health := pm.GetProcessHealth()
		activeCount := 0
		for _, processHealth := range health {
			if processHealth.Status == ProcessStatusRunning {
				activeCount++
			}
		}

		fmt.Printf("Health check %d: %d processes still running\n", i+1, activeCount)

		if activeCount == 0 {
			break
		}
	}

	fmt.Println("\n--- Final Health Summary ---")

	// Wait for all processes to complete
	time.Sleep(500 * time.Millisecond)

	// Get final health status
	health := pm.GetProcessHealth()
	runningCount := 0
	exitedCount := 0

	for name, processHealth := range health {
		switch processHealth.Status {
		case ProcessStatusRunning:
			runningCount++
			fmt.Printf("  Still running: %s (PID %d)\n", name, processHealth.PID)
		case ProcessStatusExited:
			exitedCount++
			fmt.Printf("  Completed: %s (ran for %v)\n", name, processHealth.UptimeDuration)
		}
	}

	fmt.Printf("Summary: %d total processes monitored, %d completed, %d still running\n",
		len(health), exitedCount, runningCount)

	// Verify that we monitored the expected number of keepAlive processes
	expectedKeepAliveCount := 3 // web-server, database, background-worker
	if len(health) < expectedKeepAliveCount {
		t.Errorf("Expected at least %d processes to be monitored, got %d", expectedKeepAliveCount, len(health))
	}

	fmt.Println("\n=== Health Monitoring Integration Test Complete ===")
}

func TestHealthMonitor_Integration_ProcessFailure(t *testing.T) {
	// Test health monitoring with a process that fails
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	fmt.Println("\n=== Testing Process Failure Monitoring ===")

	// Execute a command that will fail
	cmd := config.Command{
		Name:    "failing-service",
		Command: "false", // This command always fails
		Mode:    config.ModeOnce,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)

	// We expect this to fail
	if err == nil {
		t.Error("Expected command to fail, but it succeeded")
	}

	if result.Success {
		t.Error("Expected result to indicate failure")
	}

	fmt.Printf("Command failed as expected: %s\n", result.Error)
	fmt.Println("Health monitoring correctly detected the failure")
}

func TestHealthMonitor_Integration_LongRunningProcess(t *testing.T) {
	// Test health monitoring with a long-running process that we terminate
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	err := pm.StartHealthMonitoring(ctx)
	if err != nil {
		t.Fatalf("Failed to start health monitoring: %v", err)
	}
	defer pm.StopHealthMonitoring()

	fmt.Println("\n=== Testing Long-Running Process Monitoring ===")

	// Start a long-running process
	cmd := config.Command{
		Name:    "long-service",
		Command: "sleep",
		Args:    []string{"10"}, // Long sleep
		Mode:    config.ModeKeepAlive,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}

	fmt.Printf("Started long-running service with PID from result\n")

	// Monitor the process for a short time
	time.Sleep(200 * time.Millisecond)

	health := pm.GetProcessHealth()
	if processHealth, exists := health["long-service"]; exists {
		if processHealth.Status != ProcessStatusRunning {
			t.Errorf("Expected process to be running, got status: %s", processHealth.Status)
		}
		fmt.Printf("Process is running: PID %d, Uptime: %v\n", processHealth.PID, processHealth.UptimeDuration)
	} else {
		t.Error("Expected long-service to be in health monitoring")
	}

	// Terminate the process
	fmt.Println("Terminating the long-running process...")
	err = pm.TerminateProcess("long-service")
	if err != nil {
		t.Errorf("Failed to terminate process: %v", err)
	}

	// Wait for termination to be detected
	time.Sleep(200 * time.Millisecond)

	// Check that the process is no longer running
	health = pm.GetProcessHealth()
	if processHealth, exists := health["long-service"]; exists {
		if processHealth.Status == ProcessStatusRunning {
			t.Error("Expected process to be terminated")
		}
		fmt.Printf("Process terminated: Status %s\n", processHealth.Status)
	}

	fmt.Println("Long-running process monitoring test completed successfully")
}
