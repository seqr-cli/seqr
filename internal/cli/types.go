package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/seqr-cli/seqr/internal/config"
	"github.com/seqr-cli/seqr/internal/executor"
)

// CLIOptions holds all command-line configuration options
type CLIOptions struct {
	ConfigFile string // Path to queue configuration file
	Verbose    bool   // Enable verbose output
	Help       bool   // Show help message
}

// CLI represents the command-line interface
type CLI struct {
	options  CLIOptions
	flagSet  *flag.FlagSet
	executor executor.Executor
	args     []string
}

// NewCLI creates a new CLI instance with default options
func NewCLI(args []string) *CLI {
	flagSet := flag.NewFlagSet("seqr", flag.ContinueOnError)

	// Suppress automatic usage output - we'll handle it ourselves
	flagSet.SetOutput(os.Stderr)

	cli := &CLI{
		options: CLIOptions{
			ConfigFile: config.DefaultConfigFile(), // ".queue.json"
			Verbose:    false,
			Help:       false,
		},
		flagSet: flagSet,
		args:    args,
	}

	cli.setupFlags()
	return cli
}

// setupFlags configures all command-line flags
func (c *CLI) setupFlags() {
	c.flagSet.StringVar(&c.options.ConfigFile, "f", c.options.ConfigFile,
		"Path to queue configuration file")
	c.flagSet.BoolVar(&c.options.Verbose, "v", c.options.Verbose,
		"Enable verbose output with execution details")
	c.flagSet.BoolVar(&c.options.Help, "h", c.options.Help,
		"Show help message")
	c.flagSet.BoolVar(&c.options.Help, "help", c.options.Help,
		"Show help message")
}

// Parse parses command-line arguments and validates options
func (c *CLI) Parse() error {
	if err := c.flagSet.Parse(c.args); err != nil {
		return fmt.Errorf("failed to parse command-line arguments: %w", err)
	}

	return c.validateOptions()
}

// validateOptions validates the parsed command-line options
func (c *CLI) validateOptions() error {
	// If help is requested, no validation needed
	if c.options.Help {
		return nil
	}

	// Only validate config file existence during Run(), not Parse()
	// This allows CLI to be created and parsed without requiring the file to exist
	// The file will be validated when actually needed in Run()

	return nil
}

// GetOptions returns the parsed CLI options
func (c *CLI) GetOptions() CLIOptions {
	return c.options
}

// ShouldShowHelp returns true if help should be displayed
func (c *CLI) ShouldShowHelp() bool {
	return c.options.Help
}

// ShowHelp displays the help message
func (c *CLI) ShowHelp() {
	fmt.Fprintf(os.Stdout, "seqr - AI-Safe Command Queue Runner\n\n")
	fmt.Fprintf(os.Stdout, "USAGE:\n")
	fmt.Fprintf(os.Stdout, "  seqr [options]\n\n")
	fmt.Fprintf(os.Stdout, "OPTIONS:\n")
	c.flagSet.PrintDefaults()
	fmt.Fprintf(os.Stdout, "\nEXAMPLES:\n")
	fmt.Fprintf(os.Stdout, "  seqr                    # Run commands from .queue.json\n")
	fmt.Fprintf(os.Stdout, "  seqr -f my-queue.json   # Run commands from custom file\n")
	fmt.Fprintf(os.Stdout, "  seqr -v                 # Run with verbose output\n")
	fmt.Fprintf(os.Stdout, "  seqr -f queue.json -v   # Custom file with verbose output\n")
}

// Run executes the CLI application with the parsed options
func (c *CLI) Run(ctx context.Context) error {
	// Show help if requested
	if c.ShouldShowHelp() {
		c.ShowHelp()
		return nil
	}

	// Load configuration
	cfg, err := config.LoadFromFile(c.options.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create executor with CLI options
	executorOpts := executor.ExecutorOptions{
		Verbose: c.options.Verbose,
	}
	c.executor = executor.NewExecutor(executorOpts)

	// Execute the command queue
	if err := c.executor.Execute(ctx, cfg); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	return nil
}

// Stop gracefully stops the CLI execution
func (c *CLI) Stop() {
	if c.executor != nil {
		c.executor.Stop()
	}
}
