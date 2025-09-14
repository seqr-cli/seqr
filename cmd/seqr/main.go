package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/seqr-cli/seqr/internal/cli"
)

func main() {
	// Create CLI with command-line arguments (excluding program name)
	cliApp := cli.NewCLI(os.Args[1:])

	// Parse command-line arguments
	if err := cliApp.Parse(); err != nil {
		// For flag parsing errors, show usage and exit with code 2 (standard for flag errors)
		if isFlagError(err) {
			os.Exit(2)
		}
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Show help if requested
	if cliApp.ShouldShowHelp() {
		cliApp.ShowHelp()
		os.Exit(0)
	}

	// Set up context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
		cliApp.Stop()
	}()

	// Run the CLI application
	if err := cliApp.Run(ctx); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

// isFlagError checks if the error is a flag parsing error
func isFlagError(err error) bool {
	return err == flag.ErrHelp || strings.Contains(err.Error(), "flag provided but not defined") ||
		strings.Contains(err.Error(), "flag needs an argument")
}
