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

var version = "dev"

func main() {
	cliApp := cli.NewCLI(os.Args[1:])

	if err := cliApp.Parse(); err != nil {
		if isFlagError(err) {
			os.Exit(2)
		}
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}

	if cliApp.ShouldShowHelp() {
		cliApp.ShowHelp()
		os.Exit(0)
	}

	if cliApp.ShouldShowVersion() {
		cliApp.ShowVersion(version)
		os.Exit(0)
	}

	if cliApp.ShouldRunInit() {
		if err := cliApp.RunInit(); err != nil {
			os.Stderr.WriteString("Error: " + err.Error() + "\n")
			os.Exit(1)
		}
		os.Exit(0)
	}

	if cliApp.ShouldRunKill() {
		if err := cliApp.RunKill(); err != nil {
			os.Stderr.WriteString("Error: " + err.Error() + "\n")
			os.Exit(1)
		}
		os.Exit(0)
	}

	if cliApp.ShouldRunStatus() {
		if err := cliApp.RunStatus(); err != nil {
			os.Stderr.WriteString("Error: " + err.Error() + "\n")
			os.Exit(1)
		}
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		signalCount := 0
		for {
			<-sigChan
			signalCount++

			if signalCount == 1 {
				// First signal: try to detach from streaming if active
				if cliApp.TryDetachFromStreaming() {
					// Successfully detached from streaming, continue running
					continue
				} else {
					// No active streaming or detachment failed, proceed with normal shutdown
					cancel()
					cliApp.Stop()
					return
				}
			} else {
				// Second signal: force shutdown
				cancel()
				cliApp.Stop()
				return
			}
		}
	}()
	if err := cliApp.Run(ctx); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func isFlagError(err error) bool {
	return err == flag.ErrHelp || strings.Contains(err.Error(), "flag provided but not defined") ||
		strings.Contains(err.Error(), "flag needs an argument")
}
