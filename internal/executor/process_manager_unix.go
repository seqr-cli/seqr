//go:build !windows

package executor

import (
	"syscall"
)

// killProcessGroupPlatform kills an entire process group on Unix-like systems
func (pm *ProcessManager) killProcessGroupPlatform(pid int, graceful bool) error {
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
