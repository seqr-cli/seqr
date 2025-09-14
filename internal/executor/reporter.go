package executor

import (
	"fmt"
	"io"
)

type Reporter interface {
	ReportStart(totalCommands int)
	ReportCommandStart(commandName string, commandIndex int)
	ReportCommandSuccess(result ExecutionResult, commandIndex int)
	ReportCommandFailure(result ExecutionResult, commandIndex int)
	ReportExecutionComplete(status ExecutionStatus)
}

type ConsoleReporter struct {
	writer  io.Writer
	verbose bool
}

func NewConsoleReporter(writer io.Writer, verbose bool) *ConsoleReporter {
	return &ConsoleReporter{
		writer:  writer,
		verbose: verbose,
	}
}

func (r *ConsoleReporter) ReportStart(totalCommands int) {
	if r.verbose {
		fmt.Fprintf(r.writer, "Starting execution of %d commands\n", totalCommands)
	}
}

func (r *ConsoleReporter) ReportCommandStart(commandName string, commandIndex int) {
	fmt.Fprintf(r.writer, "[%d] Starting: %s\n", commandIndex+1, commandName)
}

func (r *ConsoleReporter) ReportCommandSuccess(result ExecutionResult, commandIndex int) {
	fmt.Fprintf(r.writer, "[%d] ✓ %s (%v)\n", commandIndex+1, result.Command.Name, result.Duration.Round(10))
	if r.verbose && result.Output != "" {
		fmt.Fprintf(r.writer, "    Output: %s\n", result.Output)
	}
}

func (r *ConsoleReporter) ReportCommandFailure(result ExecutionResult, commandIndex int) {
	fmt.Fprintf(r.writer, "[%d] ✗ %s failed: %s\n", commandIndex+1, result.Command.Name, result.Error)
	if r.verbose && result.Output != "" {
		fmt.Fprintf(r.writer, "    Output: %s\n", result.Output)
	}
}

func (r *ConsoleReporter) ReportExecutionComplete(status ExecutionStatus) {
	if status.State == StateSuccess {
		fmt.Fprintf(r.writer, "All commands completed successfully\n")
	} else {
		fmt.Fprintf(r.writer, "Execution failed: %s\n", status.LastError)
	}
}
