package executor

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type ExecutionReporter interface {
	ReportStart(totalCommands int)
	ReportExecutionComplete(status ExecutionStatus)
	ReportExecutionSummary(status ExecutionStatus)
}

type CommandReporter interface {
	ReportCommandStart(commandName string, commandIndex int)
	ReportCommandSuccess(result ExecutionResult, commandIndex int)
	ReportCommandFailure(result ExecutionResult, commandIndex int)
}

type ProcessStatusReporter interface {
	ReportProcessStatus(processes map[string]ProcessInfo)
	ReportProcessHealth(health map[string]ProcessHealth)
	ReportProcessLifecycleEvent(event HealthMonitorEvent)
	ReportHealthSummary(summary HealthSummary)
}

type Reporter interface {
	ExecutionReporter
	CommandReporter
	ProcessStatusReporter
}

// ConsoleReporter implements Reporter for console output
type ConsoleReporter struct {
	writer  io.Writer
	verbose bool
}

// NewConsoleReporter creates a new console reporter
func NewConsoleReporter(writer io.Writer, verbose bool) *ConsoleReporter {
	return &ConsoleReporter{
		writer:  writer,
		verbose: verbose,
	}
}

// ReportStart reports the beginning of execution
func (r *ConsoleReporter) ReportStart(totalCommands int) {
	fmt.Fprintf(r.writer, "Starting execution of %d command(s)\n", totalCommands)
	if r.verbose {
		fmt.Fprintf(r.writer, "Verbose mode enabled\n")
	}
	fmt.Fprintln(r.writer)
}

// ReportCommandStart reports when a command starts executing
func (r *ConsoleReporter) ReportCommandStart(commandName string, commandIndex int) {
	fmt.Fprintf(r.writer, "[%d] Starting '%s'", commandIndex+1, commandName)
	if r.verbose {
		fmt.Fprintf(r.writer, "...")
	}
	fmt.Fprintln(r.writer)
}

// ReportCommandSuccess reports successful command completion
func (r *ConsoleReporter) ReportCommandSuccess(result ExecutionResult, commandIndex int) {
	duration := formatDuration(result.Duration)

	if r.verbose {
		fmt.Fprintf(r.writer, "[%d] ✓ '%s' completed successfully (%s)\n",
			commandIndex+1, result.Command.Name, duration)

		if result.Command.Mode == "keepAlive" {
			fmt.Fprintf(r.writer, "    Process started and running in background\n")
		}

		if result.Output != "" && result.Command.Mode != "keepAlive" {
			// Show first few lines of output in verbose mode
			lines := strings.Split(strings.TrimSpace(result.Output), "\n")
			if len(lines) > 3 {
				for i := 0; i < 3; i++ {
					fmt.Fprintf(r.writer, "    %s\n", lines[i])
				}
				fmt.Fprintf(r.writer, "    ... (%d more lines)\n", len(lines)-3)
			} else {
				for _, line := range lines {
					fmt.Fprintf(r.writer, "    %s\n", line)
				}
			}
		}
	} else {
		fmt.Fprintf(r.writer, "[%d] ✓ '%s' (%s)\n",
			commandIndex+1, result.Command.Name, duration)
	}
}

// ReportCommandFailure reports command failure
func (r *ConsoleReporter) ReportCommandFailure(result ExecutionResult, commandIndex int) {
	duration := formatDuration(result.Duration)

	fmt.Fprintf(r.writer, "[%d] ✗ '%s' failed (%s)\n",
		commandIndex+1, result.Command.Name, duration)

	// Always show failure details
	if result.ExitCode != 0 {
		fmt.Fprintf(r.writer, "    Exit code: %d\n", result.ExitCode)
	}

	if result.Error != "" {
		fmt.Fprintf(r.writer, "    Error: %s\n", result.Error)
	}

	if r.verbose && result.ErrorDetail != nil {
		fmt.Fprintf(r.writer, "    Command: %s\n", result.ErrorDetail.CommandLine)
		if result.ErrorDetail.WorkingDir != "" {
			fmt.Fprintf(r.writer, "    Working Directory: %s\n", result.ErrorDetail.WorkingDir)
		}
		if result.ErrorDetail.Stderr != "" {
			fmt.Fprintf(r.writer, "    Stderr: %s\n", result.ErrorDetail.Stderr)
		}
	}

	if result.Output != "" && r.verbose {
		fmt.Fprintf(r.writer, "    Output: %s\n", result.Output)
	}
}

