//go:build windows

package executor

import (
	"fmt"
	"os/exec"
)

// killProcessGroupPlatform kills an entire process group on Windows
func (pm *ProcessManager) killProcessGroupPlatform(pid int, graceful bool) error {
	// On Windows, we'll use taskkill to kill the process tree
	// This is more reliable than trying to enumerate child processes manually
	var killCmd *exec.Cmd

	if graceful {
		// Try graceful termination first (though Windows doesn't have SIGTERM equivalent)
		killCmd = exec.Command("taskkill", "/T", "/PID", fmt.Sprintf("%d", pid))
	} else {
		// Force kill the entire process tree
		killCmd = exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	}

	return killCmd.Run()
}
