package config

import (
	"strings"
	"testing"
)

func TestNormalizer_NormalizeCommand(t *testing.T) {
	normalizer := NewNormalizer()

	tests := []struct {
		name        string
		input       interface{}
		cmdName     string
		mode        Mode
		workDir     string
		env         map[string]string
		wantErr     bool
		errorSubstr string
		validate    func(*testing.T, *Command)
	}{
		{
			name:    "string command with multiple args",
			input:   "npm run build --production",
			cmdName: "build-cmd",
			mode:    ModeOnce,
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
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
				if cmd.Name != "build-cmd" {
					t.Errorf("Expected name 'build-cmd', got '%s'", cmd.Name)
				}
			},
		},
		{
			name:    "string command single word",
			input:   "ls",
			cmdName: "",
			mode:    ModeOnce,
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "ls" {
					t.Errorf("Expected command 'ls', got '%s'", cmd.Command)
				}
				if len(cmd.Args) != 0 {
					t.Errorf("Expected no args, got %d", len(cmd.Args))
				}
				if cmd.Name != "ls" {
					t.Errorf("Expected auto-generated name 'ls', got '%s'", cmd.Name)
				}
			},
		},
		{
			name:    "array command format",
			input:   []interface{}{"docker", "run", "-p", "8080:80", "nginx"},
			cmdName: "web-server",
			mode:    ModeKeepAlive,
			workDir: "/app",
			env:     map[string]string{"ENV": "production"},
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "docker" {
					t.Errorf("Expected command 'docker', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"run", "-p", "8080:80", "nginx"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
				if cmd.Mode != ModeKeepAlive {
					t.Errorf("Expected mode 'keepAlive', got '%s'", cmd.Mode)
				}
				if cmd.WorkDir != "/app" {
					t.Errorf("Expected workDir '/app', got '%s'", cmd.WorkDir)
				}
				if cmd.Env["ENV"] != "production" {
					t.Errorf("Expected env ENV='production', got '%s'", cmd.Env["ENV"])
				}
			},
		},
		{
			name: "object command with args",
			input: map[string]interface{}{
				"command": "node",
				"args":    []interface{}{"server.js", "--port", "3000"},
			},
			cmdName: "api-server",
			mode:    ModeKeepAlive,
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "node" {
					t.Errorf("Expected command 'node', got '%s'", cmd.Command)
				}
				expectedArgs := []string{"server.js", "--port", "3000"}
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
			},
		},
		{
			name: "object command without args",
			input: map[string]interface{}{
				"command": "echo",
			},
			cmdName: "simple-echo",
			mode:    ModeOnce,
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Command != "echo" {
					t.Errorf("Expected command 'echo', got '%s'", cmd.Command)
				}
				if len(cmd.Args) != 0 {
					t.Errorf("Expected no args, got %d", len(cmd.Args))
				}
			},
		},
		{
			name:    "default mode when not specified",
			input:   "echo hello",
			cmdName: "test",
			mode:    "",
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Mode != ModeOnce {
					t.Errorf("Expected default mode 'once', got '%s'", cmd.Mode)
				}
			},
		},
		{
			name:    "auto-generated name with args",
			input:   "npm start",
			cmdName: "",
			mode:    ModeOnce,
			wantErr: false,
			validate: func(t *testing.T, cmd *Command) {
				if cmd.Name != "npm-start" {
					t.Errorf("Expected auto-generated name 'npm-start', got '%s'", cmd.Name)
				}
			},
		},
		{
			name:        "nil input",
			input:       nil,
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "command input cannot be nil",
		},
		{
			name:        "empty string command",
			input:       "",
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "command string cannot be empty",
		},
		{
			name:        "empty array command",
			input:       []interface{}{},
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "command array cannot be empty",
		},
		{
			name:        "array with non-string elements",
			input:       []interface{}{"npm", 123},
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "command array element 1 must be a string",
		},
		{
			name:        "array with non-string first element",
			input:       []interface{}{123, "start"},
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "first element of command array must be a string",
		},
		{
			name: "object without command field",
			input: map[string]interface{}{
				"args": []interface{}{"start"},
			},
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "object format must have a 'command' field",
		},
		{
			name: "object with non-string command",
			input: map[string]interface{}{
				"command": 123,
			},
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "'command' field must be a string",
		},
		{
			name: "object with invalid args type",
			input: map[string]interface{}{
				"command": "npm",
				"args":    "start",
			},
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "'args' field must be an array",
		},
		{
			name: "object with non-string args elements",
			input: map[string]interface{}{
				"command": "npm",
				"args":    []interface{}{"start", 123},
			},
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "args array element 1 must be a string",
		},
		{
			name:        "unsupported command type",
			input:       123,
			cmdName:     "test",
			mode:        ModeOnce,
			wantErr:     true,
			errorSubstr: "unsupported command format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := normalizer.NormalizeCommand(tt.input, tt.cmdName, tt.mode, tt.workDir, tt.env)

			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("NormalizeCommand() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}

			if !tt.wantErr {
				if cmd == nil {
					t.Error("NormalizeCommand() returned nil command without error")
				} else if tt.validate != nil {
					tt.validate(t, cmd)
				}
			}
		})
	}
}

