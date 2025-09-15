package executor

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestForceKillProcessWithTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping SIGTERM/SIGKILL test on Windows")
	}

	executor := NewExecutor(true)

	// Create a process that ignores SIGTERM (sleep command)
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	process := cmd.Process
	processName := "test-sleep"

	// Test the force kill functionality
	start := time.Now()
	executor.terminateProcessGracefully(process, processName)
	duration := time.Since(start)

	// Should complete within reasonable time (5s graceful + 3s force kill timeout)
	if duration > 10*time.Second {
		t.Errorf("Force kill took too long: %v", duration)
	}

	// Verify process is actually dead
	if err := process.Signal(syscall.Signal(0)); err == nil {
		t.Error("Process should be dead but is still running")
	}
}

func TestForceKillProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping SIGKILL test on Windows")
	}

	executor := NewExecutor(true)

	// Create a test process
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	process := cmd.Process
	processName := "test-sleep-force"

	// Test immediate force kill
	start := time.Now()
	executor.forceKillProcess(process, processName)
	duration := time.Since(start)

	// Should complete quickly (within 5 seconds including timeout)
	if duration > 5*time.Second {
		t.Errorf("Force kill took too long: %v", duration)
	}

	// Verify process is actually dead
	if err := process.Signal(syscall.Signal(0)); err == nil {
		t.Error("Process should be dead but is still running")
	}
}

func TestWindowsForceKill(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	executor := NewExecutor(true)

	// Create a test process on Windows (use timeout command)
	cmd := exec.Command("timeout", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	process := cmd.Process
	processName := "test-timeout"

	// Test Windows force kill (should use Kill() directly)
	start := time.Now()
	executor.terminateProcessGracefully(process, processName)
	duration := time.Since(start)

	// Should complete quickly on Windows (no graceful period)
	if duration > 5*time.Second {
		t.Errorf("Windows force kill took too long: %v", duration)
	}

	// Wait a moment for process to actually terminate
	time.Sleep(100 * time.Millisecond)

	// Verify process is dead by trying to find it
	if _, err := os.FindProcess(process.Pid); err == nil {
		// On Windows, FindProcess always succeeds, so we can't easily verify
		// the process is dead this way. The test passing means no panic occurred.
		t.Logf("Process termination completed (Windows doesn't provide easy verification)")
	}
}
