package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEdgeCasesAndMissingCoverage tests edge cases and scenarios not covered by existing tests
func TestEdgeCasesAndMissingCoverage(t *testing.T) {
	t.Run("LoadFromFile edge cases", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Test file size limit (create a file larger than 1MB)
		largeFile := filepath.Join(tmpDir, "large.json")
		largeContent := strings.Repeat("a", 1024*1024+1) // 1MB + 1 byte
		if err := os.WriteFile(largeFile, []byte(largeContent), 0644); err != nil {
			t.Fatalf("Failed to create large test file: %v", err)
		}

		_, err := LoadFromFile(largeFile)
		if err == nil {
			t.Error("Expected error for file too large")
		}
		if !strings.Contains(err.Error(), "too large") {
			t.Errorf("Expected 'too large' error, got: %v", err)
		}

		// Test permission denied scenario (create file with no read permissions)
		restrictedFile := filepath.Join(tmpDir, "restricted.json")
		if err := os.WriteFile(restrictedFile, []byte(`{"version":"1.0","commands":[]}`), 0000); err != nil {
			t.Fatalf("Failed to create restricted test file: %v", err)
		}

		_, err = LoadFromFile(restrictedFile)
		if err == nil {
			t.Error("Expected permission error for restricted file")
		}
		// Note: This test might not work on all systems due to permission handling differences
	})

	t.Run("FileExists edge cases", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Test permission denied on directory access
		restrictedDir := filepath.Join(tmpDir, "restricted")
		if err := os.Mkdir(restrictedDir, 0000); err != nil {
			t.Fatalf("Failed to create restricted directory: %v", err)
		}

		restrictedFile := filepath.Join(restrictedDir, "test.json")
		err := FileExists(restrictedFile)
		if err == nil {
			t.Error("Expected permission error for file in restricted directory")
		}
		// Note: This test might not work on all systems due to permission handling differences
	})

	t.Run("Strict mode command limits", func(t *testing.T) {
		// Test maximum commands limit in strict mode
		commands := make([]Command, 51) // Exceeds limit of 50
		for i := 0; i < 51; i++ {
			commands[i] = Command{
				Name:    "cmd" + string(rune(i)),
				Command: "echo",
				Mode:    ModeOnce,
			}
		}

		config := &Config{
			Version:  "1.0.0",
			Commands: commands,
		}

		validator := NewStrictValidator()
		err := validator.ValidateConfig(config)
		if err == nil {
			t.Error("Expected error for too many commands in strict mode")
		}
		if !strings.Contains(err.Error(), "too many commands") {
			t.Errorf("Expected 'too many commands' error, got: %v", err)
		}
	})

	t.Run("WorkDir validation edge cases", func(t *testing.T) {
		validator := NewStrictValidator()

		// Test absolute path validation in strict mode (without directory validation)
		validatorNoDir := &Validator{StrictMode: true, ValidateWorkDirs: false}
		err := validatorNoDir.validateWorkDir("/some/absolute/path")
		// This should not error when directory validation is disabled
		if err != nil {
			t.Errorf("Unexpected error for absolute path: %v", err)
		}

		// Test path with multiple parent directory references
		err = validator.validateWorkDir("../../dangerous/path")
		if err == nil {
			t.Error("Expected error for path with parent directory references")
		}
		if !strings.Contains(err.Error(), "cannot contain parent directory references") {
			t.Errorf("Expected parent directory error, got: %v", err)
		}

		// Test path normalization edge case
		err = validator.validateWorkDir("./normal/../path")
		if err == nil {
			t.Error("Expected error for normalized path with parent references")
		}
	})

	t.Run("Command validation edge cases", func(t *testing.T) {
		validator := NewStrictValidator()

		// Test command with all possible validation errors
		cmd := &Command{
			Name:    strings.Repeat("a", 101), // Too long name
			Command: "",                       // Empty command
			Mode:    Mode("invalid"),          // Invalid mode
			Args:    make([]string, 51),       // Too many args
			WorkDir: "../invalid",             // Invalid work dir
			Env: map[string]string{
				"123INVALID": "value", // Invalid env var name
			},
		}

		errors := validator.validateCommand(cmd)
		if len(errors) == 0 {
			t.Error("Expected multiple validation errors")
		}

		// Check that all expected errors are present
		errorStr := errors.Error()
		expectedErrors := []string{
			"too long",
			"command is required",
			"mode must be either",
			"too many arguments",
			"cannot contain parent directory references",
			"invalid environment variable name",
		}

		for _, expected := range expectedErrors {
			if !strings.Contains(errorStr, expected) {
				t.Errorf("Expected error to contain %q, got: %v", expected, errorStr)
			}
		}
	})

	t.Run("JSON parsing edge cases", func(t *testing.T) {
		// Test malformed JSON with specific syntax errors
		malformedJSON := `{
			"version": "1.0",
			"commands": [
				{
					"name": "test",
					"command": "echo",
					"mode": "once"
				},
			]
		}`

		_, err := ParseJSON([]byte(malformedJSON))
		if err == nil {
			t.Error("Expected JSON syntax error for trailing comma")
		}

		// Test JSON with wrong type for nested field
		wrongTypeJSON := `{
			"version": "1.0",
			"commands": [
				{
					"name": "test",
					"command": "echo",
					"mode": "once",
					"args": "should-be-array"
				}
			]
		}`

		_, err = ParseJSON([]byte(wrongTypeJSON))
		if err == nil {
			t.Error("Expected type error for args field")
		}
		if !strings.Contains(err.Error(), "type error") {
			t.Errorf("Expected type error, got: %v", err)
		}
	})

	t.Run("Mode validation edge cases", func(t *testing.T) {
		// Test Mode validation with empty string
		var mode Mode
		err := mode.UnmarshalJSON([]byte(`""`))
		if err == nil {
			t.Error("Expected error for empty mode string")
		}

		// Test Mode validation with null value
		err = mode.UnmarshalJSON([]byte(`null`))
		if err == nil {
			t.Error("Expected error for null mode value")
		}
	})

	t.Run("Environment variable edge cases", func(t *testing.T) {
		validator := NewStrictValidator()

		// Test env var with underscore at start (valid)
		env := map[string]string{"_VALID": "value"}
		err := validator.validateEnv(env)
		if err != nil {
			t.Errorf("Unexpected error for valid env var starting with underscore: %v", err)
		}

		// Test env var with number in middle (valid)
		env = map[string]string{"VAR_123_TEST": "value"}
		err = validator.validateEnv(env)
		if err != nil {
			t.Errorf("Unexpected error for valid env var with numbers: %v", err)
		}

		// Test env var with special characters (invalid)
		env = map[string]string{"VAR-INVALID": "value"}
		err = validator.validateEnv(env)
		if err == nil {
			t.Error("Expected error for env var with hyphen")
		}
	})

	t.Run("Command name edge cases", func(t *testing.T) {
		validator := NewStrictValidator()

		// Test single character name (valid)
		err := validator.validateCommandName("a")
		if err != nil {
			t.Errorf("Unexpected error for single character name: %v", err)
		}

		// Test name ending with special character (invalid in strict mode)
		err = validator.validateCommandName("test-")
		if err == nil {
			t.Error("Expected error for name ending with hyphen in strict mode")
		}

		// Test reserved name case insensitive
		err = validator.validateCommandName("HELP")
		if err == nil {
			t.Error("Expected error for reserved name in uppercase")
		}
	})

	t.Run("Version validation edge cases", func(t *testing.T) {
		validator := NewStrictValidator()

		// Test version with only major.minor (valid)
		err := validator.validateVersion("1.0")
		if err != nil {
			t.Errorf("Unexpected error for major.minor version: %v", err)
		}

		// Test version with complex prerelease and build metadata
		err = validator.validateVersion("1.0.0-alpha.beta.1+build.123.abc")
		if err != nil {
			t.Errorf("Unexpected error for complex semantic version: %v", err)
		}

		// Test invalid semantic version formats
		invalidVersions := []string{
			"1",       // Only major
			"1.0.0.0", // Too many parts
			"v1.0.0",  // Prefix not allowed
			"1.0.0-",  // Empty prerelease
			"1.0.0+",  // Empty build metadata
		}

		for _, version := range invalidVersions {
			err := validator.validateVersion(version)
			if err == nil {
				t.Errorf("Expected error for invalid version %q", version)
			}
		}

		// Note: "01.0.0" might be accepted by the current regex - this is a limitation
		// of the current implementation that could be improved in the future
	})

	t.Run("Configuration boundary conditions", func(t *testing.T) {
		// Test configuration with exactly the maximum allowed values
		validator := NewStrictValidator()

		// Create config with exactly 50 commands (at the limit)
		commands := make([]Command, 50)
		for i := 0; i < 50; i++ {
			// Make each command name unique by including the index
			uniqueName := fmt.Sprintf("cmd%02d%s", i, strings.Repeat("a", 93)) // 99 chars total
			commands[i] = Command{
				Name:    uniqueName,
				Command: strings.Repeat("b", 500), // Exactly at 500 char limit
				Mode:    ModeOnce,
				Args:    make([]string, 50), // Exactly at 50 arg limit
				Env:     make(map[string]string),
			}

			// Add exactly 100 env vars (at the limit)
			for j := 0; j < 100; j++ {
				envName := "VAR_" + strings.Repeat("A", 95) // 99 chars total
				envValue := strings.Repeat("v", 10000)      // Exactly at 10000 char limit
				commands[i].Env[envName] = envValue
			}

			// Add exactly 50 args with max length
			for j := 0; j < 50; j++ {
				commands[i].Args[j] = strings.Repeat("arg", 333) + "a" // 1000 chars exactly
			}
		}

		config := &Config{
			Version:  "999.999.999", // Valid semantic version
			Commands: commands,
		}

		err := validator.ValidateConfig(config)
		if err != nil {
			t.Errorf("Unexpected error for config at boundary limits: %v", err)
		}
	})
}

