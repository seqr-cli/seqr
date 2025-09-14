package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// TestRealisticScenario_WebDevelopmentWorkflow tests a realistic web development workflow
// similar to the example.queue.json but with simpler commands that don't require Docker/npm
func TestRealisticScenario_WebDevelopmentWorkflow(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	// Create temporary directories to simulate project structure
	tempDir := t.TempDir()
	backendDir := filepath.Join(tempDir, "backend")
	frontendDir := filepath.Join(tempDir, "frontend")

	err := os.MkdirAll(backendDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backend directory: %v", err)
	}

	err = os.MkdirAll(frontendDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create frontend directory: %v", err)
	}

	// Create a realistic workflow configuration
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "setup-database",
				Command: "echo",
				Args:    []string{"Starting database container..."},
				Mode:    config.ModeOnce,
				Env: map[string]string{
					"POSTGRES_PASSWORD": "testpass",
					"POSTGRES_DB":       "testdb",
				},
			},
			{
				Name:    "wait-for-db",
				Command: "sleep",
				Args:    []string{"1"}, // Shorter wait for testing
				Mode:    config.ModeOnce,
			},
			{
				Name:    "run-migrations",
				Command: "echo",
				Args:    []string{"Running database migrations..."},
				Mode:    config.ModeOnce,
				WorkDir: backendDir,
			},
			{
				Name:    "start-api-server",
				Command: "sleep",
				Args:    []string{"0.5"}, // Simulate server running
				Mode:    config.ModeKeepAlive,
				WorkDir: backendDir,
				Env: map[string]string{
					"NODE_ENV":     "development",
					"PORT":         "3000",
					"DATABASE_URL": "postgresql://postgres:testpass@localhost:5432/testdb",
				},
			},
			{
				Name:    "start-frontend",
				Command: "sleep",
				Args:    []string{"0.3"}, // Simulate frontend dev server
				Mode:    config.ModeKeepAlive,
				WorkDir: frontendDir,
				Env: map[string]string{
					"VITE_API_URL": "http://localhost:3000",
				},
			},
			{
				Name:    "health-check",
				Command: "echo",
				Args:    []string{"All services are running!"},
				Mode:    config.ModeOnce,
			},
		},
	}

	start := time.Now()
	err = executor.Execute(ctx, cfg)
	executionTime := time.Since(start)

	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected state Success, got %v", status.State)
	}

	if len(status.Results) != 6 {
		t.Fatalf("Expected 6 results, got %d", len(status.Results))
	}

	// Verify all commands succeeded
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d (%s) to be successful", i, result.Command.Name)
		}
	}

	// Verify specific command outputs and behaviors
	setupResult := status.Results[0]
	if !strings.Contains(setupResult.Output, "Starting database container") {
		t.Errorf("Expected database setup message, got: %s", setupResult.Output)
	}

	migrationResult := status.Results[2]
	if !strings.Contains(migrationResult.Output, "Running database migrations") {
		t.Errorf("Expected migration message, got: %s", migrationResult.Output)
	}

	// Verify keepAlive processes were started
	apiResult := status.Results[3]
	if !strings.Contains(apiResult.Output, "PID") {
		t.Errorf("Expected API server to show PID, got: %s", apiResult.Output)
	}

	frontendResult := status.Results[4]
	if !strings.Contains(frontendResult.Output, "PID") {
		t.Errorf("Expected frontend server to show PID, got: %s", frontendResult.Output)
	}

	healthResult := status.Results[5]
	if !strings.Contains(healthResult.Output, "All services are running") {
		t.Errorf("Expected health check message, got: %s", healthResult.Output)
	}

	// Execution should be fast (keepAlive doesn't wait)
	if executionTime > 3*time.Second {
		t.Errorf("Realistic workflow took too long: %v", executionTime)
	}

	// Wait for background processes to complete
	time.Sleep(1 * time.Second)

	t.Logf("Successfully executed realistic web development workflow in %v", executionTime)
}

