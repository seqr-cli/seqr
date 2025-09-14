package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestValidationIntegration tests the complete validation pipeline
func TestValidationIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		configJSON  string
		validator   *Validator
		wantErr     bool
		errContains string
	}{
		{
			name: "complete valid configuration",
			configJSON: `{
				"version": "1.2.3",
				"commands": [
					{
						"name": "setup-db",
						"command": "docker",
						"args": ["run", "-d", "postgres"],
						"mode": "keepAlive",
						"env": {
							"POSTGRES_PASSWORD": "secret"
						}
					},
					{
						"name": "run-tests",
						"command": "npm",
						"args": ["test"],
						"mode": "once",
						"workDir": "./tests"
					}
				]
			}`,
			validator: NewValidator(),
			wantErr:   false,
		},
		{
			name: "configuration with validation errors",
			configJSON: `{
				"version": "",
				"commands": [
					{
						"name": "",
						"command": "",
						"mode": "invalid"
					}
				]
			}`,
			validator:   NewValidator(),
			wantErr:     true,
			errContains: "multiple validation errors",
		},
		{
			name: "strict mode with semantic version",
			configJSON: `{
				"version": "1.0.0-alpha.1+build.123",
				"commands": [
					{
						"name": "valid-command",
						"command": "echo",
						"mode": "once"
					}
				]
			}`,
			validator: NewStrictValidator(),
			wantErr:   false,
		},
		{
			name: "strict mode with invalid version",
			configJSON: `{
				"version": "not-semantic",
				"commands": [
					{
						"name": "test",
						"command": "echo",
						"mode": "once"
					}
				]
			}`,
			validator:   NewStrictValidator(),
			wantErr:     true,
			errContains: "not a valid semantic version",
		},
		{
			name: "strict mode with dangerous command",
			configJSON: `{
				"version": "1.0.0",
				"commands": [
					{
						"name": "dangerous",
						"command": "rm",
						"args": ["-rf", "/"],
						"mode": "once"
					}
				]
			}`,
			validator:   NewStrictValidator(),
			wantErr:     true,
			errContains: "potentially dangerous command",
		},
		{
			name: "duplicate command names",
			configJSON: `{
				"version": "1.0.0",
				"commands": [
					{
						"name": "duplicate",
						"command": "echo",
						"mode": "once"
					},
					{
						"name": "duplicate",
						"command": "ls",
						"mode": "once"
					}
				]
			}`,
			validator:   NewValidator(),
			wantErr:     true,
			errContains: "duplicate command name",
		},
		{
			name: "invalid environment variable names in strict mode",
			configJSON: `{
				"version": "1.0.0",
				"commands": [
					{
						"name": "test",
						"command": "echo",
						"mode": "once",
						"env": {
							"123INVALID": "value",
							"VALID_VAR": "value"
						}
					}
				]
			}`,
			validator:   NewStrictValidator(),
			wantErr:     true,
			errContains: "invalid environment variable name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file
			configFile := filepath.Join(tmpDir, tt.name+".json")
			if err := os.WriteFile(configFile, []byte(tt.configJSON), 0644); err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			// Load and validate the configuration
			config, err := LoadFromFile(configFile)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("LoadFromFile() unexpected error = %v", err)
				}
				return
			}

			// Apply additional validation with the specified validator
			err = tt.validator.ValidateConfig(config)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateConfig() error = %v, expected to contain %q", err, tt.errContains)
				}
			}
		})
	}
}

// TestValidationWithRealDirectories tests validation with actual filesystem paths
func TestValidationWithRealDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	config := &Config{
		Version: "1.0.0",
		Commands: []Command{
			{
				Name:    "test-existing-dir",
				Command: "echo",
				Mode:    ModeOnce,
				WorkDir: testDir,
			},
			{
				Name:    "test-nonexistent-dir",
				Command: "echo",
				Mode:    ModeOnce,
				WorkDir: filepath.Join(tmpDir, "nonexistent"),
			},
		},
	}

	// Test with directory validation enabled
	validator := &Validator{ValidateWorkDirs: true}

	err := validator.ValidateConfig(config)
	if err == nil {
		t.Error("Expected validation error for nonexistent directory")
	}

	if !contains(err.Error(), "does not exist") {
		t.Errorf("Expected error about nonexistent directory, got: %v", err)
	}

	// Test with directory validation disabled
	validator = NewValidator()
	err = validator.ValidateConfig(config)
	if err != nil {
		t.Errorf("Unexpected error with directory validation disabled: %v", err)
	}
}

// TestValidationErrorDetails tests that validation errors provide sufficient detail
func TestValidationErrorDetails(t *testing.T) {
	config := &Config{
		Version: "", // Missing version
		Commands: []Command{
			{
				Name:    "",                 // Missing name
				Command: "",                 // Missing command
				Mode:    Mode("invalid"),    // Invalid mode
				Args:    make([]string, 51), // Too many args in strict mode
				Env: map[string]string{
					"123INVALID": "value", // Invalid env var name in strict mode
				},
			},
		},
	}

	validator := NewStrictValidator()
	err := validator.ValidateConfig(config)

	if err == nil {
		t.Fatal("Expected validation errors")
	}

	errorStr := err.Error()

	// Check that all expected errors are present
	expectedErrors := []string{
		"version is required",
		"command name is required",
		"command is required",
		"mode must be either",
		"too many arguments",
		"invalid environment variable name",
	}

	for _, expected := range expectedErrors {
		if !contains(errorStr, expected) {
			t.Errorf("Expected error message to contain %q, got: %v", expected, errorStr)
		}
	}
}

// TestValidationPerformance tests that validation doesn't have performance issues
func TestValidationPerformance(t *testing.T) {
	// Create a large but valid configuration
	commands := make([]Command, 20) // Well within limits
	for i := 0; i < 20; i++ {
		commands[i] = Command{
			Name:    fmt.Sprintf("command-%d", i),
			Command: "echo",
			Mode:    ModeOnce,
			Args:    []string{"hello", "world"},
			Env: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		}
	}

	config := &Config{
		Version:  "1.0.0",
		Commands: commands,
	}

	validator := NewStrictValidator()

	// Run validation multiple times to check for performance issues
	for i := 0; i < 100; i++ {
		err := validator.ValidateConfig(config)
		if err != nil {
			t.Errorf("Unexpected validation error on iteration %d: %v", i, err)
		}
	}
}
