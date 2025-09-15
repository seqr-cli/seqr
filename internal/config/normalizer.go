package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Normalizer provides functionality to convert various command formats to a unified internal structure
type Normalizer struct {
	StrictMode bool
}

// NewNormalizer creates a new normalizer with default settings
func NewNormalizer() *Normalizer {
	return &Normalizer{
		StrictMode: false,
	}
}

// NewStrictNormalizer creates a new normalizer with strict validation enabled
func NewStrictNormalizer() *Normalizer {
	return &Normalizer{
		StrictMode: true,
	}
}

// NormalizeCommand converts any supported command format to the unified Command structure
func (n *Normalizer) NormalizeCommand(input interface{}, name string, mode Mode, workDir string, env map[string]string) (*Command, error) {
	if input == nil {
		return nil, fmt.Errorf("command input cannot be nil")
	}

	cmd := &Command{
		Name:    name,
		Mode:    mode,
		WorkDir: workDir,
		Env:     env,
	}

	// Set default mode if not specified
	if cmd.Mode == "" {
		cmd.Mode = ModeOnce
	}

	// Normalize the command field based on its type
	switch v := input.(type) {
	case string:
		if err := n.normalizeStringCommand(v, cmd); err != nil {
			return nil, fmt.Errorf("failed to normalize string command: %w", err)
		}
	case []interface{}:
		if err := n.normalizeArrayCommand(v, cmd); err != nil {
			return nil, fmt.Errorf("failed to normalize array command: %w", err)
		}
	case map[string]interface{}:
		if err := n.normalizeObjectCommand(v, cmd); err != nil {
			return nil, fmt.Errorf("failed to normalize object command: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported command format: %T", v)
	}

	// Generate name if not provided
	if cmd.Name == "" {
		cmd.Name = n.generateCommandName(cmd.Command, cmd.Args)
	}

	// Validate the normalized command
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("normalized command validation failed: %w", err)
	}

	return cmd, nil
}

// normalizeStringCommand handles string format commands like "npm run build"
func (n *Normalizer) normalizeStringCommand(input string, cmd *Command) error {
	if input == "" {
		return fmt.Errorf("command string cannot be empty")
	}

	if n.StrictMode && len(input) > 500 {
		return fmt.Errorf("command string too long (%d characters), maximum is 500", len(input))
	}

	// Split the string into command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return fmt.Errorf("command string cannot be empty after parsing")
	}

	cmd.Command = parts[0]
	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}

	return nil
}

// normalizeArrayCommand handles array format commands like ["npm", "run", "build"]
func (n *Normalizer) normalizeArrayCommand(input []interface{}, cmd *Command) error {
	if len(input) == 0 {
		return fmt.Errorf("command array cannot be empty")
	}

	if n.StrictMode && len(input) > 50 {
		return fmt.Errorf("command array too long (%d elements), maximum is 50", len(input))
	}

	// Extract command (first element)
	cmdStr, ok := input[0].(string)
	if !ok {
		return fmt.Errorf("first element of command array must be a string, got %T", input[0])
	}
	cmd.Command = cmdStr

	// Extract arguments (remaining elements)
	if len(input) > 1 {
		cmd.Args = make([]string, len(input)-1)
		for i, arg := range input[1:] {
			argStr, ok := arg.(string)
			if !ok {
				return fmt.Errorf("command array element %d must be a string, got %T", i+1, arg)
			}
			cmd.Args[i] = argStr
		}
	}

	return nil
}

// normalizeObjectCommand handles object format commands like {"command": "npm", "args": ["run", "build"]}
func (n *Normalizer) normalizeObjectCommand(input map[string]interface{}, cmd *Command) error {
	// Extract command field
	cmdInterface, hasCommand := input["command"]
	if !hasCommand {
		return fmt.Errorf("object format must have a 'command' field")
	}

	cmdStr, ok := cmdInterface.(string)
	if !ok {
		return fmt.Errorf("'command' field must be a string, got %T", cmdInterface)
	}
	cmd.Command = cmdStr

	// Extract optional args field
	if argsInterface, hasArgs := input["args"]; hasArgs {
		argsList, ok := argsInterface.([]interface{})
		if !ok {
			return fmt.Errorf("'args' field must be an array, got %T", argsInterface)
		}

		if n.StrictMode && len(argsList) > 50 {
			return fmt.Errorf("args array too long (%d elements), maximum is 50", len(argsList))
		}

		cmd.Args = make([]string, len(argsList))
		for i, arg := range argsList {
			argStr, ok := arg.(string)
			if !ok {
				return fmt.Errorf("args array element %d must be a string, got %T", i, arg)
			}
			cmd.Args[i] = argStr
		}
	}

	return nil
}

// generateCommandName creates a name for a command based on its executable and first argument
func (n *Normalizer) generateCommandName(command string, args []string) string {
	if command == "" {
		return "unnamed-command"
	}

	if len(args) > 0 {
		return fmt.Sprintf("%s-%s", command, args[0])
	}

	return command
}

