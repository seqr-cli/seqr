package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestNewProcessManager(t *testing.T) {
	opts := ProcessManagerOptions{
		Verbose:    true,
		WorkingDir: "/tmp",
		Timeout:    30 * time.Second,
	}

	pm := NewProcessManager(opts)
	if pm == nil {
		t.Fatal("NewProcessManager returned nil")
	}

	// Verify initial state
	activeProcs := pm.GetActiveProcesses()
	if len(activeProcs) != 0 {
		t.Errorf("Expected no active processes initially, got %d", len(activeProcs))
	}
}

func TestProcessManager_ExecuteCommand_OnceMode(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{})
	ctx := context.Background()

	cmd := config.Command{
		Name:    "echo-test",
		Command: "echo",
		Args:    []string{"hello world"},
		Mode:    config.ModeOnce,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Output, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got: %s", result.Output)
	}

	// Verify no active processes after once command
	activeProcs := pm.GetActiveProcesses()
	if len(activeProcs) != 0 {
		t.Errorf("Expected no active processes after once command, got %d", len(activeProcs))
	}
}

func TestProcessManager_ExecuteCommand_KeepAliveMode(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	cmd := config.Command{
		Name:    "sleep-test",
		Command: "sleep",
		Args:    []string{"0.2"}, // Short sleep for testing
		Mode:    config.ModeKeepAlive,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Output, "PID") {
		t.Errorf("Expected output to contain PID information, got: %s", result.Output)
	}

	// Verify process is tracked
	activeProcs := pm.GetActiveProcesses()
	if len(activeProcs) != 1 {
		t.Errorf("Expected 1 active process, got %d", len(activeProcs))
	}

	if proc, exists := activeProcs["sleep-test"]; !exists {
		t.Error("Expected 'sleep-test' process to be tracked")
	} else {
		if proc.Name != "sleep-test" {
			t.Errorf("Expected process name 'sleep-test', got '%s'", proc.Name)
		}
		if proc.PID <= 0 {
			t.Errorf("Expected valid PID, got %d", proc.PID)
		}
		if proc.Command != "sleep 0.2" {
			t.Errorf("Expected command 'sleep 0.2', got '%s'", proc.Command)
		}
	}

	// Wait for process to complete and be untracked
	time.Sleep(300 * time.Millisecond)
	activeProcs = pm.GetActiveProcesses()
	if len(activeProcs) != 0 {
		t.Errorf("Expected no active processes after completion, got %d", len(activeProcs))
	}
}

func TestProcessManager_ExecuteCommand_UnsupportedMode(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{})
	ctx := context.Background()

	cmd := config.Command{
		Name:    "invalid-mode",
		Command: "echo",
		Args:    []string{"test"},
		Mode:    "invalid", // Invalid mode
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err == nil {
		t.Fatal("Expected error for unsupported mode, got nil")
	}

	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.ExitCode != -1 {
		t.Errorf("Expected exit code -1, got %d", result.ExitCode)
	}

	if !strings.Contains(err.Error(), "unsupported execution mode") {
		t.Errorf("Expected error about unsupported mode, got: %v", err)
	}

	if result.ErrorDetail == nil {
		t.Fatal("Expected error detail to be populated")
	}

	if result.ErrorDetail.Type != ErrorTypeUnsupportedMode {
		t.Errorf("Expected error type %s, got %s", ErrorTypeUnsupportedMode, result.ErrorDetail.Type)
	}
}

