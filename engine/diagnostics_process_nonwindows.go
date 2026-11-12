//go:build !windows

package engine

import (
	"errors"
	"os"
	"syscall"
)

// diagnosticProcessExited preserves ownership unless the OS confirms the process is gone.
func diagnosticProcessExited(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	_ = process.Release()
	return errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone)
}
