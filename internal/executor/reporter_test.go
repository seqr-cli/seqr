package executor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestConsoleReporter_ReportStart(t *testing.T) {
	tests := []struct {
		name           string
		verbose        bool
		totalCommands  int
		expectedOutput string
	}{
		{
			name:           "normal mode",
			verbose:        false,
			totalCommands:  3,
			expectedOutput: "Starting execution of 3 command(s)\n\n",
		},
		{
			name:           "verbose mode",
			verbose:        true,
			totalCommands:  1,
			expectedOutput: "Starting execution of 1 command(s)\nVerbose mode enabled\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewConsoleReporter(&buf, tt.verbose)

			reporter.ReportStart(tt.totalCommands)

			if buf.String() != tt.expectedOutput {
				t.Errorf("Expected output:\n%q\nGot:\n%q", tt.expectedOutput, buf.String())
			}
		})
	}
}

func TestConsoleReporter_ReportCommandStart(t *testing.T) {
	tests := []struct {
		name           string
		verbose        bool
		commandName    string
		commandIndex   int
		expectedOutput string
	}{
		{
			name:           "normal mode",
			verbose:        false,
			commandName:    "test-command",
			commandIndex:   0,
			expectedOutput: "[1] Starting 'test-command'\n",
		},
		{
			name:           "verbose mode",
			verbose:        true,
			commandName:    "another-command",
			commandIndex:   2,
			expectedOutput: "[3] Starting 'another-command'...\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewConsoleReporter(&buf, tt.verbose)

			reporter.ReportCommandStart(tt.commandName, tt.commandIndex)

			if buf.String() != tt.expectedOutput {
				t.Errorf("Expected output:\n%q\nGot:\n%q", tt.expectedOutput, buf.String())
			}
		})
	}
}

func TestConsoleReporter_ReportCommandSuccess(t *testing.T) {
	cmd := config.Command{
		Name:    "test-cmd",
		Command: "echo",
		Args:    []string{"hello"},
		Mode:    config.ModeOnce,
	}

	result := ExecutionResult{
		Command:   cmd,
		Success:   true,
		ExitCode:  0,
		Output:    "hello\nworld\ntest\nmore lines",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
	}

	tests := []struct {
		name         string
		verbose      bool
		result       ExecutionResult
		commandIndex int
		contains     []string
		notContains  []string
	}{
		{
			name:         "normal mode",
			verbose:      false,
			result:       result,
			commandIndex: 0,
			contains:     []string{"[1] ✓ 'test-cmd'", "100ms"},
			notContains:  []string{"hello", "world", "completed successfully"},
		},
		{
			name:         "verbose mode with output",
			verbose:      true,
			result:       result,
			commandIndex: 1,
			contains:     []string{"[2] ✓ 'test-cmd' completed successfully", "100ms", "hello", "world", "test", "... (1 more lines)"},
			notContains:  []string{"more lines\n"}, // The actual "more lines" content should be truncated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewConsoleReporter(&buf, tt.verbose)

			reporter.ReportCommandSuccess(tt.result, tt.commandIndex)
			output := buf.String()

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain %q, got:\n%s", notExpected, output)
				}
			}
		})
	}
}

