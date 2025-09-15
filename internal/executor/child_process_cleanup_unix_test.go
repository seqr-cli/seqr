//go:build !windows

package executor

import (
	"os/exec"
	"testing"
)

func TestProcessGroupConfigurationUnix(t *testing.T) {
	// Test that process group configuration is applied correctly on Unix
	executor := NewExecutor(false)

	// Create a simple command
	cmd := &exec.Cmd{}

	// Configure process group
	executor.configureProcessGroup(cmd)

	// Verify that SysProcAttr is set
	if cmd.SysProcAttr == nil {
		t.Error("SysProcAttr should be set after configureProcessGroup")
	}

	// On Unix, check for Setpgid
	if !cmd.SysProcAttr.Setpgid {
		t.Error("Setpgid should be true on Unix systems")
	}

	if cmd.SysProcAttr.Pgid != 0 {
		t.Error("Pgid should be 0 to use process PID as group ID")
	}
}
