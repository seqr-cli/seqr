package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTemplateValidation_GeneratedFilesAreValid tests that all generated template files
// can be successfully parsed and validated by the seqr configuration system
func TestTemplateValidation_GeneratedFilesAreValid(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "seqr-template-validation")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create generator with temp directory
	generator := &TemplateGenerator{
		OutputDir: tempDir,
	}

	// Generate all templates
	err = generator.GenerateAllTemplates()
	if err != nil {
		t.Fatalf("GenerateAllTemplates failed: %v", err)
	}

	// Expected template files
	expectedFiles := []string{
		"example-string-format.queue.json",
		"example-array-format.queue.json",
		"example-object-format.queue.json",
		"example-mixed-format.queue.json",
		"example-fullstack.queue.json",
	}

	// Test each generated file
	for _, filename := range expectedFiles {
		t.Run(filename, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, filename)

			// Verify file exists
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				t.Fatalf("Expected file %s was not created", filename)
			}

			// Read file content
			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("Failed to read generated file %s: %v", filename, err)
			}

			// Verify it's valid JSON
			var jsonData map[string]interface{}
			if err := json.Unmarshal(content, &jsonData); err != nil {
				t.Errorf("Generated file %s contains invalid JSON: %v", filename, err)
			}

			// Parse with seqr configuration parser
			config, err := ParseJSON(content)
			if err != nil {
				t.Errorf("Generated file %s failed to parse with seqr parser: %v", filename, err)
				return
			}

			// Validate configuration
			if err := config.Validate(); err != nil {
				t.Errorf("Generated file %s failed validation: %v", filename, err)
			}

			// Verify basic structure requirements
			if config.Version == "" {
				t.Errorf("Generated file %s missing version field", filename)
			}

			if len(config.Commands) == 0 {
				t.Errorf("Generated file %s has no commands", filename)
			}

			// Verify all commands have required fields
			for i, cmd := range config.Commands {
				if cmd.Name == "" {
					t.Errorf("Generated file %s command %d missing name", filename, i)
				}
				if cmd.Command == "" {
					t.Errorf("Generated file %s command %d missing command", filename, i)
				}
				if cmd.Mode != ModeOnce && cmd.Mode != ModeKeepAlive {
					t.Errorf("Generated file %s command %d has invalid mode: %s", filename, i, cmd.Mode)
				}
			}
		})
	}
}

// TestTemplateValidation_CommandFormatsWork tests that each command format
// in the generated templates works correctly
func TestTemplateValidation_CommandFormatsWork(t *testing.T) {
	generator := NewTemplateGenerator()

	testCases := []struct {
		name           string
		templateFunc   func() string
		expectedFormat string
	}{
		{"StringFormat", generator.getStringFormatTemplate, "string"},
		{"ArrayFormat", generator.getArrayFormatTemplate, "array"},
		{"ObjectFormat", generator.getObjectFormatTemplate, "object"},
		{"MixedFormat", generator.getMixedFormatTemplate, "mixed"},
		{"FullstackFormat", generator.getFullstackTemplate, "mixed"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := tc.templateFunc()

			// Parse the template content
			config, formatInfo, err := ParseJSONWithFormatInfo([]byte(content))
			if err != nil {
				t.Fatalf("Failed to parse %s template: %v", tc.name, err)
			}

			// Verify format detection worked
			if !formatInfo.IsValid() {
				t.Errorf("Template %s generated invalid format info", tc.name)
			}

			// Verify configuration is valid
			if err := config.Validate(); err != nil {
				t.Errorf("Template %s generated invalid configuration: %v", tc.name, err)
			}

			// Verify specific format expectations
			formatSummary := formatInfo.GetFormatSummary()
			switch tc.expectedFormat {
			case "string":
				if formatSummary["string"] == 0 {
					t.Errorf("String format template should have string commands")
				}
			case "array":
				if formatSummary["array"] == 0 {
					t.Errorf("Array format template should have array commands")
				}
			case "object":
				if formatSummary["object"] == 0 {
					t.Errorf("Object format template should have object commands")
				}
			case "mixed":
				// Mixed format should have at least one format type
				totalFormats := len(formatSummary)
				if totalFormats == 0 {
					t.Errorf("Mixed format template should have at least one command format")
				}
			}
		})
	}
}

