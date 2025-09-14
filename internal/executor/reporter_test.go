package executor

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestConsoleReporter_ReportStart(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true)

	reporter.ReportStart(3)

	output := buf.String()
	if !strings.Contains(output, "Starting execution of 3 commands") {
		t.Errorf("Expected start message, got: %s", output)
	}
}

func TestConsoleReporter_ReportCommandSuccess(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, false)

	result := ExecutionResult{
		Command:  config.Command{Name: "test"},
		Success:  true,
		Duration: 100 * time.Millisecond,
	}

	reporter.ReportCommandSuccess(result, 0)

	output := buf.String()
	if !strings.Contains(output, "[1] ✓ test") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

func TestConsoleReporter_ReportCommandFailure(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, false)

	result := ExecutionResult{
		Command: config.Command{Name: "test"},
		Success: false,
		Error:   "command failed",
	}

	reporter.ReportCommandFailure(result, 0)

	output := buf.String()
	if !strings.Contains(output, "[1] ✗ test failed") {
		t.Errorf("Expected failure message, got: %s", output)
	}
}
