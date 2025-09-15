package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
	"github.com/seqr-cli/seqr/internal/executor"
)

// CLIOptions holds all command-line configuration options
type CLIOptions struct {
	ConfigFile string // Path to queue configuration file
	Verbose    bool   // Enable verbose output
	Help       bool   // Show help message
	Version    bool   // Show version information
	Init       bool   // Generate example queue configuration files
	Kill       bool   // Kill running seqr processes
}

// CLI represents the command-line interface
type CLI struct {
	options  CLIOptions
	flagSet  *flag.FlagSet
	executor *executor.Executor
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
			Version:    false,
			Init:       false,
			Kill:       false,
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
	c.flagSet.BoolVar(&c.options.Verbose, "verbose", c.options.Verbose,
		"Enable verbose output with execution details")
	c.flagSet.BoolVar(&c.options.Help, "h", c.options.Help,
		"Show help message")
	c.flagSet.BoolVar(&c.options.Help, "help", c.options.Help,
		"Show help message")
	c.flagSet.BoolVar(&c.options.Version, "version", c.options.Version,
		"Show version information")
	c.flagSet.BoolVar(&c.options.Init, "init", c.options.Init,
		"Generate example queue configuration files")
	c.flagSet.BoolVar(&c.options.Kill, "kill", c.options.Kill,
		"Kill running seqr processes")
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
	// If help, version, init, or kill is requested, no validation needed
	if c.options.Help || c.options.Version || c.options.Init || c.options.Kill {
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

// ShouldShowVersion returns true if version should be displayed
func (c *CLI) ShouldShowVersion() bool {
	return c.options.Version
}

// ShouldRunInit returns true if init should be executed
func (c *CLI) ShouldRunInit() bool {
	return c.options.Init
}

// ShouldRunKill returns true if kill should be executed
func (c *CLI) ShouldRunKill() bool {
	return c.options.Kill
}

// ShowVersion displays version information
func (c *CLI) ShowVersion(version string) {
	fmt.Fprintf(os.Stdout, "seqr version %s\n", version)
}

// ShowHelp displays the help message
func (c *CLI) ShowHelp() {
	fmt.Fprintf(os.Stdout, "seqr - AI-Safe Command Queue Runner\n\n")
	fmt.Fprintf(os.Stdout, "DESCRIPTION:\n")
	fmt.Fprintf(os.Stdout, "  Execute commands sequentially from a JSON configuration file.\n")
	fmt.Fprintf(os.Stdout, "  Supports both one-time commands and long-running background processes.\n\n")
	fmt.Fprintf(os.Stdout, "USAGE:\n")
	fmt.Fprintf(os.Stdout, "  seqr [options]\n\n")
	fmt.Fprintf(os.Stdout, "OPTIONS:\n")
	c.flagSet.PrintDefaults()
	fmt.Fprintf(os.Stdout, "\nEXAMPLES:\n")
	fmt.Fprintf(os.Stdout, "  seqr                      # Run commands from .queue.json\n")
	fmt.Fprintf(os.Stdout, "  seqr -f my-queue.json     # Run commands from custom file\n")
	fmt.Fprintf(os.Stdout, "  seqr -v                   # Run with verbose output\n")
	fmt.Fprintf(os.Stdout, "  seqr --verbose            # Run with verbose output (long form)\n")
	fmt.Fprintf(os.Stdout, "  seqr -f queue.json -v     # Custom file with verbose output\n")
	fmt.Fprintf(os.Stdout, "  seqr --init               # Generate example configuration files\n")
	fmt.Fprintf(os.Stdout, "  seqr --kill               # Kill running seqr processes\n\n")
	fmt.Fprintf(os.Stdout, "CONFIGURATION:\n")
	fmt.Fprintf(os.Stdout, "  The queue file should be a JSON file with the following structure:\n")
	fmt.Fprintf(os.Stdout, "  {\n")
	fmt.Fprintf(os.Stdout, "    \"version\": \"1.0\",\n")
	fmt.Fprintf(os.Stdout, "    \"commands\": [\n")
	fmt.Fprintf(os.Stdout, "      {\n")
	fmt.Fprintf(os.Stdout, "        \"name\": \"command-name\",\n")
	fmt.Fprintf(os.Stdout, "        \"command\": \"executable\",\n")
	fmt.Fprintf(os.Stdout, "        \"args\": [\"arg1\", \"arg2\"],\n")
	fmt.Fprintf(os.Stdout, "        \"mode\": \"once|keepAlive\",\n")
	fmt.Fprintf(os.Stdout, "        \"workDir\": \"./path\" (optional),\n")
	fmt.Fprintf(os.Stdout, "        \"env\": {\"KEY\": \"value\"} (optional)\n")
	fmt.Fprintf(os.Stdout, "      }\n")
	fmt.Fprintf(os.Stdout, "    ]\n")
	fmt.Fprintf(os.Stdout, "  }\n\n")
	fmt.Fprintf(os.Stdout, "EXECUTION MODES:\n")
	fmt.Fprintf(os.Stdout, "  once      - Run command once and wait for completion\n")
	fmt.Fprintf(os.Stdout, "  keepAlive - Start command and keep running in background\n\n")
	fmt.Fprintf(os.Stdout, "EXIT CODES:\n")
	fmt.Fprintf(os.Stdout, "  0 - All commands executed successfully\n")
	fmt.Fprintf(os.Stdout, "  1 - Command execution failed or configuration error\n")
	fmt.Fprintf(os.Stdout, "  2 - Invalid command-line arguments\n")
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
	c.executor = executor.NewExecutor(c.options.Verbose)

	// Execute the command queue
	if err := c.executor.Execute(ctx, cfg); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Check if there are any active keepAlive processes running
	if c.executor.HasActiveKeepAliveProcesses() {
		if c.options.Verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [seqr] [system] KeepAlive processes running, waiting for termination signal...\n", timestamp)
		}

		// Wait for context cancellation when there are keepAlive processes
		<-ctx.Done()

		if c.options.Verbose {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] [seqr] [system] Received termination signal, shutting down...\n", timestamp)
		}
	}

	return nil
}

// RunInit generates example configuration files
func (c *CLI) RunInit() error {
	generator := config.NewTemplateGenerator()
	return generator.GenerateAllTemplates()
}

// RunKill terminates running seqr processes
func (c *CLI) RunKill() error {
	processManager := executor.NewProcessManager()

	// Get all running processes
	processes, err := processManager.GetAllRunningProcesses()
	if err != nil {
		return fmt.Errorf("failed to get running processes: %w", err)
	}

	if len(processes) == 0 {
		fmt.Fprintf(os.Stdout, "No seqr processes are currently running\n")
		return nil
	}

	fmt.Fprintf(os.Stdout, "Found %d running seqr process(es):\n", len(processes))
	for pid, info := range processes {
		fmt.Fprintf(os.Stdout, "  PID %d: %s (%s %v) - started %s\n",
			pid, info.Name, info.Command, info.Args, info.StartTime.Format("15:04:05"))
	}

	fmt.Fprintf(os.Stdout, "\nTerminating processes gracefully (SIGTERM first, then SIGKILL after timeout)...\n")

	// Kill all processes with graceful termination (SIGTERM first, then SIGKILL after timeout)
	if err := processManager.KillAllProcesses(true); err != nil {
		return fmt.Errorf("failed to kill processes: %w", err)
	}

	fmt.Fprintf(os.Stdout, "All seqr processes have been terminated\n")
	return nil
}

// Stop gracefully stops the CLI execution
func (c *CLI) Stop() {
	if c.executor != nil {
		c.executor.Stop()
	}
}