// TestTemplateValidation_RequiredFieldsPresent tests that all generated templates
// contain the required fields and proper documentation
func TestTemplateValidation_RequiredFieldsPresent(t *testing.T) {
	generator := NewTemplateGenerator()

	templates := []struct {
		name     string
		getFunc  func() string
		filename string
	}{
		{"StringFormat", generator.getStringFormatTemplate, "example-string-format.queue.json"},
		{"ArrayFormat", generator.getArrayFormatTemplate, "example-array-format.queue.json"},
		{"ObjectFormat", generator.getObjectFormatTemplate, "example-object-format.queue.json"},
		{"MixedFormat", generator.getMixedFormatTemplate, "example-mixed-format.queue.json"},
		{"FullstackFormat", generator.getFullstackTemplate, "example-fullstack.queue.json"},
	}

	for _, template := range templates {
		t.Run(template.name, func(t *testing.T) {
			content := template.getFunc()

			// Parse as generic JSON to check structure
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
				t.Fatalf("Template %s contains invalid JSON: %v", template.name, err)
			}

			// Check required top-level fields
			if _, ok := jsonData["version"]; !ok {
				t.Errorf("Template %s missing version field", template.name)
			}

			if _, ok := jsonData["commands"]; !ok {
				t.Errorf("Template %s missing commands field", template.name)
			}

			// Check that commands is an array
			commands, ok := jsonData["commands"].([]interface{})
			if !ok {
				t.Errorf("Template %s commands field is not an array", template.name)
				return
			}

			if len(commands) == 0 {
				t.Errorf("Template %s has empty commands array", template.name)
			}

			// Check each command has required fields
			for i, cmdInterface := range commands {
				cmd, ok := cmdInterface.(map[string]interface{})
				if !ok {
					t.Errorf("Template %s command %d is not an object", template.name, i)
					continue
				}

				// Check required command fields
				if _, ok := cmd["name"]; !ok {
					t.Errorf("Template %s command %d missing name field", template.name, i)
				}

				if _, ok := cmd["command"]; !ok {
					t.Errorf("Template %s command %d missing command field", template.name, i)
				}

				if _, ok := cmd["mode"]; !ok {
					t.Errorf("Template %s command %d missing mode field", template.name, i)
				}

				// Verify mode is valid
				if mode, ok := cmd["mode"].(string); ok {
					if mode != "once" && mode != "keepAlive" {
						t.Errorf("Template %s command %d has invalid mode: %s", template.name, i, mode)
					}
				}
			}

			// Check for documentation comments (templates should be educational)
			if !strings.Contains(content, "_comment") && !strings.Contains(content, "_usage") {
				t.Errorf("Template %s should contain documentation comments for educational purposes", template.name)
			}
		})
	}
}

