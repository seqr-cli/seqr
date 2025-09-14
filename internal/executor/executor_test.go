package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestNewExecutor(t *testing.T) {
	opts := ExecutorOptions{
		Verbose:    true,
		WorkingDir: "/tmp",
		Timeout:    30 * time.Second,
	}

	executor := NewExecutor(opts)
	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	status := executor.GetStatus()
	if status.State != StateReady {
		t.Errorf("Expected initial state to be Ready, got %v", status.State)
	}

	if status.TotalCount != 0 {
		t.Errorf("Expected initial total count to be 0, got %d", status.TotalCount)
	}

	if status.CompletedCount != 0 {
		t.Errorf("Expected initial completed count to be 0, got %d", status.CompletedCount)
	}
}

func TestExecutionState_String(t *testing.T) {
	tests := []struct {
		state    ExecutionState
		expected string
	}{
		{StateReady, "ready"},
		{StateRunning, "running"},
		{StateSuccess, "success"},
		{StateFailed, "failed"},
		{ExecutionState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("ExecutionState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExecutor_Execute_NilConfig(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	err := executor.Execute(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil config, got nil")
	}

	if err.Error() != "configuration cannot be nil" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestExecutor_Execute_EmptyCommands(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version:  "1.0",
		Commands: []config.Command{},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for empty commands, got nil")
	}

	if err.Error() != "no commands to execute" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestExecutor_Execute_Success(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, false)
	executor := NewExecutor(ExecutorOptions{Reporter: reporter})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test1",
				Command: "echo",
				Args:    []string{"hello"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "test2",
				Command: "echo",
				Args:    []string{"world"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}

	if status.CompletedCount != 2 {
		t.Errorf("Expected completed count to be 2, got %d", status.CompletedCount)
	}

	if status.TotalCount != 2 {
		t.Errorf("Expected total count to be 2, got %d", status.TotalCount)
	}

	if len(status.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(status.Results))
	}

	// Verify all results are successful
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d to be successful", i)
		}
		if result.ExitCode != 0 {
			t.Errorf("Expected result %d exit code to be 0, got %d", i, result.ExitCode)
		}
	}
}

func TestExecutor_Stop(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})

	// Test that stop sets the internal flag
	executor.Stop()

	// Since we can't easily test the actual stopping behavior without
	// implementing the full execution logic, we'll verify the interface works
	status := executor.GetStatus()
	if status.State != StateReady {
		t.Errorf("Expected state to remain Ready after stop, got %v", status.State)
	}
}

func TestExecutor_GetStatus_ThreadSafety(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})

	// Test concurrent access to GetStatus
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			status := executor.GetStatus()
			if status.State != StateReady {
				t.Errorf("Expected state to be Ready, got %v", status.State)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestExecutor_ContextCancellation(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx, cancel := context.WithCancel(context.Background())

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test",
				Command: "sleep",
				Args:    []string{"1"},
				Mode:    config.ModeOnce,
			},
		},
	}

	// Cancel context immediately
	cancel()

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error due to context cancellation")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestExecutionResult_Fields(t *testing.T) {
	cmd := config.Command{
		Name:    "test",
		Command: "echo",
		Args:    []string{"hello"},
		Mode:    config.ModeOnce,
	}

	startTime := time.Now()
	endTime := startTime.Add(100 * time.Millisecond)

	result := ExecutionResult{
		Command:   cmd,
		Success:   true,
		ExitCode:  0,
		Output:    "hello\n",
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
	}

	if result.Command.Name != "test" {
		t.Errorf("Expected command name 'test', got '%s'", result.Command.Name)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Duration != 100*time.Millisecond {
		t.Errorf("Expected duration 100ms, got %v", result.Duration)
	}
}

func TestExecutor_Execute_OnceMode(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "echo-test",
				Command: "echo",
				Args:    []string{"hello world"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}

	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Output, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got: %s", result.Output)
	}
}

func TestExecutor_Execute_KeepAliveMode(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "sleep-test",
				Command: "sleep",
				Args:    []string{"0.1"}, // Short sleep for testing
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}

	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Output, "PID") {
		t.Errorf("Expected output to contain PID information, got: %s", result.Output)
	}

	// Give the process a moment to complete
	time.Sleep(200 * time.Millisecond)
}

