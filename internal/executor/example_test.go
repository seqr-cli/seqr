package executor

import (
	"context"
	"fmt"
	"os"

	"github.com/seqr-cli/seqr/internal/config"
)

// ExampleConsoleReporter demonstrates the execution reporting functionality
func ExampleConsoleReporter() {
	// Create a console reporter for normal (non-verbose) output
	reporter := NewConsoleReporter(os.Stdout, false)

	// Create executor with the reporter
	executor := NewExecutor(ExecutorOptions{
		Reporter: reporter,
		Verbose:  false,
	})

	// Create a simple configuration
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "hello",
				Command: "echo",
				Args:    []string{"Hello, World!"},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "date",
				Command: "date",
				Mode:    config.ModeOnce,
			},
		},
	}

	// Execute the commands
	ctx := context.Background()
	err := executor.Execute(ctx, cfg)
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		return
	}

	fmt.Println("Execution completed successfully!")
}

// ExampleConsoleReporter_verbose demonstrates verbose execution reporting
func ExampleConsoleReporter_verbose() {
	// Create a console reporter for verbose output
	reporter := NewConsoleReporter(os.Stdout, true)

	// Create executor with the reporter
	executor := NewExecutor(ExecutorOptions{
		Reporter: reporter,
		Verbose:  true,
	})

	// Create a configuration with mixed command modes
	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "setup",
				Command: "echo",
				Args:    []string{"Setting up environment..."},
				Mode:    config.ModeOnce,
			},
			{
				Name:    "background-service",
				Command: "sleep",
				Args:    []string{"0.1"}, // Short sleep for example
				Mode:    config.ModeKeepAlive,
			},
			{
				Name:    "finish",
				Command: "echo",
				Args:    []string{"Setup complete!"},
				Mode:    config.ModeOnce,
			},
		},
	}

	// Execute the commands
	ctx := context.Background()
	err := executor.Execute(ctx, cfg)
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		return
	}

	fmt.Println("All commands executed successfully!")
}
