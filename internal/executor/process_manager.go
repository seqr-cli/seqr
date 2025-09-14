package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

// ProcessManager handles the execution of commands with different modes
// It provides mode-based routing and lifecycle management for processes
type ProcessManager interface {
	// ExecuteCommand executes a single command based on its mode
	ExecuteCommand(ctx context.Context, cmd config.Command) (ExecutionResult, error)

	// GetActiveProcesses returns information about currently running keepAlive processes
	GetActiveProcesses() map[string]ProcessInfo

	// TerminateAll gracefully terminates all active keepAlive processes
	TerminateAll() error

	// TerminateProcess terminates a specific keepAlive process by name
	TerminateProcess(name string) error

	// GetProcessHealth returns health status for all monitored processes
	GetProcessHealth() map[string]ProcessHealth

	// StartHealthMonitoring starts the background health monitoring system
	StartHealthMonitoring(ctx context.Context) error

	// StopHealthMonitoring stops the background health monitoring system
	StopHealthMonitoring() error

	// ReportCurrentStatus reports the current status of all processes
	ReportCurrentStatus()

	// ReportHealthStatus reports the health status of all monitored processes
	ReportHealthStatus()

	// GetHealthEvents returns a channel for receiving health monitoring events
	GetHealthEvents() <-chan HealthMonitorEvent

	// EnableLifecycleReporting enables automatic reporting of lifecycle events
	EnableLifecycleReporting(enabled bool)

	// GetHealthSummary returns a summary of process health status
	GetHealthSummary() HealthSummary
}

// ProcessInfo contains information about a running process
type ProcessInfo struct {
	Name      string    `json:"name"`
	PID       int       `json:"pid"`
	StartTime time.Time `json:"startTime"`
	Command   string    `json:"command"`
}

// ProcessHealth contains health monitoring information for a process
type ProcessHealth struct {
	Name           string        `json:"name"`
	PID            int           `json:"pid"`
	Status         ProcessStatus `json:"status"`
	StartTime      time.Time     `json:"startTime"`
	LastCheckTime  time.Time     `json:"lastCheckTime"`
	RestartCount   int           `json:"restartCount"`
	ExitCode       *int          `json:"exitCode,omitempty"`
	ExitTime       *time.Time    `json:"exitTime,omitempty"`
	ExitReason     string        `json:"exitReason,omitempty"`
	MemoryUsage    int64         `json:"memoryUsage,omitempty"` // in bytes
	CPUPercent     float64       `json:"cpuPercent,omitempty"`  // percentage
	UptimeDuration time.Duration `json:"uptimeDuration"`
}

// ProcessStatus represents the current status of a monitored process
type ProcessStatus string

const (
	ProcessStatusRunning   ProcessStatus = "running"
	ProcessStatusExited    ProcessStatus = "exited"
	ProcessStatusFailed    ProcessStatus = "failed"
	ProcessStatusUnknown   ProcessStatus = "unknown"
	ProcessStatusRestarted ProcessStatus = "restarted"
)

// HealthMonitorEvent represents an event from the health monitoring system
type HealthMonitorEvent struct {
	Type      HealthEventType `json:"type"`
	ProcessID string          `json:"processId"`
	Timestamp time.Time       `json:"timestamp"`
	Message   string          `json:"message"`
	Health    *ProcessHealth  `json:"health,omitempty"`
}

// HealthEventType categorizes different types of health monitoring events
type HealthEventType string

const (
	HealthEventProcessStarted    HealthEventType = "process_started"
	HealthEventProcessExited     HealthEventType = "process_exited"
	HealthEventProcessFailed     HealthEventType = "process_failed"
	HealthEventProcessRestarted  HealthEventType = "process_restarted"
	HealthEventHealthCheckFailed HealthEventType = "health_check_failed"
	HealthEventMemoryThreshold   HealthEventType = "memory_threshold"
	HealthEventCPUThreshold      HealthEventType = "cpu_threshold"
)

// HealthMonitorOptions contains configuration for the health monitoring system
type HealthMonitorOptions struct {
	CheckInterval   time.Duration `json:"checkInterval"`   // How often to check process health
	MemoryThreshold int64         `json:"memoryThreshold"` // Memory usage threshold in bytes
	CPUThreshold    float64       `json:"cpuThreshold"`    // CPU usage threshold as percentage
	EnableRestart   bool          `json:"enableRestart"`   // Whether to restart failed processes
	MaxRestarts     int           `json:"maxRestarts"`     // Maximum number of restart attempts
	RestartDelay    time.Duration `json:"restartDelay"`    // Delay between restart attempts
	EventBufferSize int           `json:"eventBufferSize"` // Size of event buffer channel
	EnableMetrics   bool          `json:"enableMetrics"`   // Whether to collect CPU/memory metrics
}

