package executor

import (
	"context"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// TestProcessManager_Integration demonstrates how the process manager
// can be used as a component within the existing executor architecture
func TestProcessManager_Integration(t *testing.T) {
	// Create a process manager
	pm := NewProcessManager(ProcessManagerOptions{
		Verbose:    true,
		WorkingDir: "",
		Timeout:    30 * time.Second,
	})

	ctx := context.Background()

	// Test mixed mode execution workflow
	commands := []config.Command{
		{
			Name:    "setup",
			Command: "echo",
			Args:    []string{"Setting up environment"},
			Mode:    config.ModeOnce,
		},
		{
			Name:    "background-service",
			Command: "sleep",
			Args:    []string{"0.2"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "verification",
			Command: "echo",
			Args:    []string{"Verifying setup"},
			Mode:    config.ModeOnce,
		},
	}

	// Execute commands sequentially
	for i, cmd := range commands {
		result, err := pm.ExecuteCommand(ctx, cmd)
		if err != nil {
			t.Fatalf("Command %d (%s) failed: %v", i+1, cmd.Name, err)
		}

		if !result.Success {
			t.Errorf("Command %d (%s) was not successful", i+1, cmd.Name)
		}

		t.Logf("Command %d (%s): %s", i+1, cmd.Name, result.Output)
	}

	// Verify process tracking
	activeProcs := pm.GetActiveProcesses()
	if len(activeProcs) != 1 {
		t.Errorf("Expected 1 active process, got %d", len(activeProcs))
	}

	if proc, exists := activeProcs["background-service"]; !exists {
		t.Error("Expected 'background-service' to be tracked")
	} else {
		t.Logf("Background service running with PID %d", proc.PID)
	}

	// Wait for background process to complete
	time.Sleep(300 * time.Millisecond)

	// Verify cleanup
	activeProcs = pm.GetActiveProcesses()
	if len(activeProcs) != 0 {
		t.Errorf("Expected no active processes after completion, got %d", len(activeProcs))
	}
}

// TestProcessManager_ErrorHandling demonstrates error handling capabilities
func TestProcessManager_ErrorHandling(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: false})
	ctx := context.Background()

	// Test command not found error
	cmd := config.Command{
		Name:    "nonexistent",
		Command: "nonexistent_command_xyz",
		Mode:    config.ModeOnce,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err == nil {
		t.Fatal("Expected error for nonexistent command")
	}

	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.ErrorDetail == nil {
		t.Fatal("Expected error detail to be populated")
	}

	if result.ErrorDetail.Type != ErrorTypeCommandNotFound {
		t.Errorf("Expected error type %s, got %s", ErrorTypeCommandNotFound, result.ErrorDetail.Type)
	}

	// Test keepAlive startup failure
	keepAliveCmd := config.Command{
		Name:    "nonexistent-keepalive",
		Command: "nonexistent_command_xyz",
		Mode:    config.ModeKeepAlive,
	}

	result, err = pm.ExecuteCommand(ctx, keepAliveCmd)
	if err == nil {
		t.Fatal("Expected error for nonexistent keepAlive command")
	}

	if result.Success {
		t.Error("Expected keepAlive command to fail")
	}

	if result.ErrorDetail.Type != ErrorTypeStartupFailure {
		t.Errorf("Expected error type %s, got %s", ErrorTypeStartupFailure, result.ErrorDetail.Type)
	}
}

// TestProcessManager_ProcessLifecycle demonstrates complete process lifecycle management
func TestProcessManager_ProcessLifecycle(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	// Start multiple long-running processes
	longRunningCommands := []config.Command{
		{
			Name:    "service-a",
			Command: "sleep",
			Args:    []string{"10"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "service-b",
			Command: "sleep",
			Args:    []string{"10"},
			Mode:    config.ModeKeepAlive,
		},
	}

	// Start all services
	for _, cmd := range longRunningCommands {
		result, err := pm.ExecuteCommand(ctx, cmd)
		if err != nil {
			t.Fatalf("Failed to start %s: %v", cmd.Name, err)
		}
		if !result.Success {
			t.Errorf("Expected %s to start successfully", cmd.Name)
		}
	}

	// Verify all processes are running
	activeProcs := pm.GetActiveProcesses()
	if len(activeProcs) != 2 {
		t.Fatalf("Expected 2 active processes, got %d", len(activeProcs))
	}

	// Terminate one specific process
	err := pm.TerminateProcess("service-a")
	if err != nil {
		t.Errorf("Failed to terminate service-a: %v", err)
	}

	// Wait for termination to complete
	time.Sleep(100 * time.Millisecond)

	// Verify only one process remains
	activeProcs = pm.GetActiveProcesses()
	if len(activeProcs) != 1 {
		t.Errorf("Expected 1 active process after terminating service-a, got %d", len(activeProcs))
	}

	if _, exists := activeProcs["service-b"]; !exists {
		t.Error("Expected service-b to still be running")
	}

	// Terminate all remaining processes
	err = pm.TerminateAll()
	if err != nil {
		t.Errorf("Failed to terminate all processes: %v", err)
	}

	// Wait for termination to complete
	time.Sleep(100 * time.Millisecond)

	// Verify no processes remain
	activeProcs = pm.GetActiveProcesses()
	if len(activeProcs) != 0 {
		t.Errorf("Expected no active processes after terminating all, got %d", len(activeProcs))
	}
}
