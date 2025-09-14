package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantErr     bool
		errorSubstr string
	}{
		{
			name: "valid configuration",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "test",
						"command": "echo",
						"args": ["hello"],
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
		},
		{
			name: "configuration with keepAlive mode",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "server",
						"command": "npm",
						"args": ["start"],
						"mode": "keepAlive",
						"workDir": "./app",
						"env": {
							"NODE_ENV": "development"
						}
					}
				]
			}`,
			wantErr: false,
		},
		{
			name: "invalid JSON syntax",
			json: `{
				"version": "1.0"
				"commands": []
			}`,
			wantErr:     true,
			errorSubstr: "JSON syntax error",
		},
		{
			name: "missing version",
			json: `{
				"commands": [
					{
						"name": "test",
						"command": "echo",
						"mode": "once"
					}
				]
			}`,
			wantErr:     true,
			errorSubstr: "validation failed",
		},
		{
			name: "invalid mode",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "test",
						"command": "echo",
						"mode": "invalid"
					}
				]
			}`,
			wantErr:     true,
			errorSubstr: "invalid mode",
		},
		{
			name: "type mismatch",
			json: `{
				"version": 1.0,
				"commands": []
			}`,
			wantErr:     true,
			errorSubstr: "type error",
		},
		{
			name:        "empty data",
			json:        "",
			wantErr:     true,
			errorSubstr: "configuration data is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseJSON([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("ParseJSON() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}

			if !tt.wantErr && config == nil {
				t.Error("ParseJSON() returned nil config without error")
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Test valid config file
	validConfig := `{
		"version": "1.0",
		"commands": [
			{
				"name": "test",
				"command": "echo",
				"mode": "once"
			}
		]
	}`

	validFile := filepath.Join(tmpDir, "valid.json")
	if err := os.WriteFile(validFile, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test invalid config file
	invalidConfig := `{
		"version": "1.0"
		"commands": []
	}`

	invalidFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFile, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test empty config file
	emptyFile := filepath.Join(tmpDir, "empty.json")
	if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}

	// Test directory instead of file
	dirPath := filepath.Join(tmpDir, "directory")
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name        string
		filename    string
		wantErr     bool
		errorSubstr string
	}{
		{
			name:     "valid config file",
			filename: validFile,
			wantErr:  false,
		},
		{
			name:        "invalid config file",
			filename:    invalidFile,
			wantErr:     true,
			errorSubstr: "JSON syntax error",
		},
		{
			name:        "non-existent file",
			filename:    filepath.Join(tmpDir, "nonexistent.json"),
			wantErr:     true,
			errorSubstr: "does not exist",
		},
		{
			name:        "empty filename",
			filename:    "",
			wantErr:     true,
			errorSubstr: "cannot be empty",
		},
		{
			name:        "empty file",
			filename:    emptyFile,
			wantErr:     true,
			errorSubstr: "is empty",
		},
		{
			name:        "directory instead of file",
			filename:    dirPath,
			wantErr:     true,
			errorSubstr: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadFromFile(tt.filename)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("LoadFromFile() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}

			if !tt.wantErr && config == nil {
				t.Error("LoadFromFile() returned nil config without error")
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDefaultConfigFile(t *testing.T) {
	expected := ".queue.json"
	if got := DefaultConfigFile(); got != expected {
		t.Errorf("DefaultConfigFile() = %v, want %v", got, expected)
	}
}
func TestFileExists(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name        string
		filename    string
		wantErr     bool
		errorSubstr string
	}{
		{
			name:     "existing file",
			filename: testFile,
			wantErr:  false,
		},
		{
			name:        "non-existent file",
			filename:    filepath.Join(tmpDir, "nonexistent.json"),
			wantErr:     true,
			errorSubstr: "does not exist",
		},
		{
			name:        "empty filename",
			filename:    "",
			wantErr:     true,
			errorSubstr: "cannot be empty",
		},
		{
			name:        "directory instead of file",
			filename:    testDir,
			wantErr:     true,
			errorSubstr: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FileExists(tt.filename)

			if (err != nil) != tt.wantErr {
				t.Errorf("FileExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("FileExists() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}
		})
	}
}
