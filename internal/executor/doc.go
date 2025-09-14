// Package executor provides a sequential command execution engine with clear state management.
//
// The executor implements a fail-fast sequential state machine that processes commands
// one at a time in the order defined in the configuration. If any command fails,
// execution stops immediately and reports the failure.
//
// # Architecture
//
// The execution engine follows a simple state machine pattern:
//
//	Ready → Running → Success → Ready (next command)
//	  ↓       ↓
//	  ↓     Failed (stop execution)
//	  ↓
//	Stopped (graceful shutdown)
//
// # States
//
//   - Ready: Executor is ready to run the next command
//   - Running: A command is currently executing
//   - Success: Last command completed successfully, ready for next
//   - Failed: A command failed, execution has stopped
//
// # Thread Safety
//
// All executor methods are thread-safe. The GetStatus() method returns a copy
// of the current status to prevent race conditions. The Stop() method can be
// called from any goroutine to request graceful shutdown.
//
// # Execution Reporting
//
// The executor provides comprehensive execution reporting through the Reporter interface.
// By default, a ConsoleReporter is used that outputs execution progress and results
// to stdout. The reporting includes:
//
//   - Execution start notification
//   - Command start notifications with progress indicators
//   - Command completion status (success/failure) with timing
//   - Detailed error information for failed commands
//   - Execution summary with statistics (in verbose mode)
//
// # Usage Example
//
//	// Create custom reporter or use default
//	reporter := NewConsoleReporter(os.Stdout, true) // verbose mode
//
//	opts := ExecutorOptions{
//		Verbose:  true,
//		Timeout:  30 * time.Second,
//		Reporter: reporter, // optional, defaults to console reporter
//	}
//
//	executor := NewExecutor(opts)
//
//	ctx := context.Background()
//	err := executor.Execute(ctx, config)
//	// Reporter automatically handles all output during execution
//	if err != nil {
//		// Additional error handling if needed
//		status := executor.GetStatus()
//		fmt.Printf("Final status: %v\n", status.State)
//	}
package executor