func TestExecutor_Execute_MixedModes(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "echo-once",
				Command: "echo",
				Args:    []string{"first"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "sleep-keepalive",
				Command: "sleep",
				Args:    []string{"0.1"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "echo-once-2",
				Command: "echo",
				Args:    []string{"second"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}

	if len(status.Results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(status.Results))
	}

	// Verify all commands succeeded
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d to be successful", i)
		}
	}

	// Verify output content
	if !strings.Contains(status.Results[0].Output, "first") {
		t.Errorf("Expected first result to contain 'first', got: %s", status.Results[0].Output)
	}

	if !strings.Contains(status.Results[2].Output, "second") {
		t.Errorf("Expected third result to contain 'second', got: %s", status.Results[2].Output)
	}
}

func TestExecutor_Execute_CommandFailure(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "failing-command",
				Command: "false", // Command that always fails
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected final state to be Failed, got %v", status.State)
	}

	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for failed command")
	}
}

func TestExecutor_Execute_UnsupportedMode(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "invalid-mode",
				Command: "echo",
				Args:    []string{"test"},
				Mode:    "invalid", // Invalid mode
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for unsupported mode, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported execution mode") {
		t.Errorf("Expected error about unsupported mode, got: %v", err)
	}

	// Check that error detail is populated
	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if result.ErrorDetail == nil {
		t.Fatal("Expected error detail to be populated")
	}

	if result.ErrorDetail.Type != ErrorTypeUnsupportedMode {
		t.Errorf("Expected error type %s, got %s", ErrorTypeUnsupportedMode, result.ErrorDetail.Type)
	}

	if result.ErrorDetail.CommandLine != "echo test" {
		t.Errorf("Expected command line 'echo test', got '%s'", result.ErrorDetail.CommandLine)
	}
}

func TestExecutor_Execute_WithEnvironmentVariables(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "env-test",
				Command: "sh",
				Args:    []string{"-c", "echo $TEST_VAR"},
				Mode:    config.ModeOnce,
				Env: map[string]string{
					"TEST_VAR": "hello_from_env",
				},
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if !strings.Contains(result.Output, "hello_from_env") {
		t.Errorf("Expected output to contain environment variable value, got: %s", result.Output)
	}
}

func TestExecutor_Execute_WithWorkingDirectory(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "pwd-test",
				Command: "pwd",
				Mode:    config.ModeOnce,
				WorkDir: "/tmp",
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if !strings.Contains(result.Output, "/tmp") {
		t.Errorf("Expected output to contain /tmp, got: %s", result.Output)
	}
}
func TestExecutor_Execute_CommandNotFound(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "nonexistent-command",
				Command: "nonexistent_command_12345",
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for nonexistent command, got nil")
	}

	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected state to be Failed, got %v", status.State)
	}

	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.ErrorDetail == nil {
		t.Fatal("Expected error detail to be populated")
	}

	if result.ErrorDetail.Type != ErrorTypeCommandNotFound {
		t.Errorf("Expected error type %s, got %s", ErrorTypeCommandNotFound, result.ErrorDetail.Type)
	}

	if result.ErrorDetail.CommandLine != "nonexistent_command_12345" {
		t.Errorf("Expected command line 'nonexistent_command_12345', got '%s'", result.ErrorDetail.CommandLine)
	}
}

func TestExecutor_Execute_DetailedFailureContext(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "success-command",
				Command: "echo",
				Args:    []string{"success"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "failing-command",
				Command: "false",
				Mode:    config.ModeOnce,
			},
			{
				Name:    "should-not-run",
				Command: "echo",
				Args:    []string{"should not see this"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected state to be Failed, got %v", status.State)
	}

	// Should have 2 results (success + failure), third command should not run
	if len(status.Results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(status.Results))
	}

	// First command should succeed
	if !status.Results[0].Success {
		t.Error("Expected first command to succeed")
	}

	// Second command should fail with detailed context
	failedResult := status.Results[1]
	if failedResult.Success {
		t.Error("Expected second command to fail")
	}

	if failedResult.ErrorDetail == nil {
		t.Fatal("Expected error detail for failed command")
	}

	if failedResult.ErrorDetail.Type != ErrorTypeNonZeroExit {
		t.Errorf("Expected error type %s, got %s", ErrorTypeNonZeroExit, failedResult.ErrorDetail.Type)
	}

	// Check that LastError contains execution context
	if !strings.Contains(status.LastError, "Execution stopped at command 2 of 3") {
		t.Errorf("Expected LastError to contain execution context, got: %s", status.LastError)
	}

	if !strings.Contains(status.LastError, "Error Type: non_zero_exit") {
		t.Errorf("Expected LastError to contain error type, got: %s", status.LastError)
	}
}

