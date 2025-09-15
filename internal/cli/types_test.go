package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestNewCLI(t *testing.T) {
	args := []string{"-f", "test.json", "-v"}
	cli := NewCLI(args)

	if cli == nil {
		t.Fatal("NewCLI returned nil")
	}

	// Check default options before parsing
	opts := cli.GetOptions()
	if opts.ConfigFile != config.DefaultConfigFile() {
		t.Errorf("Expected default config file %s, got %s", config.DefaultConfigFile(), opts.ConfigFile)
	}
	if opts.Verbose != false {
		t.Errorf("Expected verbose to be false by default, got %t", opts.Verbose)
	}
}

func TestCLI_Parse(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectError  bool
		expectedOpts CLIOptions
	}{
		{
			name:        "default options",
			args:        []string{},
			expectError: false,
			expectedOpts: CLIOptions{
				ConfigFile: ".queue.json",
				Verbose:    false,
				Help:       false,
				Init:       false,
			},
		},
		{
			name:        "verbose flag",
			args:        []string{"-v"},
			expectError: false,
			expectedOpts: CLIOptions{
				ConfigFile: ".queue.json",
				Verbose:    true,
				Help:       false,
				Init:       false,
			},
		},
		{
			name:        "verbose flag long form",
			args:        []string{"-verbose"},
			expectError: false,
			expectedOpts: CLIOptions{
				ConfigFile: ".queue.json",
				Verbose:    true,
				Help:       false,
				Init:       false,
			},
		},
		{
			name:        "custom config file",
			args:        []string{"-f", "custom.json"},
			expectError: false, // File validation happens in Run(), not Parse()
			expectedOpts: CLIOptions{
				ConfigFile: "custom.json",
				Verbose:    false,
				Help:       false,
				Init:       false,
			},
		},
		{
			name:        "help flag short",
			args:        []string{"-h"},
			expectError: false,
			expectedOpts: CLIOptions{
				ConfigFile: ".queue.json",
				Verbose:    false,
				Help:       true,
				Init:       false,
			},
		},
		{
			name:        "help flag long",
			args:        []string{"-help"},
			expectError: false,
			expectedOpts: CLIOptions{
				ConfigFile: ".queue.json",
				Verbose:    false,
				Help:       true,
				Init:       false,
			},
		},
		{
			name:        "combined flags",
			args:        []string{"-f", "test.json", "-v"},
			expectError: false, // File validation happens in Run(), not Parse()
			expectedOpts: CLIOptions{
				ConfigFile: "test.json",
				Verbose:    true,
				Help:       false,
				Init:       false,
			},
		},
		{
			name:        "init flag",
			args:        []string{"-init"},
			expectError: false,
			expectedOpts: CLIOptions{
				ConfigFile: ".queue.json",
				Verbose:    false,
				Help:       false,
				Init:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI(tt.args)
			err := cli.Parse()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			opts := cli.GetOptions()
			if opts.ConfigFile != tt.expectedOpts.ConfigFile {
				t.Errorf("Expected ConfigFile %s, got %s", tt.expectedOpts.ConfigFile, opts.ConfigFile)
			}
			if opts.Verbose != tt.expectedOpts.Verbose {
				t.Errorf("Expected Verbose %t, got %t", tt.expectedOpts.Verbose, opts.Verbose)
			}
			if opts.Help != tt.expectedOpts.Help {
				t.Errorf("Expected Help %t, got %t", tt.expectedOpts.Help, opts.Help)
			}
		})
	}
}

func TestCLI_ShouldRunInit(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no init flag",
			args:     []string{},
			expected: false,
		},
		{
			name:     "init flag",
			args:     []string{"-init"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI(tt.args)
			cli.Parse() // Ignore error for init tests

			if cli.ShouldRunInit() != tt.expected {
				t.Errorf("Expected ShouldRunInit() to return %t, got %t", tt.expected, cli.ShouldRunInit())
			}
		})
	}
}

func TestCLI_ShouldShowHelp(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no help flag",
			args:     []string{},
			expected: false,
		},
		{
			name:     "short help flag",
			args:     []string{"-h"},
			expected: true,
		},
		{
			name:     "long help flag",
			args:     []string{"-help"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI(tt.args)
			cli.Parse() // Ignore error for help tests

			if cli.ShouldShowHelp() != tt.expected {
				t.Errorf("Expected ShouldShowHelp() to return %t, got %t", tt.expected, cli.ShouldShowHelp())
			}
		})
	}
}

