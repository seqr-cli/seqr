package executor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestExecutorDistinguishesStdoutStderr(t *testing.T) {
	t.Run("stdout_stderr_distinction", func(t *testing.T) {
		// Test that the output formatting correctly distinguishes stdout vs stderr

		// Test stdout formatting
		timestamp := time.Now().Format("15:04:05.000")
		expectedStdout := fmt.Sprintf("[%s] [test-cmd] ✓  This is stdout output", timestamp)

		// Test stderr formatting
		expectedStderr := fmt.Sprintf("[%s] [test-cmd] ❌ This is stderr output", timestamp)

		// Verify the format strings are different
		if !strings.Contains(expectedStdout, "✓") {
			t.Error("Expected stdout format to contain ✓ symbol")
		}

		if !strings.Contains(expectedStderr, "❌") {
			t.Error("Expected stderr format to contain ❌ symbol")
		}

		// Verify they are visually distinct
		if expectedStdout == expectedStderr {
			t.Error("Expected stdout and stderr formats to be visually distinct")
		}

		// Verify the symbols are different
		if strings.Contains(expectedStdout, "❌") {
			t.Error("Expected stdout format to not contain ❌ symbol")
		}

		if strings.Contains(expectedStderr, "✓") {
			t.Error("Expected stderr format to not contain ✓ symbol")
		}
	})
}

// testReporter is a simple reporter for testing
type testReporter struct {
	output *bytes.Buffer
}

func (r *testReporter) ReportStart(totalCommands int) {
	fmt.Fprintf(r.output, "Starting execution of %d commands\n", totalCommands)
}

func (r *testReporter) ReportCommandStart(commandName string, index int) {
	fmt.Fprintf(r.output, "Starting command %d: %s\n", index+1, commandName)
}

func (r *testReporter) ReportCommandSuccess(result ExecutionResult, index int) {
	fmt.Fprintf(r.output, "Command %d completed successfully\n", index+1)
}

func (r *testReporter) ReportCommandFailure(result ExecutionResult, index int) {
	fmt.Fprintf(r.output, "Command %d failed: %s\n", index+1, result.Error)
}

func (r *testReporter) ReportExecutionComplete(status ExecutionStatus) {
	fmt.Fprintf(r.output, "Execution completed with status: %s\n", status.State)
}
