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

	// ShouldShowVersion returns true if version should be displayed
	ShouldShowVersion() bool

	// ShowVersion displays version information
	ShowVersion(version string)

	// ShouldRunInit returns true if init should be executed
	ShouldRunInit() bool

	// RunInit generates example configuration files
	RunInit() error

	// ShouldRunKill returns true if kill should be executed
	ShouldRunKill() bool

	// RunKill terminates running seqr processes
	RunKill() error

	// ShouldRunStatus returns true if status should be executed
	ShouldRunStatus() bool

	// RunStatus shows the status of running seqr processes
	RunStatus() error

	// ShouldRunWatch returns true if watch should be executed
	ShouldRunWatch() bool

	// RunWatch shows live output from running seqr processes
	RunWatch(ctx context.Context) error

	// Run executes the CLI application with the parsed options
	Run(ctx context.Context) error

	// Stop gracefully stops the CLI execution
	Stop()

	// TryDetachFromStreaming attempts to detach from active streaming sessions
	// Returns true if detachment was successful, false if no streaming was active
	TryDetachFromStreaming() bool

	// GetOptions returns the parsed CLI options
	GetOptions() CLIOptions
}