// TestRealisticScenario_DevOpsWorkflow tests a DevOps-style workflow with build and deploy steps
func TestRealisticScenario_DevOpsWorkflow(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	// Create temporary project directory
	tempDir := t.TempDir()

	// Create some mock files
	err := os.WriteFile(filepath.Join(tempDir, "Dockerfile"), []byte("FROM alpine\nRUN echo 'mock dockerfile'"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock Dockerfile: %v", err)
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "lint-code",
				Command: "echo",
				Args:    []string{"Running code linting..."},
				Mode:    config.ModeOnce,
				WorkDir: tempDir,
			},
			{
				Name:    "run-tests",
				Command: "echo",
				Args:    []string{"Running test suite..."},
				Mode:    config.ModeOnce,
				WorkDir: tempDir,
			},
			{
				Name:    "build-application",
				Command: "echo",
				Args:    []string{"Building application..."},
				Mode:    config.ModeOnce,
				WorkDir: tempDir,
				Env: map[string]string{
					"BUILD_ENV": "production",
					"VERSION":   "1.0.0",
				},
			},
			{
				Name:    "start-monitoring",
				Command: "sleep",
				Args:    []string{"0.2"}, // Simulate monitoring service
				Mode:    config.ModeKeepAlive,
				Env: map[string]string{
					"MONITOR_PORT": "9090",
				},
			},
			{
				Name:    "deploy-application",
				Command: "echo",
				Args:    []string{"Deploying to staging environment..."},
				Mode:    config.ModeOnce,
				WorkDir: tempDir,
			},
			{
				Name:    "run-smoke-tests",
				Command: "echo",
				Args:    []string{"Running smoke tests..."},
				Mode:    config.ModeOnce,
			},
		},
	}

	start := time.Now()
	err = executor.Execute(ctx, cfg)
	executionTime := time.Since(start)

	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected state Success, got %v", status.State)
	}

	if len(status.Results) != 6 {
		t.Fatalf("Expected 6 results, got %d", len(status.Results))
	}

	// Verify all commands succeeded
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d (%s) to be successful", i, result.Command.Name)
		}
	}

	// Verify the workflow sequence
	expectedMessages := []string{
		"Running code linting",
		"Running test suite",
		"Building application",
		"PID", // monitoring service should show PID
		"Deploying to staging",
		"Running smoke tests",
	}

	for i, expected := range expectedMessages {
		if !strings.Contains(status.Results[i].Output, expected) {
			t.Errorf("Expected result %d to contain '%s', got: %s", i, expected, status.Results[i].Output)
		}
	}

	// Wait for monitoring service to complete
	time.Sleep(300 * time.Millisecond)

	t.Logf("Successfully executed DevOps workflow in %v", executionTime)
}

// TestRealisticScenario_FailureRecovery tests how the system handles failures in realistic scenarios
func TestRealisticScenario_FailureRecovery(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "setup-environment",
				Command: "echo",
				Args:    []string{"Setting up environment..."},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "start-database",
				Command: "sleep",
				Args:    []string{"0.1"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "failing-migration",
				Command: "false", // This will fail
				Mode:    config.ModeOnce,
			},
			{
				Name:    "should-not-run",
				Command: "echo",
				Args:    []string{"This should not execute"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected execution to fail due to failing migration")
	}

	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected state Failed, got %v", status.State)
	}

	// Should have 3 results (setup, database start, failed migration)
	if len(status.Results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(status.Results))
	}

	// First two should succeed
	if !status.Results[0].Success {
		t.Error("Expected setup to succeed")
	}
	if !status.Results[1].Success {
		t.Error("Expected database start to succeed")
	}

	// Third should fail
	if status.Results[2].Success {
		t.Error("Expected migration to fail")
	}

	// Verify error details
	if status.Results[2].ErrorDetail == nil {
		t.Fatal("Expected error detail for failed migration")
	}

	if status.Results[2].ErrorDetail.Type != ErrorTypeNonZeroExit {
		t.Errorf("Expected error type %s, got %s", ErrorTypeNonZeroExit, status.Results[2].ErrorDetail.Type)
	}

	// Wait for database process to complete
	time.Sleep(200 * time.Millisecond)

	t.Logf("Successfully tested failure recovery scenario")
}