// ReportExecutionComplete reports overall execution completion
func (r *ConsoleReporter) ReportExecutionComplete(status ExecutionStatus) {
	fmt.Fprintln(r.writer)

	switch status.State {
	case StateSuccess:
		fmt.Fprintf(r.writer, "✓ All commands completed successfully (%d/%d)\n",
			status.CompletedCount, status.TotalCount)
	case StateFailed:
		fmt.Fprintf(r.writer, "✗ Execution failed at command %d of %d\n",
			status.CompletedCount, status.TotalCount)
		if status.LastError != "" && r.verbose {
			fmt.Fprintf(r.writer, "Error details: %s\n", status.LastError)
		}
	default:
		fmt.Fprintf(r.writer, "Execution stopped (%d/%d completed)\n",
			status.CompletedCount, status.TotalCount)
	}
}

// ReportExecutionSummary reports final execution summary
func (r *ConsoleReporter) ReportExecutionSummary(status ExecutionStatus) {
	if !r.verbose || len(status.Results) == 0 {
		return
	}

	fmt.Fprintln(r.writer)
	fmt.Fprintln(r.writer, "Execution Summary:")
	fmt.Fprintln(r.writer, strings.Repeat("-", 50))

	successCount := 0
	totalDuration := time.Duration(0)

	for i, result := range status.Results {
		statusIcon := "✓"
		if !result.Success {
			statusIcon = "✗"
		} else {
			successCount++
		}

		totalDuration += result.Duration

		fmt.Fprintf(r.writer, "%s [%d] %-20s %8s",
			statusIcon, i+1, result.Command.Name, formatDuration(result.Duration))

		if result.Command.Mode == "keepAlive" {
			fmt.Fprintf(r.writer, " (background)")
		}

		fmt.Fprintln(r.writer)
	}

	fmt.Fprintln(r.writer, strings.Repeat("-", 50))
	fmt.Fprintf(r.writer, "Total: %d commands, %d successful, %d failed\n",
		len(status.Results), successCount, len(status.Results)-successCount)
	fmt.Fprintf(r.writer, "Total execution time: %s\n", formatDuration(totalDuration))
}

// ReportProcessStatus reports current status of active processes
func (r *ConsoleReporter) ReportProcessStatus(processes map[string]ProcessInfo) {
	if len(processes) == 0 {
		if r.verbose {
			fmt.Fprintln(r.writer, "No active processes")
		}
		return
	}

	if r.verbose {
		fmt.Fprintln(r.writer)
		fmt.Fprintln(r.writer, "Active Processes:")
		fmt.Fprintln(r.writer, strings.Repeat("-", 60))

		for name, proc := range processes {
			uptime := time.Since(proc.StartTime)
			fmt.Fprintf(r.writer, "%-20s PID: %-8d Uptime: %-12s Command: %s\n",
				name, proc.PID, formatDuration(uptime), proc.Command)
		}
		fmt.Fprintln(r.writer, strings.Repeat("-", 60))
	} else {
		fmt.Fprintf(r.writer, "Active processes: %d\n", len(processes))
	}
}

