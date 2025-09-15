package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateGenerator_GenerateAllTemplates(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "seqr-templates-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create generator with temp directory
	generator := &TemplateGenerator{
		OutputDir: tempDir,
	}

	// Generate templates
	err = generator.GenerateAllTemplates()
	if err != nil {
		t.Fatalf("GenerateAllTemplates failed: %v", err)
	}

	// Expected files
	expectedFiles := []string{
		"example-string-format.queue.json",
		"example-array-format.queue.json",
		"example-object-format.queue.json",
		"example-mixed-format.queue.json",
		"example-fullstack.queue.json",
	}

	// Verify all files were created
	for _, filename := range expectedFiles {
		fullPath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", filename)
		}
	}
}

func TestTemplateGenerator_ValidJSONGeneration(t *testing.T) {
	generator := NewTemplateGenerator()

	testCases := []struct {
		name     string
		template func() string
	}{
		{"StringFormat", generator.getStringFormatTemplate},
		{"ArrayFormat", generator.getArrayFormatTemplate},
		{"ObjectFormat", generator.getObjectFormatTemplate},
		{"MixedFormat", generator.getMixedFormatTemplate},
		{"FullstackFormat", generator.getFullstackTemplate},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := tc.template()

			// Verify it's valid JSON
			var config map[string]interface{}
			if err := json.Unmarshal([]byte(content), &config); err != nil {
				t.Errorf("Template %s generated invalid JSON: %v", tc.name, err)
			}

			// Verify required fields
			if version, ok := config["version"].(string); !ok || version == "" {
				t.Errorf("Template %s missing or invalid version field", tc.name)
			}

			if commands, ok := config["commands"].([]interface{}); !ok || len(commands) == 0 {
				t.Errorf("Template %s missing or empty commands array", tc.name)
			}
		})
	}
}

func TestTemplateGenerator_ConfigurationParsing(t *testing.T) {
	generator := NewTemplateGenerator()

	testCases := []struct {
		name     string
		template func() string
	}{
		{"StringFormat", generator.getStringFormatTemplate},
		{"ArrayFormat", generator.getArrayFormatTemplate},
		{"ObjectFormat", generator.getObjectFormatTemplate},
		{"MixedFormat", generator.getMixedFormatTemplate},
		{"FullstackFormat", generator.getFullstackTemplate},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := tc.template()

			// Try to parse with our config parser
			config, err := ParseJSON([]byte(content))
			if err != nil {
				t.Errorf("Template %s failed to parse with seqr parser: %v", tc.name, err)
			}

			// Verify config is valid
			if err := config.Validate(); err != nil {
				t.Errorf("Template %s generated invalid config: %v", tc.name, err)
			}

			// Verify all commands have required fields
			for i, cmd := range config.Commands {
				if cmd.Name == "" {
					t.Errorf("Template %s command %d missing name", tc.name, i)
				}
				if cmd.Command == "" {
					t.Errorf("Template %s command %d missing command", tc.name, i)
				}
				if cmd.Mode != ModeOnce && cmd.Mode != ModeKeepAlive {
					t.Errorf("Template %s command %d has invalid mode: %s", tc.name, i, cmd.Mode)
				}
			}
		})
	}
}

func TestTemplateGenerator_FileConflictHandling(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "seqr-conflict-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create generator with temp directory
	generator := &TemplateGenerator{
		OutputDir: tempDir,
	}

	// Create a conflicting file
	conflictFile := filepath.Join(tempDir, "example-string-format.queue.json")
	err = os.WriteFile(conflictFile, []byte("existing content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}

	// Generate templates (should handle conflict gracefully)
	err = generator.GenerateAllTemplates()
	if err != nil {
		t.Fatalf("GenerateAllTemplates failed with conflict: %v", err)
	}

	// Verify the existing file wasn't overwritten
	content, err := os.ReadFile(conflictFile)
	if err != nil {
		t.Fatalf("Failed to read conflict file: %v", err)
	}

	if string(content) != "existing content" {
		t.Errorf("Existing file was overwritten, expected 'existing content', got '%s'", string(content))
	}
}

func TestTemplateGenerator_CommandFormats(t *testing.T) {
	generator := NewTemplateGenerator()

	// Test string format template
	stringTemplate := generator.getStringFormatTemplate()
	if !strings.Contains(stringTemplate, `"command": "npm install"`) {
		t.Error("String format template should contain string commands")
	}

	// Test array format template
	arrayTemplate := generator.getArrayFormatTemplate()
	if !strings.Contains(arrayTemplate, `"docker",`) {
		t.Error("Array format template should contain array commands")
	}

	// Test object format template
	objectTemplate := generator.getObjectFormatTemplate()
	if !strings.Contains(objectTemplate, `"command": "docker"`) {
		t.Error("Object format template should contain object commands with nested structure")
	}

	// Test mixed format template contains multiple formats
	mixedTemplate := generator.getMixedFormatTemplate()
	hasString := strings.Contains(mixedTemplate, `"echo Hello from string format!"`)
	hasArray := strings.Contains(mixedTemplate, `"ls",`)
	hasObject := strings.Contains(mixedTemplate, `"args":`)

	if !hasString || !hasArray || !hasObject {
		t.Error("Mixed format template should contain all three command formats")
	}
}

func TestNewTemplateGenerator(t *testing.T) {
	generator := NewTemplateGenerator()

	if generator == nil {
		t.Error("NewTemplateGenerator should return a non-nil generator")
	}

	if generator.OutputDir != "." {
		t.Errorf("Expected default OutputDir to be '.', got '%s'", generator.OutputDir)
	}
}