// TestRealisticScenario_ComplexEnvironmentVariables tests complex environment variable scenarios
func TestRealisticScenario_ComplexEnvironmentVariables(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "check-database-config",
				Command: getShellCommand(),
				Args:    []string{"-c", "echo \"Database: $DB_HOST:$DB_PORT/$DB_NAME (User: $DB_USER)\""},
				Mode:    config.ModeOnce,
				Env: map[string]string{
					"DB_HOST": "localhost",
					"DB_PORT": "5432",
					"DB_NAME": "myapp_production",
					"DB_USER": "app_user",
				},
			},
			{
				Name:    "check-api-config",
				Command: getShellCommand(),
				Args:    []string{"-c", "echo \"API Config: $API_URL (Auth: $AUTH_ENABLED, Debug: $DEBUG_MODE)\""},
				Mode:    config.ModeOnce,
				Env: map[string]string{
					"API_URL":      "https://api.example.com/v1",
					"AUTH_ENABLED": "true",
					"DEBUG_MODE":   "false",
				},
			},
			{
				Name:    "check-feature-flags",
				Command: getShellCommand(),
				Args:    []string{"-c", "echo \"Features: NEW_UI=$NEW_UI_ENABLED, ANALYTICS=$ANALYTICS_ENABLED\""},
				Mode:    config.ModeOnce,
				Env: map[string]string{
					"NEW_UI_ENABLED":       "true",
					"ANALYTICS_ENABLED":    "false",
					"EXPERIMENTAL_FEATURE": "beta",
				},
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected state Success, got %v", status.State)
	}

	if len(status.Results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(status.Results))
	}

	// Verify environment variables were properly set
	dbOutput := status.Results[0].Output
	if !strings.Contains(dbOutput, "localhost:5432/myapp_production") {
		t.Errorf("Expected database config in output, got: %s", dbOutput)
	}

	apiOutput := status.Results[1].Output
	if !strings.Contains(apiOutput, "https://api.example.com/v1") {
		t.Errorf("Expected API URL in output, got: %s", apiOutput)
	}
	if !strings.Contains(apiOutput, "Auth: true") {
		t.Errorf("Expected auth config in output, got: %s", apiOutput)
	}

	featureOutput := status.Results[2].Output
	if !strings.Contains(featureOutput, "NEW_UI=true") {
		t.Errorf("Expected feature flag in output, got: %s", featureOutput)
	}

	t.Logf("Successfully tested complex environment variable scenario")
}

// TestRealisticScenario_LongRunningServices tests managing multiple long-running services
func TestRealisticScenario_LongRunningServices(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{Verbose: true})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "start-redis",
				Command: "sleep",
				Args:    []string{"0.4"}, // Simulate Redis
				Mode:    config.ModeKeepAlive,
				Env: map[string]string{
					"REDIS_PORT": "6379",
				},
			},
			{
				Name:    "start-postgres",
				Command: "sleep",
				Args:    []string{"0.5"}, // Simulate PostgreSQL
				Mode:    config.ModeKeepAlive,
				Env: map[string]string{
					"POSTGRES_PORT": "5432",
				},
			},
			{
				Name:    "wait-for-services",
				Command: "sleep",
				Args:    []string{"0.1"}, // Wait for services to be ready
				Mode:    config.ModeOnce,
			},
			{
				Name:    "start-api-gateway",
				Command: "sleep",
				Args:    []string{"0.3"}, // Simulate API Gateway
				Mode:    config.ModeKeepAlive,
				Env: map[string]string{
					"GATEWAY_PORT": "8080",
				},
			},
			{
				Name:    "start-worker-queue",
				Command: "sleep",
				Args:    []string{"0.2"}, // Simulate background workers
				Mode:    config.ModeKeepAlive,
				Env: map[string]string{
					"WORKER_CONCURRENCY": "4",
				},
			},
			{
				Name:    "verify-all-services",
				Command: "echo",
				Args:    []string{"All services are running and healthy!"},
				Mode:    config.ModeOnce,
			},
		},
	}

	start := time.Now()
	err := executor.Execute(ctx, cfg)
	executionTime := time.Since(start)

	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected state Success, got %v", status.State)
	}

	if len(status.Results) != 6 {
		t.Fatalf("Expected 6 results, got %d", len(status.Results))
	}

	// Verify all commands succeeded
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d (%s) to be successful", i, result.Command.Name)
		}
	}

	// Verify keepAlive services show PID information
	keepAliveIndices := []int{0, 1, 3, 4} // Redis, Postgres, Gateway, Workers
	for _, idx := range keepAliveIndices {
		if !strings.Contains(status.Results[idx].Output, "PID") {
			t.Errorf("Expected keepAlive service %d to show PID, got: %s", idx, status.Results[idx].Output)
		}
	}

	// Verify final verification message
	if !strings.Contains(status.Results[5].Output, "All services are running") {
		t.Errorf("Expected final verification message, got: %s", status.Results[5].Output)
	}

	// Execution should be fast (doesn't wait for keepAlive processes)
	if executionTime > 2*time.Second {
		t.Errorf("Long-running services workflow took too long: %v", executionTime)
	}

	// Wait for all background processes to complete
	time.Sleep(700 * time.Millisecond)

	t.Logf("Successfully managed multiple long-running services in %v", executionTime)
}