func TestProcessManager_MultipleKeepAliveProcesses(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	commands := []config.Command{
		{
			Name:    "sleep-1",
			Command: "sleep",
			Args:    []string{"0.3"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "sleep-2",
			Command: "sleep",
			Args:    []string{"0.3"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "sleep-3",
			Command: "sleep",
			Args:    []string{"0.3"},
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

	// Verify all processes are tracked
	activeProcs := pm.GetActiveProcesses()
	if len(activeProcs) != 3 {
		t.Errorf("Expected 3 active processes, got %d", len(activeProcs))
	}

	expectedNames := []string{"sleep-1", "sleep-2", "sleep-3"}
	for _, name := range expectedNames {
		if _, exists := activeProcs[name]; !exists {
			t.Errorf("Expected process '%s' to be tracked", name)
		}
	}

	// Wait for all processes to complete with retry logic
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(50 * time.Millisecond)
		activeProcs = pm.GetActiveProcesses()
		if len(activeProcs) == 0 {
			break
		}
		if i == maxRetries-1 {
			t.Errorf("Expected no active processes after completion, got %d", len(activeProcs))
		}
	}
}

func TestProcessManager_TerminateProcess(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	cmd := config.Command{
		Name:    "long-sleep",
		Command: "sleep",
		Args:    []string{"10"}, // Long sleep
		Mode:    config.ModeKeepAlive,
	}

	// Start the process
	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	// Verify process is tracked
	activeProcs := pm.GetActiveProcesses()
	if len(activeProcs) != 1 {
		t.Fatalf("Expected 1 active process, got %d", len(activeProcs))
	}

	// Terminate the specific process
	err = pm.TerminateProcess("long-sleep")
	if err != nil {
		t.Errorf("Expected successful termination, got error: %v", err)
	}

	// Wait a bit for termination to complete
	time.Sleep(100 * time.Millisecond)

	// Verify process is no longer tracked
	activeProcs = pm.GetActiveProcesses()
	if len(activeProcs) != 0 {
		t.Errorf("Expected no active processes after termination, got %d", len(activeProcs))
	}
}

func TestProcessManager_TerminateProcess_NotFound(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{})

	err := pm.TerminateProcess("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent process, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestProcessManager_TerminateAll(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{Verbose: true})
	ctx := context.Background()

	commands := []config.Command{
		{
			Name:    "sleep-a",
			Command: "sleep",
			Args:    []string{"10"},
			Mode:    config.ModeKeepAlive,
		},
		{
			Name:    "sleep-b",
			Command: "sleep",
			Args:    []string{"10"},
			Mode:    config.ModeKeepAlive,
		},
	}

	// Start multiple processes
	for _, cmd := range commands {
		result, err := pm.ExecuteCommand(ctx, cmd)
		if err != nil {
			t.Fatalf("Expected successful execution for %s, got error: %v", cmd.Name, err)
		}
		if !result.Success {
			t.Errorf("Expected command %s to succeed", cmd.Name)
		}
	}

	// Verify processes are tracked
	activeProcs := pm.GetActiveProcesses()
	if len(activeProcs) != 2 {
		t.Fatalf("Expected 2 active processes, got %d", len(activeProcs))
	}

	// Terminate all processes
	err := pm.TerminateAll()
	if err != nil {
		t.Errorf("Expected successful termination of all processes, got error: %v", err)
	}

	// Wait a bit for termination to complete
	time.Sleep(100 * time.Millisecond)

	// Verify no processes are tracked
	activeProcs = pm.GetActiveProcesses()
	if len(activeProcs) != 0 {
		t.Errorf("Expected no active processes after terminating all, got %d", len(activeProcs))
	}
}

func TestProcessManager_OnceCommand_Failure(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{})
	ctx := context.Background()

	cmd := config.Command{
		Name:    "failing-command",
		Command: "false", // Command that always fails
		Mode:    config.ModeOnce,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for failed command")
	}

	// Verify it's a CommandExecutionError
	var cmdErr *CommandExecutionError
	if !strings.Contains(err.Error(), "command 'failing-command' failed") {
		t.Errorf("Expected CommandExecutionError, got: %v", err)
	}
	_ = cmdErr // Avoid unused variable warning
}

func TestProcessManager_KeepAliveCommand_StartupFailure(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{})
	ctx := context.Background()

	cmd := config.Command{
		Name:    "nonexistent-keepalive",
		Command: "nonexistent_command_12345",
		Mode:    config.ModeKeepAlive,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err == nil {
		t.Fatal("Expected error for nonexistent keepAlive command, got nil")
	}

	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.ExitCode != -1 {
		t.Errorf("Expected exit code -1, got %d", result.ExitCode)
	}

	if result.ErrorDetail == nil {
		t.Fatal("Expected error detail to be populated")
	}

	if result.ErrorDetail.Type != ErrorTypeStartupFailure {
		t.Errorf("Expected error type %s, got %s", ErrorTypeStartupFailure, result.ErrorDetail.Type)
	}

	// Verify it's a KeepAliveStartupError
	if !strings.Contains(err.Error(), "failed to start keepAlive command") {
		t.Errorf("Expected KeepAliveStartupError, got: %v", err)
	}
}

func TestProcessManager_WithWorkingDirectory(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{WorkingDir: "/tmp"})
	ctx := context.Background()

	cmd := config.Command{
		Name:    "pwd-test",
		Command: "pwd",
		Mode:    config.ModeOnce,
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if !strings.Contains(result.Output, "/tmp") {
		t.Errorf("Expected output to contain /tmp, got: %s", result.Output)
	}
}

func TestProcessManager_WithEnvironmentVariables(t *testing.T) {
	pm := NewProcessManager(ProcessManagerOptions{})
	ctx := context.Background()

	cmd := config.Command{
		Name:    "env-test",
		Command: "sh",
		Args:    []string{"-c", "echo $TEST_VAR"},
		Mode:    config.ModeOnce,
		Env: map[string]string{
			"TEST_VAR": "hello_from_env",
		},
	}

	result, err := pm.ExecuteCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if !strings.Contains(result.Output, "hello_from_env") {
		t.Errorf("Expected output to contain environment variable value, got: %s", result.Output)
	}
}
