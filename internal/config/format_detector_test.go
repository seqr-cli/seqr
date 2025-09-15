package config

import (
	"testing"
)

func TestCommandFormat_String(t *testing.T) {
	tests := []struct {
		format CommandFormat
		want   string
	}{
		{FormatString, "string"},
		{FormatArray, "array"},
		{FormatObject, "object"},
		{FormatStandard, "standard"},
		{FormatUnknown, "unknown"},
		{CommandFormat(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("CommandFormat.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDetector_DetectCommandFormat(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		name        string
		commandData interface{}
		wantFormat  CommandFormat
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "string format",
			commandData: "npm start",
			wantFormat:  FormatString,
			wantErr:     false,
		},
		{
			name:        "array format",
			commandData: []interface{}{"npm", "start"},
			wantFormat:  FormatArray,
			wantErr:     false,
		},
		{
			name: "object format with command field",
			commandData: map[string]interface{}{
				"command": "npm",
				"args":    []interface{}{"start"},
			},
			wantFormat: FormatObject,
			wantErr:    false,
		},
		{
			name: "object format without command field",
			commandData: map[string]interface{}{
				"args": []interface{}{"start"},
			},
			wantFormat: FormatUnknown,
			wantErr:    true,
			errSubstr:  "must have a 'command' field",
		},
		{
			name:        "nil command data",
			commandData: nil,
			wantFormat:  FormatUnknown,
			wantErr:     true,
			errSubstr:   "cannot be nil",
		},
		{
			name:        "unsupported type",
			commandData: 123,
			wantFormat:  FormatUnknown,
			wantErr:     true,
			errSubstr:   "unsupported command format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := detector.DetectCommandFormat(tt.commandData)

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectCommandFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if format != tt.wantFormat {
				t.Errorf("DetectCommandFormat() format = %v, want %v", format, tt.wantFormat)
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errSubstr) {
					t.Errorf("DetectCommandFormat() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestFormatDetector_ValidateStringFormat(t *testing.T) {
	tests := []struct {
		name        string
		detector    *FormatDetector
		commandData interface{}
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "valid string",
			detector:    NewFormatDetector(),
			commandData: "npm start",
			wantErr:     false,
		},
		{
			name:        "empty string",
			detector:    NewFormatDetector(),
			commandData: "",
			wantErr:     true,
			errSubstr:   "cannot be empty",
		},
		{
			name:        "non-string type",
			detector:    NewFormatDetector(),
			commandData: 123,
			wantErr:     true,
			errSubstr:   "expected string",
		},
		{
			name:        "too long string in strict mode",
			detector:    NewStrictFormatDetector(),
			commandData: string(make([]byte, 501)),
			wantErr:     true,
			errSubstr:   "too long",
		},
		{
			name:        "valid string in strict mode",
			detector:    NewStrictFormatDetector(),
			commandData: "npm start",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.detector.validateStringFormat(tt.commandData)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateStringFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateStringFormat() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestFormatDetector_ValidateArrayFormat(t *testing.T) {
	tests := []struct {
		name        string
		detector    *FormatDetector
		commandData interface{}
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "valid array",
			detector:    NewFormatDetector(),
			commandData: []interface{}{"npm", "start"},
			wantErr:     false,
		},
		{
			name:        "empty array",
			detector:    NewFormatDetector(),
			commandData: []interface{}{},
			wantErr:     true,
			errSubstr:   "cannot be empty",
		},
		{
			name:        "non-array type",
			detector:    NewFormatDetector(),
			commandData: "not an array",
			wantErr:     true,
			errSubstr:   "expected array",
		},
		{
			name:        "non-string first element",
			detector:    NewFormatDetector(),
			commandData: []interface{}{123, "start"},
			wantErr:     true,
			errSubstr:   "first element of command array must be a string",
		},
		{
			name:        "non-string argument",
			detector:    NewFormatDetector(),
			commandData: []interface{}{"npm", 123},
			wantErr:     true,
			errSubstr:   "command array element 1 must be a string",
		},
		{
			name:     "too many elements in strict mode",
			detector: NewStrictFormatDetector(),
			commandData: func() []interface{} {
				arr := make([]interface{}, 51)
				for i := range arr {
					arr[i] = "arg"
				}
				return arr
			}(),
			wantErr:   true,
			errSubstr: "too many elements",
		},
		{
			name:        "valid array in strict mode",
			detector:    NewStrictFormatDetector(),
			commandData: []interface{}{"npm", "start"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.detector.validateArrayFormat(tt.commandData)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateArrayFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateArrayFormat() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestFormatDetector_ValidateObjectFormat(t *testing.T) {
	tests := []struct {
		name        string
		detector    *FormatDetector
		commandData interface{}
		wantErr     bool
		errSubstr   string
	}{
		{
			name:     "valid object with command only",
			detector: NewFormatDetector(),
			commandData: map[string]interface{}{
				"command": "npm",
			},
			wantErr: false,
		},
		{
			name:     "valid object with command and args",
			detector: NewFormatDetector(),
			commandData: map[string]interface{}{
				"command": "npm",
				"args":    []interface{}{"start"},
			},
			wantErr: false,
		},
		{
			name:        "non-object type",
			detector:    NewFormatDetector(),
			commandData: "not an object",
			wantErr:     true,
			errSubstr:   "expected object",
		},
		{
			name:     "missing command field",
			detector: NewFormatDetector(),
			commandData: map[string]interface{}{
				"args": []interface{}{"start"},
			},
			wantErr:   true,
			errSubstr: "must have a 'command' field",
		},
		{
			name:     "non-string command field",
			detector: NewFormatDetector(),
			commandData: map[string]interface{}{
				"command": 123,
			},
			wantErr:   true,
			errSubstr: "'command' field must be a string",
		},
		{
			name:     "non-array args field",
			detector: NewFormatDetector(),
			commandData: map[string]interface{}{
				"command": "npm",
				"args":    "not an array",
			},
			wantErr:   true,
			errSubstr: "'args' field must be an array",
		},
		{
			name:     "non-string args element",
			detector: NewFormatDetector(),
			commandData: map[string]interface{}{
				"command": "npm",
				"args":    []interface{}{"start", 123},
			},
			wantErr:   true,
			errSubstr: "args array element 1 must be a string",
		},
		{
			name:     "too many args in strict mode",
			detector: NewStrictFormatDetector(),
			commandData: map[string]interface{}{
				"command": "npm",
				"args": func() []interface{} {
					arr := make([]interface{}, 51)
					for i := range arr {
						arr[i] = "arg"
					}
					return arr
				}(),
			},
			wantErr:   true,
			errSubstr: "too many elements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.detector.validateObjectFormat(tt.commandData)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateObjectFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateObjectFormat() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestFormatDetector_DetectAndValidateCommand(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		name        string
		commandData interface{}
		wantFormat  CommandFormat
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "valid string command",
			commandData: "npm start",
			wantFormat:  FormatString,
			wantErr:     false,
		},
		{
			name:        "valid array command",
			commandData: []interface{}{"npm", "start"},
			wantFormat:  FormatArray,
			wantErr:     false,
		},
		{
			name: "valid object command",
			commandData: map[string]interface{}{
				"command": "npm",
				"args":    []interface{}{"start"},
			},
			wantFormat: FormatObject,
			wantErr:    false,
		},
		{
			name:        "invalid string command",
			commandData: "",
			wantFormat:  FormatString,
			wantErr:     true,
			errSubstr:   "format validation failed",
		},
		{
			name:        "invalid array command",
			commandData: []interface{}{},
			wantFormat:  FormatArray,
			wantErr:     true,
			errSubstr:   "format validation failed",
		},
		{
			name: "invalid object command",
			commandData: map[string]interface{}{
				"args": []interface{}{"start"},
			},
			wantFormat: FormatUnknown,
			wantErr:    true,
			errSubstr:  "format detection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := detector.DetectAndValidateCommand(tt.commandData)

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectAndValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if format != tt.wantFormat {
				t.Errorf("DetectAndValidateCommand() format = %v, want %v", format, tt.wantFormat)
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errSubstr) {
					t.Errorf("DetectAndValidateCommand() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestFormatDetector_DetectConfigFormat(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		name       string
		configJSON string
		wantErr    bool
		errSubstr  string
		validate   func(*testing.T, *ConfigFormatInfo)
	}{
		{
			name: "mixed formats configuration",
			configJSON: `{
				"version": "1.0",
				"commands": [
					{
						"name": "string-cmd",
						"command": "npm start"
					},
					{
						"name": "array-cmd",
						"command": ["node", "server.js"]
					},
					{
						"name": "object-cmd",
						"command": {
							"command": "docker",
							"args": ["run", "nginx"]
						}
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, info *ConfigFormatInfo) {
				if info.Version != "1.0" {
					t.Errorf("Expected version '1.0', got '%s'", info.Version)
				}
				if len(info.CommandFormats) != 3 {
					t.Errorf("Expected 3 commands, got %d", len(info.CommandFormats))
				}
				if !info.MixedFormats {
					t.Error("Expected mixed formats to be true")
				}

				expectedFormats := []CommandFormat{FormatString, FormatArray, FormatObject}
				for i, expected := range expectedFormats {
					if info.CommandFormats[i].Format != expected {
						t.Errorf("Command %d: expected format %s, got %s",
							i, expected.String(), info.CommandFormats[i].Format.String())
					}
				}
			},
		},
		{
			name: "single format configuration",
			configJSON: `{
				"version": "1.0",
				"commands": [
					{
						"name": "cmd1",
						"command": "npm start"
					},
					{
						"name": "cmd2", 
						"command": "npm build"
					}
				]
			}`,
			wantErr: false,
			validate: func(t *testing.T, info *ConfigFormatInfo) {
				if info.MixedFormats {
					t.Error("Expected mixed formats to be false")
				}
				for _, cmdInfo := range info.CommandFormats {
					if cmdInfo.Format != FormatString {
						t.Errorf("Expected all commands to be string format, got %s", cmdInfo.Format.String())
					}
				}
			},
		},
		{
			name:       "invalid JSON",
			configJSON: `{"version": "1.0"`,
			wantErr:    true,
			errSubstr:  "failed to parse JSON",
		},
		{
			name: "missing commands field",
			configJSON: `{
				"version": "1.0"
			}`,
			wantErr:   true,
			errSubstr: "must have a 'commands' field",
		},
		{
			name: "non-array commands field",
			configJSON: `{
				"version": "1.0",
				"commands": "not an array"
			}`,
			wantErr:   true,
			errSubstr: "'commands' field must be an array",
		},
		{
			name: "invalid command object",
			configJSON: `{
				"version": "1.0",
				"commands": ["not an object"]
			}`,
			wantErr:   true,
			errSubstr: "command 0 must be an object",
		},
		{
			name: "command missing command field",
			configJSON: `{
				"version": "1.0",
				"commands": [
					{
						"name": "test"
					}
				]
			}`,
			wantErr:   true,
			errSubstr: "missing 'command' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := detector.DetectConfigFormat([]byte(tt.configJSON))

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectConfigFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errSubstr) {
					t.Errorf("DetectConfigFormat() error = %v, expected to contain %q", err, tt.errSubstr)
				}
			}

			if !tt.wantErr {
				if info == nil {
					t.Error("DetectConfigFormat() returned nil info without error")
				} else if tt.validate != nil {
					tt.validate(t, info)
				}
			}
		})
	}
}

func TestConfigFormatInfo_GetFormatSummary(t *testing.T) {
	info := &ConfigFormatInfo{
		CommandFormats: []CommandFormatInfo{
			{Format: FormatString},
			{Format: FormatString},
			{Format: FormatArray},
			{Format: FormatObject},
		},
	}

	summary := info.GetFormatSummary()

	expected := map[string]int{
		"string": 2,
		"array":  1,
		"object": 1,
	}

	for format, count := range expected {
		if summary[format] != count {
			t.Errorf("Expected %d %s commands, got %d", count, format, summary[format])
		}
	}
}

func TestConfigFormatInfo_IsValid(t *testing.T) {
	tests := []struct {
		name string
		info *ConfigFormatInfo
		want bool
	}{
		{
			name: "valid info",
			info: &ConfigFormatInfo{
				Version: "1.0",
				CommandFormats: []CommandFormatInfo{
					{Format: FormatString},
				},
			},
			want: true,
		},
		{
			name: "missing version",
			info: &ConfigFormatInfo{
				CommandFormats: []CommandFormatInfo{
					{Format: FormatString},
				},
			},
			want: false,
		},
		{
			name: "no commands",
			info: &ConfigFormatInfo{
				Version:        "1.0",
				CommandFormats: []CommandFormatInfo{},
			},
			want: false,
		},
		{
			name: "nil info",
			info: &ConfigFormatInfo{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.IsValid(); got != tt.want {
				t.Errorf("ConfigFormatInfo.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFormatDetector(t *testing.T) {
	detector := NewFormatDetector()
	if detector.StrictValidation {
		t.Error("NewFormatDetector() should not enable strict validation by default")
	}
}

func TestNewStrictFormatDetector(t *testing.T) {
	detector := NewStrictFormatDetector()
	if !detector.StrictValidation {
		t.Error("NewStrictFormatDetector() should enable strict validation")
	}
}
