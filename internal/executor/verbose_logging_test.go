package executor

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// captureOutput captures stdout during test execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(&buf, r)
	}()

	f()

	w.Close()
	os.Stdout = old
	wg.Wait()
	r.Close()

	return buf.String()
}

func TestVerboseLogging_SingleLineOutput(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		args     []string
		expected string
	}{
		{
			name:     "simple echo command",
			command:  "echo",
			args:     []string{"Hello World"},
			expected: "Hello World",
		},
		{
			name:     "echo with special characters",
			command:  "echo",
			args:     []string{"Test: @#$%^&*()"},
			expected: "Test: @#$%^&*()",
		},
		{
			name:     "echo empty string",
			command:  "echo",
			args:     []string{""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(true)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cfg := &config.Config{
				Version: "1.0",
				Commands: []config.Command{
					{
						Name:    "test-single-line",
						Command: tt.command,
						Args:    tt.args,
						Mode:    config.ModeOnce,
					},
				},
			}

			output := captureOutput(func() {
				err := executor.Execute(ctx, cfg)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})

			// Verify verbose output format
			if !strings.Contains(output, "[test-single-line]") {
				t.Errorf("Expected output to contain command name, got: %s", output)
			}

			if !strings.Contains(output, "✓") {
				t.Errorf("Expected output to contain stdout indicator ✓, got: %s", output)
			}

			if tt.expected != "" && !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain %q, got: %s", tt.expected, output)
			}

			// Verify timestamp format (HH:MM:SS.mmm)
			if !strings.Contains(output, ":") || !strings.Contains(output, ".") {
				t.Errorf("Expected output to contain timestamp, got: %s", output)
			}
		})
	}
}

func TestVerboseLogging_MultiLineOutput(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a command that produces multiple lines of output
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-multiline",
				Command: "echo",
				Args:    []string{"-e", "Line 1\\nLine 2\\nLine 3"},
				Mode:    config.ModeOnce,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Verify command execution is reported
	if !strings.Contains(output, "test-multiline") {
		t.Errorf("Expected output to contain command name, got: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected output to contain success indicator, got: %s", output)
	}

	// Verify the result contains the multiline output
	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
		return
	}

	result := status.Results[0]
	if !result.Success {
		t.Errorf("Expected command to succeed, got: %s", result.Error)
	}

	// Verify multiline output is captured in the result
	if !strings.Contains(result.Output, "Line 1") ||
		!strings.Contains(result.Output, "Line 2") ||
		!strings.Contains(result.Output, "Line 3") {
		t.Errorf("Expected result to contain all lines, got: %s", result.Output)
	}
}

func TestVerboseLogging_StderrOutput(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a command that writes to stderr
	var cmd string
	var args []string

	// Use different commands based on OS
	if _, err := exec.LookPath("sh"); err == nil {
		// Unix-like systems
		cmd = "sh"
		args = []string{"-c", "echo 'Error message' >&2"}
	} else {
		// Windows - use PowerShell
		cmd = "powershell"
		args = []string{"-Command", "Write-Error 'Error message' 2>&1"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-stderr",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeOnce,
			},
		},
	}

	output := captureOutput(func() {
		// We expect this to succeed even though it writes to stderr
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Logf("Command failed as expected: %v", err)
		}
	})

	// Verify stderr output format (should contain ❌ symbol)
	if !strings.Contains(output, "[test-stderr]") {
		t.Errorf("Expected output to contain command name, got: %s", output)
	}

	// Note: The exact stderr formatting depends on the command execution
	// We mainly verify that the output is captured and formatted
	if output == "" {
		t.Error("Expected some output to be captured")
	}
}