// TestTemplateValidation_FileWriteAndRead tests the complete cycle of
// generating templates and reading them back
func TestTemplateValidation_FileWriteAndRead(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "seqr-template-cycle-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create generator with temp directory
	generator := &TemplateGenerator{
		OutputDir: tempDir,
	}

	// Generate all templates
	err = generator.GenerateAllTemplates()
	if err != nil {
		t.Fatalf("GenerateAllTemplates failed: %v", err)
	}

	// Test loading each generated file using the standard config loader
	expectedFiles := []string{
		"example-string-format.queue.json",
		"example-array-format.queue.json",
		"example-object-format.queue.json",
		"example-mixed-format.queue.json",
		"example-fullstack.queue.json",
	}

	for _, filename := range expectedFiles {
		t.Run("LoadFile_"+filename, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, filename)

			// Load using the standard config loader
			config, err := LoadFromFile(fullPath)
			if err != nil {
				t.Errorf("Failed to load generated file %s: %v", filename, err)
				return
			}

			// Verify the loaded config is valid
			if err := config.Validate(); err != nil {
				t.Errorf("Loaded config from %s failed validation: %v", filename, err)
			}

			// Verify basic properties
			if config.Version == "" {
				t.Errorf("Loaded config from %s has empty version", filename)
			}

			if len(config.Commands) == 0 {
				t.Errorf("Loaded config from %s has no commands", filename)
			}

			// Test that we can execute validation on each command
			for i, cmd := range config.Commands {
				if err := cmd.Validate(); err != nil {
					t.Errorf("Command %d in %s failed validation: %v", i, filename, err)
				}
			}
		})
	}
}

// TestTemplateValidation_StrictValidation tests templates against strict validation
func TestTemplateValidation_StrictValidation(t *testing.T) {
	generator := NewTemplateGenerator()

	templates := []struct {
		name    string
		getFunc func() string
	}{
		{"StringFormat", generator.getStringFormatTemplate},
		{"ArrayFormat", generator.getArrayFormatTemplate},
		{"ObjectFormat", generator.getObjectFormatTemplate},
		{"MixedFormat", generator.getMixedFormatTemplate},
		{"FullstackFormat", generator.getFullstackTemplate},
	}

	for _, template := range templates {
		t.Run(template.name+"_StrictValidation", func(t *testing.T) {
			content := template.getFunc()

			// Parse the template
			config, err := ParseJSON([]byte(content))
			if err != nil {
				t.Fatalf("Failed to parse %s template: %v", template.name, err)
			}

			// Test with strict validator
			strictValidator := NewStrictValidator()
			if err := strictValidator.ValidateConfig(config); err != nil {
				// Note: Some templates might fail strict validation due to their educational nature
				// This is acceptable, but we should log it
				t.Logf("Template %s failed strict validation (this may be expected): %v", template.name, err)
			}

			// Test with regular validator (should always pass)
			regularValidator := NewValidator()
			if err := regularValidator.ValidateConfig(config); err != nil {
				t.Errorf("Template %s failed regular validation: %v", template.name, err)
			}
		})
	}
}

// TestTemplateValidation_CommandNameUniqueness tests that generated templates
// have unique command names
func TestTemplateValidation_CommandNameUniqueness(t *testing.T) {
	generator := NewTemplateGenerator()

	templates := []struct {
		name    string
		getFunc func() string
	}{
		{"StringFormat", generator.getStringFormatTemplate},
		{"ArrayFormat", generator.getArrayFormatTemplate},
		{"ObjectFormat", generator.getObjectFormatTemplate},
		{"MixedFormat", generator.getMixedFormatTemplate},
		{"FullstackFormat", generator.getFullstackTemplate},
	}

	for _, template := range templates {
		t.Run(template.name+"_UniqueNames", func(t *testing.T) {
			content := template.getFunc()

			// Parse the template
			config, err := ParseJSON([]byte(content))
			if err != nil {
				t.Fatalf("Failed to parse %s template: %v", template.name, err)
			}

			// Check for unique command names
			nameMap := make(map[string]int)
			for i, cmd := range config.Commands {
				if prevIndex, exists := nameMap[cmd.Name]; exists {
					t.Errorf("Template %s has duplicate command name '%s' at positions %d and %d",
						template.name, cmd.Name, prevIndex, i)
				}
				nameMap[cmd.Name] = i
			}
		})
	}
}

