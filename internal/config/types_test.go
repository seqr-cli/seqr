package config

import (
	"encoding/json"
	"testing"
)

func TestCommand_Validate(t *testing.T) {
	tests := []struct {
		name    string
		command Command
		wantErr bool
	}{
		{
			name: "valid once command",
			command: Command{
				Name:    "test",
				Command: "echo",
				Args:    []string{"hello"},
				Mode:    ModeOnce,
			},
			wantErr: false,
		},
		{
			name: "valid keepAlive command",
			command: Command{
				Name:    "server",
				Command: "npm",
				Args:    []string{"start"},
				Mode:    ModeKeepAlive,
				WorkDir: "./app",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			command: Command{
				Command: "echo",
				Mode:    ModeOnce,
			},
			wantErr: true,
		},
		{
			name: "missing command",
			command: Command{
				Name: "test",
				Mode: ModeOnce,
			},
			wantErr: true,
		},
		{
			name: "invalid mode",
			command: Command{
				Name:    "test",
				Command: "echo",
				Mode:    Mode("invalid"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.command.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Command.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
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
			name: "missing version",
			config: Config{
				Commands: []Command{
					{
						Name:    "test",
						Command: "echo",
						Mode:    ModeOnce,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no commands",
			config: Config{
				Version:  "1.0",
				Commands: []Command{},
			},
			wantErr: true,
		},
		{
			name: "invalid command",
			config: Config{
				Version: "1.0",
				Commands: []Command{
					{
						Name: "test",
						Mode: ModeOnce,
						// missing Command field
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMode_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    Mode
		wantErr bool
	}{
		{
			name: "valid once mode",
			json: `"once"`,
			want: ModeOnce,
		},
		{
			name: "valid keepAlive mode",
			json: `"keepAlive"`,
			want: ModeKeepAlive,
		},
		{
			name:    "invalid mode",
			json:    `"invalid"`,
			wantErr: true,
		},
		{
			name:    "non-string value",
			json:    `123`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mode Mode
			err := json.Unmarshal([]byte(tt.json), &mode)

			if (err != nil) != tt.wantErr {
				t.Errorf("Mode.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && mode != tt.want {
				t.Errorf("Mode.UnmarshalJSON() = %v, want %v", mode, tt.want)
			}
		})
	}
}
