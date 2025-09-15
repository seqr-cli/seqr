//go:build windows

package executor

import (
	"os/exec"
	"testing"
)

func TestProcessGroupConfigurationWindows(t *testing.T) {
	// Test that process group configuration is applied correctly on Windows
	executor := NewExecutor(false)

	// Create a simple command
	cmd := &exec.Cmd{}

	// Configure process group
	executor.configureProcessGroup(cmd)

	// Verify that SysProcAttr is set
	if cmd.SysProcAttr == nil {
		t.Error("SysProcAttr should be set after configureProcessGroup")
	}

	// On Windows, check for CREATE_NEW_PROCESS_GROUP flag
	if cmd.SysProcAttr.CreationFlags == 0 {
		t.Error("CreationFlags should be set on Windows")
	}
}
