package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidator_ValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		validator *Validator
		config    *Config
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "nil config",
			validator: NewValidator(),
			config:    nil,
			wantErr:   true,
			errSubstr: "cannot be nil",
		},
		{
			name:      "valid basic config",
			validator: NewValidator(),
			config: &Config{
				Version: "1.0",
				Commands: []Command{
					{
						Name:    "test",
						Command: "echo",
						Mode:    ModeOnce,
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "missing version",
			validator: NewValidator(),
			config: &Config{
				Commands: []Command{
					{
						Name:    "test",
						Command: "echo",
						Mode:    ModeOnce,
					},
				},
			},
			wantErr:   true,
			errSubstr: "version is required",
		},
		{
			name:      "no commands",
			validator: NewValidator(),
			config: &Config{
				Version:  "1.0",
				Commands: []Command{},
			},
			wantErr:   true,
			errSubstr: "at least one command is required",
		},
		{
			name:      "duplicate command names",
			validator: NewValidator(),
			config: &Config{
				Version: "1.0",
				Commands: []Command{
					{Name: "test", Command: "echo", Mode: ModeOnce},
					{Name: "test", Command: "ls", Mode: ModeOnce},
				},
			},
			wantErr:   true,
			errSubstr: "duplicate command name",
		},
		{
			name:      "strict mode version validation",
			validator: NewStrictValidator(),
			config: &Config{
				Version: "invalid-version",
				Commands: []Command{
					{Name: "test", Command: "echo", Mode: ModeOnce},
				},
			},
			wantErr:   true,
			errSubstr: "not a valid semantic version",
		},
		{
			name:      "valid semantic version in strict mode",
			validator: NewStrictValidator(),
			config: &Config{
				Version: "1.2.3",
				Commands: []Command{
					{Name: "test", Command: "echo", Mode: ModeOnce},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.ValidateConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("ValidateConfig() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestValidator_validateVersion(t *testing.T) {
	tests := []struct {
		name      string
		validator *Validator
		version   string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "empty version",
			validator: NewValidator(),
			version:   "",
			wantErr:   true,
			errSubstr: "version is required",
		},
		{
			name:      "valid simple version",
			validator: NewValidator(),
			version:   "1.0",
			wantErr:   false,
		},
		{
			name:      "invalid version in strict mode",
			validator: NewStrictValidator(),
			version:   "not-a-version",
			wantErr:   true,
			errSubstr: "not a valid semantic version",
		},
		{
			name:      "valid semantic version",
			validator: NewStrictValidator(),
			version:   "1.2.3",
			wantErr:   false,
		},
		{
			name:      "valid semantic version with prerelease",
			validator: NewStrictValidator(),
			version:   "1.2.3-alpha.1",
			wantErr:   false,
		},
		{
			name:      "valid semantic version with build metadata",
			validator: NewStrictValidator(),
			version:   "1.2.3+build.1",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.validateVersion(tt.version)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateVersion() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestValidator_validateCommandName(t *testing.T) {
	tests := []struct {
		name      string
		validator *Validator
		cmdName   string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "valid simple name",
			validator: NewValidator(),
			cmdName:   "test",
			wantErr:   false,
		},
		{
			name:      "valid name with hyphen",
			validator: NewValidator(),
			cmdName:   "test-command",
			wantErr:   false,
		},
		{
			name:      "valid name with underscore",
			validator: NewValidator(),
			cmdName:   "test_command",
			wantErr:   false,
		},
		{
			name:      "too long name",
			validator: NewValidator(),
			cmdName:   strings.Repeat("a", 101),
			wantErr:   true,
			errSubstr: "too long",
		},
		{
			name:      "invalid name starting with hyphen in strict mode",
			validator: NewStrictValidator(),
			cmdName:   "-invalid",
			wantErr:   true,
			errSubstr: "cannot start or end with special characters",
		},
		{
			name:      "reserved name in strict mode",
			validator: NewStrictValidator(),
			cmdName:   "help",
			wantErr:   true,
			errSubstr: "is reserved",
		},
		{
			name:      "valid name in strict mode",
			validator: NewStrictValidator(),
			cmdName:   "my-command",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.validateCommandName(tt.cmdName)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommandName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateCommandName() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestValidator_validateCommandString(t *testing.T) {
	tests := []struct {
		name      string
		validator *Validator
		command   string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "valid command",
			validator: NewValidator(),
			command:   "echo",
			wantErr:   false,
		},
		{
			name:      "empty command",
			validator: NewValidator(),
			command:   "",
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
		{
			name:      "whitespace only command",
			validator: NewValidator(),
			command:   "   ",
			wantErr:   true,
			errSubstr: "cannot be empty or whitespace only",
		},
		{
			name:      "too long command",
			validator: NewValidator(),
			command:   strings.Repeat("a", 501),
			wantErr:   true,
			errSubstr: "too long",
		},
		{
			name:      "dangerous command in strict mode",
			validator: NewStrictValidator(),
			command:   "rm",
			wantErr:   true,
			errSubstr: "potentially dangerous command",
		},
		{
			name:      "safe command in strict mode",
			validator: NewStrictValidator(),
			command:   "echo",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.validateCommandString(tt.command)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommandString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateCommandString() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestValidator_validateArgs(t *testing.T) {
	tests := []struct {
		name      string
		validator *Validator
		args      []string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "valid args",
			validator: NewValidator(),
			args:      []string{"hello", "world"},
			wantErr:   false,
		},
		{
			name:      "empty args",
			validator: NewValidator(),
			args:      []string{},
			wantErr:   false,
		},
		{
			name:      "too many args in strict mode",
			validator: NewStrictValidator(),
			args:      make([]string, 51),
			wantErr:   true,
			errSubstr: "too many arguments",
		},
		{
			name:      "too long arg in strict mode",
			validator: NewStrictValidator(),
			args:      []string{strings.Repeat("a", 1001)},
			wantErr:   true,
			errSubstr: "too long",
		},
		{
			name:      "valid args in strict mode",
			validator: NewStrictValidator(),
			args:      []string{"hello", "world"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.validateArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateArgs() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestValidator_validateWorkDir(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		validator *Validator
		workDir   string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "valid relative path",
			validator: NewValidator(),
			workDir:   "./test",
			wantErr:   false,
		},
		{
			name:      "empty workDir",
			validator: NewValidator(),
			workDir:   "",
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
		{
			name:      "whitespace only workDir",
			validator: NewValidator(),
			workDir:   "   ",
			wantErr:   true,
			errSubstr: "cannot be empty or whitespace only",
		},
		{
			name:      "parent directory reference in strict mode",
			validator: NewStrictValidator(),
			workDir:   "../test",
			wantErr:   true,
			errSubstr: "cannot contain parent directory references",
		},
		{
			name:      "existing directory with validation enabled",
			validator: &Validator{ValidateWorkDirs: true},
			workDir:   tmpDir,
			wantErr:   false,
		},
		{
			name:      "non-existing directory with validation enabled",
			validator: &Validator{ValidateWorkDirs: true},
			workDir:   filepath.Join(tmpDir, "nonexistent"),
			wantErr:   true,
			errSubstr: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.validateWorkDir(tt.workDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateWorkDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateWorkDir() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestValidator_validateEnv(t *testing.T) {
	tests := []struct {
		name      string
		validator *Validator
		env       map[string]string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "valid env vars",
			validator: NewValidator(),
			env:       map[string]string{"NODE_ENV": "development", "PORT": "3000"},
			wantErr:   false,
		},
		{
			name:      "empty env vars",
			validator: NewValidator(),
			env:       map[string]string{},
			wantErr:   false,
		},
		{
			name:      "nil env vars",
			validator: NewValidator(),
			env:       nil,
			wantErr:   false,
		},
		{
			name:      "invalid env var name in strict mode",
			validator: NewStrictValidator(),
			env:       map[string]string{"123INVALID": "value"},
			wantErr:   true,
			errSubstr: "invalid environment variable name",
		},
		{
			name:      "too long env var name in strict mode",
			validator: NewStrictValidator(),
			env:       map[string]string{strings.Repeat("A", 101): "value"},
			wantErr:   true,
			errSubstr: "too long",
		},
		{
			name:      "too long env var value in strict mode",
			validator: NewStrictValidator(),
			env:       map[string]string{"TEST": strings.Repeat("a", 10001)},
			wantErr:   true,
			errSubstr: "too long",
		},
		{
			name:      "too many env vars in strict mode",
			validator: NewStrictValidator(),
			env: func() map[string]string {
				env := make(map[string]string)
				for i := 0; i < 101; i++ {
					env[fmt.Sprintf("VAR_%d", i)] = "value"
				}
				return env
			}(),
			wantErr:   true,
			errSubstr: "too many environment variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.validateEnv(tt.env)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateEnv() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestValidator_validateCommandNameUniqueness(t *testing.T) {
	tests := []struct {
		name      string
		validator *Validator
		commands  []Command
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "unique names",
			validator: NewValidator(),
			commands: []Command{
				{Name: "cmd1", Command: "echo", Mode: ModeOnce},
				{Name: "cmd2", Command: "ls", Mode: ModeOnce},
			},
			wantErr: false,
		},
		{
			name:      "duplicate names",
			validator: NewValidator(),
			commands: []Command{
				{Name: "cmd1", Command: "echo", Mode: ModeOnce},
				{Name: "cmd1", Command: "ls", Mode: ModeOnce},
			},
			wantErr:   true,
			errSubstr: "duplicate command name",
		},
		{
			name:      "empty names ignored",
			validator: NewValidator(),
			commands: []Command{
				{Name: "", Command: "echo", Mode: ModeOnce},
				{Name: "", Command: "ls", Mode: ModeOnce},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.validateCommandNameUniqueness(tt.commands)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommandNameUniqueness() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateCommandNameUniqueness() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{
			name: "error with field",
			err:  ValidationError{Field: "name", Message: "is required"},
			want: "validation error in field 'name': is required",
		},
		{
			name: "error without field",
			err:  ValidationError{Message: "general error"},
			want: "validation error: general error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name string
		errs ValidationErrors
		want string
	}{
		{
			name: "no errors",
			errs: ValidationErrors{},
			want: "no validation errors",
		},
		{
			name: "single error",
			errs: ValidationErrors{
				{Field: "name", Message: "is required"},
			},
			want: "validation error in field 'name': is required",
		},
		{
			name: "multiple errors",
			errs: ValidationErrors{
				{Field: "name", Message: "is required"},
				{Field: "command", Message: "is empty"},
			},
			want: "multiple validation errors: validation error in field 'name': is required; validation error in field 'command': is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errs.Error(); got != tt.want {
				t.Errorf("ValidationErrors.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v.StrictMode {
		t.Error("NewValidator() should not enable strict mode by default")
	}
	if v.ValidateWorkDirs {
		t.Error("NewValidator() should not enable work dir validation by default")
	}
	if v.ValidateCommands {
		t.Error("NewValidator() should not enable command validation by default")
	}
}

func TestNewStrictValidator(t *testing.T) {
	v := NewStrictValidator()
	if !v.StrictMode {
		t.Error("NewStrictValidator() should enable strict mode")
	}
	if !v.ValidateWorkDirs {
		t.Error("NewStrictValidator() should enable work dir validation")
	}
	// Command validation is expensive, so it's kept optional even in strict mode
	if v.ValidateCommands {
		t.Error("NewStrictValidator() should not enable command validation by default")
	}
}
