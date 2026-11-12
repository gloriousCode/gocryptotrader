package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireDiagnosticOwner(t *testing.T) {
	t.Parallel()
	directory := filepath.Join("testdata", "diagnostics_owner")
	owner, err := acquireDiagnosticOwner(directory, "first")
	require.NoError(t, err, "an unclaimed journal must admit one owner")
	t.Cleanup(owner.release)
	other, err := acquireDiagnosticOwner(directory, "second")
	assert.ErrorIs(t, err, errDiagnosticOwned, "a live process should retain exclusive journal ownership")
	assert.Nil(t, other, "competing writers should not be admitted")
	owner.release()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "owner.lock"), []byte("2147483647 dead-run"), 0o600), "a definitely exited owner must be recorded")
	t.Cleanup(func() { _ = os.Remove(filepath.Join(directory, "owner.lock")) })
	type result struct {
		owner *diagnosticOwner
		err   error
	}
	start := make(chan struct{})
	done := make(chan result, 2)
	for _, runID := range []string{"replacement-one", "replacement-two"} {
		go func() {
			<-start
			owner, err := acquireDiagnosticOwner(directory, runID)
			done <- result{owner: owner, err: err}
		}()
	}
	close(start)
	winners := 0
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for range 2 {
		select {
		case outcome := <-done:
			if outcome.err != nil {
				assert.ErrorIs(t, outcome.err, errDiagnosticOwned, "a competing stale reclaim should be rejected")
				continue
			}
			winners++
			t.Cleanup(outcome.owner.release)
			data, err := os.ReadFile(outcome.owner.Path)
			require.NoError(t, err, "winning ownership must remain durable")
			assert.Equal(t, outcome.owner.Token, string(data), "stale reclaim should not remove a newer owner")
		case <-deadline.C:
			require.FailNow(t, "ownership contenders must finish")
		}
	}
	assert.Equal(t, 1, winners, "only one replacement owner should be admitted")
}

func TestDiagnosticOwnerRelease(t *testing.T) {
	t.Parallel()
	var nilOwner *diagnosticOwner
	nilOwner.release()
	file, err := os.CreateTemp(filepath.Join("testdata", "diagnostics_owner"), "test-runtime-")
	require.NoError(t, err, "the ownership test record must be created")
	path := file.Name()
	require.NoError(t, file.Close(), "ownership test handle must close")
	t.Cleanup(func() { _ = os.Remove(path) })
	owner := &diagnosticOwner{Path: path, Token: "old"}
	require.NoError(t, os.WriteFile(path, []byte("new"), 0o600), "a newer claim must be recorded")
	owner.release()
	_, err = os.Stat(path)
	assert.NoError(t, err, "an old owner should not remove a newer claim")
	owner.Token = "new"
	owner.release()
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist, "the matching owner should release its claim")
}
