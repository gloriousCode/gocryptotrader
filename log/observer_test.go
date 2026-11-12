package log

import (
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddObserver(t *testing.T) {
	// Observer registrations are a process-wide singleton.
	t.Run("nil observer", func(t *testing.T) {
		assert.Zero(t, AddObserver(nil), "a nil callback should not register")
	})
	t.Run("metadata excludes log arguments", func(t *testing.T) {
		sublogger, err := NewSubLogger("DIAGNOSTICOBSERVERMETADATA")
		require.NoError(t, err, "an isolated logger must be created")
		sublogger.setLevelsProtected(splitLevel("ERROR"))
		require.NoError(t, sublogger.setOutputProtected(&multiWriterHolder{writers: []io.Writer{io.Discard}}), "the isolated logger must discard output")
		var count atomic.Int64
		id := AddObserver(func(header, name string) {
			assert.NotContains(t, header+name, "private-secret", "the observer should receive metadata only")
			count.Add(1)
		})
		t.Cleanup(func() { RemoveObserver(id) })
		Errorln(sublogger, "private-secret")
		Errorf(sublogger, "%s", "private-secret")
		assert.GreaterOrEqual(t, count.Load(), int64(2), "both log forms should notify observers")
	})
}

func TestRemoveObserver(t *testing.T) {
	// Observer registrations are a process-wide singleton.
	sublogger, err := NewSubLogger("DIAGNOSTICOBSERVERREMOVAL")
	require.NoError(t, err, "an isolated logger must be created")
	sublogger.setLevelsProtected(splitLevel("ERROR"))
	require.NoError(t, sublogger.setOutputProtected(&multiWriterHolder{writers: []io.Writer{io.Discard}}), "the isolated logger must discard output")
	var count atomic.Int64
	id := AddObserver(func(_, _ string) { count.Add(1) })
	RemoveObserver(id)
	before := count.Load()
	Errorln(sublogger, "test")
	assert.Equal(t, before, count.Load(), "removed observers should not receive later messages")
	RemoveObserver(id)
	assert.Nil(t, observerState.Load(), "removing the final observer should restore the disabled fast path")
	t.Run("disabled logging does not acquire registration lock", func(t *testing.T) {
		observerMu.Lock()
		done := make(chan struct{})
		go func() {
			Errorln(sublogger, "disabled observer")
			Errorf(sublogger, "%s", "disabled observer")
			close(done)
		}()
		blocked := false
		select {
		case <-done:
		case <-time.After(time.Second):
			blocked = true
		}
		observerMu.Unlock()
		assert.False(t, blocked, "logging without observers should not wait for registration locks")
		select {
		case <-done:
		case <-time.After(time.Second):
			require.FailNow(t, "logging must complete after releasing the test lock")
		}
	})
}