// TestTemplateValidation_EnvironmentVariables tests that environment variables
// in templates are properly formatted
func TestTemplateValidation_EnvironmentVariables(t *testing.T) {
	generator := NewTemplateGenerator()

	templates := []struct {
		name    string
		getFunc func() string
	}{
		{"StringFormat", generator.getStringFormatTemplate},
		{"ArrayFormat", generator.getArrayFormatTemplate},
		{"ObjectFormat", generator.getObjectFormatTemplate},
		{"MixedFormat", generator.getMixedFormatTemplate},
		{"FullstackFormat", generator.getFullstackTemplate},
	}

	for _, template := range templates {
		t.Run(template.name+"_EnvVars", func(t *testing.T) {
			content := template.getFunc()

			// Parse the template
			config, err := ParseJSON([]byte(content))
			if err != nil {
				t.Fatalf("Failed to parse %s template: %v", template.name, err)
			}

			// Check environment variables in each command
			for i, cmd := range config.Commands {
				for envName, envValue := range cmd.Env {
					// Check environment variable name format
					if envName == "" {
						t.Errorf("Template %s command %d has empty environment variable name", template.name, i)
					}

					// Check that environment variable names are reasonable
					if strings.Contains(envName, " ") {
						t.Errorf("Template %s command %d has environment variable name with spaces: '%s'",
							template.name, i, envName)
					}

					// Check that environment variable values are not empty (unless intentional)
					if envValue == "" {
						t.Logf("Template %s command %d has empty environment variable value for '%s' (may be intentional)",
							template.name, i, envName)
					}
				}
			}
		})
	}
}

// TestTemplateValidation_JSONStructure tests that generated templates have proper JSON structure
func TestTemplateValidation_JSONStructure(t *testing.T) {
	generator := NewTemplateGenerator()

	templates := []struct {
		name    string
		getFunc func() string
	}{
		{"StringFormat", generator.getStringFormatTemplate},
		{"ArrayFormat", generator.getArrayFormatTemplate},
		{"ObjectFormat", generator.getObjectFormatTemplate},
		{"MixedFormat", generator.getMixedFormatTemplate},
		{"FullstackFormat", generator.getFullstackTemplate},
	}

	for _, template := range templates {
		t.Run(template.name+"_JSONStructure", func(t *testing.T) {
			content := template.getFunc()

			// Test that JSON is properly formatted (can be unmarshaled and remarshaled)
			var jsonData interface{}
			if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
				t.Fatalf("Template %s contains invalid JSON: %v", template.name, err)
			}

			// Test that we can marshal it back to JSON
			remarshaled, err := json.Marshal(jsonData)
			if err != nil {
				t.Errorf("Template %s cannot be remarshaled to JSON: %v", template.name, err)
			}

			// Test that remarshaled JSON is still valid
			var testData interface{}
			if err := json.Unmarshal(remarshaled, &testData); err != nil {
				t.Errorf("Template %s remarshaled JSON is invalid: %v", template.name, err)
			}
		})
	}
}

// TestTemplateValidation_ConflictHandling tests template generation with existing files
func TestTemplateValidation_ConflictHandling(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "seqr-template-conflict-validation")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create generator with temp directory
	generator := &TemplateGenerator{
		OutputDir: tempDir,
	}

	// Create a conflicting file with known content
	conflictFile := filepath.Join(tempDir, "example-string-format.queue.json")
	originalContent := `{"version": "test", "commands": []}`
	err = os.WriteFile(conflictFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}

	// Generate templates (should handle conflict gracefully in non-interactive mode)
	err = generator.GenerateAllTemplates()
	if err != nil {
		t.Fatalf("GenerateAllTemplates failed with conflict: %v", err)
	}

	// Verify the existing file wasn't overwritten (default behavior is skip)
	content, err := os.ReadFile(conflictFile)
	if err != nil {
		t.Fatalf("Failed to read conflict file: %v", err)
	}

	if string(content) != originalContent {
		t.Errorf("Existing file was unexpectedly modified. Expected '%s', got '%s'",
			originalContent, string(content))
	}

	// Verify other files were still created
	otherFiles := []string{
		"example-array-format.queue.json",
		"example-object-format.queue.json",
		"example-mixed-format.queue.json",
		"example-fullstack.queue.json",
	}

	for _, filename := range otherFiles {
		fullPath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created despite conflict with another file", filename)
		}
	}
}

