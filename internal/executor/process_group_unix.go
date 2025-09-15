//go:build !windows

package executor

import (
	"os/exec"
	"syscall"
)

// configureProcessGroupPlatform sets up process group on Unix-like systems
func (e *Executor) configureProcessGroupPlatform(cmd *exec.Cmd) {
	// Set up process group so we can kill the entire process tree
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
		Pgid:    0,    // Use process PID as process group ID
	}
}

// killProcessGroupPlatform kills an entire process group on Unix-like systems
func (e *Executor) killProcessGroupPlatform(pid int, graceful bool) error {
	if graceful {
		// Send SIGTERM to the entire process group
		if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
			return err
		}
	} else {
		// Send SIGKILL to the entire process group
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
			return err
		}
	}
	return nil
}