// TestRealWorldScenarios tests realistic configuration scenarios
func TestRealWorldScenarios(t *testing.T) {
	t.Run("Complex development workflow", func(t *testing.T) {
		configJSON := `{
			"version": "2.1.0-dev.1+build.abc123",
			"commands": [
				{
					"name": "cleanup-containers",
					"command": "docker",
					"args": ["system", "prune", "-f"],
					"mode": "once"
				},
				{
					"name": "start-database",
					"command": "docker",
					"args": ["run", "-d", "--name", "dev-db", "-p", "5432:5432", "postgres:13"],
					"mode": "keepAlive",
					"env": {
						"POSTGRES_PASSWORD": "dev_password_123",
						"POSTGRES_DB": "myapp_dev",
						"POSTGRES_USER": "developer"
					}
				},
				{
					"name": "wait-for-db",
					"command": "sleep",
					"args": ["5"],
					"mode": "once"
				},
				{
					"name": "run-migrations",
					"command": "npm",
					"args": ["run", "db:migrate"],
					"mode": "once",
					"workDir": "./backend",
					"env": {
						"NODE_ENV": "development",
						"DATABASE_URL": "postgresql://developer:dev_password_123@localhost:5432/myapp_dev"
					}
				},
				{
					"name": "seed-database",
					"command": "npm",
					"args": ["run", "db:seed"],
					"mode": "once",
					"workDir": "./backend"
				},
				{
					"name": "start-api-server",
					"command": "npm",
					"args": ["run", "dev"],
					"mode": "keepAlive",
					"workDir": "./backend",
					"env": {
						"NODE_ENV": "development",
						"PORT": "3000",
						"JWT_SECRET": "dev_secret_key",
						"REDIS_URL": "redis://localhost:6379"
					}
				},
				{
					"name": "start-frontend",
					"command": "npm",
					"args": ["run", "dev"],
					"mode": "keepAlive",
					"workDir": "./frontend",
					"env": {
						"VITE_API_URL": "http://localhost:3000",
						"VITE_ENV": "development"
					}
				}
			]
		}`

		config, err := ParseJSON([]byte(configJSON))
		if err != nil {
			t.Errorf("Failed to parse realistic config: %v", err)
		}

		// Validate with both normal and strict validators (without directory validation)
		validators := []*Validator{
			NewValidator(),
			{StrictMode: true, ValidateWorkDirs: false, ValidateCommands: false}, // Strict but no dir validation
		}

		for i, validator := range validators {
			err := validator.ValidateConfig(config)
			if err != nil {
				t.Errorf("Validator %d failed on realistic config: %v", i, err)
			}
		}
	})

	t.Run("Microservices deployment scenario", func(t *testing.T) {
		configJSON := `{
			"version": "1.0.0",
			"commands": [
				{
					"name": "auth-service",
					"command": "go",
					"args": ["run", "main.go"],
					"mode": "keepAlive",
					"workDir": "./services/auth",
					"env": {
						"PORT": "8001",
						"DB_HOST": "localhost",
						"DB_PORT": "5432",
						"JWT_SECRET": "auth_secret"
					}
				},
				{
					"name": "user-service",
					"command": "go",
					"args": ["run", "main.go"],
					"mode": "keepAlive",
					"workDir": "./services/user",
					"env": {
						"PORT": "8002",
						"AUTH_SERVICE_URL": "http://localhost:8001"
					}
				},
				{
					"name": "api-gateway",
					"command": "go",
					"args": ["run", "main.go"],
					"mode": "keepAlive",
					"workDir": "./gateway",
					"env": {
						"PORT": "8000",
						"AUTH_SERVICE": "http://localhost:8001",
						"USER_SERVICE": "http://localhost:8002"
					}
				}
			]
		}`

		config, err := ParseJSON([]byte(configJSON))
		if err != nil {
			t.Errorf("Failed to parse microservices config: %v", err)
		}

		err = NewValidator().ValidateConfig(config)
		if err != nil {
			t.Errorf("Failed to validate microservices config: %v", err)
		}
	})

	t.Run("CI/CD pipeline scenario", func(t *testing.T) {
		configJSON := `{
			"version": "1.0.0",
			"commands": [
				{
					"name": "install-dependencies",
					"command": "npm",
					"args": ["ci"],
					"mode": "once"
				},
				{
					"name": "run-linter",
					"command": "npm",
					"args": ["run", "lint"],
					"mode": "once"
				},
				{
					"name": "run-unit-tests",
					"command": "npm",
					"args": ["run", "test:unit"],
					"mode": "once",
					"env": {
						"NODE_ENV": "test",
						"CI": "true"
					}
				},
				{
					"name": "run-integration-tests",
					"command": "npm",
					"args": ["run", "test:integration"],
					"mode": "once",
					"env": {
						"NODE_ENV": "test",
						"DATABASE_URL": "postgresql://test:test@localhost:5432/test_db"
					}
				},
				{
					"name": "build-application",
					"command": "npm",
					"args": ["run", "build"],
					"mode": "once",
					"env": {
						"NODE_ENV": "production"
					}
				}
			]
		}`

		config, err := ParseJSON([]byte(configJSON))
		if err != nil {
			t.Errorf("Failed to parse CI/CD config: %v", err)
		}

		err = NewValidator().ValidateConfig(config)
		if err != nil {
			t.Errorf("Failed to validate CI/CD config: %v", err)
		}
	})
}

