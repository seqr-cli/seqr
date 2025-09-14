package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("multiple validation errors: %s", strings.Join(messages, "; "))
}

// Validator provides comprehensive validation for configurations
type Validator struct {
	StrictMode       bool
	ValidateWorkDirs bool
	ValidateCommands bool
}

func NewValidator() *Validator {
	return &Validator{
		StrictMode:       false,
		ValidateWorkDirs: false,
		ValidateCommands: false,
	}
}

func NewStrictValidator() *Validator {
	return &Validator{
		StrictMode:       true,
		ValidateWorkDirs: true,
		ValidateCommands: false,
	}
}

func (v *Validator) ValidateConfig(config *Config) error {
	if config == nil {
		return ValidationError{Message: "configuration cannot be nil"}
	}

	var errors ValidationErrors

	if err := v.validateVersion(config.Version); err != nil {
		errors = append(errors, ValidationError{Field: "version", Value: config.Version, Message: err.Error()})
	}

	if err := v.validateCommandsArray(config.Commands); err != nil {
		errors = append(errors, ValidationError{Field: "commands", Message: err.Error()})
	}

	for i, cmd := range config.Commands {
		if cmdErrors := v.validateCommand(&cmd); len(cmdErrors) > 0 {
			for _, cmdErr := range cmdErrors {
				cmdErr.Field = fmt.Sprintf("commands[%d].%s", i, cmdErr.Field)
				errors = append(errors, cmdErr)
			}
		}
	}

	if err := v.validateCommandNameUniqueness(config.Commands); err != nil {
		errors = append(errors, ValidationError{Field: "commands", Message: err.Error()})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func (v *Validator) validateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version is required")
	}

	if v.StrictMode {
		versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:-([a-zA-Z0-9\-\.]+))?(?:\+([a-zA-Z0-9\-\.]+))?$`)
		if !versionRegex.MatchString(version) {
			return fmt.Errorf("version '%s' is not a valid semantic version (expected format: X.Y or X.Y.Z)", version)
		}
	}

	return nil
}

func (v *Validator) validateCommandsArray(commands []Command) error {
	if len(commands) == 0 {
		return fmt.Errorf("at least one command is required")
	}

	if v.StrictMode {
		const maxCommands = 50
		if len(commands) > maxCommands {
			return fmt.Errorf("too many commands (%d), maximum allowed is %d", len(commands), maxCommands)
		}
	}

	return nil
}

func (v *Validator) validateCommand(cmd *Command) ValidationErrors {
	var errors ValidationErrors

	if cmd.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: "command name is required"})
	}

	if cmd.Command == "" {
		errors = append(errors, ValidationError{Field: "command", Message: "command is required"})
	}

	if cmd.Name != "" {
		if err := v.validateCommandName(cmd.Name); err != nil {
			errors = append(errors, ValidationError{Field: "name", Value: cmd.Name, Message: err.Error()})
		}
	}

	if cmd.Command != "" {
		if err := v.validateCommandString(cmd.Command); err != nil {
			errors = append(errors, ValidationError{Field: "command", Value: cmd.Command, Message: err.Error()})
		}
	}

	if err := v.validateMode(cmd.Mode); err != nil {
		errors = append(errors, ValidationError{Field: "mode", Value: cmd.Mode, Message: err.Error()})
	}

	if err := v.validateArgs(cmd.Args); err != nil {
		errors = append(errors, ValidationError{Field: "args", Message: err.Error()})
	}

	if cmd.WorkDir != "" {
		if err := v.validateWorkDir(cmd.WorkDir); err != nil {
			errors = append(errors, ValidationError{Field: "workDir", Value: cmd.WorkDir, Message: err.Error()})
		}
	}

	if err := v.validateEnv(cmd.Env); err != nil {
		errors = append(errors, ValidationError{Field: "env", Message: err.Error()})
	}

	return errors
}

func (v *Validator) validateCommandName(name string) error {
	if len(name) > 100 {
		return fmt.Errorf("command name too long (%d characters), maximum is 100", len(name))
	}

	if v.StrictMode {
		nameRegex := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-_]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`)
		if !nameRegex.MatchString(name) {
			return fmt.Errorf("command name '%s' must contain only alphanumeric characters, hyphens, and underscores, and cannot start or end with special characters", name)
		}

		reservedNames := []string{"help", "version", "config", "init", "setup"}
		for _, reserved := range reservedNames {
			if strings.EqualFold(name, reserved) {
				return fmt.Errorf("command name '%s' is reserved", name)
			}
		}
	}

	return nil
}