// NormalizeConfig converts a configuration with mixed command formats to a unified structure
func (n *Normalizer) NormalizeConfig(input interface{}) (*Config, error) {
	if input == nil {
		return nil, fmt.Errorf("config input cannot be nil")
	}

	configMap, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config must be an object, got %T", input)
	}

	config := &Config{}

	// Extract version
	if versionInterface, hasVersion := configMap["version"]; hasVersion {
		if version, ok := versionInterface.(string); ok {
			config.Version = version
		} else {
			return nil, fmt.Errorf("version must be a string, got %T", versionInterface)
		}
	} else {
		return nil, fmt.Errorf("config must have a 'version' field")
	}

	// Extract commands
	commandsInterface, hasCommands := configMap["commands"]
	if !hasCommands {
		return nil, fmt.Errorf("config must have a 'commands' field")
	}

	commandsList, ok := commandsInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'commands' field must be an array, got %T", commandsInterface)
	}

	if len(commandsList) == 0 {
		return nil, fmt.Errorf("config must have at least one command")
	}

	config.Commands = make([]Command, len(commandsList))

	for i, cmdInterface := range commandsList {
		cmdMap, ok := cmdInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("command %d must be an object, got %T", i, cmdInterface)
		}

		// Extract command metadata
		var name string
		if nameInterface, hasName := cmdMap["name"]; hasName {
			if nameStr, ok := nameInterface.(string); ok {
				name = nameStr
			} else {
				return nil, fmt.Errorf("command %d name must be a string, got %T", i, nameInterface)
			}
		}

		var mode Mode = ModeOnce
		if modeInterface, hasMode := cmdMap["mode"]; hasMode {
			if modeStr, ok := modeInterface.(string); ok {
				mode = Mode(modeStr)
			} else {
				return nil, fmt.Errorf("command %d mode must be a string, got %T", i, modeInterface)
			}
		}

		var workDir string
		if workDirInterface, hasWorkDir := cmdMap["workDir"]; hasWorkDir {
			if workDirStr, ok := workDirInterface.(string); ok {
				workDir = workDirStr
			} else {
				return nil, fmt.Errorf("command %d workDir must be a string, got %T", i, workDirInterface)
			}
		}

		var env map[string]string
		if envInterface, hasEnv := cmdMap["env"]; hasEnv {
			if envMap, ok := envInterface.(map[string]interface{}); ok {
				env = make(map[string]string)
				for key, value := range envMap {
					if valueStr, ok := value.(string); ok {
						env[key] = valueStr
					} else {
						return nil, fmt.Errorf("command %d env value for key '%s' must be a string, got %T", i, key, value)
					}
				}
			} else {
				return nil, fmt.Errorf("command %d env must be an object, got %T", i, envInterface)
			}
		}

		// Extract and normalize the command field
		commandField, hasCommand := cmdMap["command"]
		if !hasCommand {
			return nil, fmt.Errorf("command %d must have a 'command' field", i)
		}

		// Check if this is standard format (command as string + separate args field)
		var normalizedCmd *Command
		var err error

		if cmdStr, ok := commandField.(string); ok {
			if argsInterface, hasArgs := cmdMap["args"]; hasArgs {
				// Standard format: command is string and args are separate
				if argsList, ok := argsInterface.([]interface{}); ok {
					// Convert args to string slice
					args := make([]string, len(argsList))
					for j, arg := range argsList {
						if argStr, ok := arg.(string); ok {
							args[j] = argStr
						} else {
							return nil, fmt.Errorf("command %d args element %d must be a string, got %T", i, j, arg)
						}
					}

					// Create command directly for standard format
					normalizedCmd = &Command{
						Name:    name,
						Command: cmdStr,
						Args:    args,
						Mode:    mode,
						WorkDir: workDir,
						Env:     env,
					}

					// Set default mode if not specified
					if normalizedCmd.Mode == "" {
						normalizedCmd.Mode = ModeOnce
					}

					// Generate name if not provided
					if normalizedCmd.Name == "" {
						normalizedCmd.Name = n.generateCommandName(normalizedCmd.Command, normalizedCmd.Args)
					}
				} else {
					return nil, fmt.Errorf("command %d args must be an array, got %T", i, argsInterface)
				}
			} else {
				// String format without separate args - use normalizer
				normalizedCmd, err = n.NormalizeCommand(commandField, name, mode, workDir, env)
				if err != nil {
					return nil, fmt.Errorf("failed to normalize command %d: %w", i, err)
				}
			}
		} else {
			// Array or object format - use normalizer
			normalizedCmd, err = n.NormalizeCommand(commandField, name, mode, workDir, env)
			if err != nil {
				return nil, fmt.Errorf("failed to normalize command %d: %w", i, err)
			}
		}

		config.Commands[i] = *normalizedCmd
	}

	// Validate the entire config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("normalized config validation failed: %w", err)
	}

	return config, nil
}

// NormalizeFromJSON parses JSON data and normalizes it to a unified config structure
func (n *Normalizer) NormalizeFromJSON(data []byte) (*Config, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("JSON data cannot be empty")
	}

	// Parse the JSON into a generic structure first
	var rawConfig interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Normalize the parsed data
	config, err := n.NormalizeConfig(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize config: %w", err)
	}

	return config, nil
}