// processManager implements the ProcessManager interface
type processManager struct {
	mu                sync.RWMutex
	activeProcs       map[string]*managedProcess
	options           ProcessManagerOptions
	onceExecutor      OnceExecutor
	keepAliveExecutor KeepAliveExecutor

	// Health monitoring system
	healthMonitor  *healthMonitor
	monitorOptions HealthMonitorOptions
	eventChannel   chan HealthMonitorEvent
	monitorCtx     context.Context
	monitorCancel  context.CancelFunc
	monitorStarted bool

	// Reporting system
	reporter                  Reporter
	lifecycleReportingEnabled bool
}

// ProcessManagerOptions contains configuration for the process manager
type ProcessManagerOptions struct {
	Verbose    bool
	WorkingDir string
	Timeout    time.Duration
	Reporter   Reporter // Optional reporter for process status and lifecycle events
}

// managedProcess wraps an exec.Cmd with additional metadata
type managedProcess struct {
	cmd       *exec.Cmd
	name      string
	startTime time.Time
	command   config.Command
	done      chan error
}

// OnceExecutor handles synchronous command execution
type OnceExecutor interface {
	Execute(ctx context.Context, cmd config.Command, opts ProcessManagerOptions) (ExecutionResult, error)
}

// KeepAliveExecutor handles asynchronous command execution and monitoring
type KeepAliveExecutor interface {
	Execute(ctx context.Context, cmd config.Command, opts ProcessManagerOptions) (ExecutionResult, *managedProcess, error)
}

// NewProcessManager creates a new process manager with the specified options
func NewProcessManager(opts ProcessManagerOptions) ProcessManager {
	// Set default health monitoring options
	monitorOpts := HealthMonitorOptions{
		CheckInterval:   1 * time.Second,   // More frequent checks for better responsiveness
		MemoryThreshold: 100 * 1024 * 1024, // 100MB
		CPUThreshold:    80.0,              // 80%
		EnableRestart:   false,             // Disabled by default for safety
		MaxRestarts:     3,
		RestartDelay:    5 * time.Second,
		EventBufferSize: 100,
		EnableMetrics:   opts.Verbose, // Enable metrics in verbose mode
	}

	// Use provided reporter or default to console reporter
	reporter := opts.Reporter
	if reporter == nil {
		reporter = NewConsoleReporter(os.Stdout, opts.Verbose)
	}

	pm := &processManager{
		activeProcs:               make(map[string]*managedProcess),
		options:                   opts,
		onceExecutor:              NewOnceExecutor(),
		keepAliveExecutor:         NewKeepAliveExecutor(),
		monitorOptions:            monitorOpts,
		eventChannel:              make(chan HealthMonitorEvent, monitorOpts.EventBufferSize),
		reporter:                  reporter,
		lifecycleReportingEnabled: opts.Verbose, // Enable by default in verbose mode
	}

	pm.healthMonitor = newHealthMonitor(pm, monitorOpts)
	return pm
}

// ExecuteCommand routes command execution based on the command mode
func (pm *processManager) ExecuteCommand(ctx context.Context, cmd config.Command) (ExecutionResult, error) {
	switch cmd.Mode {
	case config.ModeOnce:
		return pm.executeOnceCommand(ctx, cmd)
	case config.ModeKeepAlive:
		return pm.executeKeepAliveCommand(ctx, cmd)
	default:
		return pm.handleUnsupportedMode(cmd)
	}
}

// executeOnceCommand handles synchronous command execution
func (pm *processManager) executeOnceCommand(ctx context.Context, cmd config.Command) (ExecutionResult, error) {
	return pm.onceExecutor.Execute(ctx, cmd, pm.options)
}

// executeKeepAliveCommand handles asynchronous command execution
func (pm *processManager) executeKeepAliveCommand(ctx context.Context, cmd config.Command) (ExecutionResult, error) {
	result, managedProc, err := pm.keepAliveExecutor.Execute(ctx, cmd, pm.options)
	if err != nil {
		return result, err
	}

	// Track the process for lifecycle management
	pm.trackProcess(managedProc)

	// Start monitoring the process in the background
	go pm.monitorProcess(managedProc)

	return result, nil
}

// handleUnsupportedMode creates an error result for unsupported execution modes
func (pm *processManager) handleUnsupportedMode(cmd config.Command) (ExecutionResult, error) {
	result := ExecutionResult{
		Command:   cmd,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   false,
		ExitCode:  -1,
		Error:     fmt.Sprintf("unsupported execution mode: %s", cmd.Mode),
	}

	result.ErrorDetail = createErrorDetail(cmd, nil,
		fmt.Errorf("unsupported execution mode: %s", cmd.Mode), "", "")
	result.ErrorDetail.Type = ErrorTypeUnsupportedMode

	enhancedErr := fmt.Errorf("command '%s' has unsupported execution mode '%s': supported modes are 'once' and 'keepAlive'\n  Command: %s",
		cmd.Name, cmd.Mode, result.ErrorDetail.CommandLine)

	return result, enhancedErr
}

