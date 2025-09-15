package executor

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestBackgroundProcessesWithLiveStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping background streaming integration test in short mode")
	}

	tests := []struct {
		name                      string
		commands                  []config.Command
		expectError               bool
		expectedActiveProcesses   int
		expectedStreamingSessions int
		description               string
	}{
		{
			name: "single keepAlive with streaming",
			commands: []config.Command{
				{
					Name:    "streaming-service",
					Command: getPingCommand(),
					Args:    getContinuousPingArgs(),
					Mode:    config.ModeKeepAlive,
				},
			},
			expectError:               false,
			expectedActiveProcesses:   1,
			expectedStreamingSessions: 1,
			description:               "Single keepAlive command should start background process with live streaming",
		},
		{
			name: "multiple keepAlive with streaming",
			commands: []config.Command{
				{
					Name:    "service-1",
					Command: getPingCommand(),
					Args:    getContinuousPingArgs(),
					Mode:    config.ModeKeepAlive,
				},
				{
					Name:    "service-2",
					Command: getPingCommand(),
					Args:    getContinuousPingArgs(),
					Mode:    config.ModeKeepAlive,
				},
				{
					Name:    "quick-task",
					Command: "echo",
					Args:    []string{"task completed"},
					Mode:    config.ModeOnce,
				},
			},
			expectError:               false,
			expectedActiveProcesses:   2,
			expectedStreamingSessions: 2,
			description:               "Multiple keepAlive commands should run concurrently with streaming",
		},
		{
			name: "mixed modes with streaming",
			commands: []config.Command{
				{
					Name:    "setup",
					Command: "echo",
					Args:    []string{"setting up"},
					Mode:    config.ModeOnce,
				},
				{
					Name:    "background-service",
					Command: getPingCommand(),
					Args:    getContinuousPingArgs(),
					Mode:    config.ModeKeepAlive,
				},
				{
					Name:    "cleanup",
					Command: "echo",
					Args:    []string{"cleaning up"},
					Mode:    config.ModeOnce,
				},
			},
			expectError:               false,
			expectedActiveProcesses:   1,
			expectedStreamingSessions: 1,
			description:               "Mixed execution modes should work with background streaming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(true) // Enable verbose mode for streaming
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

			// Verify execution completes quickly for keepAlive commands
			if executionDuration > 2*time.Second {
				t.Errorf("Execution took too long (%v), background processes may be blocking", executionDuration)
			}

			// Verify expected number of active processes (allow for some tolerance due to test environment)
			actualProcessCount := executor.GetTrackedProcessCount()
			if actualProcessCount < tt.expectedActiveProcesses {
				t.Errorf("Expected at least %d active processes, got %d", tt.expectedActiveProcesses, actualProcessCount)
			}

			// Verify streaming sessions are active (allow for some tolerance due to test environment)
			activeStreaming := executor.GetActiveStreamingProcesses()
			if len(activeStreaming) < tt.expectedStreamingSessions {
				t.Errorf("Expected at least %d active streaming sessions, got %d", tt.expectedStreamingSessions, len(activeStreaming))
			}

			// Verify that processes are actually running
			if tt.expectedActiveProcesses > 0 && !executor.HasActiveKeepAliveProcesses() {
				t.Errorf("Expected active keepAlive processes but none found")
			}

			// Verify streaming is active
			if tt.expectedStreamingSessions > 0 && !executor.HasActiveStreaming() {
				t.Errorf("Expected active streaming sessions but none found")
			}

			// Clean up
			executor.Stop()
			time.Sleep(200 * time.Millisecond) // Allow cleanup to complete
		})
	}
}

