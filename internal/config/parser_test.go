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
		validate    func(*testing.T, *Config)
	}{
		{
			name: "valid configuration - object format",
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
			validate: func(t *testing.T, config *Config) {
				if len(config.Commands) != 1 {
					t.Errorf("Expected 1 command, got %d", len(config.Commands))
				}
				cmd := config.Commands[0]
				if cmd.Name != "test" || cmd.Command != "echo" || len(cmd.Args) != 1 || cmd.Args[0] != "hello" {
					t.Errorf("Command not parsed correctly: %+v", cmd)
				}
			},
		},
		{
			name: "string format command",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "build",
						"command": "npm run build",
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "npm" || len(cmd.Args) != 2 || cmd.Args[0] != "run" || cmd.Args[1] != "build" {
					t.Errorf("String command not parsed correctly: %+v", cmd)
				}
			},
		},
		{
			name: "array format command",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "start",
						"command": ["node", "server.js", "--port", "3000"],
						"mode": "keepAlive"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "node" || len(cmd.Args) != 3 {
					t.Errorf("Array command not parsed correctly: %+v", cmd)
				}
				expectedArgs := []string{"server.js", "--port", "3000"}
				for i, arg := range expectedArgs {
					if cmd.Args[i] != arg {
						t.Errorf("Expected arg %d to be %s, got %s", i, arg, cmd.Args[i])
					}
				}
			},
		},
		{
			name: "object format with command and args",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "docker",
						"command": {
							"command": "docker",
							"args": ["run", "-p", "8080:80", "nginx"]
						},
						"mode": "keepAlive"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "docker" || len(cmd.Args) != 4 {
					t.Errorf("Object command not parsed correctly: %+v", cmd)
				}
			},
		},
		{
			name: "auto-generated name for string command",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"command": "npm start"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Name != "npm-start" {
					t.Errorf("Expected auto-generated name 'npm-start', got '%s'", cmd.Name)
				}
			},
		},
		{
			name: "auto-generated name for single command",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"command": "ls"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Name != "ls" {
					t.Errorf("Expected auto-generated name 'ls', got '%s'", cmd.Name)
				}
			},
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
			name: "empty command string",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "empty",
						"command": "",
						"mode": "once"
					}
				]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name: "empty command array",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "empty",
						"command": [],
						"mode": "once"
					}
				]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name: "invalid array element type",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "invalid",
						"command": ["npm", 123],
						"mode": "once"
					}
				]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name: "object without command field",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "invalid",
						"command": {
							"args": ["start"]
						},
						"mode": "once"
					}
				]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name: "invalid JSON syntax",
			json: `{
				"version": "1.0"
				"commands": []
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
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
			errorSubstr: "invalid configuration format detected",
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
			errorSubstr: "invalid configuration format detected",
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

			if !tt.wantErr {
				if config == nil {
					t.Error("ParseJSON() returned nil config without error")
				} else if tt.validate != nil {
					tt.validate(t, config)
				}
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
			errorSubstr: "format detection failed",
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

func TestFlexibleCommandToStandardCommand(t *testing.T) {
	tests := []struct {
		name        string
		flexCmd     FlexibleCommand
		wantErr     bool
		errorSubstr string
		validate    func(*testing.T, *Command)
	}{
		{
			name: "string command with args",
			flexCmd: FlexibleCommand{
				Name:    "build",
				Command: "npm run build",
				Mode:    ModeOnce,
			},
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "npm" || len(cmd.Args) != 2 || cmd.Args[0] != "run" || cmd.Args[1] != "build" {
					t.Errorf("String command not converted correctly: %+v", cmd)
				}
			},
		},
		{
			name: "string command single word",
			flexCmd: FlexibleCommand{
				Name:    "list",
				Command: "ls",
				Mode:    ModeOnce,
			},
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "ls" || len(cmd.Args) != 0 {
					t.Errorf("Single word command not converted correctly: %+v", cmd)
				}
			},
		},
		{
			name: "array command",
			flexCmd: FlexibleCommand{
				Name:    "server",
				Command: []interface{}{"node", "server.js", "--port", "3000"},
				Mode:    ModeKeepAlive,
			},
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "node" || len(cmd.Args) != 3 {
					t.Errorf("Array command not converted correctly: %+v", cmd)
				}
				expectedArgs := []string{"server.js", "--port", "3000"}
				for i, arg := range expectedArgs {
					if cmd.Args[i] != arg {
						t.Errorf("Expected arg %d to be %s, got %s", i, arg, cmd.Args[i])
					}
				}
			},
		},
		{
			name: "object command with args",
			flexCmd: FlexibleCommand{
				Name: "docker",
				Command: map[string]interface{}{
					"command": "docker",
					"args":    []interface{}{"run", "-p", "8080:80", "nginx"},
				},
				Mode: ModeKeepAlive,
			},
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "docker" || len(cmd.Args) != 4 {
					t.Errorf("Object command not converted correctly: %+v", cmd)
				}
			},
		},
		{
			name: "object command without args",
			flexCmd: FlexibleCommand{
				Name: "simple",
				Command: map[string]interface{}{
					"command": "echo",
				},
				Mode: ModeOnce,
			},
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "echo" || len(cmd.Args) != 0 {
					t.Errorf("Object command without args not converted correctly: %+v", cmd)
				}
			},
		},
		{
			name: "default mode when not specified",
			flexCmd: FlexibleCommand{
				Name:    "test",
				Command: "echo hello",
			},
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Mode != ModeOnce {
					t.Errorf("Expected default mode to be 'once', got %s", cmd.Mode)
				}
			},
		},
		{
			name: "empty string command",
			flexCmd: FlexibleCommand{
				Name:    "empty",
				Command: "",
				Mode:    ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "command string cannot be empty",
		},
		{
			name: "empty array command",
			flexCmd: FlexibleCommand{
				Name:    "empty",
				Command: []interface{}{},
				Mode:    ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "command array cannot be empty",
		},
		{
			name: "array with non-string elements",
			flexCmd: FlexibleCommand{
				Name:    "invalid",
				Command: []interface{}{"npm", 123},
				Mode:    ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "command array elements must be strings",
		},
		{
			name: "array with non-string first element",
			flexCmd: FlexibleCommand{
				Name:    "invalid",
				Command: []interface{}{123, "start"},
				Mode:    ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "first element of command array must be a string",
		},
		{
			name: "object without command field",
			flexCmd: FlexibleCommand{
				Name: "invalid",
				Command: map[string]interface{}{
					"args": []interface{}{"start"},
				},
				Mode: ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "object format must have a 'command' field",
		},
		{
			name: "object with non-string command",
			flexCmd: FlexibleCommand{
				Name: "invalid",
				Command: map[string]interface{}{
					"command": 123,
				},
				Mode: ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "object format must have a 'command' field of type string",
		},
		{
			name: "object with invalid args type",
			flexCmd: FlexibleCommand{
				Name: "invalid",
				Command: map[string]interface{}{
					"command": "npm",
					"args":    "start",
				},
				Mode: ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "'args' field must be an array",
		},
		{
			name: "object with non-string args elements",
			flexCmd: FlexibleCommand{
				Name: "invalid",
				Command: map[string]interface{}{
					"command": "npm",
					"args":    []interface{}{"start", 123},
				},
				Mode: ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "args array elements must be strings",
		},
		{
			name: "invalid command type",
			flexCmd: FlexibleCommand{
				Name:    "invalid",
				Command: 123,
				Mode:    ModeOnce,
			},
			wantErr:     true,
			errorSubstr: "command field must be a string, array, or object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := tt.flexCmd.ToStandardCommand()

			if (err != nil) != tt.wantErr {
				t.Errorf("ToStandardCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("ToStandardCommand() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}

			if !tt.wantErr {
				if cmd == nil {
					t.Error("ToStandardCommand() returned nil command without error")
				} else if tt.validate != nil {
					tt.validate(t, cmd)
				}
			}
		})
	}
}

func TestParseJSONWithFormatInfo(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantErr     bool
		errorSubstr string
		validate    func(*testing.T, *Config, *ConfigFormatInfo)
	}{
		{
			name: "mixed formats with format info",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "string-cmd",
						"command": "npm start",
						"mode": "once"
					},
					{
						"name": "array-cmd",
						"command": ["node", "server.js"],
						"mode": "keepAlive"
					},
					{
						"name": "object-cmd",
						"command": {
							"command": "docker",
							"args": ["run", "nginx"]
						},
						"mode": "keepAlive"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config, formatInfo *ConfigFormatInfo) {
				if len(config.Commands) != 3 {
					t.Errorf("Expected 3 commands, got %d", len(config.Commands))
				}
				if !formatInfo.MixedFormats {
					t.Error("Expected mixed formats to be true")
				}
				if len(formatInfo.CommandFormats) != 3 {
					t.Errorf("Expected 3 command format infos, got %d", len(formatInfo.CommandFormats))
				}

				expectedFormats := []CommandFormat{FormatString, FormatArray, FormatObject}
				for i, expected := range expectedFormats {
					if formatInfo.CommandFormats[i].Format != expected {
						t.Errorf("Command %d: expected format %s, got %s",
							i, expected.String(), formatInfo.CommandFormats[i].Format.String())
					}
				}
			},
		},
		{
			name: "single format with format info",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "cmd1",
						"command": "npm start",
						"mode": "once"
					},
					{
						"name": "cmd2",
						"command": "npm build",
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config, formatInfo *ConfigFormatInfo) {
				if formatInfo.MixedFormats {
					t.Error("Expected mixed formats to be false")
				}
				summary := formatInfo.GetFormatSummary()
				if summary["string"] != 2 {
					t.Errorf("Expected 2 string commands, got %d", summary["string"])
				}
			},
		},
		{
			name: "invalid format detection",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "invalid",
						"command": 123,
						"mode": "once"
					}
				]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
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
			config, formatInfo, err := ParseJSONWithFormatInfo([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSONWithFormatInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("ParseJSONWithFormatInfo() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}

			if !tt.wantErr {
				if config == nil {
					t.Error("ParseJSONWithFormatInfo() returned nil config without error")
				}
				if formatInfo == nil {
					t.Error("ParseJSONWithFormatInfo() returned nil formatInfo without error")
				}
				if tt.validate != nil {
					tt.validate(t, config, formatInfo)
				}
			}
		})
	}
}

func TestFlexibleConfigToStandardConfig(t *testing.T) {
	tests := []struct {
		name        string
		flexConfig  FlexibleConfig
		wantErr     bool
		errorSubstr string
		validate    func(*testing.T, *Config)
	}{
		{
			name: "mixed command formats",
			flexConfig: FlexibleConfig{
				Version: "1.0",
				Commands: []FlexibleCommand{
					{
						Name:    "build",
						Command: "npm run build",
						Mode:    ModeOnce,
					},
					{
						Name:    "start",
						Command: []interface{}{"node", "server.js"},
						Mode:    ModeKeepAlive,
					},
					{
						Name: "docker",
						Command: map[string]interface{}{
							"command": "docker",
							"args":    []interface{}{"run", "nginx"},
						},
						Mode: ModeKeepAlive,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if len(config.Commands) != 3 {
					t.Errorf("Expected 3 commands, got %d", len(config.Commands))
				}

				// Check first command (string format)
				cmd1 := config.Commands[0]
				if cmd1.Name != "build" || cmd1.Command != "npm" || len(cmd1.Args) != 2 {
					t.Errorf("First command not converted correctly: %+v", cmd1)
				}

				// Check second command (array format)
				cmd2 := config.Commands[1]
				if cmd2.Name != "start" || cmd2.Command != "node" || len(cmd2.Args) != 1 {
					t.Errorf("Second command not converted correctly: %+v", cmd2)
				}

				// Check third command (object format)
				cmd3 := config.Commands[2]
				if cmd3.Name != "docker" || cmd3.Command != "docker" || len(cmd3.Args) != 2 {
					t.Errorf("Third command not converted correctly: %+v", cmd3)
				}
			},
		},
		{
			name: "auto-generated names",
			flexConfig: FlexibleConfig{
				Version: "1.0",
				Commands: []FlexibleCommand{
					{
						Command: "npm start",
					},
					{
						Command: "ls",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if config.Commands[0].Name != "npm-start" {
					t.Errorf("Expected auto-generated name 'npm-start', got '%s'", config.Commands[0].Name)
				}
				if config.Commands[1].Name != "ls" {
					t.Errorf("Expected auto-generated name 'ls', got '%s'", config.Commands[1].Name)
				}
			},
		},
		{
			name: "error in command conversion",
			flexConfig: FlexibleConfig{
				Version: "1.0",
				Commands: []FlexibleCommand{
					{
						Name:    "valid",
						Command: "echo hello",
						Mode:    ModeOnce,
					},
					{
						Name:    "invalid",
						Command: "",
						Mode:    ModeOnce,
					},
				},
			},
			wantErr:     true,
			errorSubstr: "error converting command 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := tt.flexConfig.ToStandardConfig()

			if (err != nil) != tt.wantErr {
				t.Errorf("ToStandardConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("ToStandardConfig() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}

			if !tt.wantErr {
				if config == nil {
					t.Error("ToStandardConfig() returned nil config without error")
				} else if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}
