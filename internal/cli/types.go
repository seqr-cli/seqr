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
	Status     bool   // Show status of running seqr processes
	Watch      bool   // Watch live processes and their output
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
			Status:     false,
			Watch:      false,
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
	c.flagSet.BoolVar(&c.options.Status, "status", c.options.Status,
		"Show status of running seqr processes")
	c.flagSet.BoolVar(&c.options.Watch, "watch", c.options.Watch,
		"Watch live processes and their real-time output")
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
	// If help, version, init, kill, status, or watch is requested, no validation needed
	if c.options.Help || c.options.Version || c.options.Init || c.options.Kill || c.options.Status || c.options.Watch {
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

// ShouldRunStatus returns true if status should be executed
func (c *CLI) ShouldRunStatus() bool {
	return c.options.Status
}

// ShouldRunWatch returns true if watch should be executed
func (c *CLI) ShouldRunWatch() bool {
	return c.options.Watch
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
	fmt.Fprintf(os.Stdout, "  seqr --kill               # Kill running seqr processes\n")
	fmt.Fprintf(os.Stdout, "  seqr --status             # Show status of running seqr processes\n")
	fmt.Fprintf(os.Stdout, "  seqr --watch              # Watch live processes and their output\n\n")
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
	fmt.Fprintf(os.Stdout, "        \"concurrent\": true|false (optional),\n")
	fmt.Fprintf(os.Stdout, "        \"workDir\": \"./path\" (optional),\n")
	fmt.Fprintf(os.Stdout, "        \"env\": {\"KEY\": \"value\"} (optional)\n")
	fmt.Fprintf(os.Stdout, "      }\n")
	fmt.Fprintf(os.Stdout, "    ]\n")
	fmt.Fprintf(os.Stdout, "  }\n\n")
	fmt.Fprintf(os.Stdout, "EXECUTION MODES:\n")
	fmt.Fprintf(os.Stdout, "  once      - Run command once and wait for completion\n")
	fmt.Fprintf(os.Stdout, "  keepAlive - Start command and keep running in background\n\n")
	fmt.Fprintf(os.Stdout, "CONCURRENT EXECUTION:\n")
	fmt.Fprintf(os.Stdout, "  Commands with \"concurrent\": true will run in parallel with other\n")
	fmt.Fprintf(os.Stdout, "  concurrent commands. Sequential commands (concurrent: false or omitted)\n")
	fmt.Fprintf(os.Stdout, "  will wait for all previous commands to complete before starting.\n\n")
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
			fmt.Printf("[%s] [seqr] [system] KeepAlive processes running in background, main execution complete\n", timestamp)
			fmt.Printf("[%s] [seqr] [system] Use 'seqr --kill' to terminate background processes\n", timestamp)
		}

		// For background execution, we don't wait for keepAlive processes
		// They continue running in the background while the main execution completes
		// Users can use 'seqr --kill' to terminate them when needed
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

// RunStatus shows the status of running seqr processes
func (c *CLI) RunStatus() error {
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

	fmt.Fprintf(os.Stdout, "seqr Process Status\n")
	fmt.Fprintf(os.Stdout, "==================\n\n")
	fmt.Fprintf(os.Stdout, "Found %d running seqr process(es):\n\n", len(processes))

	for pid, info := range processes {
		uptime := time.Since(info.StartTime)
		fmt.Fprintf(os.Stdout, "PID %d: %s\n", pid, info.Name)
		fmt.Fprintf(os.Stdout, "  Command: %s %v\n", info.Command, info.Args)
		fmt.Fprintf(os.Stdout, "  Mode: %s\n", info.Mode)
		if info.WorkDir != "" {
			fmt.Fprintf(os.Stdout, "  Working Directory: %s\n", info.WorkDir)
		}
		fmt.Fprintf(os.Stdout, "  Started: %s (%s ago)\n",
			info.StartTime.Format("2006-01-02 15:04:05"),
			uptime.Round(time.Second))
		fmt.Fprintf(os.Stdout, "  Status: Running\n")
		fmt.Fprintf(os.Stdout, "\n")
	}

	fmt.Fprintf(os.Stdout, "Use 'seqr --kill' to terminate all processes\n")
	return nil
}

// RunWatch shows live output from running seqr processes
func (c *CLI) RunWatch(ctx context.Context) error {
	processManager := executor.NewProcessManager()
	logger := executor.NewBackgroundLogger()

	// Get all running processes
	processes, err := processManager.GetAllRunningProcesses()
	if err != nil {
		return fmt.Errorf("failed to get running processes: %w", err)
	}

	// Get all available logs (including from stopped processes)
	availableLogs, err := logger.ListAvailableLogs()
	if err != nil {
		// Don't fail if we can't read logs, just continue
		availableLogs = []string{}
	}

	totalProcesses := len(processes)
	totalLogs := len(availableLogs)

	if totalProcesses == 0 && totalLogs == 0 {
		fmt.Fprintf(os.Stdout, "No seqr processes are currently running\n")
		fmt.Fprintf(os.Stdout, "Use 'seqr' to start processes, then 'seqr --watch' to monitor them\n")
		return nil
	}

	if totalProcesses > 0 {
		fmt.Fprintf(os.Stdout, "🔍 Watching %d running seqr process(es):\n\n", totalProcesses)

		for pid, info := range processes {
			uptime := time.Since(info.StartTime)
			fmt.Fprintf(os.Stdout, "📊 PID %d: %s\n", pid, info.Name)
			fmt.Fprintf(os.Stdout, "   Command: %s %v\n", info.Command, info.Args)
			fmt.Fprintf(os.Stdout, "   Mode: %s\n", info.Mode)
			if info.WorkDir != "" {
				fmt.Fprintf(os.Stdout, "   Working Directory: %s\n", info.WorkDir)
			}
			fmt.Fprintf(os.Stdout, "   Started: %s (%s ago)\n",
				info.StartTime.Format("2006-01-02 15:04:05"),
				uptime.Round(time.Second))
			fmt.Fprintf(os.Stdout, "   Status: Running\n")

			// Show recent logs for this process
			recentLogs, err := logger.ReadRecentLogs(info.Name, 5)
			if err == nil && len(recentLogs) > 0 {
				fmt.Fprintf(os.Stdout, "   Recent Output:\n")
				for _, logLine := range recentLogs {
					fmt.Fprintf(os.Stdout, "     %s\n", logLine)
				}
			}
			fmt.Fprintf(os.Stdout, "\n")
		}
	}

	// Show information about available logs from stopped processes
	stoppedProcessLogs := []string{}
	for _, logName := range availableLogs {
		isRunning := false
		for _, info := range processes {
			if info.Name == logName {
				isRunning = true
				break
			}
		}
		if !isRunning {
			stoppedProcessLogs = append(stoppedProcessLogs, logName)
		}
	}

	if len(stoppedProcessLogs) > 0 {
		fmt.Fprintf(os.Stdout, "📁 Log files available from %d stopped process(es):\n", len(stoppedProcessLogs))
		for _, logName := range stoppedProcessLogs {
			logInfo, err := logger.GetLogInfo(logName)
			if err == nil {
				fmt.Fprintf(os.Stdout, "   📄 %s (%s, %s)\n",
					logName,
					logInfo.ModTime().Format("2006-01-02 15:04:05"),
					formatFileSize(logInfo.Size()))
			} else {
				fmt.Fprintf(os.Stdout, "   📄 %s\n", logName)
			}
		}
		fmt.Fprintf(os.Stdout, "\n")
	}

	if totalProcesses > 0 {
		fmt.Fprintf(os.Stdout, "🎯 Live output will appear below as processes generate it:\n")
		fmt.Fprintf(os.Stdout, "💡 Press Ctrl+C to stop watching\n\n")

		// For now, just show current status and recent logs. In a future enhancement, we could
		// implement real-time monitoring of process output by connecting to
		// running processes' stdout/stderr streams.

		// Wait for context cancellation (Ctrl+C)
		<-ctx.Done()
		fmt.Fprintf(os.Stdout, "\n👋 Stopped watching processes\n")
	} else {
		fmt.Fprintf(os.Stdout, "💡 Use 'seqr' to start processes, then 'seqr --watch' to monitor them\n")
		fmt.Fprintf(os.Stdout, "📁 Log files are preserved in: %s\n", logger.GetLogDir())
	}

	return nil
}

// formatFileSize formats a file size in human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Stop gracefully stops the CLI execution
func (c *CLI) Stop() {
	if c.executor != nil {
		c.executor.Stop()
	}
}

// TryDetachFromStreaming attempts to detach from active streaming sessions
// Returns true if detachment was successful, false if no streaming was active
func (c *CLI) TryDetachFromStreaming() bool {
	if c.executor == nil {
		return false
	}

	if !c.executor.HasActiveStreaming() {
		return false
	}

	c.executor.DetachFromStreaming()
	return true
}
