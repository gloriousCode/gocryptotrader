package engine

import (
	"context"
	"math"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
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
		{name: "inaccessible system process", pid: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exited, diagnosticProcessExited(tc.pid), "process ownership should only expire after confirmed exit")
		})
	}
	t.Run("PID cannot wrap to an absent process", func(t *testing.T) {
		t.Parallel()
		if strconv.IntSize < 64 {
			t.Skip("an int cannot hold an out-of-range Windows PID on this platform")
		}
		pid := uint64(math.MaxUint32) + 1 + 2147483647
		assert.False(t, diagnosticProcessExited(int(pid)), "out-of-range PIDs should preserve ownership")
	})
	t.Run("terminated process with retained handle", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		child := exec.CommandContext(ctx, os.Args[0], "-test.run=^$", "-test.count=1") //nolint:gosec // Runs only this test binary with no selected tests, never GCT.
		require.NoError(t, child.Start(), "the owned child must start before its process handle is retained")
		t.Cleanup(func() { _ = child.Wait() })
		pid := child.Process.Pid
		if pid <= 0 || uint64(pid) > math.MaxUint32 {
			require.FailNow(t, "the owned child must have a valid Windows process ID")
			return
		}
		handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
		require.NoError(t, err, "the child process handle must remain open through its exit")
		t.Cleanup(func() { require.NoError(t, windows.CloseHandle(handle), "the retained process handle must close") })
		require.NoError(t, child.Wait(), "the owned child must exit before its liveness is checked")
		assert.True(t, diagnosticProcessExited(child.Process.Pid), "a signalled process should permit recovery while its process object still exists")
	})
}