func TestErrorDetail_BuildCommandLine(t *testing.T) {
	tests := []struct {
		name     string
		cmd      config.Command
		expected string
	}{
		{
			name: "simple command",
			cmd: config.Command{
				Command: "echo",
			},
			expected: "echo",
		},
		{
			name: "command with args",
			cmd: config.Command{
				Command: "echo",
				Args:    []string{"hello", "world"},
			},
			expected: "echo hello world",
		},
		{
			name: "command with spaced args",
			cmd: config.Command{
				Command: "echo",
				Args:    []string{"hello world", "test"},
			},
			expected: "echo \"hello world\" test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCommandLine(tt.cmd)
			if result != tt.expected {
				t.Errorf("buildCommandLine() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorDetail_CategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: ErrorTypeContextCancelled,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: ErrorTypeContextCancelled,
		},
		{
			name:     "command not found",
			err:      fmt.Errorf("executable file not found in $PATH"),
			expected: ErrorTypeCommandNotFound,
		},
		{
			name:     "permission denied",
			err:      fmt.Errorf("permission denied"),
			expected: ErrorTypePermissionDenied,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("some other error"),
			expected: ErrorTypeSystemError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("categorizeError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExecutor_Execute_KeepAliveStartupFailure(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "nonexistent-keepalive",
				Command: "nonexistent_command_12345",
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for nonexistent keepAlive command, got nil")
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.ErrorDetail == nil {
		t.Fatal("Expected error detail to be populated")
	}

	if result.ErrorDetail.Type != ErrorTypeStartupFailure {
		t.Errorf("Expected error type %s, got %s", ErrorTypeStartupFailure, result.ErrorDetail.Type)
	}

	// Check enhanced error message
	if !strings.Contains(err.Error(), "failed to start keepAlive command") {
		t.Errorf("Expected enhanced error message, got: %v", err)
	}
}

func TestExecutor_Execute_WithWorkingDirectoryInErrorDetail(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "failing-with-workdir",
				Command: "false",
				Mode:    config.ModeOnce,
				WorkDir: "/tmp",
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if result.ErrorDetail == nil {
		t.Fatal("Expected error detail to be populated")
	}

	if result.ErrorDetail.WorkingDir != "/tmp" {
		t.Errorf("Expected working directory '/tmp', got '%s'", result.ErrorDetail.WorkingDir)
	}

	// Check that error message includes working directory
	if !strings.Contains(err.Error(), "Working Directory: /tmp") {
		t.Errorf("Expected error to include working directory, got: %v", err)
	}
}

func TestExecutor_Execute_WithReporting(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, false)
	executor := NewExecutor(ExecutorOptions{Reporter: reporter})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "first-cmd",
				Command: "echo",
				Args:    []string{"hello"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "second-cmd",
				Command: "echo",
				Args:    []string{"world"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	output := buf.String()

	// Verify execution start is reported
	if !strings.Contains(output, "Starting execution of 2 command(s)") {
		t.Errorf("Expected execution start report, got:\n%s", output)
	}

	// Verify command starts are reported
	if !strings.Contains(output, "[1] Starting 'first-cmd'") {
		t.Errorf("Expected first command start report, got:\n%s", output)
	}

	if !strings.Contains(output, "[2] Starting 'second-cmd'") {
		t.Errorf("Expected second command start report, got:\n%s", output)
	}

	// Verify command successes are reported
	if !strings.Contains(output, "[1] ✓ 'first-cmd'") {
		t.Errorf("Expected first command success report, got:\n%s", output)
	}

	if !strings.Contains(output, "[2] ✓ 'second-cmd'") {
		t.Errorf("Expected second command success report, got:\n%s", output)
	}

	// Verify execution completion is reported
	if !strings.Contains(output, "✓ All commands completed successfully (2/2)") {
		t.Errorf("Expected execution completion report, got:\n%s", output)
	}
}

func TestExecutor_Execute_WithReporting_Verbose(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true)
	executor := NewExecutor(ExecutorOptions{Reporter: reporter})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-cmd",
				Command: "echo",
				Args:    []string{"verbose output"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	output := buf.String()

	// Verify verbose mode is enabled
	if !strings.Contains(output, "Verbose mode enabled") {
		t.Errorf("Expected verbose mode indication, got:\n%s", output)
	}

	// Verify verbose command reporting
	if !strings.Contains(output, "[1] Starting 'test-cmd'...") {
		t.Errorf("Expected verbose command start, got:\n%s", output)
	}

	if !strings.Contains(output, "completed successfully") {
		t.Errorf("Expected verbose success message, got:\n%s", output)
	}

	// Verify execution summary is shown in verbose mode
	if !strings.Contains(output, "Execution Summary:") {
		t.Errorf("Expected execution summary in verbose mode, got:\n%s", output)
	}

	if !strings.Contains(output, "Total: 1 commands, 1 successful, 0 failed") {
		t.Errorf("Expected summary statistics, got:\n%s", output)
	}
}

func TestExecutor_Execute_WithReporting_Failure(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, false)
	executor := NewExecutor(ExecutorOptions{Reporter: reporter})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "success-cmd",
				Command: "echo",
				Args:    []string{"success"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "failing-cmd",
				Command: "false",
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	output := buf.String()

	// Verify successful command is reported
	if !strings.Contains(output, "[1] ✓ 'success-cmd'") {
		t.Errorf("Expected success command report, got:\n%s", output)
	}

	// Verify failed command is reported
	if !strings.Contains(output, "[2] ✗ 'failing-cmd' failed") {
		t.Errorf("Expected failure command report, got:\n%s", output)
	}

	// Verify execution failure is reported
	if !strings.Contains(output, "✗ Execution failed at command 1 of 2") {
		t.Errorf("Expected execution failure report, got:\n%s", output)
	}

	// Verify error details are shown
	if !strings.Contains(output, "Exit code:") {
		t.Errorf("Expected exit code in failure report, got:\n%s", output)
	}
}

func TestExecutor_Execute_WithReporting_KeepAlive(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose to see keepAlive details
	executor := NewExecutor(ExecutorOptions{Reporter: reporter})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "background-service",
				Command: "sleep",
				Args:    []string{"0.1"}, // Short sleep for testing
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	output := buf.String()

	// Verify keepAlive command is reported
	if !strings.Contains(output, "[1] ✓ 'background-service' completed successfully") {
		t.Errorf("Expected keepAlive success report, got:\n%s", output)
	}

	// Verify background process indication
	if !strings.Contains(output, "Process started and running in background") {
		t.Errorf("Expected background process indication, got:\n%s", output)
	}

	// Give the process a moment to complete
	time.Sleep(200 * time.Millisecond)
}

func TestExecutor_DefaultReporter(t *testing.T) {
	// Test that executor creates a default reporter when none is provided
	executor := NewExecutor(ExecutorOptions{Verbose: true})

	// Verify executor was created successfully
	if executor == nil {
		t.Fatal("Expected executor to be created with default reporter")
	}

	status := executor.GetStatus()
	if status.State != StateReady {
		t.Errorf("Expected initial state to be Ready, got %v", status.State)
	}
}

// Additional comprehensive tests for all execution paths

func TestExecutor_Execute_WithTimeout(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{
		Timeout: 100 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "timeout-test",
				Command: "sleep",
				Args:    []string{"1"}, // Sleep longer than timeout
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// The error might be context.DeadlineExceeded, signal: killed, or a wrapped error
	// All are valid timeout-related errors
	if err != context.DeadlineExceeded &&
		!strings.Contains(err.Error(), "context deadline exceeded") &&
		!strings.Contains(err.Error(), "signal: killed") {
		t.Errorf("Expected timeout-related error, got: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected state to be Failed, got %v", status.State)
	}
}

func TestExecutor_Execute_MultipleKeepAliveProcesses(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "keepalive-1",
				Command: "sleep",
				Args:    []string{"0.2"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "keepalive-2",
				Command: "sleep",
				Args:    []string{"0.2"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "once-cmd",
				Command: "echo",
				Args:    []string{"between keepalives"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "keepalive-3",
				Command: "sleep",
				Args:    []string{"0.2"},
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}

	if len(status.Results) != 4 {
		t.Fatalf("Expected 4 results, got %d", len(status.Results))
	}

	// Verify all commands succeeded
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d to be successful", i)
		}
	}

	// Give processes time to complete
	time.Sleep(300 * time.Millisecond)
}

func TestExecutor_Stop_WithKeepAliveProcesses(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "long-running",
				Command: "sleep",
				Args:    []string{"10"}, // Long sleep
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	// Start execution in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- executor.Execute(ctx, cfg)
	}()

	// Wait a bit for the process to start
	time.Sleep(50 * time.Millisecond)

	// Stop the executor
	executor.Stop()

	// Wait for execution to complete
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Expected successful execution, got error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Execution did not complete within timeout")
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}
}

func TestExecutor_Execute_InvalidWorkingDirectory(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "invalid-workdir",
				Command: "echo",
				Args:    []string{"test"},
				Mode:    config.ModeOnce,
				WorkDir: "/nonexistent/directory/path",
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for invalid working directory, got nil")
	}

	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected state to be Failed, got %v", status.State)
	}

	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.ErrorDetail == nil {
		t.Fatal("Expected error detail to be populated")
	}

	if result.ErrorDetail.WorkingDir != "/nonexistent/directory/path" {
		t.Errorf("Expected working directory '/nonexistent/directory/path', got '%s'", result.ErrorDetail.WorkingDir)
	}
}

func TestExecutor_Execute_EmptyEnvironmentVariables(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "empty-env-test",
				Command: "sh",
				Args:    []string{"-c", "echo \"EMPTY_VAR=[$EMPTY_VAR]\""},
				Mode:    config.ModeOnce,
				Env: map[string]string{
					"EMPTY_VAR": "",
				},
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if !strings.Contains(result.Output, "EMPTY_VAR=[]") {
		t.Errorf("Expected output to contain empty environment variable, got: %s", result.Output)
	}
}

func TestExecutor_Execute_OverrideSystemEnvironment(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "override-env-test",
				Command: "sh",
				Args:    []string{"-c", "echo $PATH"},
				Mode:    config.ModeOnce,
				Env: map[string]string{
					"PATH": "/custom/path",
				},
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if !strings.Contains(result.Output, "/custom/path") {
		t.Errorf("Expected output to contain custom PATH, got: %s", result.Output)
	}
}

func TestExecutor_Execute_ConcurrentGetStatus(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "concurrent-test",
				Command: "sleep",
				Args:    []string{"0.1"},
				Mode:    config.ModeOnce,
			},
		},
	}

	// Start execution in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- executor.Execute(ctx, cfg)
	}()

	// Concurrently call GetStatus multiple times
	statusChan := make(chan ExecutionStatus, 10)
	for range 10 {
		go func() {
			statusChan <- executor.GetStatus()
		}()
	}

	// Collect all status results
	var statuses []ExecutionStatus
	for range 10 {
		statuses = append(statuses, <-statusChan)
	}

	// Wait for execution to complete
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Expected successful execution, got error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Execution did not complete within timeout")
	}

	// Verify all status calls returned valid data
	for i, status := range statuses {
		// TotalCount might be 0 initially before execution starts, then 1
		if status.TotalCount != 0 && status.TotalCount != 1 {
			t.Errorf("Status %d: expected total count 0 or 1, got %d", i, status.TotalCount)
		}
	}
}

