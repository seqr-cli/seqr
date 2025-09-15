package executor

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestLiveStreamingMaintenance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live streaming test in short mode")
	}

	tests := []struct {
		name        string
		command     config.Command
		expectError bool
		minDuration time.Duration
	}{
		{
			name: "live streaming with once mode",
			command: config.Command{
				Name:    "test-live-once",
				Command: getPingCommand(),
				Args:    getPingArgs(3),
				Mode:    config.ModeOnce,
			},
			expectError: false,
			minDuration: 2 * time.Second,
		},
		{
			name: "live streaming with keepAlive mode",
			command: config.Command{
				Name:    "test-live-keepalive",
				Command: getPingCommand(),
				Args:    getPingArgs(2),
				Mode:    config.ModeKeepAlive,
			},
			expectError: false,
			minDuration: 1 * time.Millisecond, // keepAlive starts quickly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(true) // Enable verbose mode for streaming

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cfg := &config.Config{
				Version:  "1.0",
				Commands: []config.Command{tt.command},
			}

			startTime := time.Now()
			err := executor.Execute(ctx, cfg)
			duration := time.Since(startTime)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify execution took at least the minimum expected duration
			// This indicates that streaming was maintained during execution
			if duration < tt.minDuration {
				t.Errorf("Execution completed too quickly (%v), expected at least %v", duration, tt.minDuration)
			}

			status := executor.GetStatus()
			if len(status.Results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(status.Results))
				return
			}

			result := status.Results[0]
			if !tt.expectError && !result.Success {
				t.Errorf("Expected success but got failure: %s", result.Error)
			}

			// For keepAlive commands, verify process was started
			if tt.command.Mode == config.ModeKeepAlive && !strings.Contains(result.Output, "PID") {
				t.Errorf("Expected keepAlive command to report PID, got: %s", result.Output)
			}

			// Clean up any running processes
			executor.Stop()
		})
	}
}

func TestStreamingWithInterruption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming interruption test in short mode")
	}

	executor := NewExecutor(true)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-interruption",
				Command: getPingCommand(),
				Args:    getPingArgs(10), // Long-running command
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	// Start execution in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- executor.Execute(ctx, cfg)
	}()

	// Wait a bit for the command to start
	time.Sleep(500 * time.Millisecond)

	// Stop the executor to test graceful interruption
	executor.Stop()

	// Wait for execution to complete
	select {
	case err := <-errChan:
		// Execution should complete without panic
		if err != nil && !strings.Contains(err.Error(), "execution stopped") {
			t.Errorf("Unexpected error during interruption: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Execution did not complete within timeout after interruption")
	}
}

func TestStreamingBufferHandling(t *testing.T) {
	executor := NewExecutor(true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a command that produces multiple lines of output
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-buffer",
				Command: getEchoCommand(),
				Args:    getMultiLineEchoArgs(),
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
		return
	}

	result := status.Results[0]
	if !result.Success {
		t.Errorf("Expected success but got failure: %s", result.Error)
	}

	// Verify that multi-line output was captured
	if !strings.Contains(result.Output, "Line") {
		t.Errorf("Expected multi-line output to be captured, got: %s", result.Output)
	}
}

// Helper functions to get platform-specific commands
func getPingCommand() string {
	if runtime.GOOS == "windows" {
		return "ping"
	}
	return "ping"
}

func getPingArgs(count int) []string {
	if runtime.GOOS == "windows" {
		return []string{"-n", "3", "127.0.0.1"}
	}
	return []string{"-c", "3", "127.0.0.1"}
}

func getEchoCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "echo"
}

func getMultiLineEchoArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"/c", "echo Line 1 & echo Line 2 & echo Line 3"}
	}
	return []string{"-e", "Line 1\\nLine 2\\nLine 3"}
}
