package stream

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRelay(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { NewRelay(0) }, "buffer size should be greater than 0")
	r := NewRelay(5)
	require.NotNil(t, r)
	assert.Equal(t, 5, cap(r.comm))
}

func TestSend(t *testing.T) {
	t.Parallel()
	r := NewRelay(1)
	require.NotNil(t, r)
	assert.NoError(t, r.Send(t.Context(), "test"))
	assert.ErrorIs(t, r.Send(t.Context(), "overflow"), errChannelBufferFull)
}

func TestRead(t *testing.T) {
	t.Parallel()
	r := NewRelay(1)
	require.NotNil(t, r)
	require.Empty(t, r.C)
	assert.NoError(t, r.Send(t.Context(), "test"))
	require.Len(t, r.C, 1)
	assert.Equal(t, "test", (<-r.C).Data)
}

func TestClose(t *testing.T) {
	t.Parallel()
	r := NewRelay(1)
	require.NotNil(t, r)
	r.Close()
	_, ok := <-r.C
	assert.False(t, ok)
}

var errDirectHandlerTest = errors.New("direct handler test")

func TestSetDirectHandler(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		handled bool
		err     error
	}{
		{name: "handled", handled: true}, {name: "fallback"}, {name: "handler error", err: errDirectHandlerTest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			relay := NewRelay(1)
			relay.SetDirectHandler(func(ctx context.Context, value any) (bool, error) {
				require.Equal(t, "payload", value)
				require.Equal(t, t.Context(), ctx)
				return tc.handled, tc.err
			})
			require.ErrorIs(t, relay.Send(t.Context(), "payload"), tc.err)
			if tc.handled || tc.err != nil {
				require.Empty(t, relay.C)
			} else {
				require.Equal(t, "payload", (<-relay.C).Data)
			}
			relay.SetDirectHandler(nil)
			require.NoError(t, relay.Send(t.Context(), "fallback"))
			require.Equal(t, "fallback", (<-relay.C).Data)
		})
	}
}
