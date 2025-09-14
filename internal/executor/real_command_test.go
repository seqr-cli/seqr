package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// TestRealCommands_OnceMode tests "once" mode with real system commands
func TestRealCommands_OnceMode(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	tests := []struct {
		name           string
		command        string
		args           []string
		expectSuccess  bool
		expectedOutput string
		workDir        string
		env            map[string]string
	}{
		{
			name:           "echo command",
			command:        "echo",
			args:           []string{"Hello, World!"},
			expectSuccess:  true,
			expectedOutput: "Hello, World!",
		},
		{
			name:          "pwd command",
			command:       "pwd",
			args:          []string{},
			expectSuccess: true,
			workDir:       "/tmp",
		},
		{
			name:           "environment variable test",
			command:        getShellCommand(),
			args:           []string{"-c", "echo $TEST_VAR"},
			expectSuccess:  true,
			expectedOutput: "test_value",
			env:            map[string]string{"TEST_VAR": "test_value"},
		},
		{
			name:          "failing command",
			command:       "false",
			args:          []string{},
			expectSuccess: false,
		},
		{
			name:          "nonexistent command",
			command:       "nonexistent_command_xyz123",
			args:          []string{},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Version: "1.0",
				Commands: []config.Command{
					{
						Name:    tt.name,
						Command: tt.command,
						Args:    tt.args,
						Mode:    config.ModeOnce,
						WorkDir: tt.workDir,
						Env:     tt.env,
					},
				},
			}

			err := executor.Execute(ctx, cfg)
			status := executor.GetStatus()

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if status.State != StateSuccess {
					t.Errorf("Expected state Success, got %v", status.State)
				}
				if len(status.Results) != 1 || !status.Results[0].Success {
					t.Errorf("Expected successful result")
				}
				if tt.expectedOutput != "" && !strings.Contains(status.Results[0].Output, tt.expectedOutput) {
					t.Errorf("Expected output to contain '%s', got '%s'", tt.expectedOutput, status.Results[0].Output)
				}
			} else {
				if err == nil {
					t.Errorf("Expected failure but got success")
				}
				if status.State != StateFailed {
					t.Errorf("Expected state Failed, got %v", status.State)
				}
				if len(status.Results) != 1 || status.Results[0].Success {
					t.Errorf("Expected failed result")
				}
			}
		})
	}
}

// TestRealCommands_KeepAliveMode tests "keepAlive" mode with real system commands
func TestRealCommands_KeepAliveMode(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	tests := []struct {
		name          string
		command       string
		args          []string
		expectSuccess bool
		duration      time.Duration
	}{
		{
			name:          "short sleep",
			command:       "sleep",
			args:          []string{"0.1"},
			expectSuccess: true,
			duration:      200 * time.Millisecond,
		},
		{
			name:          "medium sleep",
			command:       "sleep",
			args:          []string{"0.3"},
			expectSuccess: true,
			duration:      500 * time.Millisecond,
		},
		{
			name:          "nonexistent keepalive command",
			command:       "nonexistent_keepalive_xyz123",
			args:          []string{},
			expectSuccess: false,
			duration:      100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Version: "1.0",
				Commands: []config.Command{
					{
						Name:    tt.name,
						Command: tt.command,
						Args:    tt.args,
						Mode:    config.ModeKeepAlive,
					},
				},
			}

			start := time.Now()
			err := executor.Execute(ctx, cfg)
			executionTime := time.Since(start)

			status := executor.GetStatus()

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if status.State != StateSuccess {
					t.Errorf("Expected state Success, got %v", status.State)
				}
				if len(status.Results) != 1 || !status.Results[0].Success {
					t.Errorf("Expected successful result")
				}
				// KeepAlive should return quickly, not wait for process completion
				if executionTime > 1*time.Second {
					t.Errorf("KeepAlive execution took too long: %v", executionTime)
				}
				// Verify PID is mentioned in output
				if !strings.Contains(status.Results[0].Output, "PID") {
					t.Errorf("Expected output to contain PID information, got: %s", status.Results[0].Output)
				}
			} else {
				if err == nil {
					t.Errorf("Expected failure but got success")
				}
				if status.State != StateFailed {
					t.Errorf("Expected state Failed, got %v", status.State)
				}
			}

			// Wait for background process to complete if it was started
			if tt.expectSuccess {
				time.Sleep(tt.duration)
			}
		})
	}
}