func TestNormalizer_NormalizeConfig(t *testing.T) {
	normalizer := NewNormalizer()

	tests := []struct {
		name        string
		input       interface{}
		wantErr     bool
		errorSubstr string
		validate    func(*testing.T, *Config)
	}{
		{
			name: "mixed command formats",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"name":    "string-cmd",
						"command": "npm run build",
						"mode":    "once",
					},
					map[string]interface{}{
						"name":    "array-cmd",
						"command": []interface{}{"node", "server.js"},
						"mode":    "keepAlive",
					},
					map[string]interface{}{
						"name": "object-cmd",
						"command": map[string]interface{}{
							"command": "docker",
							"args":    []interface{}{"run", "nginx"},
						},
						"mode": "keepAlive",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if len(config.Commands) != 3 {
					t.Errorf("Expected 3 commands, got %d", len(config.Commands))
				}

				// Check string command
				cmd1 := config.Commands[0]
				if cmd1.Name != "string-cmd" || cmd1.Command != "npm" || len(cmd1.Args) != 2 {
					t.Errorf("String command not normalized correctly: %+v", cmd1)
				}

				// Check array command
				cmd2 := config.Commands[1]
				if cmd2.Name != "array-cmd" || cmd2.Command != "node" || len(cmd2.Args) != 1 {
					t.Errorf("Array command not normalized correctly: %+v", cmd2)
				}

				// Check object command
				cmd3 := config.Commands[2]
				if cmd3.Name != "object-cmd" || cmd3.Command != "docker" || len(cmd3.Args) != 2 {
					t.Errorf("Object command not normalized correctly: %+v", cmd3)
				}
			},
		},
		{
			name: "config with environment variables and work directories",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"name":    "server",
						"command": "npm start",
						"mode":    "keepAlive",
						"workDir": "./backend",
						"env": map[string]interface{}{
							"NODE_ENV": "development",
							"PORT":     "3000",
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				cmd := config.Commands[0]
				if cmd.WorkDir != "./backend" {
					t.Errorf("Expected workDir './backend', got '%s'", cmd.WorkDir)
				}
				if cmd.Env["NODE_ENV"] != "development" {
					t.Errorf("Expected NODE_ENV='development', got '%s'", cmd.Env["NODE_ENV"])
				}
				if cmd.Env["PORT"] != "3000" {
					t.Errorf("Expected PORT='3000', got '%s'", cmd.Env["PORT"])
				}
			},
		},
		{
			name: "auto-generated command names",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"command": "npm start",
					},
					map[string]interface{}{
						"command": "ls",
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
			name:        "nil input",
			input:       nil,
			wantErr:     true,
			errorSubstr: "config input cannot be nil",
		},
		{
			name:        "non-object input",
			input:       "invalid",
			wantErr:     true,
			errorSubstr: "config must be an object",
		},
		{
			name: "missing version",
			input: map[string]interface{}{
				"commands": []interface{}{},
			},
			wantErr:     true,
			errorSubstr: "config must have a 'version' field",
		},
		{
			name: "non-string version",
			input: map[string]interface{}{
				"version":  123,
				"commands": []interface{}{},
			},
			wantErr:     true,
			errorSubstr: "version must be a string",
		},
		{
			name: "missing commands",
			input: map[string]interface{}{
				"version": "1.0",
			},
			wantErr:     true,
			errorSubstr: "config must have a 'commands' field",
		},
		{
			name: "non-array commands",
			input: map[string]interface{}{
				"version":  "1.0",
				"commands": "invalid",
			},
			wantErr:     true,
			errorSubstr: "'commands' field must be an array",
		},
		{
			name: "empty commands array",
			input: map[string]interface{}{
				"version":  "1.0",
				"commands": []interface{}{},
			},
			wantErr:     true,
			errorSubstr: "config must have at least one command",
		},
		{
			name: "non-object command",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					"invalid",
				},
			},
			wantErr:     true,
			errorSubstr: "command must be an object",
		},
		{
			name: "command with invalid name type",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"name":    123,
						"command": "echo",
					},
				},
			},
			wantErr:     true,
			errorSubstr: "name must be a string",
		},
		{
			name: "command with invalid mode type",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"command": "echo",
						"mode":    123,
					},
				},
			},
			wantErr:     true,
			errorSubstr: "mode must be a string",
		},
		{
			name: "command with invalid workDir type",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"command": "echo",
						"workDir": 123,
					},
				},
			},
			wantErr:     true,
			errorSubstr: "workDir must be a string",
		},
		{
			name: "command with invalid env type",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"command": "echo",
						"env":     "invalid",
					},
				},
			},
			wantErr:     true,
			errorSubstr: "env must be an object",
		},
		{
			name: "command with invalid env value type",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"command": "echo",
						"env": map[string]interface{}{
							"KEY": 123,
						},
					},
				},
			},
			wantErr:     true,
			errorSubstr: "env value for key 'KEY' must be a string",
		},
		{
			name: "command missing command field",
			input: map[string]interface{}{
				"version": "1.0",
				"commands": []interface{}{
					map[string]interface{}{
						"name": "test",
					},
				},
			},
			wantErr:     true,
			errorSubstr: "must have a 'command' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := normalizer.NormalizeConfig(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("NormalizeConfig() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}

			if !tt.wantErr {
				if config == nil {
					t.Error("NormalizeConfig() returned nil config without error")
				} else if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestNormalizer_NormalizeFromJSON(t *testing.T) {
	normalizer := NewNormalizer()

	tests := []struct {
		name        string
		json        string
		wantErr     bool
		errorSubstr string
		validate    func(*testing.T, *Config)
	}{
		{
			name: "valid JSON with mixed formats",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "build",
						"command": "npm run build",
						"mode": "once"
					},
					{
						"name": "start",
						"command": ["node", "server.js"],
						"mode": "keepAlive"
					},
					{
						"name": "docker",
						"command": {
							"command": "docker",
							"args": ["run", "nginx"]
						},
						"mode": "keepAlive"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if len(config.Commands) != 3 {
					t.Errorf("Expected 3 commands, got %d", len(config.Commands))
				}
				if config.Version != "1.0" {
					t.Errorf("Expected version '1.0', got '%s'", config.Version)
				}
			},
		},
		{
			name:        "empty JSON data",
			json:        "",
			wantErr:     true,
			errorSubstr: "JSON data cannot be empty",
		},
		{
			name:        "invalid JSON syntax",
			json:        `{"version": "1.0"`,
			wantErr:     true,
			errorSubstr: "failed to parse JSON",
		},
		{
			name: "valid JSON but invalid config",
			json: `{
				"version": "1.0",
				"commands": []
			}`,
			wantErr:     true,
			errorSubstr: "failed to normalize config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := normalizer.NormalizeFromJSON([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("NormalizeFromJSON() error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}

			if !tt.wantErr {
				if config == nil {
					t.Error("NormalizeFromJSON() returned nil config without error")
				} else if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestNormalizer_StrictMode(t *testing.T) {
	strictNormalizer := NewStrictNormalizer()

	tests := []struct {
		name        string
		input       interface{}
		wantErr     bool
		errorSubstr string
	}{
		{
			name:        "string command too long in strict mode",
			input:       strings.Repeat("a", 501),
			wantErr:     true,
			errorSubstr: "command string too long",
		},
		{
			name:        "array command too long in strict mode",
			input:       make([]interface{}, 51),
			wantErr:     true,
			errorSubstr: "command array too long",
		},
		{
			name: "object args too long in strict mode",
			input: map[string]interface{}{
				"command": "test",
				"args":    make([]interface{}, 51),
			},
			wantErr:     true,
			errorSubstr: "args array too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill array with strings for valid test data
			if arr, ok := tt.input.([]interface{}); ok {
				for i := range arr {
					arr[i] = "arg"
				}
			}
			if obj, ok := tt.input.(map[string]interface{}); ok {
				if args, hasArgs := obj["args"].([]interface{}); hasArgs {
					for i := range args {
						args[i] = "arg"
					}
				}
			}

			_, err := strictNormalizer.NormalizeCommand(tt.input, "test", ModeOnce, "", nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeCommand() in strict mode error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("NormalizeCommand() in strict mode error = %v, expected to contain %q", err, tt.errorSubstr)
				}
			}
		})
	}
}

func TestNormalizer_generateCommandName(t *testing.T) {
	normalizer := NewNormalizer()

	tests := []struct {
		name     string
		command  string
		args     []string
		expected string
	}{
		{
			name:     "command with args",
			command:  "npm",
			args:     []string{"start", "--port", "3000"},
			expected: "npm-start",
		},
		{
			name:     "command without args",
			command:  "ls",
			args:     []string{},
			expected: "ls",
		},
		{
			name:     "empty command",
			command:  "",
			args:     []string{"arg1"},
			expected: "unnamed-command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.generateCommandName(tt.command, tt.args)
			if result != tt.expected {
				t.Errorf("generateCommandName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
