//go:build !windows

package engine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiagnosticProcessExited(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		pid    int
		exited bool
	}{
		{name: "zero PID"},
		{name: "negative PID", pid: -1},
		{name: "current process", pid: os.Getpid()},
		{name: "absent PID", pid: 2147483647, exited: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exited, diagnosticProcessExited(tc.pid), "process ownership should only expire after confirmed exit")
		})
	}
}
