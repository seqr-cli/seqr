package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// MockReporter implements Reporter interface for testing
type MockReporter struct {
	mu                     sync.Mutex
	startCalls             []int
	commandStartCalls      []CommandStartCall
	commandSuccessCalls    []CommandResultCall
	commandFailureCalls    []CommandResultCall
	executionCompleteCalls []ExecutionStatus
	executionSummaryCalls  []ExecutionStatus
}

type CommandStartCall struct {
	CommandName  string
	CommandIndex int
}

type CommandResultCall struct {
	Result       ExecutionResult
	CommandIndex int
}

func NewMockReporter() *MockReporter {
	return &MockReporter{
		startCalls:             make([]int, 0),
		commandStartCalls:      make([]CommandStartCall, 0),
		commandSuccessCalls:    make([]CommandResultCall, 0),
		commandFailureCalls:    make([]CommandResultCall, 0),
		executionCompleteCalls: make([]ExecutionStatus, 0),
		executionSummaryCalls:  make([]ExecutionStatus, 0),
	}
}

func (m *MockReporter) ReportStart(totalCommands int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalls = append(m.startCalls, totalCommands)
}

func (m *MockReporter) ReportCommandStart(commandName string, commandIndex int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandStartCalls = append(m.commandStartCalls, CommandStartCall{
		CommandName:  commandName,
		CommandIndex: commandIndex,
	})
}

func (m *MockReporter) ReportCommandSuccess(result ExecutionResult, commandIndex int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandSuccessCalls = append(m.commandSuccessCalls, CommandResultCall{
		Result:       result,
		CommandIndex: commandIndex,
	})
}

func (m *MockReporter) ReportCommandFailure(result ExecutionResult, commandIndex int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandFailureCalls = append(m.commandFailureCalls, CommandResultCall{
		Result:       result,
		CommandIndex: commandIndex,
	})
}

func (m *MockReporter) ReportExecutionComplete(status ExecutionStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executionCompleteCalls = append(m.executionCompleteCalls, status)
}

func (m *MockReporter) ReportExecutionSummary(status ExecutionStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executionSummaryCalls = append(m.executionSummaryCalls, status)
}

// ProcessStatusReporter interface methods
func (m *MockReporter) ReportProcessStatus(processes map[string]ProcessInfo) {
	// Mock implementation - just track that it was called
}

func (m *MockReporter) ReportProcessHealth(health map[string]ProcessHealth) {
	// Mock implementation - just track that it was called
}

func (m *MockReporter) ReportProcessLifecycleEvent(event HealthMonitorEvent) {
	// Mock implementation - just track that it was called
}

func (m *MockReporter) ReportHealthSummary(summary HealthSummary) {
	// Mock implementation - just track that it was called
}

func (m *MockReporter) GetCalls() ([]int, []CommandStartCall, []CommandResultCall, []CommandResultCall, []ExecutionStatus, []ExecutionStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCalls, m.commandStartCalls, m.commandSuccessCalls, m.commandFailureCalls, m.executionCompleteCalls, m.executionSummaryCalls
}