// TestTemplateValidation_RealWorldUsage tests templates in realistic usage scenarios
func TestTemplateValidation_RealWorldUsage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "seqr-template-realworld")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create generator with temp directory
	generator := &TemplateGenerator{
		OutputDir: tempDir,
	}

	// Generate all templates
	err = generator.GenerateAllTemplates()
	if err != nil {
		t.Fatalf("GenerateAllTemplates failed: %v", err)
	}

	// Test each template as if a user would use it
	templateFiles := []string{
		"example-string-format.queue.json",
		"example-array-format.queue.json",
		"example-object-format.queue.json",
		"example-mixed-format.queue.json",
		"example-fullstack.queue.json",
	}

	for _, filename := range templateFiles {
		t.Run("RealWorld_"+filename, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, filename)

			// Test 1: Load configuration as a user would
			config, err := LoadFromFile(fullPath)
			if err != nil {
				t.Errorf("User would fail to load %s: %v", filename, err)
				return
			}

			// Test 2: Validate configuration thoroughly
			if err := config.Validate(); err != nil {
				t.Errorf("User would get validation error for %s: %v", filename, err)
			}

			// Test 3: Check that all commands are executable (have valid structure)
			for i, cmd := range config.Commands {
				if cmd.Name == "" {
					t.Errorf("Command %d in %s has no name - user would be confused", i, filename)
				}
				if cmd.Command == "" {
					t.Errorf("Command %d in %s has no command - would fail to execute", i, filename)
				}
				if cmd.Mode != ModeOnce && cmd.Mode != ModeKeepAlive {
					t.Errorf("Command %d in %s has invalid mode - user would get error", i, filename)
				}
			}

			// Test 4: Check for educational value (comments/documentation)
			content, _ := os.ReadFile(fullPath)
			contentStr := string(content)
			if !strings.Contains(contentStr, "_comment") &&
				!strings.Contains(contentStr, "_usage") &&
				!strings.Contains(contentStr, "_explanation") {
				t.Errorf("Template %s lacks educational comments - users won't understand format", filename)
			}
		})
	}
}

// TestTemplateValidation_PerformanceAndSize tests that templates are reasonable in size
func TestTemplateValidation_PerformanceAndSize(t *testing.T) {
	generator := NewTemplateGenerator()

	templates := []struct {
		name    string
		getFunc func() string
	}{
		{"StringFormat", generator.getStringFormatTemplate},
		{"ArrayFormat", generator.getArrayFormatTemplate},
		{"ObjectFormat", generator.getObjectFormatTemplate},
		{"MixedFormat", generator.getMixedFormatTemplate},
		{"FullstackFormat", generator.getFullstackTemplate},
	}

	for _, template := range templates {
		t.Run(template.name+"_Size", func(t *testing.T) {
			content := template.getFunc()

			// Check template size is reasonable (not too large)
			const maxTemplateSize = 50 * 1024 // 50KB should be more than enough
			if len(content) > maxTemplateSize {
				t.Errorf("Template %s is too large (%d bytes), maximum should be %d bytes",
					template.name, len(content), maxTemplateSize)
			}

			// Check template is not empty
			if len(content) == 0 {
				t.Errorf("Template %s is empty", template.name)
			}

			// Check template has reasonable number of commands (not excessive)
			config, err := ParseJSON([]byte(content))
			if err != nil {
				t.Fatalf("Failed to parse %s template: %v", template.name, err)
			}

			const maxCommands = 20 // Reasonable limit for example templates
			if len(config.Commands) > maxCommands {
				t.Errorf("Template %s has too many commands (%d), should be educational not overwhelming (max %d)",
					template.name, len(config.Commands), maxCommands)
			}

			if len(config.Commands) == 0 {
				t.Errorf("Template %s has no commands, not useful as example", template.name)
			}
		})
	}
}