// TestRealCommands_MixedModes tests both execution modes in a single configuration
func TestRealCommands_MixedModes(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "setup-echo",
				Command: "echo",
				Args:    []string{"Setting up..."},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "background-sleep",
				Command: "sleep",
				Args:    []string{"0.2"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "check-date",
				Command: "date",
				Mode:    config.ModeOnce,
			},
			{
				Name:    "another-background",
				Command: "sleep",
				Args:    []string{"0.1"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "final-echo",
				Command: "echo",
				Args:    []string{"All done!"},
				Mode:    config.ModeOnce,
			},
		},
	}

	start := time.Now()
	err := executor.Execute(ctx, cfg)
	executionTime := time.Since(start)

	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected state Success, got %v", status.State)
	}

	if len(status.Results) != 5 {
		t.Fatalf("Expected 5 results, got %d", len(status.Results))
	}

	// Verify all commands succeeded
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d (%s) to be successful", i, result.Command.Name)
		}
	}

	// Verify specific outputs
	if !strings.Contains(status.Results[0].Output, "Setting up...") {
		t.Errorf("Expected first command output to contain 'Setting up...', got: %s", status.Results[0].Output)
	}

	if !strings.Contains(status.Results[4].Output, "All done!") {
		t.Errorf("Expected last command output to contain 'All done!', got: %s", status.Results[4].Output)
	}

	// Verify keepAlive commands have PID information
	if !strings.Contains(status.Results[1].Output, "PID") {
		t.Errorf("Expected keepAlive result to contain PID, got: %s", status.Results[1].Output)
	}

	if !strings.Contains(status.Results[3].Output, "PID") {
		t.Errorf("Expected keepAlive result to contain PID, got: %s", status.Results[3].Output)
	}

	// Execution should be fast (keepAlive doesn't wait)
	if executionTime > 2*time.Second {
		t.Errorf("Mixed mode execution took too long: %v", executionTime)
	}

	// Wait for background processes to complete
	time.Sleep(400 * time.Millisecond)

	t.Logf("Successfully executed mixed mode commands in %v", executionTime)
}

// TestRealCommands_EdgeCases tests various edge cases with real commands
func TestRealCommands_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) *config.Config
		expectError bool
		errorCheck  func(t *testing.T, err error, status ExecutionStatus)
	}{
		{
			name: "command with special characters in args",
			setupFunc: func(t *testing.T) *config.Config {
				return &config.Config{
					Version: "1.0",
					Commands: []config.Command{
						{
							Name:    "special-chars",
							Command: "echo",
							Args:    []string{"Hello & World | Test > Output"},
							Mode:    config.ModeOnce,
						},
					},
				}
			},
			expectError: false,
		},
		{
			name: "command with empty args",
			setupFunc: func(t *testing.T) *config.Config {
				return &config.Config{
					Version: "1.0",
					Commands: []config.Command{
						{
							Name:    "empty-args",
							Command: "echo",
							Args:    []string{},
							Mode:    config.ModeOnce,
						},
					},
				}
			},
			expectError: false,
		},
		{
			name: "command with nonexistent working directory",
			setupFunc: func(t *testing.T) *config.Config {
				return &config.Config{
					Version: "1.0",
					Commands: []config.Command{
						{
							Name:    "bad-workdir",
							Command: "echo",
							Args:    []string{"test"},
							Mode:    config.ModeOnce,
							WorkDir: "/nonexistent/directory/path/xyz123",
						},
					},
				}
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error, status ExecutionStatus) {
				if !strings.Contains(err.Error(), "no such file or directory") &&
					!strings.Contains(err.Error(), "cannot find the path") {
					t.Errorf("Expected directory not found error, got: %v", err)
				}
			},
		},
		{
			name: "command that produces large output",
			setupFunc: func(t *testing.T) *config.Config {
				return &config.Config{
					Version: "1.0",
					Commands: []config.Command{
						{
							Name:    "large-output",
							Command: getShellCommand(),
							Args:    []string{"-c", "for i in {1..100}; do echo \"Line $i of large output\"; done"},
							Mode:    config.ModeOnce,
						},
					},
				}
			},
			expectError: false,
		},
		{
			name: "keepAlive command that exits immediately",
			setupFunc: func(t *testing.T) *config.Config {
				return &config.Config{
					Version: "1.0",
					Commands: []config.Command{
						{
							Name:    "immediate-exit",
							Command: "echo",
							Args:    []string{"immediate exit"},
							Mode:    config.ModeKeepAlive,
						},
					},
				}
			},
			expectError: false,
		},
		{
			name: "command with multiple environment variables",
			setupFunc: func(t *testing.T) *config.Config {
				return &config.Config{
					Version: "1.0",
					Commands: []config.Command{
						{
							Name:    "multi-env",
							Command: getShellCommand(),
							Args:    []string{"-c", "echo \"VAR1=$VAR1, VAR2=$VAR2, VAR3=$VAR3\""},
							Mode:    config.ModeOnce,
							Env: map[string]string{
								"VAR1": "value1",
								"VAR2": "value with spaces",
								"VAR3": "value_with_special_chars!@#$%",
							},
						},
					},
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(ExecutorOptions{Verbose: true})
			ctx := context.Background()
			cfg := tt.setupFunc(t)

			err := executor.Execute(ctx, cfg)
			status := executor.GetStatus()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got success")
				}
				if status.State != StateFailed {
					t.Errorf("Expected state Failed, got %v", status.State)
				}
				if tt.errorCheck != nil {
					tt.errorCheck(t, err, status)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if status.State != StateSuccess {
					t.Errorf("Expected state Success, got %v", status.State)
				}
				if len(status.Results) == 0 || !status.Results[0].Success {
					t.Errorf("Expected successful result")
				}
			}
		})
	}
}

