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
		return nil, n.createDetailedError("command input cannot be nil", input, "Ensure the command field is not null")
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
			return nil, n.enhanceNormalizationError("string", err, v)
		}
	case []interface{}:
		if err := n.normalizeArrayCommand(v, cmd); err != nil {
			return nil, n.enhanceNormalizationError("array", err, v)
		}
	case map[string]interface{}:
		if err := n.normalizeObjectCommand(v, cmd); err != nil {
			return nil, n.enhanceNormalizationError("object", err, v)
		}
	default:
		return nil, n.createDetailedError(
			fmt.Sprintf("unsupported command format: %T", v),
			input,
			"Use string (\"npm start\"), array ([\"npm\", \"start\"]), or object ({\"command\": \"npm\", \"args\": [\"start\"]}) format",
		)
	}

	// Generate name if not provided
	if cmd.Name == "" {
		cmd.Name = n.generateCommandName(cmd.Command, cmd.Args)
	}

	// Validate the normalized command
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("normalized command validation failed: %w\nSuggestion: Check that the command name and executable are valid", err)
	}

	return cmd, nil
}

// createDetailedError creates an error with additional context and suggestions
func (n *Normalizer) createDetailedError(message string, input interface{}, suggestion string) error {
	return fmt.Errorf("%s\nInput received: %+v\nSuggestion: %s", message, input, suggestion)
}

// enhanceNormalizationError provides format-specific error enhancement
func (n *Normalizer) enhanceNormalizationError(format string, originalErr error, input interface{}) error {
	baseMsg := fmt.Sprintf("failed to normalize %s command: %v", format, originalErr)

	switch format {
	case "string":
		if str, ok := input.(string); ok && str == "" {
			return fmt.Errorf("%s\nInput: \"%s\"\nSuggestion: Provide a non-empty command string like \"npm start\" or \"echo hello\"", baseMsg, str)
		}
		return fmt.Errorf("%s\nInput: \"%v\"\nSuggestion: Ensure the command string contains valid executable and arguments", baseMsg, input)
	case "array":
		if arr, ok := input.([]interface{}); ok {
			if len(arr) == 0 {
				return fmt.Errorf("%s\nInput: %v\nSuggestion: Provide at least one element like [\"npm\", \"start\"]", baseMsg, arr)
			}
			return fmt.Errorf("%s\nInput: %v\nSuggestion: Ensure all array elements are strings representing the command and its arguments", baseMsg, arr)
		}
		return fmt.Errorf("%s\nSuggestion: Array format should contain strings only", baseMsg)
	case "object":
		if obj, ok := input.(map[string]interface{}); ok {
			if _, hasCmd := obj["command"]; !hasCmd {
				return fmt.Errorf("%s\nInput: %v\nSuggestion: Add a 'command' field like {\"command\": \"npm\", \"args\": [\"start\"]}", baseMsg, obj)
			}
			return fmt.Errorf("%s\nInput: %v\nSuggestion: Ensure 'command' is a string and 'args' (if present) is an array of strings", baseMsg, obj)
		}
		return fmt.Errorf("%s\nSuggestion: Object format requires 'command' field", baseMsg)
	default:
		return fmt.Errorf("%s", baseMsg)
	}
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

// ConfigNormalizationError represents errors that occur during config normalization with enhanced context
type ConfigNormalizationError struct {
	Message      string
	CommandIndex int
	Field        string
	Value        interface{}
	Suggestion   string
}

func (e ConfigNormalizationError) Error() string {
	if e.CommandIndex >= 0 {
		return fmt.Sprintf("command %d: %s (field: %s, value: %v)\nSuggestion: %s",
			e.CommandIndex, e.Message, e.Field, e.Value, e.Suggestion)
	}
	return fmt.Sprintf("%s\nSuggestion: %s", e.Message, e.Suggestion)
}

// NormalizeConfig converts a configuration with mixed command formats to a unified structure
func (n *Normalizer) NormalizeConfig(input interface{}) (*Config, error) {
	if input == nil {
		return nil, ConfigNormalizationError{
			Message:      "config input cannot be nil",
			CommandIndex: -1,
			Suggestion:   "Provide a valid configuration object with 'version' and 'commands' fields",
		}
	}

	configMap, ok := input.(map[string]interface{})
	if !ok {
		return nil, ConfigNormalizationError{
			Message:      fmt.Sprintf("config must be an object, got %T", input),
			CommandIndex: -1,
			Value:        input,
			Suggestion:   "Ensure the root configuration is a JSON object: {\"version\": \"1.0\", \"commands\": [...]}",
		}
	}

	config := &Config{}
	var errors []error

	// Extract version
	if versionInterface, hasVersion := configMap["version"]; hasVersion {
		if version, ok := versionInterface.(string); ok {
			config.Version = version
		} else {
			errors = append(errors, ConfigNormalizationError{
				Message:      fmt.Sprintf("version must be a string, got %T", versionInterface),
				CommandIndex: -1,
				Field:        "version",
				Value:        versionInterface,
				Suggestion:   "Set version to a string like \"1.0\" or \"2.1.0\"",
			})
		}
	} else {
		errors = append(errors, ConfigNormalizationError{
			Message:      "config must have a 'version' field",
			CommandIndex: -1,
			Field:        "version",
			Suggestion:   "Add a version field: {\"version\": \"1.0\", ...}",
		})
	}

	// Extract commands
	commandsInterface, hasCommands := configMap["commands"]
	if !hasCommands {
		errors = append(errors, ConfigNormalizationError{
			Message:      "config must have a 'commands' field",
			CommandIndex: -1,
			Field:        "commands",
			Suggestion:   "Add a commands array: {\"commands\": [{\"name\": \"example\", \"command\": \"echo hello\"}]}",
		})
	} else {
		commandsList, ok := commandsInterface.([]interface{})
		if !ok {
			errors = append(errors, ConfigNormalizationError{
				Message:      fmt.Sprintf("'commands' field must be an array, got %T", commandsInterface),
				CommandIndex: -1,
				Field:        "commands",
				Value:        commandsInterface,
				Suggestion:   "Commands should be an array: \"commands\": [{...}, {...}]",
			})
		} else if len(commandsList) == 0 {
			errors = append(errors, ConfigNormalizationError{
				Message:      "config must have at least one command",
				CommandIndex: -1,
				Field:        "commands",
				Value:        commandsList,
				Suggestion:   "Add at least one command: \"commands\": [{\"name\": \"example\", \"command\": \"echo hello\"}]",
			})
		} else {
			// Process commands
			config.Commands = make([]Command, len(commandsList))

			for i, cmdInterface := range commandsList {
				if err := n.normalizeConfigCommand(cmdInterface, i, &config.Commands[i]); err != nil {
					errors = append(errors, err)
				}
			}
		}
	}

	// Return aggregated errors if any
	if len(errors) > 0 {
		return nil, n.aggregateConfigErrors(errors)
	}

	// Validate the entire config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("normalized config validation failed: %w\nSuggestion: Review the validation errors above and ensure all commands have valid names, executables, and modes", err)
	}

	return config, nil
}

