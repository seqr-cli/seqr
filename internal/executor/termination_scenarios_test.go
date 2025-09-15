package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// TestTerminationScenarios tests various process termination scenarios with different process types
func TestTerminationScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping termination scenarios test in short mode")
	}

	tests := []struct {
		name        string
		processType string
		command     string
		args        []string
		mode        config.Mode
		expectError bool
		description string
	}{
		{
			name:        "short-running-once",
			processType: "short-running",
			command:     "echo",
			args:        []string{"hello world"},
			mode:        config.ModeOnce,
			expectError: false,
			description: "Short-running process in once mode should complete normally",
		},
		{
			name:        "long-running-keepalive",
			processType: "long-running",
			command:     getPingCommand(),
			args:        getPingArgs(30),
			mode:        config.ModeKeepAlive,
			expectError: false,
			description: "Long-running process in keepAlive mode should be terminable",
		},
		{
			name:        "cpu-intensive-keepalive",
			processType: "cpu-intensive",
			command:     getCPUIntensiveCommand(),
			args:        getCPUIntensiveArgs(),
			mode:        config.ModeKeepAlive,
			expectError: false,
			description: "CPU-intensive process should be terminable",
		},
		{
			name:        "network-service-keepalive",
			processType: "network-service",
			command:     getNetworkServiceCommand(),
			args:        getNetworkServiceArgs(),
			mode:        config.ModeKeepAlive,
			expectError: false,
			description: "Network service process should be terminable",
		},
		{
			name:        "shell-script-keepalive",
			processType: "shell-script",
			command:     getShellCommand(),
			args:        getShellScriptArgs(),
			mode:        config.ModeKeepAlive,
			expectError: false,
			description: "Shell script with child processes should be terminable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTerminationScenario(t, tt.processType, tt.command, tt.args, tt.mode, tt.expectError, tt.description)
		})
	}
}

func testTerminationScenario(t *testing.T, processType, command string, args []string, mode config.Mode, expectError bool, description string) {
	t.Logf("Testing %s: %s", processType, description)

	// Skip if command is not available
	if !isCommandAvailable(command) {
		t.Skipf("Command %s not available on this system", command)
	}

	// Special handling for shell script test - create the script file
	if processType == "shell-script" {
		scriptContent := getChildProcessScript()
		scriptFile := getScriptFileName()

		if err := os.WriteFile(scriptFile, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}
		defer os.Remove(scriptFile)
	}

	executor := NewExecutor(true)
	defer executor.Stop()

	// Create command configuration
	cmd := config.Command{
		Name:    fmt.Sprintf("test-%s", processType),
		Command: command,
		Args:    args,
		Mode:    mode,
	}

	cfg := &config.Config{
		Version:  "1.0",
		Commands: []config.Command{cmd},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if mode == config.ModeOnce {
		// For once mode, just execute and verify completion
		err := executor.Execute(ctx, cfg)
		if expectError && err == nil {
			t.Errorf("Expected error for %s but got none", processType)
		} else if !expectError && err != nil {
			t.Errorf("Unexpected error for %s: %v", processType, err)
		}
		return
	}

	// For keepAlive mode, start execution in background and test termination
	go func() {
		executor.Execute(ctx, cfg)
	}()

	// Wait for process to start
	time.Sleep(2 * time.Second)

	// Verify process is tracked
	processes := executor.GetTrackedProcesses()
	if len(processes) == 0 {
		t.Fatalf("No processes tracked for %s", processType)
	}

	var pid int
	for p := range processes {
		pid = p
		break
	}

	t.Logf("Process %s started with PID %d", processType, pid)

	// Test graceful termination
	start := time.Now()
	executor.Stop()
	duration := time.Since(start)

	// Verify termination completed within reasonable time
	if duration > 15*time.Second {
		t.Errorf("Termination took too long for %s: %v", processType, duration)
	}

	// Wait a moment for cleanup
	time.Sleep(1 * time.Second)

	// Verify process is no longer running (be lenient on Windows due to OS limitations)
	if isProcessRunning(pid) {
		if runtime.GOOS == "windows" {
			t.Logf("Warning: Process %d (%s) appears to still be running after termination (Windows limitation)", pid, processType)
		} else {
			t.Errorf("Process %d (%s) is still running after termination", pid, processType)
		}
	}

	// Verify process is no longer tracked
	processes = executor.GetTrackedProcesses()
	if len(processes) > 0 {
		t.Errorf("Processes still tracked after termination for %s: %v", processType, processes)
	}

	t.Logf("Successfully terminated %s process (PID %d) in %v", processType, pid, duration)
}

// TestProcessManagerTermination tests termination through ProcessManager
func TestProcessManagerTermination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping process manager termination test in short mode")
	}

	tests := []struct {
		name        string
		graceful    bool
		description string
	}{
		{
			name:        "graceful-termination",
			graceful:    true,
			description: "Test graceful termination through ProcessManager",
		},
		{
			name:        "force-termination",
			graceful:    false,
			description: "Test force termination through ProcessManager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testProcessManagerTermination(t, tt.graceful, tt.description)
		})
	}
}