// TestRealCommands_ProcessManagement tests process lifecycle management
func TestRealCommands_ProcessManagement(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	// Test multiple keepAlive processes
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "process-1",
				Command: "sleep",
				Args:    []string{"0.3"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "process-2",
				Command: "sleep",
				Args:    []string{"0.2"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "process-3",
				Command: "sleep",
				Args:    []string{"0.1"},
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	start := time.Now()
	err := executor.Execute(ctx, cfg)
	executionTime := time.Since(start)

	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected state Success, got %v", status.State)
	}

	// All processes should start successfully
	if len(status.Results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(status.Results))
	}

	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d to be successful", i)
		}
		if !strings.Contains(result.Output, "PID") {
			t.Errorf("Expected result %d to contain PID information", i)
		}
	}

	// Execution should be fast (doesn't wait for processes)
	if executionTime > 1*time.Second {
		t.Errorf("Process management execution took too long: %v", executionTime)
	}

	// Wait for all processes to complete
	time.Sleep(500 * time.Millisecond)

	t.Logf("Successfully managed 3 keepAlive processes in %v", executionTime)
}

// TestRealCommands_StopDuringExecution tests stopping execution with active processes
func TestRealCommands_StopDuringExecution(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "quick-setup",
				Command: "echo",
				Args:    []string{"setup complete"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "long-running",
				Command: "sleep",
				Args:    []string{"5"}, // Long enough to be stopped
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "should-not-reach",
				Command: "echo",
				Args:    []string{"this should not run"},
				Mode:    config.ModeOnce,
			},
		},
	}

	// Start execution in background
	done := make(chan error, 1)
	go func() {
		done <- executor.Execute(ctx, cfg)
	}()

	// Wait for first command and keepAlive to start
	time.Sleep(100 * time.Millisecond)

	// Stop execution
	executor.Stop()

	// Wait for execution to complete
	select {
	case err := <-done:
		// Execution might complete successfully if it was fast enough
		// or might be stopped - both are acceptable
		t.Logf("Execution completed with: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Execution did not complete within timeout after stop")
	}

	status := executor.GetStatus()

	// Should have at least the first command result
	if len(status.Results) == 0 {
		t.Error("Expected at least one result")
	}

	// First command should have succeeded
	if len(status.Results) > 0 && !status.Results[0].Success {
		t.Error("Expected first command to succeed")
	}

	t.Logf("Stop test completed with %d results", len(status.Results))
}

// TestRealCommands_ContextCancellation tests context cancellation during execution
func TestRealCommands_ContextCancellation(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx, cancel := context.WithCancel(context.Background())

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
				Args:    []string{"5"},
				Mode:    config.ModeOnce,
			},
		},
	}

	// Start execution in background
	done := make(chan error, 1)
	go func() {
		done <- executor.Execute(ctx, cfg)
	}()

	// Cancel context after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for execution to complete
	select {
	case err := <-done:
		if err == nil {
			t.Error("Expected error due to context cancellation")
		}
		// Context cancellation can manifest as different errors depending on timing
		if err != context.Canceled &&
			!strings.Contains(err.Error(), "context canceled") &&
			!strings.Contains(err.Error(), "signal: killed") {
			t.Errorf("Expected context cancellation or signal killed error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Execution did not complete within timeout after cancellation")
	}

	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected state Failed after cancellation, got %v", status.State)
	}

	t.Logf("Context cancellation test completed")
}

// TestRealCommands_FileOperations tests commands that work with files
func TestRealCommands_FileOperations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "create-file",
				Command: getShellCommand(),
				Args:    []string{"-c", fmt.Sprintf("echo 'Hello from file' > %s", testFile)},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "read-file",
				Command: "cat",
				Args:    []string{testFile},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "list-directory",
				Command: "ls",
				Args:    []string{"-la", tempDir},
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
		t.Errorf("Expected state Success, got %v", status.State)
	}

	if len(status.Results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(status.Results))
	}

	// Verify file was created and read correctly
	if !strings.Contains(status.Results[1].Output, "Hello from file") {
		t.Errorf("Expected file content in output, got: %s", status.Results[1].Output)
	}

	// Verify directory listing shows the file
	if !strings.Contains(status.Results[2].Output, "test.txt") {
		t.Errorf("Expected file to appear in directory listing, got: %s", status.Results[2].Output)
	}

	t.Logf("File operations test completed successfully")
}

// getShellCommand returns the appropriate shell command for the current OS
func getShellCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}
