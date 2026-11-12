package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/internal/diagnostics"
)

var errExpectedWrite = errors.New("expected diagnostic write failure")

func TestRun(t *testing.T) {
	// Child environment and the committed output directory are process-wide test fixtures.
	if mode := os.Getenv("GCT_DIAGNOSTICRUN_TEST_CHILD"); mode != "" {
		switch mode {
		case "success":
			fmt.Fprintln(os.Stdout, "private-stdout")
			fmt.Fprintln(os.Stderr, "private-stderr")
		case "panic":
			fmt.Fprint(os.Stderr, "pan")
			fmt.Fprintln(os.Stderr, "ic: private-stderr")
			os.Exit(7)
		case "wait":
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()
			fmt.Fprintln(os.Stdout, "child-ready")
			<-ctx.Done()
		case "main-help":
			os.Args = []string{"gctdiagnosticrun", "--help"}
			main()
		case "main-invalid":
			os.Args = []string{"gctdiagnosticrun"}
			main()
		}
		return
	}
	const directory = "testdata/runtime"
	for _, tc := range []struct {
		name       string
		args       []string
		mode       string
		writeFault string
		lock       bool
		want       error
		wantState  string
		wantCode   int
		wantPanic  bool
	}{
		{name: "help", args: []string{"--help"}},
		{name: "unknown flag", args: []string{"--unknown"}},
		{name: "missing executable", want: errMissingExecutable},
		{name: "missing separator", args: []string{os.Args[0]}, want: errCommandSeparator},
		{name: "empty directory", args: []string{"--diagnostics-dir", " ", "--", os.Args[0]}, want: errDiagnosticsPath},
		{name: "existing launcher lock", lock: true, want: errSupervisorLocked},
		{name: "intent failure prevents launch", mode: "success", writeFault: "intent", want: errExpectedWrite},
		{name: "failed executable is recorded", args: []string{"--diagnostics-dir", directory, "--", "./testdata/no-such-executable"}, want: errChildStart, wantState: "unclean"},
		{name: "successful child", mode: "success", wantState: "stopped"},
		{name: "panic marker and exit code", mode: "panic", wantState: "unclean", wantCode: 7, wantPanic: true},
		{name: "running record failure stops own child", mode: "wait", writeFault: "running", want: errExpectedWrite},
		{name: "completion write failure is reported", mode: "success", writeFault: "completion", want: errExpectedWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(directory, "crash.json")
			lockPath := filepath.Join(directory, "supervisor.lock")
			t.Cleanup(func() {
				for _, file := range []string{path, lockPath} {
					err := os.Remove(file)
					if errors.Is(err, os.ErrNotExist) {
						continue
					}
					assert.NoError(t, err, "test output should be removed")
				}
			})
			t.Setenv("GCT_DIAGNOSTICRUN_TEST_CHILD", tc.mode)
			t.Setenv("GCT_DIAGNOSTICRUN_PRIVATE_TEST_VALUE", "private-environment")
			args := tc.args
			if tc.mode != "" || tc.lock {
				args = []string{"--diagnostics-dir", directory, "--", os.Args[0], "-test.run=^TestRun$", "--", "private-argument"}
			}
			if tc.lock {
				require.NoError(t, os.WriteFile(lockPath, []byte("previous launcher"), 0o600), "existing lock fixture must write")
			}
			var stdout, stderr bytes.Buffer
			write := func(path string, snapshot *diagnostics.Snapshot) error {
				record, ok := snapshot.Views["crash_report"].(*crashRecord)
				require.True(t, ok, "launcher evidence must use its fixed crash schema")
				if tc.writeFault == "intent" && record.ProcessID == 0 || tc.writeFault == "running" && record.ProcessID != 0 && record.FinishedAt.IsZero() || tc.writeFault == "completion" && !record.FinishedAt.IsZero() {
					return errExpectedWrite
				}
				return diagnostics.Write(path, snapshot)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			err := run(ctx, args, &stdout, &stderr, write)
			switch {
			case tc.want != nil:
				assert.ErrorIs(t, err, tc.want, "launch failure should retain its cause")
			case tc.name == "unknown flag" || tc.wantCode != 0:
				assert.Error(t, err, "failed child or invalid flags should return an error")
			default:
				assert.NoError(t, err, "valid launch should complete")
			}
			if tc.writeFault == "intent" {
				assert.Empty(t, stdout.String(), "intent failure should prevent the child from starting")
			}
			if tc.wantState == "" {
				return
			}
			snapshot, readErr := diagnostics.Read(path)
			require.NoError(t, readErr, "completed launch evidence must be readable")
			assert.Equal(t, tc.wantState, snapshot.State, "snapshot should describe the child outcome")
			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr, "saved evidence must be readable")
			for _, secret := range []string{"private-stdout", "private-stderr", "private-argument", "private-environment", "GCT_DIAGNOSTICRUN_TEST_CHILD"} {
				assert.NotContains(t, string(data), secret, "saved evidence should omit output, arguments and environment")
			}
			encoded, readErr := json.Marshal(snapshot.Views["crash_report"])
			require.NoError(t, readErr, "crash view must serialise")
			var record crashRecord
			require.NoError(t, json.Unmarshal(encoded, &record), "crash view must match the fixed schema")
			assert.Equal(t, tc.wantPanic, record.PanicObserved, "panic marker should remain an observation")
			assert.False(t, record.FinishedAt.IsZero(), "finished evidence should include its observation time")
			if tc.want == errChildStart {
				assert.True(t, record.StartFailed, "failed starts should be explicit")
				assert.Nil(t, record.ExitCode, "unstarted child should not receive a fabricated exit code")
			} else {
				require.NotNil(t, record.ExitCode, "completed child must include its actual exit code")
				assert.Equal(t, tc.wantCode, *record.ExitCode, "recorded exit code should be the actual child result")
				assert.Positive(t, record.ProcessID, "started child should have a process identity")
			}
			info, readErr := os.Stat(path)
			require.NoError(t, readErr, "saved evidence permissions must be inspectable")
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "evidence should be owner-only")
			_, readErr = os.Stat(lockPath)
			assert.ErrorIs(t, readErr, os.ErrNotExist, "finished launcher should release its lock")
		})
	}
	t.Run("cancellation interrupts only its own child", func(t *testing.T) {
		t.Setenv("GCT_DIAGNOSTICRUN_TEST_CHILD", "wait")
		path := filepath.Join(directory, "crash.json")
		t.Cleanup(func() { _ = os.Remove(path) })
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		reader, writer := io.Pipe()
		defer reader.Close()
		defer writer.Close()
		ready := make(chan string, 1)
		go func() {
			buffered := bufio.NewReader(reader)
			line, _ := buffered.ReadString('\n')
			ready <- line
			cancel()
			_, _ = io.Copy(io.Discard, buffered)
		}()
		err := run(ctx, []string{"--diagnostics-dir", directory, "--", os.Args[0], "-test.run=^TestRun$"}, writer, io.Discard, diagnostics.Write)
		assert.ErrorIs(t, err, context.Canceled, "cancelled supervision should retain cancellation")
		select {
		case line := <-ready:
			assert.Equal(t, "child-ready\n", line, "child should have installed its interrupt handler before cancellation")
		case <-time.After(time.Second):
			assert.Fail(t, "child readiness should be observed before the deadline")
		}
		snapshot, err := diagnostics.Read(path)
		require.NoError(t, err, "interrupted launch evidence must be readable")
		assert.Equal(t, "stopped", snapshot.State, "gracefully interrupted child should have a clean exit")
		data, err := json.Marshal(snapshot.Views["crash_report"])
		require.NoError(t, err, "interrupted crash view must serialise")
		var record crashRecord
		require.NoError(t, json.Unmarshal(data, &record), "interrupted crash view must decode")
		assert.True(t, record.Interrupted, "record should distinguish operator cancellation")
	})
}

