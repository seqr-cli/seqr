package config

import (
	"strings"
	"testing"
)

func TestEnhancedErrorHandling_FormatDetection(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		name           string
		commandData    interface{}
		wantErr        bool
		expectedSubstr []string
	}{
		{
			name:        "nil command with suggestion",
			commandData: nil,
			wantErr:     true,
			expectedSubstr: []string{
				"format detection failed",
				"Suggestion: Ensure the command field is not null",
				"String: \"npm start\"",
				"Array: [\"npm\", \"start\"]",
				"Object: {\"command\": \"npm\", \"args\": [\"start\"]}",
			},
		},
		{
			name:        "empty object with suggestion",
			commandData: map[string]interface{}{},
			wantErr:     true,
			expectedSubstr: []string{
				"format detection failed",
				"Suggestion: Object format requires a 'command' field",
				"Example:",
				"{\"command\": \"npm\", \"args\": [\"start\"]}",
			},
		},
		{
			name:        "empty array with suggestion",
			commandData: []interface{}{},
			wantErr:     true,
			expectedSubstr: []string{
				"format validation failed",
				"Format: Array",
				"Suggestion: Ensure all array elements are strings and the array is not empty",
				"Example:",
			},
		},
		{
			name:        "empty string with suggestion",
			commandData: "",
			wantErr:     true,
			expectedSubstr: []string{
				"format validation failed",
				"Format: String",
				"Suggestion: Ensure the command string is not empty",
				"Example:",
			},
		},
		{
			name:        "invalid type with suggestion",
			commandData: 123,
			wantErr:     true,
			expectedSubstr: []string{
				"format detection failed",
				"Suggestion: Command must be a string, array, or object",
				"Received type: int",
				"Valid formats:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := detector.DetectAndValidateCommand(tt.commandData)

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectAndValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				errMsg := err.Error()
				for _, substr := range tt.expectedSubstr {
					if !strings.Contains(errMsg, substr) {
						t.Errorf("Expected error to contain %q, but got: %s", substr, errMsg)
					}
				}
			}
		})
	}
}

func TestEnhancedErrorHandling_FormatValidation(t *testing.T) {
	tests := []struct {
		name           string
		commandData    interface{}
		wantErr        bool
		expectedSubstr []string
	}{
		{
			name:        "invalid string format with enhanced error",
			commandData: strings.Repeat("a", 501),
			wantErr:     true,
			expectedSubstr: []string{
				"format validation failed",
				"Format: String",
				"Suggestion: Ensure the command string is not empty",
				"Example: \"npm run build --production\"",
			},
		},
		{
			name:        "invalid array format with enhanced error",
			commandData: []interface{}{"npm", 123},
			wantErr:     true,
			expectedSubstr: []string{
				"format validation failed",
				"Format: Array",
				"Suggestion: Ensure all array elements are strings",
				"Example: [\"docker\", \"run\", \"-p\", \"8080:80\", \"nginx\"]",
			},
		},
		{
			name: "invalid object format with enhanced error",
			commandData: map[string]interface{}{
				"command": 123,
			},
			wantErr: true,
			expectedSubstr: []string{
				"format validation failed",
				"Format: Object",
				"Suggestion: Ensure the object has a 'command' field (string)",
				"Example: {\"command\": \"node\", \"args\": [\"server.js\", \"--port\", \"3000\"]}",
			},
		},
	}

	// Use strict detector for validation errors
	strictDetector := NewStrictFormatDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := strictDetector.DetectAndValidateCommand(tt.commandData)

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectAndValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				errMsg := err.Error()
				for _, substr := range tt.expectedSubstr {
					if !strings.Contains(errMsg, substr) {
						t.Errorf("Expected error to contain %q, but got: %s", substr, errMsg)
					}
				}
			}
		})
	}
}

func TestEnhancedErrorHandling_Normalization(t *testing.T) {
	normalizer := NewNormalizer()

	tests := []struct {
		name           string
		input          interface{}
		wantErr        bool
		expectedSubstr []string
	}{
		{
			name:    "nil input with detailed error",
			input:   nil,
			wantErr: true,
			expectedSubstr: []string{
				"command input cannot be nil",
				"Input received:",
				"Suggestion: Ensure the command field is not null",
			},
		},
		{
			name:    "empty string with detailed error",
			input:   "",
			wantErr: true,
			expectedSubstr: []string{
				"failed to normalize string command",
				"Input: \"\"",
				"Suggestion: Provide a non-empty command string",
			},
		},
		{
			name:    "empty array with detailed error",
			input:   []interface{}{},
			wantErr: true,
			expectedSubstr: []string{
				"failed to normalize array command",
				"Input: []",
				"Suggestion: Provide at least one element like [\"npm\", \"start\"]",
			},
		},
		{
			name: "object missing command field",
			input: map[string]interface{}{
				"args": []interface{}{"start"},
			},
			wantErr: true,
			expectedSubstr: []string{
				"failed to normalize object command",
				"Suggestion: Add a 'command' field like {\"command\": \"npm\", \"args\": [\"start\"]}",
			},
		},
		{
			name:    "unsupported type with detailed error",
			input:   123,
			wantErr: true,
			expectedSubstr: []string{
				"unsupported command format: int",
				"Input received: 123",
				"Suggestion: Use string",
				"array",
				"object",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizer.NormalizeCommand(tt.input, "test", ModeOnce, "", nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				errMsg := err.Error()
				for _, substr := range tt.expectedSubstr {
					if !strings.Contains(errMsg, substr) {
						t.Errorf("Expected error to contain %q, but got: %s", substr, errMsg)
					}
				}
			}
		})
	}
}