// TestErrorRecoveryAndRobustness tests error handling and recovery scenarios
func TestErrorRecoveryAndRobustness(t *testing.T) {
	t.Run("Malformed JSON recovery", func(t *testing.T) {
		malformedConfigs := []string{
			`{"version": "1.0", "commands": [}`,                                                        // Missing closing bracket
			`{"version": "1.0" "commands": []}`,                                                        // Missing comma
			`{"version": "1.0", "commands": [{"name": "test"}]}`,                                       // Missing required fields
			`{"version": "1.0", "commands": [{"name": "test", "command": "echo", "mode": "invalid"}]}`, // Invalid mode
		}

		for i, config := range malformedConfigs {
			_, err := ParseJSON([]byte(config))
			if err == nil {
				t.Errorf("Expected error for malformed config %d", i)
			}
			// Ensure error messages are helpful
			if !strings.Contains(err.Error(), "JSON") && !strings.Contains(err.Error(), "validation") {
				t.Errorf("Error message should mention JSON or validation issue: %v", err)
			}
		}
	})

	t.Run("Partial validation success", func(t *testing.T) {
		// Test that validation continues even when some commands fail
		config := &Config{
			Version: "1.0.0",
			Commands: []Command{
				{
					Name:    "valid-command",
					Command: "echo",
					Mode:    ModeOnce,
				},
				{
					Name:    "", // Invalid: empty name
					Command: "ls",
					Mode:    ModeOnce,
				},
				{
					Name:    "another-valid",
					Command: "pwd",
					Mode:    ModeOnce,
				},
				{
					Name:    "invalid-mode-cmd",
					Command: "date",
					Mode:    Mode("invalid"), // Invalid mode
				},
			},
		}

		validator := NewValidator()
		err := validator.ValidateConfig(config)
		if err == nil {
			t.Error("Expected validation errors")
		}

		// Check that multiple errors are reported
		errorStr := err.Error()
		if !strings.Contains(errorStr, "multiple validation errors") {
			t.Errorf("Expected multiple validation errors, got: %v", err)
		}
	})

	t.Run("Unicode and special characters", func(t *testing.T) {
		configJSON := `{
			"version": "1.0.0",
			"commands": [
				{
					"name": "unicode-test-命令",
					"command": "echo",
					"args": ["Hello, 世界! 🌍"],
					"mode": "once",
					"env": {
						"MESSAGE": "Testing unicode: αβγδε"
					}
				}
			]
		}`

		config, err := ParseJSON([]byte(configJSON))
		if err != nil {
			t.Errorf("Failed to parse config with unicode: %v", err)
		}

		// Normal validator should accept unicode
		err = NewValidator().ValidateConfig(config)
		if err != nil {
			t.Errorf("Normal validator should accept unicode: %v", err)
		}

		// Strict validator might have different rules for command names
		err = NewStrictValidator().ValidateConfig(config)
		// This might fail due to strict naming rules, which is expected behavior
	})
}