func TestConsoleReporter_ReportCommandSuccess_KeepAlive(t *testing.T) {
	cmd := config.Command{
		Name:    "background-service",
		Command: "sleep",
		Args:    []string{"10"},
		Mode:    config.ModeKeepAlive,
	}

	result := ExecutionResult{
		Command:   cmd,
		Success:   true,
		ExitCode:  0,
		Output:    "keepAlive process started with PID 12345",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(50 * time.Millisecond),
		Duration:  50 * time.Millisecond,
	}

	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	reporter.ReportCommandSuccess(result, 0)
	output := buf.String()

	expectedContains := []string{
		"[1] ✓ 'background-service' completed successfully",
		"50ms",
		"Process started and running in background",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// Should not show PID output for keepAlive commands
	if strings.Contains(output, "keepAlive process started with PID") {
		t.Errorf("Expected output to NOT show PID details for keepAlive, got:\n%s", output)
	}
}

func TestConsoleReporter_ReportCommandFailure(t *testing.T) {
	cmd := config.Command{
		Name:    "failing-cmd",
		Command: "false",
		Mode:    config.ModeOnce,
	}

	errorDetail := &ErrorDetail{
		Type:        ErrorTypeNonZeroExit,
		Message:     "exit status 1",
		CommandLine: "false",
		WorkingDir:  "/tmp",
		Stderr:      "command failed",
	}

	result := ExecutionResult{
		Command:     cmd,
		Success:     false,
		ExitCode:    1,
		Output:      "some output",
		Error:       "exit status 1",
		ErrorDetail: errorDetail,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(200 * time.Millisecond),
		Duration:    200 * time.Millisecond,
	}

	tests := []struct {
		name        string
		verbose     bool
		contains    []string
		notContains []string
	}{
		{
			name:    "normal mode",
			verbose: false,
			contains: []string{
				"[1] ✗ 'failing-cmd' failed",
				"200ms",
				"Exit code: 1",
				"Error: exit status 1",
			},
			notContains: []string{
				"Command: false",
				"Working Directory: /tmp",
				"Stderr: command failed",
				"Output: some output",
			},
		},
		{
			name:    "verbose mode",
			verbose: true,
			contains: []string{
				"[1] ✗ 'failing-cmd' failed",
				"200ms",
				"Exit code: 1",
				"Error: exit status 1",
				"Command: false",
				"Working Directory: /tmp",
				"Stderr: command failed",
				"Output: some output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewConsoleReporter(&buf, tt.verbose)

			reporter.ReportCommandFailure(result, 0)
			output := buf.String()

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain %q, got:\n%s", notExpected, output)
				}
			}
		})
	}
}

