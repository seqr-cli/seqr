package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestExecuteOnceWithRealTimeOutput(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		command     config.Command
		expectError bool
	}{
		{
			name:    "verbose mode with echo command",
			verbose: true,
			command: config.Command{
				Name:    "test-echo",
				Command: "echo",
				Args:    []string{"Hello, World!"},
				Mode:    config.ModeOnce,
			},
			expectError: false,
		},
		{
			name:    "non-verbose mode with echo command",
			verbose: false,
			command: config.Command{
				Name:    "test-echo",
				Command: "echo",
				Args:    []string{"Hello, World!"},
				Mode:    config.ModeOnce,
			},
			expectError: false,
		},
		{
			name:    "verbose mode with multi-line output",
			verbose: true,
			command: config.Command{
				Name:    "test-multiline",
				Command: "echo",
				Args:    []string{"-e", "Line 1\\nLine 2\\nLine 3"},
				Mode:    config.ModeOnce,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.verbose)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cfg := &config.Config{
				Version:  "1.0",
				Commands: []config.Command{tt.command},
			}

			err := executor.Execute(ctx, cfg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			status := executor.GetStatus()
			if len(status.Results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(status.Results))
				return
			}

			result := status.Results[0]
			if !tt.expectError && !result.Success {
				t.Errorf("Expected success but got failure: %s", result.Error)
			}

			if tt.expectError && result.Success {
				t.Error("Expected failure but got success")
			}

			// Verify output is captured
			if !tt.expectError && result.Output == "" {
				t.Error("Expected output to be captured but got empty string")
			}
		})
	}
}

func TestStreamOutput(t *testing.T) {
	executor := NewExecutor(true)

	// Create a test pipe with some content
	testContent := "Line 1\nLine 2\nLine 3"
	reader := &testReadCloser{strings.NewReader(testContent)}

	var outputBuilder strings.Builder

	// We can't easily test the streaming directly since it writes to stdout,
	// but we can test the output building functionality
	executor.streamOutput(reader, &outputBuilder, "test-command", "stdout")

	capturedOutput := strings.TrimSpace(outputBuilder.String())
	expectedOutput := strings.ReplaceAll(testContent, "\n", "\n") + "\n"
	expectedOutput = strings.TrimSpace(expectedOutput)

	if capturedOutput != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, capturedOutput)
	}
}

// testReadCloser wraps a strings.Reader to implement io.ReadCloser
type testReadCloser struct {
	*strings.Reader
}

func (t *testReadCloser) Close() error {
	return nil
}

func TestRealTimeOutputWithKeepAlive(t *testing.T) {
	// Skip this test on systems where we can't easily create a long-running process
	if testing.Short() {
		t.Skip("Skipping keepAlive test in short mode")
	}

	executor := NewExecutor(true)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use a command that will run for a bit and produce output
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-keepalive",
				Command: "ping",
				Args:    []string{"-c", "2", "127.0.0.1"}, // 2 pings to localhost
				Mode:    config.ModeKeepAlive,
			},
		},
	}

	err := executor.Execute(ctx, cfg)

	// For keepAlive commands, we expect success even if the process is still running
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	status := executor.GetStatus()
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(status.Results))
		return
	}

	result := status.Results[0]
	if !result.Success {
		t.Errorf("Expected success but got failure: %s", result.Error)
	}

	// Verify the process was started
	if !strings.Contains(result.Output, "PID") {
		t.Errorf("Expected output to contain PID information, got: %s", result.Output)
	}
}
