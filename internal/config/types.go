package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Mode string

const (
	ModeOnce      Mode = "once"
	ModeKeepAlive Mode = "keepAlive"
)

type Command struct {
	Name       string            `json:"name"`
	Command    string            `json:"command"`
	Args       []string          `json:"args,omitempty"`
	Mode       Mode              `json:"mode"`
	WorkDir    string            `json:"workDir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Concurrent bool              `json:"concurrent,omitempty"` // Allow concurrent execution with other concurrent commands
}

// FlexibleCommand represents a command that can be parsed from multiple formats
type FlexibleCommand struct {
	Name       string            `json:"name,omitempty"`
	Command    interface{}       `json:"command"` // Can be string, []string, or object
	Mode       Mode              `json:"mode,omitempty"`
	WorkDir    string            `json:"workDir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Concurrent bool              `json:"concurrent,omitempty"` // Allow concurrent execution with other concurrent commands
}

// FlexibleConfig represents a configuration that supports multiple command formats
type FlexibleConfig struct {
	Version  string            `json:"version"`
	Commands []FlexibleCommand `json:"commands"`
}

type Config struct {
	Version  string    `json:"version"`
	Commands []Command `json:"commands"`
}

func (c *Config) Validate() error {
	validator := NewValidator()
	return validator.ValidateConfig(c)
}

func (c *Command) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("command name is required")
	}

	if c.Command == "" {
		return fmt.Errorf("command is required")
	}

	if c.Mode != ModeOnce && c.Mode != ModeKeepAlive {
		return fmt.Errorf("mode must be either 'once' or 'keepAlive', got '%s'", c.Mode)
	}

	return nil
}

func (m *Mode) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	mode := Mode(s)
	if mode != ModeOnce && mode != ModeKeepAlive {
		return fmt.Errorf("invalid mode: %s", s)
	}

	*m = mode
	return nil
}

// ToStandardCommand converts a FlexibleCommand to a standard Command
func (fc *FlexibleCommand) ToStandardCommand() (*Command, error) {
	cmd := &Command{
		Name:       fc.Name,
		Mode:       fc.Mode,
		WorkDir:    fc.WorkDir,
		Env:        fc.Env,
		Concurrent: fc.Concurrent,
	}

	// Set default mode if not specified
	if cmd.Mode == "" {
		cmd.Mode = ModeOnce
	}

	// Parse the command field based on its type
	switch v := fc.Command.(type) {
	case string:
		// String format: "npm start" or "echo hello world"
		parts := strings.Fields(v)
		if len(parts) == 0 {
			return nil, fmt.Errorf("command string cannot be empty")
		}
		cmd.Command = parts[0]
		if len(parts) > 1 {
			cmd.Args = parts[1:]
		}

	case []interface{}:
		// Array format: ["npm", "start"] or ["echo", "hello", "world"]
		if len(v) == 0 {
			return nil, fmt.Errorf("command array cannot be empty")
		}

		// Convert first element to command
		if cmdStr, ok := v[0].(string); ok {
			cmd.Command = cmdStr
		} else {
			return nil, fmt.Errorf("first element of command array must be a string")
		}

		// Convert remaining elements to args
		if len(v) > 1 {
			cmd.Args = make([]string, len(v)-1)
			for i, arg := range v[1:] {
				if argStr, ok := arg.(string); ok {
					cmd.Args[i] = argStr
				} else {
					return nil, fmt.Errorf("command array elements must be strings")
				}
			}
		}

	case map[string]interface{}:
		// Object format: {"command": "npm", "args": ["start"]}
		if cmdStr, ok := v["command"].(string); ok {
			cmd.Command = cmdStr
		} else {
			return nil, fmt.Errorf("object format must have a 'command' field of type string")
		}

		if argsInterface, ok := v["args"]; ok {
			if argsList, ok := argsInterface.([]interface{}); ok {
				cmd.Args = make([]string, len(argsList))
				for i, arg := range argsList {
					if argStr, ok := arg.(string); ok {
						cmd.Args[i] = argStr
					} else {
						return nil, fmt.Errorf("args array elements must be strings")
					}
				}
			} else {
				return nil, fmt.Errorf("'args' field must be an array")
			}
		}

	default:
		return nil, fmt.Errorf("command field must be a string, array, or object")
	}

	return cmd, nil
}

// ToStandardConfig converts a FlexibleConfig to a standard Config
func (fc *FlexibleConfig) ToStandardConfig() (*Config, error) {
	config := &Config{
		Version:  fc.Version,
		Commands: make([]Command, len(fc.Commands)),
	}

	for i, flexCmd := range fc.Commands {
		stdCmd, err := flexCmd.ToStandardCommand()
		if err != nil {
			return nil, fmt.Errorf("error converting command %d: %w", i, err)
		}

		// Generate name if not provided
		if stdCmd.Name == "" {
			if len(stdCmd.Args) > 0 {
				stdCmd.Name = fmt.Sprintf("%s-%s", stdCmd.Command, stdCmd.Args[0])
			} else {
				stdCmd.Name = stdCmd.Command
			}
		}

		config.Commands[i] = *stdCmd
	}

	return config, nil
}