func TestExecutor_Integration_CompleteWorkflow(t *testing.T) {
	mockReporter := NewMockReporter()
	executor := NewExecutor(ExecutorOptions{
		Verbose:  true,
		Reporter: mockReporter,
	})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "setup",
				Command: "echo",
				Args:    []string{"Setting up environment"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "background-service",
				Command: "sleep",
				Args:    []string{"0.1"},
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "main-task",
				Command: "echo",
				Args:    []string{"Running main task"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	// Verify all reporter calls were made correctly
	starts, commandStarts, successes, failures, completes, summaries := mockReporter.GetCalls()

	// Check ReportStart was called once with correct total
	if len(starts) != 1 || starts[0] != 3 {
		t.Errorf("Expected 1 start call with 3 commands, got %v", starts)
	}

	// Check ReportCommandStart was called for each command
	if len(commandStarts) != 3 {
		t.Fatalf("Expected 3 command start calls, got %d", len(commandStarts))
	}

	expectedCommandNames := []string{"setup", "background-service", "main-task"}
	for i, call := range commandStarts {
		if call.CommandName != expectedCommandNames[i] {
			t.Errorf("Expected command start %d to be '%s', got '%s'", i, expectedCommandNames[i], call.CommandName)
		}
		if call.CommandIndex != i {
			t.Errorf("Expected command start %d to have index %d, got %d", i, i, call.CommandIndex)
		}
	}

	// Check ReportCommandSuccess was called for each successful command
	if len(successes) != 3 {
		t.Fatalf("Expected 3 command success calls, got %d", len(successes))
	}

	// Check no failures were reported
	if len(failures) != 0 {
		t.Errorf("Expected 0 command failure calls, got %d", len(failures))
	}

	// Check ReportExecutionComplete was called once
	if len(completes) != 1 {
		t.Fatalf("Expected 1 execution complete call, got %d", len(completes))
	}

	if completes[0].State != StateSuccess {
		t.Errorf("Expected execution complete state to be Success, got %v", completes[0].State)
	}

	// Check ReportExecutionSummary was called once
	if len(summaries) != 1 {
		t.Fatalf("Expected 1 execution summary call, got %d", len(summaries))
	}

	if summaries[0].State != StateSuccess {
		t.Errorf("Expected execution summary state to be Success, got %v", summaries[0].State)
	}

	// Wait for background process to complete
	time.Sleep(200 * time.Millisecond)
}

func TestExecutor_Integration_FailureWorkflow(t *testing.T) {
	mockReporter := NewMockReporter()
	executor := NewExecutor(ExecutorOptions{
		Reporter: mockReporter,
	})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "success-cmd",
				Command: "echo",
				Args:    []string{"This will succeed"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "failing-cmd",
				Command: "false",
				Mode:    config.ModeOnce,
			},
			{
				Name:    "should-not-run",
				Command: "echo",
				Args:    []string{"This should not run"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected execution to fail, got nil error")
	}

	// Verify reporter calls for failure scenario
	starts, commandStarts, successes, failures, completes, summaries := mockReporter.GetCalls()

	// Should have started execution
	if len(starts) != 1 || starts[0] != 3 {
		t.Errorf("Expected 1 start call with 3 commands, got %v", starts)
	}

	// Should have started 2 commands (success + failure)
	if len(commandStarts) != 2 {
		t.Errorf("Expected 2 command start calls, got %d", len(commandStarts))
	}

	// Should have 1 success and 1 failure
	if len(successes) != 1 {
		t.Errorf("Expected 1 command success call, got %d", len(successes))
	}

	if len(failures) != 1 {
		t.Errorf("Expected 1 command failure call, got %d", len(failures))
	}

	// Check failure details
	if failures[0].Result.Command.Name != "failing-cmd" {
		t.Errorf("Expected failure to be for 'failing-cmd', got '%s'", failures[0].Result.Command.Name)
	}

	// Should have reported execution complete with failure
	if len(completes) != 1 {
		t.Fatalf("Expected 1 execution complete call, got %d", len(completes))
	}

	if completes[0].State != StateFailed {
		t.Errorf("Expected execution complete state to be Failed, got %v", completes[0].State)
	}

	if completes[0].CompletedCount != 1 {
		t.Errorf("Expected completed count to be 1, got %d", completes[0].CompletedCount)
	}

	// Check execution summary was also called
	if len(summaries) != 1 {
		t.Fatalf("Expected 1 execution summary call, got %d", len(summaries))
	}
}

func TestExecutor_Integration_ConcurrentAccess(t *testing.T) {
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "concurrent-test",
				Command: "sleep",
				Args:    []string{"0.2"},
				Mode:    config.ModeOnce,
			},
		},
	}

	// Start execution in background
	done := make(chan error, 1)
	go func() {
		done <- executor.Execute(ctx, cfg)
	}()

	// Concurrently access status and stop methods
	var wg sync.WaitGroup
	statusResults := make([]ExecutionStatus, 20)

	// Multiple goroutines calling GetStatus
	for i := range 20 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			time.Sleep(time.Duration(index) * time.Millisecond) // Stagger calls
			statusResults[index] = executor.GetStatus()
		}(i)
	}

	// One goroutine calling Stop (but execution should complete normally)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		executor.Stop() // This should be safe even if execution is nearly done
	}()

	wg.Wait()

	// Wait for execution to complete
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Expected successful execution, got error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Execution did not complete within timeout")
	}

	// Verify all status calls returned valid data
	for i, status := range statusResults {
		if status.TotalCount != 1 {
			t.Errorf("Status call %d: expected total count 1, got %d", i, status.TotalCount)
		}
		// State can be Ready, Running, or Success depending on timing
		if status.State != StateReady && status.State != StateRunning && status.State != StateSuccess {
			t.Errorf("Status call %d: unexpected state %v", i, status.State)
		}
	}
}