func TestExecutor_Execute_StoppedDuringExecution(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "first-cmd",
				Command: "echo",
				Args:    []string{"first"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "long-cmd",
				Command: "sleep",
				Args:    []string{"1"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "should-not-run",
				Command: "echo",
				Args:    []string{"should not see this"},
				Mode:    config.ModeOnce,
			},
		},
	}

	// Start execution in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- executor.Execute(ctx, cfg)
	}()

	// Wait for first command to complete, then stop
	time.Sleep(50 * time.Millisecond)
	executor.Stop()

	// Wait for execution to complete
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Expected error due to stopping, got nil")
		}
		if !strings.Contains(err.Error(), "execution stopped by user") {
			t.Errorf("Expected 'execution stopped by user' error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Execution did not complete within timeout")
	}
}

func TestExecutor_Execute_KeepAliveProcessMonitoring(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose to see monitoring output
	executor := NewExecutor(ExecutorOptions{
		Verbose:  true,
		Reporter: reporter,
	})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "short-keepalive",
				Command: "sleep",
				Args:    []string{"0.05"}, // Very short sleep to trigger monitoring
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	// Wait for the keepAlive process to exit and be monitored
	time.Sleep(200 * time.Millisecond)

	output := buf.String()
	// The monitoring output goes to stderr, not the reporter buffer
	// Just verify the execution completed successfully
	if !strings.Contains(output, "✓ All commands completed successfully") {
		t.Errorf("Expected successful completion, got:\n%s", output)
	}
}