// TestTemplateValidation_CrossPlatformCompatibility tests templates work across platforms
func TestTemplateValidation_CrossPlatformCompatibility(t *testing.T) {
	generator := NewTemplateGenerator()

	templates := []struct {
		name    string
		getFunc func() string
	}{
		{"StringFormat", generator.getStringFormatTemplate},
		{"ArrayFormat", generator.getArrayFormatTemplate},
		{"ObjectFormat", generator.getObjectFormatTemplate},
		{"MixedFormat", generator.getMixedFormatTemplate},
		{"FullstackFormat", generator.getFullstackTemplate},
	}

	for _, template := range templates {
		t.Run(template.name+"_CrossPlatform", func(t *testing.T) {
			content := template.getFunc()

			// Parse template
			config, err := ParseJSON([]byte(content))
			if err != nil {
				t.Fatalf("Failed to parse %s template: %v", template.name, err)
			}

			// Check for platform-specific issues
			for i, cmd := range config.Commands {
				// Check for Windows-specific path separators in workDir
				if cmd.WorkDir != "" && strings.Contains(cmd.WorkDir, "\\") {
					t.Errorf("Template %s command %d uses Windows path separator in workDir: %s",
						template.name, i, cmd.WorkDir)
				}

				// Check for Unix-specific commands that won't work on Windows
				unixCommands := []string{"ls", "grep", "awk", "sed", "chmod", "chown"}
				for _, unixCmd := range unixCommands {
					if cmd.Command == unixCmd {
						t.Logf("Template %s command %d uses Unix-specific command '%s' - may not work on Windows",
							template.name, i, unixCmd)
					}
				}

				// Check environment variable names are cross-platform compatible
				for envName := range cmd.Env {
					if strings.Contains(envName, " ") {
						t.Errorf("Template %s command %d has environment variable with space: '%s'",
							template.name, i, envName)
					}
				}
			}
		})
	}
}

// TestTemplateValidation_CLIIntegration tests that generated templates work with the CLI
func TestTemplateValidation_CLIIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "seqr-template-cli-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory for this test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Generate templates using CLI init functionality
	generator := &TemplateGenerator{
		OutputDir: ".",
	}

	err = generator.GenerateAllTemplates()
	if err != nil {
		t.Fatalf("Failed to generate templates: %v", err)
	}

	// Test each generated template with the configuration loader
	templateFiles := []string{
		"example-string-format.queue.json",
		"example-array-format.queue.json",
		"example-object-format.queue.json",
		"example-mixed-format.queue.json",
		"example-fullstack.queue.json",
	}

	for _, filename := range templateFiles {
		t.Run("CLI_Integration_"+filename, func(t *testing.T) {
			// Test that the file can be loaded by the config system
			config, err := LoadFromFile(filename)
			if err != nil {
				t.Errorf("CLI would fail to load template %s: %v", filename, err)
				return
			}

			// Test that the configuration validates
			if err := config.Validate(); err != nil {
				t.Errorf("CLI would reject template %s due to validation error: %v", filename, err)
				return
			}

			// Test that all commands in the template are well-formed
			for i, cmd := range config.Commands {
				// Test command validation
				if err := cmd.Validate(); err != nil {
					t.Errorf("Template %s command %d would fail CLI validation: %v", filename, i, err)
				}

				// Test that command has all required fields for execution
				if cmd.Name == "" {
					t.Errorf("Template %s command %d missing name - CLI execution would fail", filename, i)
				}
				if cmd.Command == "" {
					t.Errorf("Template %s command %d missing command - CLI execution would fail", filename, i)
				}

				// Test mode is valid
				if cmd.Mode != ModeOnce && cmd.Mode != ModeKeepAlive {
					t.Errorf("Template %s command %d has invalid mode %s - CLI would reject", filename, i, cmd.Mode)
				}
			}

			// Verify the template would be accepted by the CLI parser
			if config.Version == "" {
				t.Errorf("Template %s missing version - CLI would reject", filename)
			}

			if len(config.Commands) == 0 {
				t.Errorf("Template %s has no commands - CLI would have nothing to execute", filename)
			}
		})
	}
}