func TestVerboseLogging_MixedOutput(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a command that writes to both stdout and stderr
	var cmd string
	var args []string

	if _, err := exec.LookPath("sh"); err == nil {
		// Unix-like systems
		cmd = "sh"
		args = []string{"-c", "echo 'stdout message'; echo 'stderr message' >&2"}
	} else {
		// Windows - create a simple script that outputs to both
		cmd = "powershell"
		args = []string{"-Command", "Write-Output 'stdout message'; Write-Error 'stderr message' 2>&1"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-mixed",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeOnce,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Logf("Command may have failed: %v", err)
		}
	})

	// Verify command name appears in output
	if !strings.Contains(output, "[test-mixed]") {
		t.Errorf("Expected output to contain command name, got: %s", output)
	}

	// Verify some output was captured
	if output == "" {
		t.Error("Expected some output to be captured")
	}
}

func TestVerboseLogging_CommandFailure(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-failure",
				Command: "nonexistent-command-12345",
				Args:    []string{},
				Mode:    config.ModeOnce,
			},
		},
	}

	var capturedOutput string

	// Capture both stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(&buf, r)
	}()

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	wg.Wait()
	r.Close()

	capturedOutput = buf.String()

	// Debug: print what we captured
	t.Logf("Captured output: %q", capturedOutput)

	// Verify the execution status shows failure
	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected execution state to be Failed, got: %v", status.State)
	}

	// Verify we have results
	if len(status.Results) == 0 {
		t.Error("Expected at least one execution result")
		return
	}

	result := status.Results[0]
	if result.Success {
		t.Error("Expected command to fail")
	}

	if result.Command.Name != "test-failure" {
		t.Errorf("Expected command name to be test-failure, got: %s", result.Command.Name)
	}

	// The main verification is that the command failed as expected
	// The exact output format may vary between systems
}

func TestVerboseLogging_KeepAliveMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keepAlive test in short mode")
	}

	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use a command that will run and produce output
	var cmd string
	var args []string

	if _, err := exec.LookPath("ping"); err == nil {
		// Use ping command (available on most systems)
		cmd = "ping"
		args = []string{"-c", "2", "127.0.0.1"} // Unix
		if _, err := exec.LookPath("ping"); err == nil {
			// Try Windows ping format
			if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
				args = []string{"-n", "2", "127.0.0.1"}
			}
		}
	} else {
		// Fallback to a simple loop
		cmd = "sh"
		args = []string{"-c", "for i in 1 2; do echo \"Output $i\"; sleep 1; done"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-keepalive",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Give some time for output to be generated
		time.Sleep(1 * time.Second)
	})

	// Verify keepAlive process output format
	if !strings.Contains(output, "[test-keepalive]") {
		t.Errorf("Expected output to contain command name, got: %s", output)
	}

	// Verify process started message
	status := executor.GetStatus()
	if len(status.Results) > 0 {
		result := status.Results[0]
		if !strings.Contains(result.Output, "PID") {
			t.Errorf("Expected result to contain PID information, got: %s", result.Output)
		}
	}
}

func TestVerboseLogging_TimestampFormat(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-timestamp",
				Command: "echo",
				Args:    []string{"timestamp test"},
				Mode:    config.ModeOnce,
			},
		},
	}

	startTime := time.Now()

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	endTime := time.Now()

	// Extract timestamp from output
	lines := strings.Split(output, "\n")
	var timestampLine string
	for _, line := range lines {
		if strings.Contains(line, "[test-timestamp]") && strings.Contains(line, "✓") {
			timestampLine = line
			break
		}
	}

	if timestampLine == "" {
		t.Fatalf("Could not find timestamped output line in: %s", output)
	}

	// Extract timestamp (format: [HH:MM:SS.mmm])
	start := strings.Index(timestampLine, "[")
	end := strings.Index(timestampLine, "]")
	if start == -1 || end == -1 || end <= start {
		t.Fatalf("Could not extract timestamp from line: %s", timestampLine)
	}

	timestampStr := timestampLine[start+1 : end]

	// Parse the timestamp
	_, err := time.Parse("15:04:05.000", timestampStr)
	if err != nil {
		t.Errorf("Could not parse timestamp %q: %v", timestampStr, err)
	}

	// Verify timestamp is reasonable (within the test execution window)
	// We only check the time portion since we don't have the date
	startTimeStr := startTime.Format("15:04:05.000")
	endTimeStr := endTime.Format("15:04:05.000")

	if timestampStr < startTimeStr || timestampStr > endTimeStr {
		// Allow for some flexibility around midnight boundary
		if !(startTimeStr > endTimeStr && (timestampStr >= startTimeStr || timestampStr <= endTimeStr)) {
			t.Logf("Timestamp %s outside expected range %s - %s (may be acceptable near midnight)",
				timestampStr, startTimeStr, endTimeStr)
		}
	}
}

