package executor

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestExecutorProcessTracking(t *testing.T) {
	executor := NewExecutor(true)

	// Clean up any existing tracking file
	defer os.Remove(executor.tracker.filePath)

	// Create ping arguments based on platform
	var pingArgs []string
	if runtime.GOOS == "windows" {
		pingArgs = []string{"127.0.0.1", "-n", "10"}
	} else {
		pingArgs = []string{"127.0.0.1", "-c", "10"}
	}

	// Create a config with a keepAlive command
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-tracking",
				Command: "ping",
				Args:    pingArgs,
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	// Start execution in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		executor.Execute(ctx, cfg)
	}()

	// Wait a bit for the process to start
	time.Sleep(100 * time.Millisecond)

	// Verify process is tracked
	trackedProcesses := executor.GetTrackedProcesses()
	if len(trackedProcesses) != 1 {
		t.Fatalf("Expected 1 tracked process, got %d", len(trackedProcesses))
	}

	// Get the process info
	var processInfo *ProcessInfo
	for _, info := range trackedProcesses {
		processInfo = info
		break
	}

	if processInfo.Name != "test-tracking" {
		t.Errorf("Expected process name 'test-tracking', got '%s'", processInfo.Name)
	}

	if processInfo.Command != "ping" {
		t.Errorf("Expected command 'ping', got '%s'", processInfo.Command)
	}

	if processInfo.Mode != "keepAlive" {
		t.Errorf("Expected mode 'keepAlive', got '%s'", processInfo.Mode)
	}

	// Verify process count
	count := executor.GetTrackedProcessCount()
	if count != 1 {
		t.Errorf("Expected tracked process count 1, got %d", count)
	}

	// Stop the executor
	executor.Stop()

	// Wait a bit for cleanup
	time.Sleep(200 * time.Millisecond)

	// Verify process is no longer tracked
	count = executor.GetTrackedProcessCount()
	if count != 0 {
		t.Errorf("Expected tracked process count 0 after stop, got %d", count)
	}
}

func TestProcessTrackerPersistence(t *testing.T) {
	// Create first executor and start a process
	executor1 := NewExecutor(false)
	defer os.Remove(executor1.tracker.filePath)

	// Create ping arguments based on platform
	var pingArgs []string
	if runtime.GOOS == "windows" {
		pingArgs = []string{"127.0.0.1", "-n", "5"}
	} else {
		pingArgs = []string{"127.0.0.1", "-c", "5"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-persistence",
				Command: "ping",
				Args:    pingArgs,
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		executor1.Execute(ctx, cfg)
	}()

	// Wait for process to start
	time.Sleep(100 * time.Millisecond)

	// Verify process is tracked
	count1 := executor1.GetTrackedProcessCount()
	if count1 != 1 {
		t.Fatalf("Expected 1 tracked process in first executor, got %d", count1)
	}

	// Create second executor (should load from file)
	executor2 := NewExecutor(false)

	// Verify process is loaded in second executor
	count2 := executor2.GetTrackedProcessCount()
	if count2 != 1 {
		t.Errorf("Expected 1 tracked process in second executor, got %d", count2)
	}

	// Get process info from second executor
	processes := executor2.GetTrackedProcesses()
	if len(processes) != 1 {
		t.Fatalf("Expected 1 process in second executor, got %d", len(processes))
	}

	var processInfo *ProcessInfo
	for _, info := range processes {
		processInfo = info
		break
	}

	if processInfo.Name != "test-persistence" {
		t.Errorf("Expected process name 'test-persistence', got '%s'", processInfo.Name)
	}

	// Stop first executor
	executor1.Stop()

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)
}
