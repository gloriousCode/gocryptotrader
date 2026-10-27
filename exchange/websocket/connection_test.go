package websocket

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestMatchReturnResponses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		closeManager  bool
		cancelRequest bool
		removeMatcher bool
	}{
		{name: "connection shutdown"},
		{name: "manager shutdown", closeManager: true},
		{name: "caller cancellation retains priority", cancelRequest: true},
		{name: "removed matcher cannot produce an empty success", removeMatcher: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				conn := &connection{Match: NewMatch(), ResponseMaxLimit: time.Hour, done: make(chan struct{}), shutdown: make(chan struct{})}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				pending := make([]<-chan MatchedResponse, 8)
				for i := range pending {
					var err error
					pending[i], err = conn.MatchReturnResponses(ctx, i, 1)
					require.NoError(t, err, "pending matcher registration must succeed")
				}
				synctest.Wait()
				started := time.Now()
				switch {
				case tc.cancelRequest:
					cancel()
					require.NoError(t, conn.Shutdown(), "connection retirement must succeed")
				case tc.closeManager:
					close(conn.shutdown)
				case tc.removeMatcher:
					for i := range pending {
						conn.Match.RemoveSignature(i)
					}
				default:
					require.NoError(t, conn.Shutdown(), "connection retirement must succeed")
				}
				synctest.Wait()
				for _, results := range pending {
					select {
					case result := <-results:
						switch {
						case tc.cancelRequest:
							assert.ErrorIs(t, result.Err, context.Canceled, "already cancelled requests should preserve caller cancellation")
						case tc.removeMatcher:
							assert.ErrorIs(t, result.Err, ErrSignatureNotMatched, "removed matchers should fail without manufacturing an empty response")
						default:
							assert.ErrorIs(t, result.Err, errWebsocketIsDisconnected, "retirement should immediately release pending response waiters")
						}
						assert.Empty(t, result.Responses, "retired requests should not claim response data")
					default:
						assert.Fail(t, "retirement should release every waiter without waiting for its response timeout")
					}
				}
				assert.Equal(t, started, time.Now(), "retirement should not advance the response timeout clock")
				conn.Match.mu.Lock()
				assert.Empty(t, conn.Match.m, "retired requests should remove all matcher registrations")
				conn.Match.mu.Unlock()
			})
		})
	}

	conn := connection{Match: NewMatch()}
	_, err := conn.MatchReturnResponses(t.Context(), nil, 0)
	require.ErrorIs(t, err, errInvalidBufferSize)

	ch, err := conn.MatchReturnResponses(t.Context(), nil, 1)
	require.NoError(t, err)

	require.ErrorIs(t, (<-ch).Err, ErrSignatureTimeout)
	conn.ResponseMaxLimit = time.Second

	ch, err = conn.MatchReturnResponses(t.Context(), nil, 1)
	require.NoError(t, err)

	exp := []byte("test")
	require.True(t, conn.Match.IncomingWithData(nil, exp))
	resp := <-ch
	require.NoError(t, resp.Err)
	require.NotEmpty(t, resp.Responses, "must have response data")
	assert.Equal(t, exp, resp.Responses[0])
}

func TestWebsocketConnectionRequireMatchWithData(t *testing.T) {
	t.Parallel()
	ws := connection{Match: NewMatch()}
	err := ws.RequireMatchWithData(0, nil)
	require.ErrorIs(t, err, ErrSignatureNotMatched)

	ch, err := ws.Match.Set(0, 1)
	require.NoError(t, err)

	err = ws.RequireMatchWithData(0, []byte("test"))
	require.NoError(t, err)
	require.Len(t, ch, 1, "must have one item in channel")
	assert.Equal(t, []byte("test"), <-ch)
}

func TestIncomingWithData(t *testing.T) {
	t.Parallel()
	ws := connection{Match: NewMatch()}
	require.False(t, ws.IncomingWithData(0, nil))

	ch, err := ws.Match.Set(0, 1)
	require.NoError(t, err)

	require.True(t, ws.IncomingWithData(0, []byte("test")))
	require.Len(t, ch, 1, "must have one item in channel")
	assert.Equal(t, []byte("test"), <-ch)
}

func TestConnectionSubscriptions(t *testing.T) {
	t.Parallel()
	ws := &connection{}
	require.Nil(t, ws.Subscriptions())
	ws.subscriptions = subscription.NewStore()
	require.NotNil(t, ws.Subscriptions())
	testsubs.EqualLists(t, ws.subscriptions.List(), ws.Subscriptions().List())
}

func TestDialNilDialer(t *testing.T) {
	t.Parallel()
	err := (&connection{}).Dial(t.Context(), nil, nil, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Dial should error on a nil dialer")
}

func TestDialHandshakeTimeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// the server accepts connections but never answers a request, so only the handshake timeout can end the dial
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		wc := &connection{URL: "ws" + mock.URL[len("http"):] + "/ws"}

		start := time.Now()
		err := wc.Dial(t.Context(), dialer, nil, nil)
		require.ErrorIs(t, err, os.ErrDeadlineExceeded, "Dial must time out when the upgrade is never answered")
		assert.Equal(t, 45*time.Second, time.Since(start), "Dial should give up after the default handshake timeout")
		assert.Zero(t, dialer.HandshakeTimeout, "Dial should not set a handshake timeout on the caller's dialer")

		wc.ProxyURL = "http://proxy.invalid"
		dialer.HandshakeTimeout = time.Second
		var dialled []string
		dial := dialer.NetDialContext
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialled = append(dialled, addr)
			return dial(ctx, network, addr)
		}
		start = time.Now()
		err = wc.Dial(t.Context(), dialer, nil, nil)
		require.ErrorIs(t, err, os.ErrDeadlineExceeded, "Dial must time out when the proxy never answers")
		assert.Equal(t, time.Second, time.Since(start), "Dial should use the handshake timeout set on the dialer")
		assert.Nil(t, dialer.Proxy, "Dial should not set a proxy on the caller's dialer")
		assert.Equal(t, []string{"proxy.invalid:80"}, dialled, "Dial should connect through the configured proxy")
	})
}
