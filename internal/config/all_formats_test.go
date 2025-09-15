package config

import (
	"testing"
)

// TestAllSupportedFormats provides comprehensive testing for all supported configuration formats
// This test ensures that all documented command formats work correctly and produce the expected results
func TestAllSupportedFormats(t *testing.T) {
	tests := []struct {
		name        string
		description string
		json        string
		wantErr     bool
		validate    func(*testing.T, *Config)
	}{
		// String Format Tests
		{
			name:        "string_format_simple_command",
			description: "Simple string command without arguments",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "list",
						"command": "ls",
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "ls" {
					t.Errorf("Expected command 'ls', got '%s'", cmd.Command)
				}
				if len(cmd.Args) != 0 {
					t.Errorf("Expected no args, got %d", len(cmd.Args))
				}
			},
		},
		{
			name:        "string_format_command_with_args",
			description: "String command with multiple arguments",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "build",
						"command": "npm run build --production",
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "npm" {
					t.Errorf("Expected command 'npm', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"run", "build", "--production"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
				for i, expected := range expectedArgs {
					if cmd.Args[i] != expected {
						t.Errorf("Expected arg %d to be '%s', got '%s'", i, expected, cmd.Args[i])
					}
				}
			},
		},
		{
			name:        "string_format_complex_command",
			description: "Complex string command with flags and values",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "docker-run",
						"command": "docker run -p 8080:80 --name web-server nginx:latest",
						"mode": "keepAlive"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "docker" {
					t.Errorf("Expected command 'docker', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"run", "-p", "8080:80", "--name", "web-server", "nginx:latest"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
			},
		},

		// Array Format Tests
		{
			name:        "array_format_simple_command",
			description: "Simple array command without arguments",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "list",
						"command": ["ls"],
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "ls" {
					t.Errorf("Expected command 'ls', got '%s'", cmd.Command)
				}
				if len(cmd.Args) != 0 {
					t.Errorf("Expected no args, got %d", len(cmd.Args))
				}
			},
		},
		{
			name:        "array_format_command_with_args",
			description: "Array command with multiple arguments",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "server",
						"command": ["node", "server.js", "--port", "3000", "--env", "development"],
						"mode": "keepAlive"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "node" {
					t.Errorf("Expected command 'node', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"server.js", "--port", "3000", "--env", "development"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
				for i, expected := range expectedArgs {
					if cmd.Args[i] != expected {
						t.Errorf("Expected arg %d to be '%s', got '%s'", i, expected, cmd.Args[i])
					}
				}
			},
		},
		{
			name:        "array_format_complex_command",
			description: "Complex array command with many arguments",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "docker-compose",
						"command": ["docker-compose", "-f", "docker-compose.yml", "-f", "docker-compose.override.yml", "up", "-d", "--build"],
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "docker-compose" {
					t.Errorf("Expected command 'docker-compose', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"-f", "docker-compose.yml", "-f", "docker-compose.override.yml", "up", "-d", "--build"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
			},
		},

		// Object Format Tests
		{
			name:        "object_format_command_only",
			description: "Object format with command field only",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "list",
						"command": {
							"command": "ls"
						},
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "ls" {
					t.Errorf("Expected command 'ls', got '%s'", cmd.Command)
				}
				if len(cmd.Args) != 0 {
					t.Errorf("Expected no args, got %d", len(cmd.Args))
				}
			},
		},
		{
			name:        "object_format_command_with_args",
			description: "Object format with command and args fields",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "build",
						"command": {
							"command": "npm",
							"args": ["run", "build", "--production"]
						},
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "npm" {
					t.Errorf("Expected command 'npm', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"run", "build", "--production"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
				for i, expected := range expectedArgs {
					if cmd.Args[i] != expected {
						t.Errorf("Expected arg %d to be '%s', got '%s'", i, expected, cmd.Args[i])
					}
				}
			},
		},
		{
			name:        "object_format_complex_command",
			description: "Complex object format with many arguments",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "database",
						"command": {
							"command": "docker",
							"args": ["run", "-d", "--name", "postgres-db", "-e", "POSTGRES_PASSWORD=secret", "-p", "5432:5432", "postgres:13"]
						},
						"mode": "keepAlive"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "docker" {
					t.Errorf("Expected command 'docker', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"run", "-d", "--name", "postgres-db", "-e", "POSTGRES_PASSWORD=secret", "-p", "5432:5432", "postgres:13"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
			},
		},

		// Standard Format Tests (command and args as separate fields)
		{
			name:        "standard_format_simple",
			description: "Standard format with command and args as separate fields",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "echo",
						"command": "echo",
						"args": ["Hello", "World"],
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "echo" {
					t.Errorf("Expected command 'echo', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"Hello", "World"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
				for i, expected := range expectedArgs {
					if cmd.Args[i] != expected {
						t.Errorf("Expected arg %d to be '%s', got '%s'", i, expected, cmd.Args[i])
					}
				}
			},
		},
		{
			name:        "standard_format_complex",
			description: "Standard format with complex command and many arguments",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "webpack",
						"command": "webpack",
						"args": ["--mode", "production", "--config", "webpack.prod.js", "--optimize-minimize"],
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.Command != "webpack" {
					t.Errorf("Expected command 'webpack', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"--mode", "production", "--config", "webpack.prod.js", "--optimize-minimize"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
			},
		},

		// Mixed Format Tests
		{
			name:        "mixed_formats_all_types",
			description: "Configuration with all supported formats mixed together",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "string-cmd",
						"command": "npm run dev --watch",
						"mode": "keepAlive"
					},
					{
						"name": "array-cmd",
						"command": ["docker", "build", "-t", "myapp:latest", "."],
						"mode": "once"
					},
					{
						"name": "object-cmd",
						"command": {
							"command": "python",
							"args": ["-m", "pytest", "tests/", "-v", "--cov=src"]
						},
						"mode": "once"
					},
					{
						"name": "standard-cmd",
						"command": "go",
						"args": ["test", "./...", "-race", "-coverprofile=coverage.out"],
						"mode": "once"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if len(config.Commands) != 4 {
					t.Errorf("Expected 4 commands, got %d", len(config.Commands))
				}

				// Validate string format command
				cmd1 := config.Commands[0]
				if cmd1.Name != "string-cmd" || cmd1.Command != "npm" {
					t.Errorf("String command not parsed correctly: %+v", cmd1)
				}

				// Validate array format command
				cmd2 := config.Commands[1]
				if cmd2.Name != "array-cmd" || cmd2.Command != "docker" {
					t.Errorf("Array command not parsed correctly: %+v", cmd2)
				}

				// Validate object format command
				cmd3 := config.Commands[2]
				if cmd3.Name != "object-cmd" || cmd3.Command != "python" {
					t.Errorf("Object command not parsed correctly: %+v", cmd3)
				}

				// Validate standard format command
				cmd4 := config.Commands[3]
				if cmd4.Name != "standard-cmd" || cmd4.Command != "go" {
					t.Errorf("Standard command not parsed correctly: %+v", cmd4)
				}
			},
		},

		// Auto-generated Names Tests
		{
			name:        "auto_generated_names_all_formats",
			description: "Test auto-generated names for all formats when name is not provided",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"command": "npm start"
					},
					{
						"command": ["node", "server.js"]
					},
					{
						"command": {
							"command": "docker",
							"args": ["run", "nginx"]
						}
					},
					{
						"command": "python",
						"args": ["app.py"]
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				expectedNames := []string{"npm-start", "node-server.js", "docker-run", "python-app.py"}
				for i, expected := range expectedNames {
					if config.Commands[i].Name != expected {
						t.Errorf("Command %d: expected name '%s', got '%s'", i, expected, config.Commands[i].Name)
					}
				}
			},
		},

		// Environment Variables and Working Directory Tests
		{
			name:        "all_formats_with_env_and_workdir",
			description: "All formats with environment variables and working directories",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "frontend",
						"command": "npm run dev",
						"mode": "keepAlive",
						"workDir": "./frontend",
						"env": {
							"NODE_ENV": "development",
							"PORT": "3000"
						}
					},
					{
						"name": "backend",
						"command": ["python", "-m", "uvicorn", "main:app", "--reload"],
						"mode": "keepAlive",
						"workDir": "./backend",
						"env": {
							"PYTHONPATH": ".",
							"DEBUG": "true"
						}
					},
					{
						"name": "database",
						"command": {
							"command": "docker",
							"args": ["run", "-p", "5432:5432", "postgres"]
						},
						"mode": "keepAlive",
						"env": {
							"POSTGRES_PASSWORD": "secret",
							"POSTGRES_DB": "myapp"
						}
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				// Validate frontend command
				frontend := config.Commands[0]
				if frontend.WorkDir != "./frontend" {
					t.Errorf("Expected workDir './frontend', got '%s'", frontend.WorkDir)
				}
				if frontend.Env["NODE_ENV"] != "development" {
					t.Errorf("Expected NODE_ENV='development', got '%s'", frontend.Env["NODE_ENV"])
				}

				// Validate backend command
				backend := config.Commands[1]
				if backend.WorkDir != "./backend" {
					t.Errorf("Expected workDir './backend', got '%s'", backend.WorkDir)
				}
				if backend.Env["PYTHONPATH"] != "." {
					t.Errorf("Expected PYTHONPATH='.', got '%s'", backend.Env["PYTHONPATH"])
				}

				// Validate database command
				database := config.Commands[2]
				if database.Env["POSTGRES_PASSWORD"] != "secret" {
					t.Errorf("Expected POSTGRES_PASSWORD='secret', got '%s'", database.Env["POSTGRES_PASSWORD"])
				}
			},
		},

		// Mode Tests
		{
			name:        "all_formats_with_different_modes",
			description: "All formats with different execution modes",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "build",
						"command": "npm run build",
						"mode": "once"
					},
					{
						"name": "test",
						"command": ["npm", "test", "--watch"],
						"mode": "keepAlive"
					},
					{
						"name": "lint",
						"command": {
							"command": "eslint",
							"args": ["src/", "--fix"]
						},
						"mode": "once"
					},
					{
						"name": "server",
						"command": "node",
						"args": ["server.js"],
						"mode": "keepAlive"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				expectedModes := []Mode{ModeOnce, ModeKeepAlive, ModeOnce, ModeKeepAlive}
				for i, expected := range expectedModes {
					if config.Commands[i].Mode != expected {
						t.Errorf("Command %d: expected mode '%s', got '%s'", i, expected, config.Commands[i].Mode)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseJSON([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
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

// TestFormatCompatibility ensures that all formats produce equivalent results when they represent the same command
func TestFormatCompatibility(t *testing.T) {
	// Test that different formats representing the same command produce identical results
	testCases := []struct {
		name        string
		description string
		formats     []string
	}{
		{
			name:        "simple_command_equivalence",
			description: "Simple command 'ls' in all formats should produce identical results",
			formats: []string{
				// String format
				`{
					"version": "1.0",
					"commands": [{"name": "test", "command": "ls", "mode": "once"}]
				}`,
				// Array format
				`{
					"version": "1.0",
					"commands": [{"name": "test", "command": ["ls"], "mode": "once"}]
				}`,
				// Object format
				`{
					"version": "1.0",
					"commands": [{"name": "test", "command": {"command": "ls"}, "mode": "once"}]
				}`,
				// Standard format
				`{
					"version": "1.0",
					"commands": [{"name": "test", "command": "ls", "args": [], "mode": "once"}]
				}`,
			},
		},
		{
			name:        "command_with_args_equivalence",
			description: "Command 'npm run build' in all formats should produce identical results",
			formats: []string{
				// String format
				`{
					"version": "1.0",
					"commands": [{"name": "test", "command": "npm run build", "mode": "once"}]
				}`,
				// Array format
				`{
					"version": "1.0",
					"commands": [{"name": "test", "command": ["npm", "run", "build"], "mode": "once"}]
				}`,
				// Object format
				`{
					"version": "1.0",
					"commands": [{"name": "test", "command": {"command": "npm", "args": ["run", "build"]}, "mode": "once"}]
				}`,
				// Standard format
				`{
					"version": "1.0",
					"commands": [{"name": "test", "command": "npm", "args": ["run", "build"], "mode": "once"}]
				}`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var configs []*Config

			// Parse all formats
			for i, format := range tc.formats {
				config, err := ParseJSON([]byte(format))
				if err != nil {
					t.Fatalf("Format %d failed to parse: %v", i, err)
				}
				configs = append(configs, config)
			}

			// Compare all configs to ensure they produce identical results
			baseConfig := configs[0]
			baseCmd := baseConfig.Commands[0]

			for i := 1; i < len(configs); i++ {
				cmd := configs[i].Commands[0]

				if cmd.Name != baseCmd.Name {
					t.Errorf("Format %d: name mismatch. Expected '%s', got '%s'", i, baseCmd.Name, cmd.Name)
				}

				if cmd.Command != baseCmd.Command {
					t.Errorf("Format %d: command mismatch. Expected '%s', got '%s'", i, baseCmd.Command, cmd.Command)
				}

				if len(cmd.Args) != len(baseCmd.Args) {
					t.Errorf("Format %d: args length mismatch. Expected %d, got %d", i, len(baseCmd.Args), len(cmd.Args))
				} else {
					for j, arg := range cmd.Args {
						if arg != baseCmd.Args[j] {
							t.Errorf("Format %d: arg %d mismatch. Expected '%s', got '%s'", i, j, baseCmd.Args[j], arg)
						}
					}
				}

				if cmd.Mode != baseCmd.Mode {
					t.Errorf("Format %d: mode mismatch. Expected '%s', got '%s'", i, baseCmd.Mode, cmd.Mode)
				}
			}
		})
	}
}

// TestFormatValidation tests validation for all supported formats
func TestFormatValidation(t *testing.T) {
	tests := []struct {
		name        string
		description string
		json        string
		wantErr     bool
		errorSubstr string
	}{
		// String format validation
		{
			name:        "string_format_empty_command",
			description: "String format with empty command should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": "", "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name:        "string_format_whitespace_only",
			description: "String format with whitespace-only command should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": "   ", "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "normalization failed",
		},

		// Array format validation
		{
			name:        "array_format_empty_array",
			description: "Array format with empty array should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": [], "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name:        "array_format_non_string_element",
			description: "Array format with non-string element should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": ["npm", 123], "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},

		// Object format validation
		{
			name:        "object_format_missing_command_field",
			description: "Object format without command field should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": {"args": ["start"]}, "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name:        "object_format_non_string_command",
			description: "Object format with non-string command field should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": {"command": 123}, "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name:        "object_format_invalid_args_type",
			description: "Object format with invalid args type should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": {"command": "npm", "args": "start"}, "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},

		// Invalid format types
		{
			name:        "invalid_command_type_number",
			description: "Command as number should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": 123, "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name:        "invalid_command_type_boolean",
			description: "Command as boolean should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": true, "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
		{
			name:        "invalid_command_type_null",
			description: "Command as null should fail",
			json: `{
				"version": "1.0",
				"commands": [{"name": "test", "command": null, "mode": "once"}]
			}`,
			wantErr:     true,
			errorSubstr: "format detection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSON([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("ParseJSON() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}
		})
	}
}