func TestCLI_RunWithValidConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.queue.json")

	configContent := `{
		"version": "1.0",
		"commands": [
			{
				"name": "test-command",
				"command": "echo",
				"args": ["hello"],
				"mode": "once"
			}
		]
	}`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test CLI with valid config
	cli := NewCLI([]string{"-f", configFile})
	if err := cli.Parse(); err != nil {
		t.Fatalf("Failed to parse CLI args: %v", err)
	}

	// Run with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cli.Run(ctx); err != nil {
		t.Errorf("CLI Run failed: %v", err)
	}
}

func TestCLI_Stop(t *testing.T) {
	cli := NewCLI([]string{})

	// Should not panic when called without executor
	cli.Stop()

	// Parse and create executor
	cli.Parse() // Ignore error since we don't have a valid config file

	// Should not panic when called with executor
	cli.Stop()
}

func TestCLI_ParseInvalidArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "unknown flag",
			args:        []string{"-unknown"},
			expectError: true,
		},
		{
			name:        "double dash unknown flag",
			args:        []string{"--unknown"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI(tt.args)
			err := cli.Parse()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestCLI_RunWithNonexistentConfig(t *testing.T) {
	cli := NewCLI([]string{"-f", "nonexistent.json"})
	if err := cli.Parse(); err != nil {
		t.Fatalf("Failed to parse CLI args: %v", err)
	}

	ctx := context.Background()
	err := cli.Run(ctx)
	if err == nil {
		t.Error("Expected error for nonexistent config file, but got none")
	}

	expectedSubstring := "does not exist"
	if !contains(err.Error(), expectedSubstring) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestCLI_RunWithHelp(t *testing.T) {
	cli := NewCLI([]string{"-h"})
	if err := cli.Parse(); err != nil {
		t.Fatalf("Failed to parse CLI args: %v", err)
	}

	ctx := context.Background()
	err := cli.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error when running with help flag, got: %v", err)
	}
}

func TestCLI_DefaultConfigFile(t *testing.T) {
	cli := NewCLI([]string{})
	opts := cli.GetOptions()

	expected := ".queue.json"
	if opts.ConfigFile != expected {
		t.Errorf("Expected default config file to be '%s', got '%s'", expected, opts.ConfigFile)
	}
}

func TestCLI_FlagCombinations(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedOpts CLIOptions
	}{
		{
			name: "all flags together",
			args: []string{"-f", "custom.json", "-v", "-h"},
			expectedOpts: CLIOptions{
				ConfigFile: "custom.json",
				Verbose:    true,
				Help:       true,
			},
		},
		{
			name: "flags in different order",
			args: []string{"-v", "-f", "test.json"},
			expectedOpts: CLIOptions{
				ConfigFile: "test.json",
				Verbose:    true,
				Help:       false,
			},
		},
		{
			name: "long help flag with other options",
			args: []string{"-help", "-v"},
			expectedOpts: CLIOptions{
				ConfigFile: ".queue.json",
				Verbose:    true,
				Help:       true,
			},
		},
		{
			name: "long verbose flag with config file",
			args: []string{"-f", "test.json", "-verbose"},
			expectedOpts: CLIOptions{
				ConfigFile: "test.json",
				Verbose:    true,
				Help:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI(tt.args)
			err := cli.Parse()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			opts := cli.GetOptions()
			if opts.ConfigFile != tt.expectedOpts.ConfigFile {
				t.Errorf("Expected ConfigFile %s, got %s", tt.expectedOpts.ConfigFile, opts.ConfigFile)
			}
			if opts.Verbose != tt.expectedOpts.Verbose {
				t.Errorf("Expected Verbose %t, got %t", tt.expectedOpts.Verbose, opts.Verbose)
			}
			if opts.Help != tt.expectedOpts.Help {
				t.Errorf("Expected Help %t, got %t", tt.expectedOpts.Help, opts.Help)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
func TestCLI_ShowHelp(t *testing.T) {
	// Capture stdout to verify help output
	cli := NewCLI([]string{"-h"})
	cli.Parse()

	// This test verifies that ShowHelp doesn't panic and contains expected content
	// We can't easily capture stdout in a unit test without complex setup,
	// but we can at least verify the method runs without error
	cli.ShowHelp()
}

func TestCLI_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectParseErr bool
		expectRunErr   bool
	}{
		{
			name:           "missing config file argument",
			args:           []string{"-f"},
			expectParseErr: true,
			expectRunErr:   false,
		},
		{
			name:           "invalid flag format",
			args:           []string{"--invalid-flag"},
			expectParseErr: true,
			expectRunErr:   false,
		},
		{
			name:           "nonexistent config file",
			args:           []string{"-f", "does-not-exist.json"},
			expectParseErr: false,
			expectRunErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI(tt.args)
			parseErr := cli.Parse()

			if tt.expectParseErr && parseErr == nil {
				t.Errorf("Expected parse error but got none")
			}
			if !tt.expectParseErr && parseErr != nil {
				t.Errorf("Unexpected parse error: %v", parseErr)
			}

			// Only test Run if Parse succeeded
			if parseErr == nil && !cli.ShouldShowHelp() {
				ctx := context.Background()
				runErr := cli.Run(ctx)

				if tt.expectRunErr && runErr == nil {
					t.Errorf("Expected run error but got none")
				}
				if !tt.expectRunErr && runErr != nil {
					t.Errorf("Unexpected run error: %v", runErr)
				}
			}
		})
	}
}

func TestCLI_ContextCancellation(t *testing.T) {
	// Create a temporary config file with a long-running command
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.queue.json")

	configContent := `{
		"version": "1.0",
		"commands": [
			{
				"name": "long-running-command",
				"command": "sleep",
				"args": ["10"],
				"mode": "once"
			}
		]
	}`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cli := NewCLI([]string{"-f", configFile})
	if err := cli.Parse(); err != nil {
		t.Fatalf("Failed to parse CLI args: %v", err)
	}

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := cli.Run(ctx)
	if err == nil {
		t.Error("Expected error due to context cancellation, but got none")
	}

	// The error should be related to context cancellation or signal termination
	errStr := err.Error()
	if !contains(errStr, "context") && !contains(errStr, "timeout") && !contains(errStr, "cancelled") &&
		!contains(errStr, "killed") && !contains(errStr, "signal") {
		t.Errorf("Expected context/signal-related error, got: %v", err)
	}
}

func TestCLI_VerboseOutput(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.queue.json")

	configContent := `{
		"version": "1.0",
		"commands": [
			{
				"name": "test-echo",
				"command": "echo",
				"args": ["test output"],
				"mode": "once"
			}
		]
	}`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test verbose mode
	cli := NewCLI([]string{"-f", configFile, "-v"})
	if err := cli.Parse(); err != nil {
		t.Fatalf("Failed to parse CLI args: %v", err)
	}

	opts := cli.GetOptions()
	if !opts.Verbose {
		t.Error("Expected verbose mode to be enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cli.Run(ctx); err != nil {
		t.Errorf("CLI Run failed in verbose mode: %v", err)
	}
}

func TestCLI_EmptyArgs(t *testing.T) {
	cli := NewCLI([]string{})
	if err := cli.Parse(); err != nil {
		t.Errorf("Unexpected error with empty args: %v", err)
	}

	opts := cli.GetOptions()
	if opts.ConfigFile != ".queue.json" {
		t.Errorf("Expected default config file, got %s", opts.ConfigFile)
	}
	if opts.Verbose {
		t.Error("Expected verbose to be false by default")
	}
	if opts.Help {
		t.Error("Expected help to be false by default")
	}
}

func TestCLI_MultipleHelpFlags(t *testing.T) {
	// Test that multiple help flags work correctly
	tests := []struct {
		name string
		args []string
	}{
		{"short help", []string{"-h"}},
		{"long help", []string{"-help"}},
		{"both help flags", []string{"-h", "-help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI(tt.args)
			if err := cli.Parse(); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !cli.ShouldShowHelp() {
				t.Error("Expected help to be requested")
			}

			// Running with help should not error
			ctx := context.Background()
			if err := cli.Run(ctx); err != nil {
				t.Errorf("Unexpected error when running with help: %v", err)
			}
		})
	}
}
