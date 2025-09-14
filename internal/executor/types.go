package executor

import (
	"time"

	"github.com/seqr-cli/seqr/internal/config"
)

type ExecutionState int

const (
	StateReady ExecutionState = iota
	StateRunning
	StateSuccess
	StateFailed
)

func (s ExecutionState) String() string {
	switch s {
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StateSuccess:
		return "success"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type ExecutionResult struct {
	Command   config.Command `json:"command"`
	Success   bool           `json:"success"`
	ExitCode  int            `json:"exitCode"`
	Output    string         `json:"output,omitempty"`
	Error     string         `json:"error,omitempty"`
	StartTime time.Time      `json:"startTime"`
	EndTime   time.Time      `json:"endTime"`
	Duration  time.Duration  `json:"duration"`
}

type ExecutionStatus struct {
	State          ExecutionState    `json:"state"`
	CurrentCommand *config.Command   `json:"currentCommand,omitempty"`
	CompletedCount int               `json:"completedCount"`
	TotalCount     int               `json:"totalCount"`
	Results        []ExecutionResult `json:"results"`
	LastError      string            `json:"lastError,omitempty"`
}