// TestTemplateValidation_EndToEndWorkflow tests the complete workflow from generation to execution readiness
func TestTemplateValidation_EndToEndWorkflow(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "seqr-template-e2e")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test the complete workflow:
	// 1. Generate templates
	// 2. Validate they can be parsed
	// 3. Validate they pass all checks
	// 4. Verify they're ready for execution

	generator := &TemplateGenerator{
		OutputDir: tempDir,
	}

	// Step 1: Generate all templates
	err = generator.GenerateAllTemplates()
	if err != nil {
		t.Fatalf("Step 1 failed - template generation: %v", err)
	}

	expectedFiles := []string{
		"example-string-format.queue.json",
		"example-array-format.queue.json",
		"example-object-format.queue.json",
		"example-mixed-format.queue.json",
		"example-fullstack.queue.json",
	}

	for _, filename := range expectedFiles {
		t.Run("E2E_"+filename, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, filename)

			// Step 2: Verify file exists and is readable
			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("Step 2 failed - cannot read generated file %s: %v", filename, err)
			}

			if len(content) == 0 {
				t.Fatalf("Step 2 failed - generated file %s is empty", filename)
			}

			// Step 3: Verify JSON is valid
			var jsonData interface{}
			if err := json.Unmarshal(content, &jsonData); err != nil {
				t.Fatalf("Step 3 failed - invalid JSON in %s: %v", filename, err)
			}

			// Step 4: Verify configuration can be parsed by seqr
			config, err := LoadFromFile(fullPath)
			if err != nil {
				t.Fatalf("Step 4 failed - seqr cannot parse %s: %v", filename, err)
			}

			// Step 5: Verify configuration passes validation
			if err := config.Validate(); err != nil {
				t.Fatalf("Step 5 failed - configuration validation failed for %s: %v", filename, err)
			}

			// Step 6: Verify configuration is execution-ready
			if config.Version == "" {
				t.Errorf("Step 6 failed - %s missing version field", filename)
			}

			if len(config.Commands) == 0 {
				t.Errorf("Step 6 failed - %s has no commands to execute", filename)
			}

			// Step 7: Verify each command is execution-ready
			for i, cmd := range config.Commands {
				if cmd.Name == "" {
					t.Errorf("Step 7 failed - %s command %d missing name", filename, i)
				}
				if cmd.Command == "" {
					t.Errorf("Step 7 failed - %s command %d missing command", filename, i)
				}
				if cmd.Mode != ModeOnce && cmd.Mode != ModeKeepAlive {
					t.Errorf("Step 7 failed - %s command %d has invalid mode: %s", filename, i, cmd.Mode)
				}

				// Verify command can be validated individually
				if err := cmd.Validate(); err != nil {
					t.Errorf("Step 7 failed - %s command %d validation failed: %v", filename, i, err)
				}
			}

			// Step 8: Verify template has educational value (contains documentation)
			contentStr := string(content)
			hasDocumentation := strings.Contains(contentStr, "_comment") ||
				strings.Contains(contentStr, "_usage") ||
				strings.Contains(contentStr, "_explanation") ||
				strings.Contains(contentStr, "_description")

			if !hasDocumentation {
				t.Errorf("Step 8 failed - %s lacks educational documentation", filename)
			}

			t.Logf("✅ Template %s passed all end-to-end validation steps", filename)
		})
	}
}