func TestPanicWriterWrite(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		parts []string
		panic bool
	}{
		{name: "ordinary output", parts: []string{"ordinary\n", "output\n"}},
		{name: "one panic chunk", parts: []string{"panic: private-detail\n"}, panic: true},
		{name: "split marker", parts: []string{"pa", "ni", "c", ": private-detail\n"}, panic: true},
		{name: "large output remains bounded", parts: []string{strings.Repeat("private-detail", 10000)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			writer := &panicWriter{output: &output}
			for _, part := range tc.parts {
				n, err := writer.Write([]byte(part))
				require.NoError(t, err, "passthrough output must write")
				assert.Equal(t, len(part), n, "writer should preserve the downstream count")
				assert.LessOrEqual(t, len(writer.tail), 5, "retained marker tail should remain bounded")
			}
			assert.Equal(t, strings.Join(tc.parts, ""), output.String(), "raw output should pass through unchanged")
			assert.Equal(t, tc.panic, writer.seen, "only observed panic markers should be recorded")
		})
	}
	t.Run("passthrough write failure", func(t *testing.T) {
		t.Parallel()
		reader, output := io.Pipe()
		require.NoError(t, reader.Close(), "test pipe must close")
		t.Cleanup(func() { _ = output.Close() })
		writer := &panicWriter{output: output}
		n, err := writer.Write([]byte("panic: detail"))
		assert.ErrorIs(t, err, io.ErrClosedPipe, "downstream errors should remain explicit")
		assert.Zero(t, n, "failed output should retain the downstream count")
		assert.True(t, writer.seen, "observed stderr marker should remain recorded despite output failure")
	})
}

func TestMainExitStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mode string
		text string
		fail bool
	}{
		{mode: "main-help", text: "Usage of gctdiagnosticrun"},
		{mode: "main-invalid", text: errMissingExecutable.Error(), fail: true},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRun$") //nolint:gosec // Re-runs this test binary to exercise main without launching GCT.
			command.Env = append(os.Environ(), "GCT_DIAGNOSTICRUN_TEST_CHILD="+tc.mode)
			output, err := command.CombinedOutput()
			if tc.fail {
				assert.Error(t, err, "invalid command should produce a non-zero exit status")
			} else {
				assert.NoError(t, err, "help should have a zero exit status")
			}
			assert.Contains(t, string(output), tc.text, "main should explain its result")
		})
	}
}
