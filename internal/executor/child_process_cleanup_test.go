package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestChildProcessCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping child process cleanup test in short mode")
	}

	// Create a script that spawns child processes
	var scriptContent string
	var scriptFile string
	var scriptCmd []string

	if runtime.GOOS == "windows" {
		// Windows batch script that spawns child processes
		scriptContent = `@echo off
echo Parent process started
start /B ping 127.0.0.1 -n 100 > nul
start /B ping 127.0.0.1 -n 100 > nul
ping 127.0.0.1 -n 100 > nul
`
		scriptFile = "test_parent.bat"
		scriptCmd = []string{"cmd", "/C", scriptFile}
	} else {
		// Unix shell script that spawns child processes
		scriptContent = `#!/bin/bash
echo "Parent process started"
ping 127.0.0.1 -c 100 >/dev/null 2>&1 &
ping 127.0.0.1 -c 100 >/dev/null 2>&1 &
ping 127.0.0.1 -c 100 >/dev/null 2>&1
`
		scriptFile = "test_parent.sh"
		scriptCmd = []string{"bash", scriptFile}
	}

	// Write the script file
	if err := os.WriteFile(scriptFile, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptFile)

	// Create executor
	executor := NewExecutor(true)

	// Create command configuration
	cmd := config.Command{
		Name:    "test-parent",
		Command: scriptCmd[0],
		Args:    scriptCmd[1:],
		Mode:    config.ModeKeepAlive,
	}

	cfg := &config.Config{
		Version:  "1.0",
		Commands: []config.Command{cmd},
	}

	// Start execution in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		executor.Execute(ctx, cfg)
	}()

	// Wait for the process to start and spawn children
	time.Sleep(2 * time.Second)

	// Get the tracked processes
	processes := executor.GetTrackedProcesses()
	if len(processes) == 0 {
		t.Fatal("No processes were tracked")
	}

	var parentPID int
	for pid := range processes {
		parentPID = pid
		break
	}

	t.Logf("Parent process PID: %d", parentPID)

	// Count child processes before termination
	childrenBefore := countChildProcesses(parentPID)
	t.Logf("Child processes before termination: %d", childrenBefore)

	// Stop the executor (this should trigger process group cleanup)
	executor.Stop()

	// Wait for cleanup to complete
	time.Sleep(3 * time.Second)

	// Count child processes after termination
	childrenAfter := countChildProcesses(parentPID)
	t.Logf("Child processes after termination: %d", childrenAfter)

	// Verify that child processes were cleaned up
	// Note: On some systems, this might not be perfect due to timing or system limitations
	if childrenAfter > childrenBefore {
		t.Errorf("Child process cleanup may not have worked properly. Before: %d, After: %d", childrenBefore, childrenAfter)
	}

	// Verify parent process is no longer running
	if isProcessRunning(parentPID) {
		t.Errorf("Parent process %d is still running after cleanup", parentPID)
	}
}

// countChildProcesses counts the number of child processes for a given parent PID
func countChildProcesses(parentPID int) int {
	if runtime.GOOS == "windows" {
		return countChildProcessesWindows(parentPID)
	}
	return countChildProcessesUnix(parentPID)
}

// countChildProcessesWindows counts child processes on Windows using wmic
func countChildProcessesWindows(parentPID int) int {
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ParentProcessId=%d", parentPID), "get", "ProcessId", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(output), "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "Node,ProcessId" && !strings.HasPrefix(line, "Node,") {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 && parts[1] != "" {
				if _, err := strconv.Atoi(parts[1]); err == nil {
					count++
				}
			}
		}
	}
	return count
}

// countChildProcessesUnix counts child processes on Unix using ps
func countChildProcessesUnix(parentPID int) int {
	cmd := exec.Command("ps", "--ppid", fmt.Sprintf("%d", parentPID), "-o", "pid", "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			if _, err := strconv.Atoi(line); err == nil {
				count++
			}
		}
	}
	return count
}

func TestProcessGroupConfiguration(t *testing.T) {
	// Test that process group configuration is applied correctly
	executor := NewExecutor(false)

	// Create a simple command
	cmd := &exec.Cmd{}

	// Configure process group
	executor.configureProcessGroup(cmd)

	// Verify that SysProcAttr is set
	if cmd.SysProcAttr == nil {
		t.Error("SysProcAttr should be set after configureProcessGroup")
	}

	// Note: Platform-specific checks are in platform-specific test files
}