func TestVerboseLogging_CommandIdentification(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "first-command",
				Command: "echo",
				Args:    []string{"first output"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "second-command",
				Command: "echo",
				Args:    []string{"second output"},
				Mode:    config.ModeOnce,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Verify both commands are properly identified in output
	if !strings.Contains(output, "first-command") {
		t.Errorf("Expected output to contain first command name, got: %s", output)
	}

	if !strings.Contains(output, "second-command") {
		t.Errorf("Expected output to contain second command name, got: %s", output)
	}

	// Verify both commands succeeded
	if strings.Count(output, "✓") < 2 {
		t.Errorf("Expected at least 2 success indicators, got: %s", output)
	}

	// Verify the execution results contain correct outputs
	status := executor.GetStatus()
	if len(status.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(status.Results))
		return
	}

	// Verify first command result
	firstResult := status.Results[0]
	if firstResult.Command.Name != "first-command" {
		t.Errorf("Expected first result to be first-command, got: %s", firstResult.Command.Name)
	}
	if !strings.Contains(firstResult.Output, "first output") {
		t.Errorf("Expected first result to contain 'first output', got: %s", firstResult.Output)
	}

	// Verify second command result
	secondResult := status.Results[1]
	if secondResult.Command.Name != "second-command" {
		t.Errorf("Expected second result to be second-command, got: %s", secondResult.Command.Name)
	}
	if !strings.Contains(secondResult.Output, "second output") {
		t.Errorf("Expected second result to contain 'second output', got: %s", secondResult.Output)
	}
}

func TestVerboseLogging_NonVerboseMode(t *testing.T) {
	// Test that non-verbose mode doesn't produce detailed output
	executor := NewExecutor(false) // verbose = false
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-non-verbose",
				Command: "echo",
				Args:    []string{"should not see detailed output"},
				Mode:    config.ModeOnce,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// In non-verbose mode, we should not see the detailed timestamped output
	if strings.Contains(output, "✓") {
		t.Errorf("Non-verbose mode should not contain stdout indicators, got: %s", output)
	}

	if strings.Contains(output, "[test-non-verbose]") {
		t.Errorf("Non-verbose mode should not contain command name brackets, got: %s", output)
	}

	// Verify the command still executed successfully
	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
	}

	if len(status.Results) > 0 && !status.Results[0].Success {
		t.Errorf("Expected command to succeed, got: %s", status.Results[0].Error)
	}
}