func TestExecutor_Integration_CustomReporter(t *testing.T) {
	// Test with a custom reporter that writes to a buffer
	var buf bytes.Buffer
	customReporter := &CustomTestReporter{writer: &buf}

	executor := NewExecutor(ExecutorOptions{
		Reporter: customReporter,
	})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test-cmd",
				Command: "echo",
				Args:    []string{"custom reporter test"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	output := buf.String()
	expectedParts := []string{
		"CUSTOM START: 1",
		"CUSTOM COMMAND START: test-cmd",
		"CUSTOM SUCCESS: test-cmd",
		"CUSTOM COMPLETE: success",
		"CUSTOM SUMMARY",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected custom reporter output to contain %q, got:\n%s", expected, output)
		}
	}
}

// CustomTestReporter is a test implementation of Reporter
type CustomTestReporter struct {
	writer io.Writer
}

func (r *CustomTestReporter) ReportStart(totalCommands int) {
	fmt.Fprintf(r.writer, "CUSTOM START: %d\n", totalCommands)
}

func (r *CustomTestReporter) ReportCommandStart(commandName string, commandIndex int) {
	fmt.Fprintf(r.writer, "CUSTOM COMMAND START: %s\n", commandName)
}

func (r *CustomTestReporter) ReportCommandSuccess(result ExecutionResult, commandIndex int) {
	fmt.Fprintf(r.writer, "CUSTOM SUCCESS: %s\n", result.Command.Name)
}

func (r *CustomTestReporter) ReportCommandFailure(result ExecutionResult, commandIndex int) {
	fmt.Fprintf(r.writer, "CUSTOM FAILURE: %s\n", result.Command.Name)
}

func (r *CustomTestReporter) ReportExecutionComplete(status ExecutionStatus) {
	fmt.Fprintf(r.writer, "CUSTOM COMPLETE: %s\n", status.State.String())
}

func (r *CustomTestReporter) ReportExecutionSummary(status ExecutionStatus) {
	fmt.Fprintf(r.writer, "CUSTOM SUMMARY\n")
}

// ProcessStatusReporter interface methods
func (r *CustomTestReporter) ReportProcessStatus(processes map[string]ProcessInfo) {
	fmt.Fprintf(r.writer, "CUSTOM PROCESS STATUS\n")
}

func (r *CustomTestReporter) ReportProcessHealth(health map[string]ProcessHealth) {
	fmt.Fprintf(r.writer, "CUSTOM PROCESS HEALTH\n")
}

func (r *CustomTestReporter) ReportProcessLifecycleEvent(event HealthMonitorEvent) {
	fmt.Fprintf(r.writer, "CUSTOM LIFECYCLE EVENT: %s\n", event.Type)
}

func (r *CustomTestReporter) ReportHealthSummary(summary HealthSummary) {
	fmt.Fprintf(r.writer, "CUSTOM HEALTH SUMMARY\n")
}