func TestEnhancedErrorHandling_ConfigNormalization(t *testing.T) {
	normalizer := NewNormalizer()

	tests := []struct {
		name           string
		input          interface{}
		wantErr        bool
		expectedSubstr []string
	}{
		{
			name:    "nil config with enhanced error",
			input:   nil,
			wantErr: true,
			expectedSubstr: []string{
				"config input cannot be nil",
				"Suggestion: Provide a valid configuration object",
			},
		},
		{
			name:    "non-object config",
			input:   "invalid",
			wantErr: true,
			expectedSubstr: []string{
				"config must be an object, got string",
				"Suggestion: Ensure the root configuration is a JSON object",
			},
		},
		{
			name: "missing version field",
			input: map[string]interface{}{
				"commands": []interface{}{},
			},
			wantErr: true,
			expectedSubstr: []string{
				"config must have a 'version' field",
				"Suggestion: Add a version field: {\"version\": \"1.0\", ...}",
			},
		},
		{
			name: "invalid version type",
			input: map[string]interface{}{
				"version":  123,
				"commands": []interface{}{},
			},
			wantErr: true,
			expectedSubstr: []string{
				"version must be a string, got int",
				"Suggestion: Set version to a string like \"1.0\"",
			},
		},
		{
			name: "missing commands field",
			input: map[string]interface{}{
				"version": "1.0",
			},
			wantErr: true,
			expectedSubstr: []string{
				"config must have a 'commands' field",
				"Suggestion: Add a commands array",
			},
		},
		{
			name: "invalid commands type",
			input: map[string]interface{}{
				"version":  "1.0",
				"commands": "invalid",
			},
			wantErr: true,
			expectedSubstr: []string{
				"'commands' field must be an array, got string",
				"Suggestion: Commands should be an array",
			},
		},
		{
			name: "empty commands array",
			input: map[string]interface{}{
				"version":  "1.0",
				"commands": []interface{}{},
			},
			wantErr: true,
			expectedSubstr: []string{
				"config must have at least one command",
				"Suggestion: Add at least one command",
			},
		},
		{
			name: "multiple errors aggregated",
			input: map[string]interface{}{
				"version": 123,
				"commands": []interface{}{
					"invalid_command",
					map[string]interface{}{
						"name":    456,
						"command": "echo",
					},
				},
			},
			wantErr: true,
			expectedSubstr: []string{
				"Multiple configuration errors found:",
				"version must be a string, got int",
				"command 0: command must be an object",
				"command 1: name must be a string, got int",
				"Suggestions:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizer.NormalizeConfig(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				errMsg := err.Error()
				for _, substr := range tt.expectedSubstr {
					if !strings.Contains(errMsg, substr) {
						t.Errorf("Expected error to contain %q, but got: %s", substr, errMsg)
					}
				}
			}
		})
	}
}

func TestEnhancedErrorHandling_ParseJSON(t *testing.T) {
	tests := []struct {
		name           string
		json           string
		wantErr        bool
		expectedSubstr []string
	}{
		{
			name:    "empty JSON with suggestion",
			json:    "",
			wantErr: true,
			expectedSubstr: []string{
				"configuration data is empty",
				"Suggestion: Provide a valid JSON configuration",
			},
		},
		{
			name:    "invalid JSON syntax",
			json:    `{"version": "1.0"`,
			wantErr: true,
			expectedSubstr: []string{
				"format detection failed",
				"JSON Preview:",
				"Suggestion: Check for missing commas, quotes, or brackets",
			},
		},
		{
			name: "missing version with suggestion",
			json: `{
				"commands": [
					{"name": "test", "command": "echo"}
				]
			}`,
			wantErr: true,
			expectedSubstr: []string{
				"invalid configuration format detected",
				"Suggestion: Ensure your configuration has a 'version' field",
			},
		},
		{
			name: "missing commands with suggestion",
			json: `{
				"version": "1.0"
			}`,
			wantErr: true,
			expectedSubstr: []string{
				"format detection failed",
				"Suggestion: Add a commands array to your configuration",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSON([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				errMsg := err.Error()
				for _, substr := range tt.expectedSubstr {
					if !strings.Contains(errMsg, substr) {
						t.Errorf("Expected error to contain %q, but got: %s", substr, errMsg)
					}
				}
			}
		})
	}
}

func TestConfigNormalizationError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ConfigNormalizationError
		want string
	}{
		{
			name: "error with command index",
			err: ConfigNormalizationError{
				Message:      "invalid field",
				CommandIndex: 2,
				Field:        "name",
				Value:        123,
				Suggestion:   "Use a string value",
			},
			want: "command 2: invalid field (field: name, value: 123)\nSuggestion: Use a string value",
		},
		{
			name: "error without command index",
			err: ConfigNormalizationError{
				Message:      "config error",
				CommandIndex: -1,
				Suggestion:   "Fix the configuration",
			},
			want: "config error\nSuggestion: Fix the configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ConfigNormalizationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