// trackProcess adds a process to the active processes map
func (pm *processManager) trackProcess(proc *managedProcess) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.activeProcs[proc.name] = proc

	// Notify health monitor about new process immediately
	if pm.healthMonitor != nil {
		procInfo := ProcessInfo{
			Name:      proc.name,
			PID:       proc.cmd.Process.Pid,
			StartTime: proc.startTime,
			Command:   buildCommandLine(proc.command),
		}
		pm.healthMonitor.NotifyProcessStarted(proc.name, procInfo)

		// Also send event if monitoring is started
		if pm.monitorStarted {
			event := HealthMonitorEvent{
				Type:      HealthEventProcessStarted,
				ProcessID: proc.name,
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("Process started (PID %d)", proc.cmd.Process.Pid),
			}

			select {
			case pm.eventChannel <- event:
			default:
				// Channel full, continue without blocking
			}
		}
	}
}

// untrackProcess removes a process from the active processes map
func (pm *processManager) untrackProcess(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.activeProcs, name)
}

// monitorProcess monitors a keepAlive process and handles its lifecycle
func (pm *processManager) monitorProcess(proc *managedProcess) {
	err := proc.cmd.Wait()
	pm.untrackProcess(proc.name)

	// Send completion signal
	select {
	case proc.done <- err:
	default:
		// Channel might be closed or not being read
	}

	// Notify health monitor about process exit immediately
	exitTime := time.Now()
	if pm.healthMonitor != nil {
		pm.healthMonitor.NotifyProcessExited(proc.name, exitTime, pm.formatProcessExitMessage(proc, err))

		// Also send event if monitoring is started
		if pm.monitorStarted {
			event := HealthMonitorEvent{
				Type:      HealthEventProcessExited,
				ProcessID: proc.name,
				Timestamp: exitTime,
				Message:   pm.formatProcessExitMessage(proc, err),
			}

			select {
			case pm.eventChannel <- event:
			default:
				// Channel full, continue without blocking
			}
		}
	}

	// Log process exit if verbose mode is enabled
	if pm.options.Verbose {
		pm.logProcessExit(proc, err)
	}
}

// formatProcessExitMessage creates a formatted message for process exit events
func (pm *processManager) formatProcessExitMessage(proc *managedProcess, err error) string {
	if err != nil {
		return fmt.Sprintf("Process exited with error (PID %d): %v", proc.cmd.Process.Pid, err)
	}
	return fmt.Sprintf("Process exited cleanly (PID %d)", proc.cmd.Process.Pid)
}

// logProcessExit logs information about a process that has exited
func (pm *processManager) logProcessExit(proc *managedProcess, err error) {
	if err != nil {
		fmt.Printf("keepAlive process '%s' exited unexpectedly (PID %d): %v\n",
			proc.name, proc.cmd.Process.Pid, err)
	} else {
		fmt.Printf("keepAlive process '%s' exited cleanly (PID %d)\n",
			proc.name, proc.cmd.Process.Pid)
	}
}

// GetActiveProcesses returns information about currently running processes
func (pm *processManager) GetActiveProcesses() map[string]ProcessInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	processes := make(map[string]ProcessInfo)
	for name, proc := range pm.activeProcs {
		processes[name] = ProcessInfo{
			Name:      name,
			PID:       proc.cmd.Process.Pid,
			StartTime: proc.startTime,
			Command:   buildCommandLine(proc.command),
		}
	}

	return processes
}

