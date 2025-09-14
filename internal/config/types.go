package config

import (
	"encoding/json"
	"fmt"
)

type Mode string

const (
	ModeOnce      Mode = "once"
	ModeKeepAlive Mode = "keepAlive"
)

type Command struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Mode    Mode              `json:"mode"`
	WorkDir string            `json:"workDir,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
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