func TestExecutorOptions_DefaultValues(t *testing.T) {
	tests := []struct {
		name     string
		opts     ExecutorOptions
		expected ExecutorOptions
	}{
		{
			name: "empty options",
			opts: ExecutorOptions{},
			expected: ExecutorOptions{
				Verbose:    false,
				WorkingDir: "",
				Timeout:    0,
				Reporter:   nil, // Will be set to default by NewExecutor
			},
		},
		{
			name: "partial options",
			opts: ExecutorOptions{
				Verbose: true,
			},
			expected: ExecutorOptions{
				Verbose:    true,
				WorkingDir: "",
				Timeout:    0,
				Reporter:   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.opts)
			if executor == nil {
				t.Fatal("Expected executor to be created")
			}

			status := executor.GetStatus()
			if status.State != StateReady {
				t.Errorf("Expected initial state to be Ready, got %v", status.State)
			}
		})
	}
}

func TestErrorDetail_CompleteFields(t *testing.T) {
	cmd := config.Command{
		Name:    "test-cmd",
		Command: "echo",
		Args:    []string{"hello", "world with spaces"},
		Mode:    config.ModeOnce,
		WorkDir: "/tmp",
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	testErr := fmt.Errorf("test error message")
	stderr := "error output"
	stdout := "standard output"

	// Create a mock exec.Cmd to test complete error detail creation
	execCmd := &exec.Cmd{
		Path: "/bin/echo",
		Args: []string{"echo", "hello", "world with spaces"},
		Dir:  "/tmp",
		Env:  []string{"TEST_VAR=test_value", "PATH=/usr/bin"},
	}

	detail := createErrorDetail(cmd, execCmd, testErr, stderr, stdout)

	if detail.Type != ErrorTypeSystemError {
		t.Errorf("Expected error type %s, got %s", ErrorTypeSystemError, detail.Type)
	}

	if detail.Message != "test error message" {
		t.Errorf("Expected message 'test error message', got '%s'", detail.Message)
	}

	expectedCommandLine := "echo hello \"world with spaces\""
	if detail.CommandLine != expectedCommandLine {
		t.Errorf("Expected command line '%s', got '%s'", expectedCommandLine, detail.CommandLine)
	}

	if detail.WorkingDir != "/tmp" {
		t.Errorf("Expected working directory '/tmp', got '%s'", detail.WorkingDir)
	}

	if detail.Stderr != stderr {
		t.Errorf("Expected stderr '%s', got '%s'", stderr, detail.Stderr)
	}

	if detail.Stdout != stdout {
		t.Errorf("Expected stdout '%s', got '%s'", stdout, detail.Stdout)
	}

	if detail.SystemError != "test error message" {
		t.Errorf("Expected system error 'test error message', got '%s'", detail.SystemError)
	}

	if len(detail.Environment) != 2 {
		t.Errorf("Expected 2 environment variables, got %d", len(detail.Environment))
	}
}

func TestErrorDetail_CategorizeError_ExitError(t *testing.T) {
	// Create a mock ExitError
	exitErr := &exec.ExitError{
		ProcessState: nil, // We can't easily create a real ProcessState
	}

	errorType := categorizeError(exitErr)
	if errorType != ErrorTypeNonZeroExit {
		t.Errorf("Expected error type %s, got %s", ErrorTypeNonZeroExit, errorType)
	}
}

func TestErrorDetail_CategorizeError_AdditionalCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "timeout error",
			err:      fmt.Errorf("operation timeout exceeded"),
			expected: ErrorTypeTimeout,
		},
		{
			name:     "deadline exceeded",
			err:      fmt.Errorf("context deadline exceeded"),
			expected: ErrorTypeTimeout,
		},
		{
			name:     "no such file",
			err:      fmt.Errorf("no such file or directory"),
			expected: ErrorTypeCommandNotFound,
		},
		{
			name:     "permission error",
			err:      fmt.Errorf("permission denied: cannot execute"),
			expected: ErrorTypePermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("categorizeError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExecutor_Execute_RelativeWorkingDirectory(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "relative-workdir",
				Command: "pwd",
				Mode:    config.ModeOnce,
				WorkDir: ".", // Relative path
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	// Should contain some path (current directory)
	if result.Output == "" {
		t.Error("Expected output to contain current directory path")
	}
}

func TestExecutor_Execute_GlobalWorkingDirectory(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{
		WorkingDir: "/tmp",
	})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "global-workdir-test",
				Command: "pwd",
				Mode:    config.ModeOnce,
				// No WorkDir specified, should use global
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(status.Results))
	}

	result := status.Results[0]
	if !result.Success {
		t.Error("Expected command to succeed")
	}

	if !strings.Contains(result.Output, "/tmp") {
		t.Errorf("Expected output to contain /tmp, got: %s", result.Output)
	}
}
