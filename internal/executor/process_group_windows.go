//go:build windows

package executor

import (
	"fmt"
	"os/exec"
	"syscall"
)

// configureProcessGroupPlatform sets up process group on Windows
func (e *Executor) configureProcessGroupPlatform(cmd *exec.Cmd) {
	// On Windows, we use CREATE_NEW_PROCESS_GROUP to create a new process group
	// This allows us to send signals to the entire process tree
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// killProcessGroupPlatform kills an entire process group on Windows
func (e *Executor) killProcessGroupPlatform(pid int, graceful bool) error {
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