// TerminateAll gracefully terminates all active keepAlive processes
func (pm *processManager) TerminateAll() error {
	pm.mu.RLock()
	processes := make([]*managedProcess, 0, len(pm.activeProcs))
	for _, proc := range pm.activeProcs {
		processes = append(processes, proc)
	}
	pm.mu.RUnlock()

	var lastErr error
	for _, proc := range processes {
		if err := pm.terminateProcess(proc); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// TerminateProcess terminates a specific keepAlive process by name
func (pm *processManager) TerminateProcess(name string) error {
	pm.mu.RLock()
	proc, exists := pm.activeProcs[name]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process '%s' not found", name)
	}

	return pm.terminateProcess(proc)
}

// terminateProcess terminates a managed process
func (pm *processManager) terminateProcess(proc *managedProcess) error {
	if proc.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	if pm.options.Verbose {
		fmt.Printf("Terminating keepAlive process '%s' (PID %d)\n",
			proc.name, proc.cmd.Process.Pid)
	}

	// Try graceful termination first
	if err := proc.cmd.Process.Signal(nil); err != nil {
		// Process might already be dead, try kill
		return proc.cmd.Process.Kill()
	}

	// Wait a short time for graceful shutdown
	select {
	case <-proc.done:
		return nil
	case <-time.After(2 * time.Second):
		// Force kill if graceful shutdown takes too long
		return proc.cmd.Process.Kill()
	}
}

// GetProcessHealth returns health status for all monitored processes
func (pm *processManager) GetProcessHealth() map[string]ProcessHealth {
	if pm.healthMonitor == nil {
		return make(map[string]ProcessHealth)
	}
	return pm.healthMonitor.GetAllHealth()
}

// StartHealthMonitoring starts the background health monitoring system
func (pm *processManager) StartHealthMonitoring(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.monitorStarted {
		return fmt.Errorf("health monitoring is already started")
	}

	pm.monitorCtx, pm.monitorCancel = context.WithCancel(ctx)
	pm.monitorStarted = true

	// Start the health monitor
	go pm.healthMonitor.Start(pm.monitorCtx, pm.eventChannel)

	// Start event processor if verbose mode is enabled
	if pm.options.Verbose {
		go pm.processHealthEvents()
	}

	if pm.options.Verbose {
		fmt.Printf("Health monitoring started with %v check interval\n", pm.monitorOptions.CheckInterval)
	}

	return nil
}

// StopHealthMonitoring stops the background health monitoring system
func (pm *processManager) StopHealthMonitoring() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.monitorStarted {
		return fmt.Errorf("health monitoring is not started")
	}

	if pm.monitorCancel != nil {
		pm.monitorCancel()
	}

	pm.monitorStarted = false

	if pm.options.Verbose {
		fmt.Println("Health monitoring stopped")
	}

	return nil
}

// processHealthEvents processes health monitoring events in the background
func (pm *processManager) processHealthEvents() {
	for {
		select {
		case event := <-pm.eventChannel:
			pm.handleHealthEvent(event)
		case <-pm.monitorCtx.Done():
			return
		}
	}
}

// handleHealthEvent processes individual health monitoring events
func (pm *processManager) handleHealthEvent(event HealthMonitorEvent) {
	// Report lifecycle events if enabled
	if pm.lifecycleReportingEnabled && pm.reporter != nil {
		pm.reporter.ReportProcessLifecycleEvent(event)
	}

	// Legacy verbose output for backward compatibility
	if pm.options.Verbose {
		switch event.Type {
		case HealthEventProcessExited:
			fmt.Printf("Health Monitor: Process '%s' exited - %s\n", event.ProcessID, event.Message)
		case HealthEventProcessFailed:
			fmt.Printf("Health Monitor: Process '%s' failed - %s\n", event.ProcessID, event.Message)
		case HealthEventMemoryThreshold:
			fmt.Printf("Health Monitor: Process '%s' exceeded memory threshold - %s\n", event.ProcessID, event.Message)
		case HealthEventCPUThreshold:
			fmt.Printf("Health Monitor: Process '%s' exceeded CPU threshold - %s\n", event.ProcessID, event.Message)
		case HealthEventHealthCheckFailed:
			fmt.Printf("Health Monitor: Health check failed for process '%s' - %s\n", event.ProcessID, event.Message)
		}
	}
}

// ReportCurrentStatus reports the current status of all processes
func (pm *processManager) ReportCurrentStatus() {
	if pm.reporter == nil {
		return
	}

	activeProcs := pm.GetActiveProcesses()
	pm.reporter.ReportProcessStatus(activeProcs)
}

// ReportHealthStatus reports the health status of all monitored processes
func (pm *processManager) ReportHealthStatus() {
	if pm.reporter == nil {
		return
	}

	health := pm.GetProcessHealth()
	pm.reporter.ReportProcessHealth(health)
}

// GetHealthEvents returns a channel for receiving health monitoring events
func (pm *processManager) GetHealthEvents() <-chan HealthMonitorEvent {
	return pm.eventChannel
}

// EnableLifecycleReporting enables or disables automatic reporting of lifecycle events
func (pm *processManager) EnableLifecycleReporting(enabled bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.lifecycleReportingEnabled = enabled
}

// GetHealthSummary returns a summary of process health status
func (pm *processManager) GetHealthSummary() HealthSummary {
	if pm.healthMonitor == nil {
		return HealthSummary{
			TotalProcesses: 0,
			Timestamp:      time.Now(),
		}
	}
	return pm.healthMonitor.GetHealthSummary()
}