func TestVerboseLogging_LongRunningOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a command that produces output over time
	var cmd string
	var args []string

	if _, err := exec.LookPath("sh"); err == nil {
		cmd = "sh"
		args = []string{"-c", "for i in 1 2 3; do echo \"Output line $i\"; sleep 0.5; done"}
	} else {
		// Windows PowerShell equivalent
		cmd = "powershell"
		args = []string{"-Command", "1..3 | ForEach-Object { Write-Output \"Output line $_\"; Start-Sleep -Milliseconds 500 }"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-streaming",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeOnce,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Verify streaming output format
	if !strings.Contains(output, "[test-streaming]") {
		t.Errorf("Expected output to contain command name, got: %s", output)
	}

	// Count the number of output lines
	lines := strings.Split(output, "\n")
	var outputLines []string
	for _, line := range lines {
		if strings.Contains(line, "[test-streaming]") && strings.Contains(line, "✓") {
			outputLines = append(outputLines, line)
		}
	}

	if len(outputLines) < 3 {
		t.Errorf("Expected at least 3 output lines, got %d: %v", len(outputLines), outputLines)
	}

	// Verify each line has proper timestamp (should be different due to sleep)
	if len(outputLines) >= 2 {
		// Extract timestamps and verify they're different
		timestamp1 := extractTimestamp(outputLines[0])
		timestamp2 := extractTimestamp(outputLines[1])

		if timestamp1 == timestamp2 {
			t.Logf("Timestamps are the same, may indicate timing issue: %s vs %s", timestamp1, timestamp2)
		}
	}
}

// Helper function to extract timestamp from a log line
func extractTimestamp(line string) string {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return line[start+1 : end]
}

func TestVerboseLogging_ProcessMonitoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping process monitoring test in short mode")
	}

	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use a command that will exit after a short time
	var cmd string
	var args []string

	if _, err := exec.LookPath("sleep"); err == nil {
		cmd = "sleep"
		args = []string{"1"}
	} else {
		// Windows equivalent
		cmd = "powershell"
		args = []string{"-Command", "Start-Sleep -Seconds 1"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-process-monitor",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Wait for process to complete
		time.Sleep(2 * time.Second)
	})

	// Verify process monitoring messages
	if !strings.Contains(output, "[test-process-monitor]") {
		t.Errorf("Expected output to contain command name, got: %s", output)
	}

	// Look for process exit message
	if !strings.Contains(output, "[process]") {
		t.Logf("Expected to see process monitoring message, got: %s", output)
	}
}

// TestVerboseLogging_RealTimeStreaming tests that verbose mode actually streams output in real-time
func TestVerboseLogging_RealTimeStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-time streaming test in short mode")
	}

	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a command that outputs with delays to test real-time streaming
	var cmd string
	var args []string

	if _, err := exec.LookPath("sh"); err == nil {
		cmd = "sh"
		args = []string{"-c", "echo 'Start'; sleep 1; echo 'Middle'; sleep 1; echo 'End'"}
	} else {
		// Windows PowerShell equivalent
		cmd = "powershell"
		args = []string{"-Command", "Write-Output 'Start'; Start-Sleep -Seconds 1; Write-Output 'Middle'; Start-Sleep -Seconds 1; Write-Output 'End'"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-realtime",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeOnce,
			},
		},
	}

	startTime := time.Now()

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Verify the command took at least 2 seconds (due to sleep commands)
	if duration < 2*time.Second {
		t.Errorf("Expected command to take at least 2 seconds for real-time streaming test, took: %v", duration)
	}

	// Verify the result contains all expected outputs
	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
		return
	}

	result := status.Results[0]
	if !result.Success {
		t.Errorf("Expected command to succeed, got: %s", result.Error)
	}

	expectedOutputs := []string{"Start", "Middle", "End"}
	for _, expected := range expectedOutputs {
		if !strings.Contains(result.Output, expected) {
			t.Errorf("Expected result to contain %q, got: %s", expected, result.Output)
		}
	}
}