// ReportProcessHealth reports health status of monitored processes
func (r *ConsoleReporter) ReportProcessHealth(health map[string]ProcessHealth) {
	if len(health) == 0 {
		if r.verbose {
			fmt.Fprintln(r.writer, "No processes being monitored")
		}
		return
	}

	if r.verbose {
		fmt.Fprintln(r.writer)
		fmt.Fprintln(r.writer, "Process Health Status:")
		fmt.Fprintln(r.writer, strings.Repeat("-", 80))

		for name, h := range health {
			statusIcon := r.getStatusIcon(h.Status)
			fmt.Fprintf(r.writer, "%s %-20s PID: %-8d Status: %-10s Uptime: %s",
				statusIcon, name, h.PID, h.Status, formatDuration(h.UptimeDuration))

			if h.RestartCount > 0 {
				fmt.Fprintf(r.writer, " Restarts: %d", h.RestartCount)
			}

			if h.ExitCode != nil {
				fmt.Fprintf(r.writer, " Exit: %d", *h.ExitCode)
			}

			fmt.Fprintln(r.writer)

			if h.ExitReason != "" && h.Status != ProcessStatusRunning {
				fmt.Fprintf(r.writer, "    Reason: %s\n", h.ExitReason)
			}
		}
		fmt.Fprintln(r.writer, strings.Repeat("-", 80))
	}
}

// ReportProcessLifecycleEvent reports individual process lifecycle events
func (r *ConsoleReporter) ReportProcessLifecycleEvent(event HealthMonitorEvent) {
	if !r.verbose {
		return
	}

	timestamp := event.Timestamp.Format("15:04:05")
	eventIcon := r.getEventIcon(event.Type)

	fmt.Fprintf(r.writer, "[%s] %s Process '%s': %s\n",
		timestamp, eventIcon, event.ProcessID, event.Message)

	// Show additional health details for certain events
	if event.Health != nil && (event.Type == HealthEventProcessFailed ||
		event.Type == HealthEventMemoryThreshold || event.Type == HealthEventCPUThreshold) {

		if event.Health.MemoryUsage > 0 {
			fmt.Fprintf(r.writer, "    Memory: %s\n", formatBytes(event.Health.MemoryUsage))
		}
		if event.Health.CPUPercent > 0 {
			fmt.Fprintf(r.writer, "    CPU: %.1f%%\n", event.Health.CPUPercent)
		}
	}
}

// ReportHealthSummary reports overall health summary
func (r *ConsoleReporter) ReportHealthSummary(summary HealthSummary) {
	if r.verbose {
		fmt.Fprintln(r.writer)
		fmt.Fprintln(r.writer, "Health Summary:")
		fmt.Fprintf(r.writer, "  Total: %d, Running: %d, Exited: %d, Failed: %d\n",
			summary.TotalProcesses, summary.RunningProcesses,
			summary.ExitedProcesses, summary.FailedProcesses)
		fmt.Fprintf(r.writer, "  Last updated: %s\n", summary.Timestamp.Format("15:04:05"))
	}
}

// getStatusIcon returns an icon for process status
func (r *ConsoleReporter) getStatusIcon(status ProcessStatus) string {
	switch status {
	case ProcessStatusRunning:
		return "🟢"
	case ProcessStatusExited:
		return "⚪"
	case ProcessStatusFailed:
		return "🔴"
	case ProcessStatusRestarted:
		return "🔄"
	default:
		return "❓"
	}
}

// getEventIcon returns an icon for event type
func (r *ConsoleReporter) getEventIcon(eventType HealthEventType) string {
	switch eventType {
	case HealthEventProcessStarted:
		return "🚀"
	case HealthEventProcessExited:
		return "🏁"
	case HealthEventProcessFailed:
		return "💥"
	case HealthEventProcessRestarted:
		return "🔄"
	case HealthEventMemoryThreshold:
		return "📈"
	case HealthEventCPUThreshold:
		return "⚡"
	case HealthEventHealthCheckFailed:
		return "⚠️"
	default:
		return "ℹ️"
	}
}

// formatDuration formats a duration for human-readable output
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1000000.0)
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
}

// formatBytes formats bytes for human-readable output
func formatBytes(bytes int64) string {
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
