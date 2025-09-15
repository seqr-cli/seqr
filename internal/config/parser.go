package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadFromFile loads and parses a configuration file
func LoadFromFile(filename string) (*Config, error) {
	if filename == "" {
		return nil, fmt.Errorf("config filename cannot be empty")
	}

	cleanPath := filepath.Clean(filename)

	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file '%s' does not exist", cleanPath)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading config file '%s'", cleanPath)
		}
		return nil, fmt.Errorf("failed to access config file '%s': %w", cleanPath, err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("'%s' is a directory, not a file", cleanPath)
	}

	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("config file '%s' is empty", cleanPath)
	}

	const maxFileSize = 1024 * 1024
	if fileInfo.Size() > maxFileSize {
		return nil, fmt.Errorf("config file '%s' is too large (%d bytes), maximum allowed is %d bytes",
			cleanPath, fileInfo.Size(), maxFileSize)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", cleanPath, err)
	}

	config, err := ParseJSON(data)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file '%s': %w", cleanPath, err)
	}

	return config, nil
}

// ParseJSON parses JSON data into a Config struct, supporting multiple command formats
func ParseJSON(data []byte) (*Config, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("configuration data is empty\nSuggestion: Provide a valid JSON configuration with 'version' and 'commands' fields")
	}

	// First, detect and validate formats before parsing
	detector := NewFormatDetector()
	formatInfo, err := detector.DetectConfigFormat(data)
	if err != nil {
		return nil, enhanceParseError("format detection failed", err, data)
	}

	if !formatInfo.IsValid() {
		return nil, fmt.Errorf("invalid configuration format detected\nSuggestion: Ensure your configuration has a 'version' field and at least one command in the 'commands' array")
	}

	// Use the normalizer to handle all format variations
	normalizer := NewNormalizer()
	config, err := normalizer.NormalizeFromJSON(data)
	if err != nil {
		return nil, enhanceParseError("normalization failed", err, data)
	}

	return config, nil
}

// enhanceParseError provides enhanced error messages for parsing failures
func enhanceParseError(context string, originalErr error, data []byte) error {
	baseMsg := fmt.Sprintf("%s: %v", context, originalErr)

	// Try to provide context about the JSON structure
	var preview string
	if len(data) > 200 {
		preview = string(data[:200]) + "..."
	} else {
		preview = string(data)
	}

	// Check for common JSON syntax errors
	if strings.Contains(originalErr.Error(), "invalid character") ||
		strings.Contains(originalErr.Error(), "unexpected end") ||
		strings.Contains(originalErr.Error(), "failed to parse JSON") {
		return fmt.Errorf("%s\nJSON Preview: %s\nSuggestion: Check for missing commas, quotes, or brackets in your JSON configuration", baseMsg, preview)
	}

	// Check for missing required fields
	if strings.Contains(originalErr.Error(), "must have a 'version' field") {
		return fmt.Errorf("%s\nSuggestion: Add a version field to your configuration:\n{\n  \"version\": \"1.0\",\n  \"commands\": [...]\n}", baseMsg)
	}

	if strings.Contains(originalErr.Error(), "must have a 'commands' field") {
		return fmt.Errorf("%s\nSuggestion: Add a commands array to your configuration:\n{\n  \"version\": \"1.0\",\n  \"commands\": [\n    {\"name\": \"example\", \"command\": \"echo hello\"}\n  ]\n}", baseMsg)
	}

	return fmt.Errorf("%s\nJSON Preview: %s", baseMsg, preview)
}

// ParseJSONWithFormatInfo parses JSON data and returns both the config and format information
func ParseJSONWithFormatInfo(data []byte) (*Config, *ConfigFormatInfo, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("configuration data is empty")
	}

	// Detect and validate formats
	detector := NewFormatDetector()
	formatInfo, err := detector.DetectConfigFormat(data)
	if err != nil {
		return nil, nil, fmt.Errorf("format detection failed: %w", err)
	}

	if !formatInfo.IsValid() {
		return nil, nil, fmt.Errorf("invalid configuration format detected")
	}

	// Use the normalizer to handle all format variations
	normalizer := NewNormalizer()
	config, err := normalizer.NormalizeFromJSON(data)
	if err != nil {
		return nil, nil, fmt.Errorf("normalization failed: %w", err)
	}

	return config, formatInfo, nil
}

func DefaultConfigFile() string {
	return ".queue.json"
}

func FileExists(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	cleanPath := filepath.Clean(filename)

	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file '%s' does not exist", cleanPath)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing file '%s'", cleanPath)
		}
		return fmt.Errorf("cannot access file '%s': %w", cleanPath, err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a file", cleanPath)
	}

	return nil
}