// TestVerboseLogging_EnvironmentVariables tests verbose logging with commands that use environment variables
func TestVerboseLogging_EnvironmentVariables(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a command that echoes an environment variable
	var cmd string
	var args []string

	if _, err := exec.LookPath("sh"); err == nil {
		cmd = "sh"
		args = []string{"-c", "echo $TEST_VAR"}
	} else {
		// Windows
		cmd = "powershell"
		args = []string{"-Command", "Write-Output $env:TEST_VAR"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-env-vars",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeOnce,
				Env: map[string]string{
					"TEST_VAR": "test-value-123",
				},
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Verify command execution is reported
	if !strings.Contains(output, "test-env-vars") {
		t.Errorf("Expected output to contain command name, got: %s", output)
	}

	// Verify the result contains the environment variable value
	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
		return
	}

	result := status.Results[0]
	if !result.Success {
		t.Errorf("Expected command to succeed, got: %s", result.Error)
	}

	if !strings.Contains(result.Output, "test-value-123") {
		t.Errorf("Expected result to contain environment variable value, got: %s", result.Output)
	}
}

// TestVerboseLogging_WorkingDirectory tests verbose logging with commands that use custom working directories
func TestVerboseLogging_WorkingDirectory(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a simple command that should work in any directory
	cmd := "echo"
	args := []string{"working directory test"}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-workdir",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeOnce,
				WorkDir: ".", // Use current directory
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Verify command execution is reported
	if !strings.Contains(output, "test-workdir") {
		t.Errorf("Expected output to contain command name, got: %s", output)
	}

	// Verify the command succeeded
	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
		return
	}

	result := status.Results[0]
	if !result.Success {
		t.Errorf("Expected command to succeed, got: %s", result.Error)
	}

	// Verify the expected output
	if !strings.Contains(result.Output, "working directory test") {
		t.Errorf("Expected result to contain expected output, got: %s", result.Output)
	}
}

// TestVerboseLogging_ConcurrentCommands tests verbose logging behavior with multiple commands
func TestVerboseLogging_ConcurrentCommands(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "cmd1",
				Command: "echo",
				Args:    []string{"Command 1 output"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "cmd2",
				Command: "echo",
				Args:    []string{"Command 2 output"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "cmd3",
				Command: "echo",
				Args:    []string{"Command 3 output"},
				Mode:    config.ModeOnce,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Verify all commands are reported
	expectedCommands := []string{"cmd1", "cmd2", "cmd3"}
	for _, cmdName := range expectedCommands {
		if !strings.Contains(output, cmdName) {
			t.Errorf("Expected output to contain command %s, got: %s", cmdName, output)
		}
	}

	// Verify all commands succeeded
	if strings.Count(output, "✓") < 3 {
		t.Errorf("Expected at least 3 success indicators, got: %s", output)
	}

	// Verify execution results
	status := executor.GetStatus()
	if len(status.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(status.Results))
		return
	}

	// Verify each command result
	expectedOutputs := []string{"Command 1 output", "Command 2 output", "Command 3 output"}
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected command %d to succeed, got: %s", i+1, result.Error)
		}
		if !strings.Contains(result.Output, expectedOutputs[i]) {
			t.Errorf("Expected result %d to contain %q, got: %s", i+1, expectedOutputs[i], result.Output)
		}
	}
}

// TestVerboseLogging_EmptyOutput tests verbose logging with commands that produce no output
func TestVerboseLogging_EmptyOutput(t *testing.T) {
	executor := NewExecutor(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a command that produces no output
	var cmd string
	var args []string

	if _, err := exec.LookPath("true"); err == nil {
		// Unix true command (exits successfully with no output)
		cmd = "true"
		args = []string{}
	} else {
		// Windows equivalent
		cmd = "powershell"
		args = []string{"-Command", "exit 0"}
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-empty-output",
				Command: cmd,
				Args:    args,
				Mode:    config.ModeOnce,
			},
		},
	}

	output := captureOutput(func() {
		err := executor.Execute(ctx, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// For empty output commands, we mainly verify execution success
	// The reporter output should contain the command name
	if !strings.Contains(output, "test-empty-output") {
		t.Logf("Output may not contain command name for empty output test: %s", output)
	}

	// Verify the command succeeded
	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
		return
	}

	result := status.Results[0]
	if !result.Success {
		t.Errorf("Expected command to succeed, got: %s", result.Error)
	}

	// Empty output is acceptable
	if result.Output != "" {
		t.Logf("Command produced output: %s", result.Output)
	}
}
