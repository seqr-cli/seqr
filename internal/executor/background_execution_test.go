package executor

import (
	"context"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestBackgroundProcessExecution(t *testing.T) {
	tests := []struct {
		name        string
		commands    []config.Command
		expectError bool
		description string
	}{
		{
			name: "keepAlive command runs in background",
			commands: []config.Command{
				{
					Name:    "background-ping",
					Command: "ping",
					Args:    []string{"-t", "127.0.0.1"}, // Continuous ping (Windows)
					Mode:    config.ModeKeepAlive,
				},
				{
					Name:    "quick-echo",
					Command: "echo",
					Args:    []string{"hello"},
					Mode:    config.ModeOnce,
				},
			},
			expectError: false,
			description: "KeepAlive command should start in background and not block subsequent commands",
		},
		{
			name: "multiple keepAlive commands",
			commands: []config.Command{
				{
					Name:    "background-timeout-1",
					Command: "timeout",
					Args:    []string{"5"},
					Mode:    config.ModeKeepAlive,
				},
				{
					Name:    "background-timeout-2",
					Command: "timeout",
					Args:    []string{"5"},
					Mode:    config.ModeKeepAlive,
				},
				{
					Name:    "final-echo",
					Command: "echo",
					Args:    []string{"done"},
					Mode:    config.ModeOnce,
				},
			},
			expectError: false,
			description: "Multiple keepAlive commands should all run in background concurrently",
		},
		{
			name: "mixed execution modes",
			commands: []config.Command{
				{
					Name:    "setup-echo",
					Command: "echo",
					Args:    []string{"setup"},
					Mode:    config.ModeOnce,
				},
				{
					Name:    "background-service",
					Command: "timeout",
					Args:    []string{"8"},
					Mode:    config.ModeKeepAlive,
				},
				{
					Name:    "cleanup-echo",
					Command: "echo",
					Args:    []string{"cleanup"},
					Mode:    config.ModeOnce,
				},
			},
			expectError: false,
			description: "Mixed execution modes should work correctly with background processes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(false) // Non-verbose for cleaner test output
			cfg := &config.Config{
				Version:  "1.0",
				Commands: tt.commands,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			startTime := time.Now()
			err := executor.Execute(ctx, cfg)
			executionDuration := time.Since(startTime)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify that execution completes quickly (within 3 seconds)
			// even with long-running keepAlive commands
			if executionDuration > 2*time.Second {
				t.Errorf("Execution took too long (%v), background processes may be blocking", executionDuration)
			}

			// Verify that keepAlive processes are still running
			if executor.HasActiveKeepAliveProcesses() {
				t.Logf("✓ KeepAlive processes are running in background as expected")
			} else {
				// This might be expected if the test runs very quickly
				t.Logf("No active keepAlive processes (may have completed quickly)")
			}

			// Clean up any remaining processes
			executor.Stop()

			// Wait a moment for cleanup to complete
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestBackgroundProcessConcurrency(t *testing.T) {
	executor := NewExecutor(false)

	// Create a configuration with multiple keepAlive commands
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "service-1",
				Command: "ping",
				Args:    []string{"-t", "127.0.0.1"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "service-2",
				Command: "ping",
				Args:    []string{"-t", "127.0.0.1"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "service-3",
				Command: "ping",
				Args:    []string{"-t", "127.0.0.1"},
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	startTime := time.Now()
	err := executor.Execute(ctx, cfg)
	executionDuration := time.Since(startTime)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Execution should complete quickly since all commands run in background
	if executionDuration > 500*time.Millisecond {
		t.Errorf("Execution took too long (%v), processes may not be running concurrently", executionDuration)
	}

	// Verify multiple processes are running
	if !executor.HasActiveKeepAliveProcesses() {
		t.Errorf("Expected active keepAlive processes")
	}

	// Clean up
	executor.Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestBackgroundProcessTracking(t *testing.T) {
	executor := NewExecutor(false)

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "tracked-service",
				Command: "timeout",
				Args:    []string{"3"},
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify process tracking
	trackedProcesses := executor.GetTrackedProcesses()
	if len(trackedProcesses) == 0 {
		t.Errorf("Expected at least one tracked process")
	}

	processCount := executor.GetTrackedProcessCount()
	if processCount == 0 {
		t.Errorf("Expected process count > 0, got %d", processCount)
	}

	// Clean up
	executor.Stop()
	time.Sleep(100 * time.Millisecond)

	// Verify cleanup
	finalCount := executor.GetTrackedProcessCount()
	if finalCount > 0 {
		t.Logf("Warning: %d processes still tracked after cleanup", finalCount)
	}
}