// normalizeConfigCommand handles normalization of a single command within the config
func (n *Normalizer) normalizeConfigCommand(cmdInterface interface{}, index int, result *Command) error {
	cmdMap, ok := cmdInterface.(map[string]interface{})
	if !ok {
		return ConfigNormalizationError{
			Message:      fmt.Sprintf("command must be an object, got %T", cmdInterface),
			CommandIndex: index,
			Value:        cmdInterface,
			Suggestion:   "Each command should be an object: {\"name\": \"example\", \"command\": \"echo hello\"}",
		}
	}

	// Extract command metadata with error handling
	name, err := n.extractStringField(cmdMap, "name", index, true)
	if err != nil {
		return err
	}

	mode, err := n.extractModeField(cmdMap, "mode", index)
	if err != nil {
		return err
	}

	workDir, err := n.extractStringField(cmdMap, "workDir", index, true)
	if err != nil {
		return err
	}

	env, err := n.extractEnvField(cmdMap, "env", index)
	if err != nil {
		return err
	}

	// Extract and normalize the command field
	commandField, hasCommand := cmdMap["command"]
	if !hasCommand {
		return ConfigNormalizationError{
			Message:      "must have a 'command' field",
			CommandIndex: index,
			Field:        "command",
			Suggestion:   "Add a command field with string, array, or object format",
		}
	}

	// Check if this is standard format (command as string + separate args field)
	var normalizedCmd *Command
	var normErr error

	if cmdStr, ok := commandField.(string); ok {
		if argsInterface, hasArgs := cmdMap["args"]; hasArgs {
			// Standard format: command is string and args are separate
			args, err := n.extractArgsField(argsInterface, index)
			if err != nil {
				return err
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
			// String format without separate args - use normalizer
			normalizedCmd, normErr = n.NormalizeCommand(commandField, name, mode, workDir, env)
		}
	} else {
		// Array or object format - use normalizer
		normalizedCmd, normErr = n.NormalizeCommand(commandField, name, mode, workDir, env)
	}

	if normErr != nil {
		return ConfigNormalizationError{
			Message:      fmt.Sprintf("failed to normalize command: %v", normErr),
			CommandIndex: index,
			Field:        "command",
			Value:        commandField,
			Suggestion:   "Check the command format and ensure it follows one of the supported patterns",
		}
	}

	*result = *normalizedCmd
	return nil
}

// Helper methods for field extraction with enhanced error handling
func (n *Normalizer) extractStringField(cmdMap map[string]interface{}, fieldName string, index int, optional bool) (string, error) {
	if fieldInterface, hasField := cmdMap[fieldName]; hasField {
		if fieldStr, ok := fieldInterface.(string); ok {
			return fieldStr, nil
		}
		return "", ConfigNormalizationError{
			Message:      fmt.Sprintf("%s must be a string, got %T", fieldName, fieldInterface),
			CommandIndex: index,
			Field:        fieldName,
			Value:        fieldInterface,
			Suggestion:   fmt.Sprintf("Set %s to a string value", fieldName),
		}
	}
	return "", nil
}

func (n *Normalizer) extractModeField(cmdMap map[string]interface{}, fieldName string, index int) (Mode, error) {
	if modeInterface, hasMode := cmdMap[fieldName]; hasMode {
		if modeStr, ok := modeInterface.(string); ok {
			mode := Mode(modeStr)
			if mode != ModeOnce && mode != ModeKeepAlive {
				return "", ConfigNormalizationError{
					Message:      fmt.Sprintf("invalid mode value: %s", modeStr),
					CommandIndex: index,
					Field:        fieldName,
					Value:        modeInterface,
					Suggestion:   "Mode must be either \"once\" or \"keepAlive\"",
				}
			}
			return mode, nil
		}
		return "", ConfigNormalizationError{
			Message:      fmt.Sprintf("mode must be a string, got %T", modeInterface),
			CommandIndex: index,
			Field:        fieldName,
			Value:        modeInterface,
			Suggestion:   "Set mode to \"once\" or \"keepAlive\"",
		}
	}
	return ModeOnce, nil
}

func (n *Normalizer) extractEnvField(cmdMap map[string]interface{}, fieldName string, index int) (map[string]string, error) {
	if envInterface, hasEnv := cmdMap[fieldName]; hasEnv {
		if envMap, ok := envInterface.(map[string]interface{}); ok {
			env := make(map[string]string)
			for key, value := range envMap {
				if valueStr, ok := value.(string); ok {
					env[key] = valueStr
				} else {
					return nil, ConfigNormalizationError{
						Message:      fmt.Sprintf("env value for key '%s' must be a string, got %T", key, value),
						CommandIndex: index,
						Field:        fmt.Sprintf("env.%s", key),
						Value:        value,
						Suggestion:   "All environment variable values must be strings",
					}
				}
			}
			return env, nil
		}
		return nil, ConfigNormalizationError{
			Message:      fmt.Sprintf("env must be an object, got %T", envInterface),
			CommandIndex: index,
			Field:        fieldName,
			Value:        envInterface,
			Suggestion:   "Environment variables should be an object: \"env\": {\"KEY\": \"value\"}",
		}
	}
	return nil, nil
}

func (n *Normalizer) extractArgsField(argsInterface interface{}, index int) ([]string, error) {
	if argsList, ok := argsInterface.([]interface{}); ok {
		args := make([]string, len(argsList))
		for j, arg := range argsList {
			if argStr, ok := arg.(string); ok {
				args[j] = argStr
			} else {
				return nil, ConfigNormalizationError{
					Message:      fmt.Sprintf("args element %d must be a string, got %T", j, arg),
					CommandIndex: index,
					Field:        fmt.Sprintf("args[%d]", j),
					Value:        arg,
					Suggestion:   "All arguments must be strings",
				}
			}
		}
		return args, nil
	}
	return nil, ConfigNormalizationError{
		Message:      fmt.Sprintf("args must be an array, got %T", argsInterface),
		CommandIndex: index,
		Field:        "args",
		Value:        argsInterface,
		Suggestion:   "Arguments should be an array of strings: \"args\": [\"--port\", \"3000\"]",
	}
}

// aggregateConfigErrors combines multiple configuration errors into a single comprehensive error
func (n *Normalizer) aggregateConfigErrors(errors []error) error {
	if len(errors) == 1 {
		return errors[0]
	}

	var messages []string
	var suggestions []string

	for _, err := range errors {
		messages = append(messages, err.Error())

		if configErr, ok := err.(ConfigNormalizationError); ok && configErr.Suggestion != "" {
			suggestions = append(suggestions, configErr.Suggestion)
		}
	}

	errorMsg := fmt.Sprintf("Multiple configuration errors found:\n%s", strings.Join(messages, "\n"))

	if len(suggestions) > 0 {
		// Remove duplicate suggestions
		uniqueSuggestions := make(map[string]bool)
		var finalSuggestions []string
		for _, suggestion := range suggestions {
			if !uniqueSuggestions[suggestion] {
				uniqueSuggestions[suggestion] = true
				finalSuggestions = append(finalSuggestions, suggestion)
			}
		}
		errorMsg += fmt.Sprintf("\n\nSuggestions:\n- %s", strings.Join(finalSuggestions, "\n- "))
	}

	return fmt.Errorf("%s", errorMsg)
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
