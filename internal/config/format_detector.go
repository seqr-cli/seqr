package config

import (
	"encoding/json"
	"fmt"
)

// CommandFormat represents the different supported command formats
type CommandFormat int

const (
	FormatUnknown  CommandFormat = iota
	FormatString                 // "npm start"
	FormatArray                  // ["npm", "start"]
	FormatObject                 // {"command": "npm", "args": ["start"]}
	FormatStandard               // {"command": "npm", "args": ["start"]} with separate fields
)

// String returns the string representation of the command format
func (f CommandFormat) String() string {
	switch f {
	case FormatString:
		return "string"
	case FormatArray:
		return "array"
	case FormatObject:
		return "object"
	case FormatStandard:
		return "standard"
	default:
		return "unknown"
	}
}

// FormatDetector provides functionality to detect and validate command formats
type FormatDetector struct {
	StrictValidation bool
}

// NewFormatDetector creates a new format detector
func NewFormatDetector() *FormatDetector {
	return &FormatDetector{
		StrictValidation: false,
	}
}

// NewStrictFormatDetector creates a new format detector with strict validation
func NewStrictFormatDetector() *FormatDetector {
	return &FormatDetector{
		StrictValidation: true,
	}
}

// DetectCommandFormat detects the format of a command field from raw JSON data
func (fd *FormatDetector) DetectCommandFormat(commandData interface{}) (CommandFormat, error) {
	if commandData == nil {
		return FormatUnknown, fmt.Errorf("command data cannot be nil")
	}

	switch v := commandData.(type) {
	case string:
		return FormatString, nil
	case []interface{}:
		return FormatArray, nil
	case map[string]interface{}:
		// Check if it's a standard format (has separate command and args fields)
		if _, hasCommand := v["command"]; hasCommand {
			return FormatObject, nil
		}
		return FormatUnknown, fmt.Errorf("object format must have a 'command' field")
	default:
		return FormatUnknown, fmt.Errorf("unsupported command format: %T", v)
	}
}

// ValidateCommandFormat validates a command based on its detected format
func (fd *FormatDetector) ValidateCommandFormat(commandData interface{}, format CommandFormat) error {
	switch format {
	case FormatString:
		return fd.validateStringFormat(commandData)
	case FormatArray:
		return fd.validateArrayFormat(commandData)
	case FormatObject:
		return fd.validateObjectFormat(commandData)
	default:
		return fmt.Errorf("cannot validate unknown format: %s", format)
	}
}

// validateStringFormat validates string command format
func (fd *FormatDetector) validateStringFormat(commandData interface{}) error {
	str, ok := commandData.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", commandData)
	}

	if str == "" {
		return fmt.Errorf("string command cannot be empty")
	}

	if fd.StrictValidation {
		// Additional strict validation for string format
		if len(str) > 500 {
			return fmt.Errorf("string command too long (%d characters), maximum is 500", len(str))
		}
	}

	return nil
}

// validateArrayFormat validates array command format
func (fd *FormatDetector) validateArrayFormat(commandData interface{}) error {
	arr, ok := commandData.([]interface{})
	if !ok {
		return fmt.Errorf("expected array, got %T", commandData)
	}

	if len(arr) == 0 {
		return fmt.Errorf("array command cannot be empty")
	}

	// Validate first element (command)
	if _, ok := arr[0].(string); !ok {
		return fmt.Errorf("first element of command array must be a string, got %T", arr[0])
	}

	// Validate remaining elements (arguments)
	for i, arg := range arr[1:] {
		if _, ok := arg.(string); !ok {
			return fmt.Errorf("command array element %d must be a string, got %T", i+1, arg)
		}
	}

	if fd.StrictValidation {
		// Additional strict validation for array format
		if len(arr) > 50 {
			return fmt.Errorf("array command has too many elements (%d), maximum is 50", len(arr))
		}
	}

	return nil
}

// validateObjectFormat validates object command format
func (fd *FormatDetector) validateObjectFormat(commandData interface{}) error {
	obj, ok := commandData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected object, got %T", commandData)
	}

	// Validate required 'command' field
	cmdField, hasCommand := obj["command"]
	if !hasCommand {
		return fmt.Errorf("object format must have a 'command' field")
	}

	if _, ok := cmdField.(string); !ok {
		return fmt.Errorf("'command' field must be a string, got %T", cmdField)
	}

	// Validate optional 'args' field
	if argsField, hasArgs := obj["args"]; hasArgs {
		argsList, ok := argsField.([]interface{})
		if !ok {
			return fmt.Errorf("'args' field must be an array, got %T", argsField)
		}

		for i, arg := range argsList {
			if _, ok := arg.(string); !ok {
				return fmt.Errorf("args array element %d must be a string, got %T", i, arg)
			}
		}

		if fd.StrictValidation && len(argsList) > 50 {
			return fmt.Errorf("args array has too many elements (%d), maximum is 50", len(argsList))
		}
	}

	return nil
}

// DetectAndValidateCommand detects and validates a command format in one step
func (fd *FormatDetector) DetectAndValidateCommand(commandData interface{}) (CommandFormat, error) {
	format, err := fd.DetectCommandFormat(commandData)
	if err != nil {
		return FormatUnknown, fd.enhanceFormatDetectionError(err, commandData)
	}

	if err := fd.ValidateCommandFormat(commandData, format); err != nil {
		return format, fd.enhanceFormatValidationError(err, format, commandData)
	}

	return format, nil
}

