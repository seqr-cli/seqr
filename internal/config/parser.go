package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		return nil, fmt.Errorf("configuration data is empty")
	}

	var config Config

	if err := json.Unmarshal(data, &config); err != nil {
		switch err := err.(type) {
		case *json.SyntaxError:
			return nil, fmt.Errorf("JSON syntax error at byte offset %d: %w", err.Offset, err)
		case *json.UnmarshalTypeError:
			return nil, fmt.Errorf("JSON type error: cannot unmarshal %s into field '%s' of type %s",
				err.Value, err.Field, err.Type)
		default:
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
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