func TestExecutor_Integration_KeepAliveProcessLifecycle(t *testing.T) {
	// Test the complete lifecycle of keepAlive processes
	var buf bytes.Buffer
	reporter := NewConsoleReporter(&buf, true) // verbose to see process monitoring

	executor := NewExecutor(ExecutorOptions{
		Verbose:  true,
		Reporter: reporter,
	})
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "short-lived-service",
				Command: "sleep",
				Args:    []string{"0.05"}, // Very short to trigger monitoring
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "after-service",
				Command: "echo",
				Args:    []string{"Service started"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	// Wait for the keepAlive process to exit and monitoring to kick in
	time.Sleep(200 * time.Millisecond)

	output := buf.String()

	// Verify execution completed successfully
	if !strings.Contains(output, "✓ All commands completed successfully") {
		t.Errorf("Expected successful completion message, got:\n%s", output)
	}

	// Verify keepAlive process was started
	if !strings.Contains(output, "short-lived-service") {
		t.Errorf("Expected keepAlive service to be mentioned, got:\n%s", output)
	}

	// The process monitoring output might appear in stderr, so we check the execution was successful
	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}

	if len(status.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(status.Results))
	}

	// Both commands should have succeeded
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d to be successful", i)
		}
	}
}

func TestExecutor_Integration_StressTest(t *testing.T) {
	// Test with many commands to ensure no resource leaks or race conditions
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	// Create a config with many commands
	commands := make([]config.Command, 20)
	for i := range commands {
		commands[i] = config.Command{
			Name:    fmt.Sprintf("cmd-%d", i+1),
			Command: "echo",
			Args:    []string{fmt.Sprintf("Command %d output", i+1)},
			Mode:    config.ModeOnce,
		}
	}

	cfg := &config.Config{
		Version:  "1.0",
		Commands: commands,
	}

	start := time.Now()
	err := executor.Execute(ctx, cfg)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}

	if len(status.Results) != 20 {
		t.Errorf("Expected 20 results, got %d", len(status.Results))
	}

	// Verify all commands succeeded
	for i, result := range status.Results {
		if !result.Success {
			t.Errorf("Expected result %d to be successful", i)
		}
		expectedOutput := fmt.Sprintf("Command %d output", i+1)
		if !strings.Contains(result.Output, expectedOutput) {
			t.Errorf("Expected result %d output to contain '%s', got '%s'", i, expectedOutput, result.Output)
		}
	}

	// Execution should be reasonably fast (less than 5 seconds for 20 echo commands)
	if duration > 5*time.Second {
		t.Errorf("Execution took too long: %v", duration)
	}

	t.Logf("Successfully executed 20 commands in %v", duration)
}

func TestExecutor_Integration_ErrorRecovery(t *testing.T) {
	// Test that executor can be reused after a failure
	executor := NewExecutor(ExecutorOptions{})
	ctx := context.Background()

	// First execution that fails
	failingCfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "failing-command",
				Command: "false",
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, failingCfg)
	if err == nil {
		t.Fatal("Expected first execution to fail, got nil error")
	}

	// Verify failed state
	status := executor.GetStatus()
	if status.State != StateFailed {
		t.Errorf("Expected state to be Failed after first execution, got %v", status.State)
	}

	// Second execution that succeeds
	successCfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "success-command",
				Command: "echo",
				Args:    []string{"recovery test"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err = executor.Execute(ctx, successCfg)
	if err != nil {
		t.Fatalf("Expected second execution to succeed, got error: %v", err)
	}

	// Verify successful state
	status = executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected state to be Success after second execution, got %v", status.State)
	}

	// Should have results from second execution only
	if len(status.Results) != 1 {
		t.Errorf("Expected 1 result from second execution, got %d", len(status.Results))
	}

	if !status.Results[0].Success {
		t.Error("Expected second execution result to be successful")
	}
}