// enhanceFormatDetectionError provides more helpful error messages for format detection failures
func (fd *FormatDetector) enhanceFormatDetectionError(originalErr error, commandData interface{}) error {
	baseMsg := fmt.Sprintf("format detection failed: %v", originalErr)

	switch commandData.(type) {
	case nil:
		return fmt.Errorf("%s\nSuggestion: Ensure the command field is not null. Use one of these formats:\n  - String: \"npm start\"\n  - Array: [\"npm\", \"start\"]\n  - Object: {\"command\": \"npm\", \"args\": [\"start\"]}", baseMsg)
	case map[string]interface{}:
		return fmt.Errorf("%s\nSuggestion: Object format requires a 'command' field. Example:\n  {\"command\": \"npm\", \"args\": [\"start\"]}", baseMsg)
	case []interface{}:
		if arr := commandData.([]interface{}); len(arr) == 0 {
			return fmt.Errorf("%s\nSuggestion: Array format cannot be empty. Example:\n  [\"npm\", \"start\"]", baseMsg)
		}
		return fmt.Errorf("%s\nSuggestion: Array format requires all elements to be strings. Example:\n  [\"npm\", \"start\", \"--port\", \"3000\"]", baseMsg)
	case string:
		if str := commandData.(string); str == "" {
			return fmt.Errorf("%s\nSuggestion: String format cannot be empty. Example:\n  \"npm start\"", baseMsg)
		}
		return fmt.Errorf("%s\nSuggestion: Check for invalid characters in command string", baseMsg)
	default:
		return fmt.Errorf("%s\nSuggestion: Command must be a string, array, or object. Received type: %T\nValid formats:\n  - String: \"npm start\"\n  - Array: [\"npm\", \"start\"]\n  - Object: {\"command\": \"npm\", \"args\": [\"start\"]}", baseMsg, commandData)
	}
}

// enhanceFormatValidationError provides more helpful error messages for format validation failures
func (fd *FormatDetector) enhanceFormatValidationError(originalErr error, format CommandFormat, commandData interface{}) error {
	baseMsg := fmt.Sprintf("format validation failed: %v", originalErr)

	switch format {
	case FormatString:
		return fmt.Errorf("%s\nFormat: String\nSuggestion: Ensure the command string is not empty and contains valid characters.\nExample: \"npm run build --production\"", baseMsg)
	case FormatArray:
		return fmt.Errorf("%s\nFormat: Array\nSuggestion: Ensure all array elements are strings and the array is not empty.\nExample: [\"docker\", \"run\", \"-p\", \"8080:80\", \"nginx\"]", baseMsg)
	case FormatObject:
		return fmt.Errorf("%s\nFormat: Object\nSuggestion: Ensure the object has a 'command' field (string) and optional 'args' field (array of strings).\nExample: {\"command\": \"node\", \"args\": [\"server.js\", \"--port\", \"3000\"]}", baseMsg)
	default:
		return fmt.Errorf("%s\nUnknown format detected. This should not happen.", baseMsg)
	}
}

// DetectConfigFormat analyzes an entire configuration and returns format information
func (fd *FormatDetector) DetectConfigFormat(configData []byte) (*ConfigFormatInfo, error) {
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(configData, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	info := &ConfigFormatInfo{
		Version:        "",
		CommandFormats: make([]CommandFormatInfo, 0),
		MixedFormats:   false,
	}

	// Extract version
	if version, ok := rawConfig["version"].(string); ok {
		info.Version = version
	}

	// Extract commands
	commandsInterface, ok := rawConfig["commands"]
	if !ok {
		return nil, fmt.Errorf("configuration must have a 'commands' field")
	}

	commands, ok := commandsInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'commands' field must be an array")
	}

	formatCounts := make(map[CommandFormat]int)

	for i, cmdInterface := range commands {
		cmd, ok := cmdInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("command %d must be an object", i)
		}

		cmdInfo := CommandFormatInfo{
			Index: i,
		}

		// Extract command name if present
		if name, ok := cmd["name"].(string); ok {
			cmdInfo.Name = name
		}

		// Detect command format
		if commandField, hasCommand := cmd["command"]; hasCommand {
			format, err := fd.DetectAndValidateCommand(commandField)
			if err != nil {
				return nil, fmt.Errorf("command %d format error: %w", i, err)
			}
			cmdInfo.Format = format
			formatCounts[format]++
		} else {
			return nil, fmt.Errorf("command %d missing 'command' field", i)
		}

		info.CommandFormats = append(info.CommandFormats, cmdInfo)
	}

	// Determine if mixed formats are used
	info.MixedFormats = len(formatCounts) > 1

	return info, nil
}

// ConfigFormatInfo contains information about the formats used in a configuration
type ConfigFormatInfo struct {
	Version        string              `json:"version"`
	CommandFormats []CommandFormatInfo `json:"commandFormats"`
	MixedFormats   bool                `json:"mixedFormats"`
}

// CommandFormatInfo contains information about a single command's format
type CommandFormatInfo struct {
	Index  int           `json:"index"`
	Name   string        `json:"name,omitempty"`
	Format CommandFormat `json:"format"`
}

// GetFormatSummary returns a summary of formats used in the configuration
func (cfi *ConfigFormatInfo) GetFormatSummary() map[string]int {
	summary := make(map[string]int)
	for _, cmdInfo := range cfi.CommandFormats {
		formatName := cmdInfo.Format.String()
		summary[formatName]++
	}
	return summary
}

// IsValid returns true if the configuration format information is valid
func (cfi *ConfigFormatInfo) IsValid() bool {
	return cfi.Version != "" && len(cfi.CommandFormats) > 0
}