func TestConsoleReporter_ReportExecutionComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		verbose  bool
		contains []string
	}{
		{
			name: "successful execution",
			status: ExecutionStatus{
				State:          StateSuccess,
				CompletedCount: 3,
				TotalCount:     3,
			},
			verbose:  false,
			contains: []string{"✓ All commands completed successfully (3/3)"},
		},
		{
			name: "failed execution",
			status: ExecutionStatus{
				State:          StateFailed,
				CompletedCount: 2,
				TotalCount:     5,
				LastError:      "Command failed with exit code 1",
			},
			verbose:  false,
			contains: []string{"✗ Execution failed at command 2 of 5"},
		},
		{
			name: "failed execution verbose",
			status: ExecutionStatus{
				State:          StateFailed,
				CompletedCount: 1,
				TotalCount:     3,
				LastError:      "Detailed error information",
			},
			verbose:  true,
			contains: []string{"✗ Execution failed at command 1 of 3", "Error details: Detailed error information"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewConsoleReporter(&buf, tt.verbose)

			reporter.ReportExecutionComplete(tt.status)
			output := buf.String()

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestConsoleReporter_ReportExecutionSummary(t *testing.T) {
	results := []ExecutionResult{
		{
			Command:  config.Command{Name: "cmd1", Mode: config.ModeOnce},
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		{
			Command:  config.Command{Name: "cmd2", Mode: config.ModeKeepAlive},
			Success:  true,
			Duration: 50 * time.Millisecond,
		},
		{
			Command:  config.Command{Name: "cmd3", Mode: config.ModeOnce},
			Success:  false,
			Duration: 200 * time.Millisecond,
		},
	}

	status := ExecutionStatus{
		State:   StateFailed,
		Results: results,
	}

	tests := []struct {
		name         string
		verbose      bool
		expectOutput bool
		contains     []string
	}{
		{
			name:         "non-verbose mode",
			verbose:      false,
			expectOutput: false,
		},
		{
			name:         "verbose mode with results",
			verbose:      true,
			expectOutput: true,
			contains: []string{
				"Execution Summary:",
				"✓ [1] cmd1",
				"100ms",
				"✓ [2] cmd2",
				"50ms",
				"(background)",
				"✗ [3] cmd3",
				"200ms",
				"Total: 3 commands, 2 successful, 1 failed",
				"Total execution time: 350ms",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewConsoleReporter(&buf, tt.verbose)

			reporter.ReportExecutionSummary(status)
			output := buf.String()

			if !tt.expectOutput {
				if output != "" {
					t.Errorf("Expected no output in non-verbose mode, got:\n%s", output)
				}
				return
			}

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestConsoleReporter_ReportExecutionSummary_EmptyResults(t *testing.T) {
	status := ExecutionStatus{
		State:   StateReady,
		Results: []ExecutionResult{},
	}

	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	reporter.ReportExecutionSummary(status)
	output := buf.String()

	if output != "" {
		t.Errorf("Expected no output for empty results, got:\n%s", output)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "microseconds",
			duration: 500 * time.Nanosecond,
			expected: "0.50μs",
		},
		{
			name:     "milliseconds",
			duration: 150 * time.Millisecond,
			expected: "150ms",
		},
		{
			name:     "seconds",
			duration: 2500 * time.Millisecond,
			expected: "2.50s",
		},
		{
			name:     "minutes",
			duration: 90 * time.Second,
			expected: "1.5m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, result, tt.expected)
			}
		})
	}
}

// Additional comprehensive reporter tests

func TestConsoleReporter_ReportCommandSuccess_NoOutput(t *testing.T) {
	cmd := config.Command{
		Name:    "no-output-cmd",
		Command: "true",
		Mode:    config.ModeOnce,
	}

	result := ExecutionResult{
		Command:   cmd,
		Success:   true,
		ExitCode:  0,
		Output:    "", // No output
		StartTime: time.Now(),
		EndTime:   time.Now().Add(50 * time.Millisecond),
		Duration:  50 * time.Millisecond,
	}

	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	reporter.ReportCommandSuccess(result, 0)
	output := buf.String()

	expectedContains := []string{
		"[1] ✓ 'no-output-cmd' completed successfully",
		"50ms",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// Should not show empty output section
	if strings.Contains(output, "    \n") {
		t.Errorf("Expected no empty output lines, got:\n%s", output)
	}
}

func TestConsoleReporter_ReportCommandSuccess_LongOutput(t *testing.T) {
	cmd := config.Command{
		Name:    "long-output-cmd",
		Command: "echo",
		Mode:    config.ModeOnce,
	}

	// Create output with many lines
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d of output", i+1)
	}
	longOutput := strings.Join(lines, "\n")

	result := ExecutionResult{
		Command:   cmd,
		Success:   true,
		ExitCode:  0,
		Output:    longOutput,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
	}

	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	reporter.ReportCommandSuccess(result, 0)
	output := buf.String()

	// Should show first 3 lines and truncation message
	expectedContains := []string{
		"Line 1 of output",
		"Line 2 of output",
		"Line 3 of output",
		"... (7 more lines)",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// Should not show all lines
	if strings.Contains(output, "Line 9 of output") {
		t.Errorf("Expected output to be truncated, but found last line:\n%s", output)
	}
}

func TestConsoleReporter_ReportCommandFailure_MinimalError(t *testing.T) {
	cmd := config.Command{
		Name:    "minimal-fail",
		Command: "false",
		Mode:    config.ModeOnce,
	}

	result := ExecutionResult{
		Command:     cmd,
		Success:     false,
		ExitCode:    1,
		Output:      "",
		Error:       "exit status 1",
		ErrorDetail: nil, // No detailed error info
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(10 * time.Millisecond),
		Duration:    10 * time.Millisecond,
	}

	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, false) // non-verbose mode

	reporter.ReportCommandFailure(result, 0)
	output := buf.String()

	expectedContains := []string{
		"[1] ✗ 'minimal-fail' failed",
		"10ms",
		"Exit code: 1",
		"Error: exit status 1",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// Should not show verbose details
	notExpectedContains := []string{
		"Command:",
		"Working Directory:",
		"Stderr:",
	}

	for _, notExpected := range notExpectedContains {
		if strings.Contains(output, notExpected) {
			t.Errorf("Expected output to NOT contain %q in non-verbose mode, got:\n%s", notExpected, output)
		}
	}
}

func TestConsoleReporter_ReportCommandFailure_ZeroExitCode(t *testing.T) {
	cmd := config.Command{
		Name:    "zero-exit-fail",
		Command: "test",
		Mode:    config.ModeOnce,
	}

	result := ExecutionResult{
		Command:   cmd,
		Success:   false,
		ExitCode:  0, // Zero exit code but still failed
		Output:    "",
		Error:     "command failed for other reasons",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Millisecond),
		Duration:  5 * time.Millisecond,
	}

	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, false)

	reporter.ReportCommandFailure(result, 0)
	output := buf.String()

	// Should not show exit code line for zero exit code
	if strings.Contains(output, "Exit code: 0") {
		t.Errorf("Expected no exit code line for zero exit code, got:\n%s", output)
	}

	// Should still show error
	if !strings.Contains(output, "Error: command failed for other reasons") {
		t.Errorf("Expected error message, got:\n%s", output)
	}
}

func TestConsoleReporter_ReportExecutionComplete_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		verbose  bool
		contains []string
	}{
		{
			name: "zero commands completed",
			status: ExecutionStatus{
				State:          StateFailed,
				CompletedCount: 0,
				TotalCount:     3,
				LastError:      "Failed immediately",
			},
			verbose:  false,
			contains: []string{"✗ Execution failed at command 0 of 3"},
		},
		{
			name: "unknown state",
			status: ExecutionStatus{
				State:          ExecutionState(999), // Unknown state
				CompletedCount: 2,
				TotalCount:     5,
			},
			verbose:  false,
			contains: []string{"Execution stopped (2/5 completed)"},
		},
		{
			name: "failed with empty error message",
			status: ExecutionStatus{
				State:          StateFailed,
				CompletedCount: 1,
				TotalCount:     2,
				LastError:      "", // Empty error
			},
			verbose:  true,
			contains: []string{"✗ Execution failed at command 1 of 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewConsoleReporter(&buf, tt.verbose)

			reporter.ReportExecutionComplete(tt.status)
			output := buf.String()

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestConsoleReporter_ReportExecutionSummary_MixedResults(t *testing.T) {
	results := []ExecutionResult{
		{
			Command:  config.Command{Name: "success-1", Mode: config.ModeOnce},
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		{
			Command:  config.Command{Name: "failure-1", Mode: config.ModeOnce},
			Success:  false,
			Duration: 50 * time.Millisecond,
		},
		{
			Command:  config.Command{Name: "success-2", Mode: config.ModeKeepAlive},
			Success:  true,
			Duration: 25 * time.Millisecond,
		},
		{
			Command:  config.Command{Name: "failure-2", Mode: config.ModeKeepAlive},
			Success:  false,
			Duration: 75 * time.Millisecond,
		},
	}

	status := ExecutionStatus{
		State:   StateFailed,
		Results: results,
	}

	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	reporter.ReportExecutionSummary(status)
	output := buf.String()

	expectedContains := []string{
		"Execution Summary:",
		"✓ [1] success-1",
		"100ms",
		"✗ [2] failure-1",
		"50ms",
		"✓ [3] success-2",
		"25ms",
		"(background)",
		"✗ [4] failure-2",
		"75ms",
		"Total: 4 commands, 2 successful, 2 failed",
		"Total execution time: 250ms",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestConsoleReporter_ReportExecutionSummary_AllSuccessful(t *testing.T) {
	results := []ExecutionResult{
		{
			Command:  config.Command{Name: "cmd1", Mode: config.ModeOnce},
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		{
			Command:  config.Command{Name: "cmd2", Mode: config.ModeOnce},
			Success:  true,
			Duration: 200 * time.Millisecond,
		},
	}

	status := ExecutionStatus{
		State:   StateSuccess,
		Results: results,
	}

	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose mode

	reporter.ReportExecutionSummary(status)
	output := buf.String()

	expectedContains := []string{
		"Total: 2 commands, 2 successful, 0 failed",
		"Total execution time: 300ms",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestFormatDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0.00μs",
		},
		{
			name:     "exactly 1 millisecond",
			duration: time.Millisecond,
			expected: "1ms",
		},
		{
			name:     "exactly 1 second",
			duration: time.Second,
			expected: "1.00s",
		},
		{
			name:     "exactly 1 minute",
			duration: time.Minute,
			expected: "1.0m",
		},
		{
			name:     "very small nanoseconds",
			duration: 1 * time.Nanosecond,
			expected: "0.00μs",
		},
		{
			name:     "large duration",
			duration: 2*time.Hour + 30*time.Minute,
			expected: "150.0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, result, tt.expected)
			}
		})
	}
}
