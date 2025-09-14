package cli

import "context"

// Interface defines the contract for CLI implementations
type Interface interface {
	// Parse parses command-line arguments and validates options
	Parse() error

	// ShouldShowHelp returns true if help should be displayed
	ShouldShowHelp() bool

	// ShowHelp displays the help message
	ShowHelp()

	// Run executes the CLI application with the parsed options
	Run(ctx context.Context) error

	// Stop gracefully stops the CLI execution
	Stop()

	// GetOptions returns the parsed CLI options
	GetOptions() CLIOptions
}