func (v *Validator) validateCommandString(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("command cannot be empty or whitespace only")
	}

	if len(command) > 500 {
		return fmt.Errorf("command too long (%d characters), maximum is 500", len(command))
	}

	if v.StrictMode {
		dangerousCommands := []string{"rm", "del", "format", "fdisk", "mkfs", "dd"}
		cmdLower := strings.ToLower(strings.TrimSpace(command))
		if slices.Contains(dangerousCommands, cmdLower) {
			return fmt.Errorf("potentially dangerous command '%s' not allowed in strict mode", command)
		}
	}

	return nil
}

func (v *Validator) validateMode(mode Mode) error {
	if mode != ModeOnce && mode != ModeKeepAlive {
		return fmt.Errorf("mode must be either 'once' or 'keepAlive', got '%s'", mode)
	}
	return nil
}

func (v *Validator) validateArgs(args []string) error {
	if v.StrictMode {
		const maxArgs = 50
		if len(args) > maxArgs {
			return fmt.Errorf("too many arguments (%d), maximum allowed is %d", len(args), maxArgs)
		}

		for i, arg := range args {
			if len(arg) > 1000 {
				return fmt.Errorf("argument %d too long (%d characters), maximum is 1000", i, len(arg))
			}
		}
	}

	return nil
}

func (v *Validator) validateWorkDir(workDir string) error {
	if strings.TrimSpace(workDir) == "" {
		return fmt.Errorf("workDir cannot be empty or whitespace only")
	}

	cleanPath := filepath.Clean(workDir)

	if v.StrictMode {
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("workDir '%s' cannot contain parent directory references (..)", workDir)
		}

		if filepath.IsAbs(cleanPath) && !strings.HasPrefix(cleanPath, "/tmp") && !strings.HasPrefix(cleanPath, "/var/tmp") {
		}
	}

	if v.ValidateWorkDirs {
		if _, err := os.Stat(cleanPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("workDir '%s' does not exist", cleanPath)
			}
			return fmt.Errorf("cannot access workDir '%s': %v", cleanPath, err)
		}
	}

	return nil
}

func (v *Validator) validateEnv(env map[string]string) error {
	if v.StrictMode {
		const maxEnvVars = 100
		if len(env) > maxEnvVars {
			return fmt.Errorf("too many environment variables (%d), maximum allowed is %d", len(env), maxEnvVars)
		}

		envNameRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
		for name, value := range env {
			if !envNameRegex.MatchString(name) {
				return fmt.Errorf("invalid environment variable name '%s': must start with letter or underscore, contain only alphanumeric characters and underscores", name)
			}

			if len(name) > 100 {
				return fmt.Errorf("environment variable name '%s' too long (%d characters), maximum is 100", name, len(name))
			}

			if len(value) > 10000 {
				return fmt.Errorf("environment variable '%s' value too long (%d characters), maximum is 10000", name, len(value))
			}
		}
	}

	return nil
}

func (v *Validator) validateCommandNameUniqueness(commands []Command) error {
	nameMap := make(map[string]int)

	for i, cmd := range commands {
		if cmd.Name == "" {
			continue
		}

		if prevIndex, exists := nameMap[cmd.Name]; exists {
			return fmt.Errorf("duplicate command name '%s' found at positions %d and %d", cmd.Name, prevIndex, i)
		}
		nameMap[cmd.Name] = i
	}

	return nil
}