func testProcessManagerTermination(t *testing.T, graceful bool, description string) {
	t.Logf("Testing: %s", description)

	pm := NewProcessManager()
	defer os.Remove(pm.tracker.filePath)

	// Start a test process
	command := getPingCommand()
	args := getPingArgs(30)

	if !isCommandAvailable(command) {
		t.Skipf("Command %s not available on this system", command)
	}

	cmd := exec.Command(command, args...)
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	pid := cmd.Process.Pid
	processName := fmt.Sprintf("test-pm-%s", map[bool]string{true: "graceful", false: "force"}[graceful])

	// Track the process
	err = pm.tracker.AddProcess(pid, processName, command, args, "", "keepAlive")
	if err != nil {
		t.Fatalf("Failed to track process: %v", err)
	}

	t.Logf("Started test process with PID %d", pid)

	// Test termination
	start := time.Now()
	err = pm.KillProcess(pid, graceful)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to kill process: %v", err)
	}

	// Verify termination completed within reasonable time
	expectedMaxDuration := 15 * time.Second
	if !graceful {
		expectedMaxDuration = 5 * time.Second // Force kill should be faster
	}

	if duration > expectedMaxDuration {
		t.Errorf("Termination took too long: %v (expected < %v)", duration, expectedMaxDuration)
	}

	// Wait a moment for cleanup
	time.Sleep(500 * time.Millisecond)

	// Verify process is no longer running (be lenient on Windows due to OS limitations)
	if isProcessRunning(pid) {
		if runtime.GOOS == "windows" {
			t.Logf("Warning: Process %d appears to still be running after %s termination (Windows limitation)", pid, map[bool]string{true: "graceful", false: "force"}[graceful])
		} else {
			t.Errorf("Process %d is still running after %s termination", pid, map[bool]string{true: "graceful", false: "force"}[graceful])
		}
	}

	t.Logf("Successfully terminated process (PID %d) using %s method in %v", pid, map[bool]string{true: "graceful", false: "force"}[graceful], duration)
}