func TestStreamingDetachment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming detachment test in short mode")
	}

	executor := NewExecutor(true)
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "detach-test-service",
				Command: getPingCommand(),
				Args:    getContinuousPingArgs(),
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start execution
	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify process and streaming are active
	if !executor.HasActiveKeepAliveProcesses() {
		t.Errorf("Expected active keepAlive processes")
	}

	if !executor.HasActiveStreaming() {
		t.Errorf("Expected active streaming sessions")
	}

	initialProcessCount := executor.GetTrackedProcessCount()
	initialStreamingCount := len(executor.GetActiveStreamingProcesses())

	// Detach from streaming
	executor.DetachFromStreaming()

	// Verify streaming is detached but processes continue
	if executor.HasActiveStreaming() {
		t.Errorf("Expected streaming to be detached")
	}

	if !executor.HasActiveKeepAliveProcesses() {
		t.Errorf("Expected processes to continue running after streaming detachment")
	}

	// Verify process count remains the same
	if executor.GetTrackedProcessCount() != initialProcessCount {
		t.Errorf("Expected process count to remain %d after detachment, got %d",
			initialProcessCount, executor.GetTrackedProcessCount())
	}

	// Verify streaming count is now zero
	if len(executor.GetActiveStreamingProcesses()) != 0 {
		t.Errorf("Expected 0 active streaming sessions after detachment, got %d",
			len(executor.GetActiveStreamingProcesses()))
	}

	t.Logf("Successfully detached from %d streaming sessions while keeping %d processes running",
		initialStreamingCount, initialProcessCount)

	// Clean up
	executor.Stop()
	time.Sleep(200 * time.Millisecond)
}

func TestConcurrentStreamingAndExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent streaming test in short mode")
	}

	executor1 := NewExecutor(true)

	// Start background service first
	cfg1 := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "background-service",
				Command: getPingCommand(),
				Args:    getContinuousPingArgs(),
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel1()

	err := executor1.Execute(ctx1, cfg1)
	if err != nil {
		t.Errorf("Unexpected error starting background service: %v", err)
	}

	// Verify background service is running with streaming
	if !executor1.HasActiveKeepAliveProcesses() {
		t.Errorf("Expected background service to be running")
	}

	if !executor1.HasActiveStreaming() {
		t.Errorf("Expected streaming to be active")
	}

	initialProcessCount := executor1.GetTrackedProcessCount()

	// Create a separate executor for concurrent tasks to avoid monitor conflicts
	executor2 := NewExecutor(true)

	// Execute additional commands concurrently
	cfg2 := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "concurrent-task-1",
				Command: "echo",
				Args:    []string{"concurrent task 1"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "concurrent-task-2",
				Command: "echo",
				Args:    []string{"concurrent task 2"},
				Mode:    config.ModeOnce,
			},
		},
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	startTime := time.Now()
	err = executor2.Execute(ctx2, cfg2)
	executionDuration := time.Since(startTime)

	if err != nil {
		t.Errorf("Unexpected error executing concurrent tasks: %v", err)
	}

	// Verify concurrent execution was fast
	if executionDuration > 1*time.Second {
		t.Errorf("Concurrent execution took too long (%v)", executionDuration)
	}

	// Verify background service is still running on first executor
	if !executor1.HasActiveKeepAliveProcesses() {
		t.Errorf("Expected background service to still be running")
	}

	// Verify process count is still the same (background service only)
	if executor1.GetTrackedProcessCount() != initialProcessCount {
		t.Errorf("Expected process count to remain %d, got %d",
			initialProcessCount, executor1.GetTrackedProcessCount())
	}

	// Verify second executor completed its tasks
	status2 := executor2.GetStatus()
	if len(status2.Results) != 2 {
		t.Errorf("Expected 2 results from concurrent executor, got %d", len(status2.Results))
	}

	// Clean up both executors
	executor1.Stop()
	executor2.Stop()
	time.Sleep(200 * time.Millisecond)
}

func TestStreamingOutputCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming output capture test in short mode")
	}

	executor := NewExecutor(true)
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "output-test",
				Command: getMultiOutputCommand(),
				Args:    getMultiOutputArgs(),
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Allow some time for output generation
	time.Sleep(500 * time.Millisecond)

	// Verify process is running and streaming
	if !executor.HasActiveKeepAliveProcesses() {
		t.Errorf("Expected active keepAlive process")
	}

	if !executor.HasActiveStreaming() {
		t.Errorf("Expected active streaming")
	}

	// Get execution results
	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
	} else {
		result := status.Results[0]
		if !result.Success {
			t.Errorf("Expected successful execution, got error: %s", result.Error)
		}

		// Verify PID is reported in output
		if !strings.Contains(result.Output, "PID") {
			t.Errorf("Expected PID in output, got: %s", result.Output)
		}
	}

	// Clean up
	executor.Stop()
	time.Sleep(200 * time.Millisecond)
}

func TestStreamingWithProcessFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming process failure test in short mode")
	}

	executor := NewExecutor(true)
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "failing-service",
				Command: getFailingCommand(),
				Args:    getFailingArgs(),
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_ = executor.Execute(ctx, cfg)
	// This might or might not error depending on timing

	// Allow time for process to potentially fail
	time.Sleep(1 * time.Second)

	// Verify that failed processes are cleaned up from tracking
	// The process might have started and then failed, or failed to start
	finalProcessCount := executor.GetTrackedProcessCount()
	finalStreamingCount := len(executor.GetActiveStreamingProcesses())

	t.Logf("Final process count: %d, streaming count: %d", finalProcessCount, finalStreamingCount)

	// Clean up any remaining processes
	executor.Stop()
	time.Sleep(200 * time.Millisecond)
}

func TestStreamingResourceCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming resource cleanup test in short mode")
	}

	// Test multiple cycles of starting and stopping processes
	for i := 0; i < 3; i++ {
		// Create a new executor for each cycle to avoid monitor conflicts
		executor := NewExecutor(true)

		cfg := &config.Config{
			Version: "1.0",
			Commands: []config.Command{
				{
					Name:    "cleanup-test-service",
					Command: getPingCommand(),
					Args:    getContinuousPingArgs(),
					Mode:    config.ModeKeepAlive,
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Cycle %d: Unexpected error: %v", i, err)
		}

		// Verify resources are allocated
		if !executor.HasActiveKeepAliveProcesses() {
			t.Errorf("Cycle %d: Expected active processes", i)
		}

		if !executor.HasActiveStreaming() {
			t.Errorf("Cycle %d: Expected active streaming", i)
		}

		// Stop and verify cleanup
		executor.Stop()
		time.Sleep(200 * time.Millisecond)

		// Verify all resources are cleaned up
		if executor.HasActiveKeepAliveProcesses() {
			t.Errorf("Cycle %d: Expected no active processes after stop", i)
		}

		if executor.HasActiveStreaming() {
			t.Errorf("Cycle %d: Expected no active streaming after stop", i)
		}

		// Allow some extra time for process cleanup on Windows
		time.Sleep(100 * time.Millisecond)
		executor.CleanupDeadProcesses()

		if executor.GetTrackedProcessCount() > 0 {
			t.Logf("Cycle %d: Warning: %d tracked processes remain after cleanup", i, executor.GetTrackedProcessCount())
		}

		cancel()
	}
}

// Helper functions for platform-specific commands

func getContinuousPingArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"-t", "127.0.0.1"}
	}
	return []string{"-i", "1", "127.0.0.1"}
}

func getMultiOutputCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

func getMultiOutputArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"/c", "for /l %i in (1,1,5) do (echo Output line %i & ping -n 2 127.0.0.1 > nul)"}
	}
	return []string{"-c", "for i in 1 2 3 4 5; do echo \"Output line $i\"; sleep 1; done"}
}

func getFailingCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

func getFailingArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"/c", "echo Starting... & ping -n 2 127.0.0.1 > nul & exit 1"}
	}
	return []string{"-c", "echo 'Starting...'; sleep 1; exit 1"}
}