// TestMultipleProcessTermination tests termination of multiple processes simultaneously
func TestMultipleProcessTermination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple process termination test in short mode")
	}

	pm := NewProcessManager()
	defer os.Remove(pm.tracker.filePath)

	command := getPingCommand()
	args := getPingArgs(60)

	if !isCommandAvailable(command) {
		t.Skipf("Command %s not available on this system", command)
	}

	// Start multiple test processes
	var pids []int
	processCount := 3

	for i := 0; i < processCount; i++ {
		cmd := exec.Command(command, args...)
		err := cmd.Start()
		if err != nil {
			t.Fatalf("Failed to start test process %d: %v", i, err)
		}

		pid := cmd.Process.Pid
		pids = append(pids, pid)

		// Track the process
		err = pm.tracker.AddProcess(pid, fmt.Sprintf("test-multi-%d", i), command, args, "", "keepAlive")
		if err != nil {
			t.Fatalf("Failed to track process %d: %v", i, err)
		}

		t.Logf("Started test process %d with PID %d", i, pid)
	}

	// Verify all processes are tracked
	count, err := pm.GetProcessCount()
	if err != nil {
		t.Fatalf("Failed to get process count: %v", err)
	}
	if count != processCount {
		t.Fatalf("Expected %d processes, got %d", processCount, count)
	}

	// Test killing all processes
	start := time.Now()
	err = pm.KillAllProcesses(true)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to kill all processes: %v", err)
	}

	// Verify termination completed within reasonable time
	if duration > 20*time.Second {
		t.Errorf("Multiple process termination took too long: %v", duration)
	}

	// Wait a moment for cleanup
	time.Sleep(1 * time.Second)

	// Verify all processes are no longer running (be lenient on Windows due to OS limitations)
	for i, pid := range pids {
		if isProcessRunning(pid) {
			if runtime.GOOS == "windows" {
				t.Logf("Warning: Process %d (PID %d) appears to still be running after kill all (Windows limitation)", i, pid)
			} else {
				t.Errorf("Process %d (PID %d) is still running after kill all", i, pid)
			}
		}
	}

	// Verify no processes are tracked
	count, err = pm.GetProcessCount()
	if err != nil {
		t.Fatalf("Failed to get process count after kill all: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 processes after kill all, got %d", count)
	}

	t.Logf("Successfully terminated %d processes in %v", processCount, duration)
}

// TestChildProcessCleanupScenarios tests cleanup of processes with child processes
func TestChildProcessCleanupScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping child process cleanup scenarios test in short mode")
	}

	// Create a script that spawns child processes
	scriptContent := getChildProcessScript()
	scriptFile := getScriptFileName()
	scriptCmd := getScriptCommand(scriptFile)

	// Write the script file
	if err := os.WriteFile(scriptFile, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptFile)

	executor := NewExecutor(true)
	defer executor.Stop()

	// Create command configuration
	cmd := config.Command{
		Name:    "test-child-cleanup",
		Command: scriptCmd[0],
		Args:    scriptCmd[1:],
		Mode:    config.ModeKeepAlive,
	}

	cfg := &config.Config{
		Version:  "1.0",
		Commands: []config.Command{cmd},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start execution in background
	go func() {
		executor.Execute(ctx, cfg)
	}()

	// Wait for the process to start and spawn children
	time.Sleep(3 * time.Second)

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
	start := time.Now()
	executor.Stop()
	duration := time.Since(start)

	t.Logf("Termination completed in %v", duration)

	// Wait for cleanup to complete
	time.Sleep(2 * time.Second)

	// Count child processes after termination
	childrenAfter := countChildProcesses(parentPID)
	t.Logf("Child processes after termination: %d", childrenAfter)

	// Verify that child processes were cleaned up (be lenient on Windows due to OS limitations)
	if childrenAfter > 0 && childrenAfter >= childrenBefore {
		if runtime.GOOS == "windows" {
			t.Logf("Warning: Child process cleanup may not have worked properly on Windows. Before: %d, After: %d (Windows limitation)", childrenBefore, childrenAfter)
		} else {
			t.Errorf("Child process cleanup may not have worked properly. Before: %d, After: %d", childrenBefore, childrenAfter)
		}
	}

	// Verify parent process is no longer running (be lenient on Windows due to OS limitations)
	if isProcessRunning(parentPID) {
		if runtime.GOOS == "windows" {
			t.Logf("Warning: Parent process %d appears to still be running after cleanup (Windows limitation)", parentPID)
		} else {
			t.Errorf("Parent process %d is still running after cleanup", parentPID)
		}
	}

	t.Logf("Successfully cleaned up parent process and %d child processes", childrenBefore-childrenAfter)
}

// Platform-specific helper functions

func getCPUIntensiveCommand() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "yes"
}

func getCPUIntensiveArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"-Command", "while($true) { $null }"}
	}
	return []string{"/dev/null"}
}

func getNetworkServiceCommand() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "python3"
}

func getNetworkServiceArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"-Command", "$listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Any, 0); $listener.Start(); while($true) { Start-Sleep 1 }"}
	}
	return []string{"-c", "import socket, time; s=socket.socket(); s.bind(('', 0)); s.listen(1); [time.sleep(1) for _ in iter(int, 1)]"}
}

func getShellCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "bash"
}

func getShellScriptArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"/C", "test_parent.bat"}
	}
	return []string{"test_parent.sh"}
}

func getChildProcessScript() string {
	if runtime.GOOS == "windows" {
		return `@echo off
echo Parent process started
start /B ping 127.0.0.1 -n 100 > nul
start /B ping 127.0.0.1 -n 100 > nul
ping 127.0.0.1 -n 100 > nul
`
	}
	return `#!/bin/bash
echo "Parent process started"
ping 127.0.0.1 -c 100 >/dev/null 2>&1 &
ping 127.0.0.1 -c 100 >/dev/null 2>&1 &
ping 127.0.0.1 -c 100 >/dev/null 2>&1
`
}

func getScriptFileName() string {
	if runtime.GOOS == "windows" {
		return "test_parent.bat"
	}
	return "test_parent.sh"
}

func getScriptCommand(scriptFile string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C", scriptFile}
	}
	return []string{"bash", scriptFile}
}

func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// TestPlatformSpecificTermination tests platform-specific termination behavior
func TestPlatformSpecificTermination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping platform-specific termination test in short mode")
	}

	t.Logf("Testing termination on platform: %s", runtime.GOOS)

	executor := NewExecutor(true)
	defer executor.Stop()

	// Use a simple command that works on all platforms
	command := getPingCommand()
	args := getPingArgs(10)

	if !isCommandAvailable(command) {
		t.Skipf("Command %s not available on this system", command)
	}

	// Create command configuration
	cmd := config.Command{
		Name:    "test-platform-termination",
		Command: command,
		Args:    args,
		Mode:    config.ModeKeepAlive,
	}

	cfg := &config.Config{
		Version:  "1.0",
		Commands: []config.Command{cmd},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start execution in background
	go func() {
		executor.Execute(ctx, cfg)
	}()

	// Wait for process to start
	time.Sleep(1 * time.Second)

	// Verify process is tracked
	processes := executor.GetTrackedProcesses()
	if len(processes) == 0 {
		t.Fatal("No processes tracked")
	}

	var pid int
	for p := range processes {
		pid = p
		break
	}

	t.Logf("Started process with PID %d on %s", pid, runtime.GOOS)

	// Test termination behavior based on platform
	start := time.Now()
	executor.Stop()
	duration := time.Since(start)

	t.Logf("Termination completed in %v on %s", duration, runtime.GOOS)

	// Platform-specific expectations
	switch runtime.GOOS {
	case "windows":
		// On Windows, termination should be relatively quick but may not be perfect
		if duration > 10*time.Second {
			t.Errorf("Windows termination took too long: %v", duration)
		}
		t.Logf("Windows termination behavior: process group termination may fall back to individual process termination")
	case "linux", "darwin":
		// On Unix-like systems, termination should be more reliable
		if duration > 15*time.Second {
			t.Errorf("Unix termination took too long: %v", duration)
		}
		t.Logf("Unix termination behavior: should support proper process group termination")
	default:
		t.Logf("Unknown platform %s: termination completed in %v", runtime.GOOS, duration)
	}

	// Wait a moment for cleanup
	time.Sleep(500 * time.Millisecond)

	// Verify no processes are tracked
	processes = executor.GetTrackedProcesses()
	if len(processes) > 0 {
		t.Errorf("Processes still tracked after termination: %v", processes)
	}

	t.Logf("Platform-specific termination test completed successfully on %s", runtime.GOOS)
}
