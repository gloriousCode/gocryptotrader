package gateio

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var errGateRecoveryFixture = errors.New("offline recovery failure")

type gateRecoveryConnection struct {
	FixtureConnection
	failEvent        string
	failing          atomic.Bool
	unsubscribeCalls atomic.Int64
	subscribeCalls   atomic.Int64
	unsubscribeState atomic.Uint32
	target           *subscription.Subscription
	cancelled        atomic.Bool
	blockedEvent     string
	blocked          chan struct{}
	release          chan struct{}
}

func (c *gateRecoveryConnection) SendMessageReturnResponse(ctx context.Context, limit request.EndpointLimit, signature, payload any) ([]byte, error) {
	var event string
	switch p := payload.(type) {
	case *WsInput:
		event = p.Event
	case WsInput:
		event = p.Event
	}
	if event == unsubscribeEvent {
		c.unsubscribeCalls.Add(1)
		if c.target != nil {
			c.unsubscribeState.Store(uint32(c.target.State()))
		}
	} else {
		c.subscribeCalls.Add(1)
	}
	if c.failEvent == event && c.failing.Load() {
		return nil, errGateRecoveryFixture
	}
	blockedEvent := c.blockedEvent
	if blockedEvent == "" {
		blockedEvent = unsubscribeEvent
	}
	if c.blocked != nil && event == blockedEvent && ((event == unsubscribeEvent && c.unsubscribeCalls.Load() == 1) || (event == subscribeEvent && c.subscribeCalls.Load() == 1)) {
		close(c.blocked)
		if c.release != nil {
			<-c.release
		} else {
			<-ctx.Done()
			c.cancelled.Store(true)
			return nil, ctx.Err()
		}
	}
	return c.FixtureConnection.SendMessageReturnResponse(ctx, limit, signature, payload)
}

func testResubscribeFailureRecovery(t *testing.T) {
	t.Helper()
	t.Parallel()
	for _, failedEvent := range []string{unsubscribeEvent, subscribeEvent, "", "missing manager target"} {
		t.Run("failure_"+failedEvent, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				e := new(Exchange)
				require.NoError(t, testexch.Setup(e), "exchange setup must succeed")
				e.Name = t.Name()
				e.Features.Subscriptions = subscription.List{{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50}}
				subs, err := e.Features.Subscriptions.ExpandTemplates(e)
				require.NoError(t, err, "subscriptions must expand")
				subs = slices.DeleteFunc(subs, func(s *subscription.Subscription) bool { return s.QualifiedChannel != "ob.BTC_USDT.50" })
				require.Len(t, subs, 1, "recovery fixture must select one instrument")
				base, err := e.Websocket.CreateTestConnection(asset.Spot)
				require.NoError(t, err, "connection must initialise")
				conn := &gateRecoveryConnection{Connection: base, failEvent: failedEvent, target: subs[0]}
				conn.failing.Store(true)
				require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn), "connection must register")
				require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(conn, subs...), "subscriptions must register")
				if failedEvent == "missing manager target" {
					require.NoError(t, conn.Subscriptions().Add(subs[0]), "connection must retain its target")
					require.NoError(t, e.Websocket.RemoveSubscriptions(conn, subs...), "prior unsubscribe must remove the manager target")
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				pair := currency.NewBTCUSDT()
				require.NoError(t, e.wsOBResubMgr.Resubscribe(ctx, e, conn, "ob.BTC_USDT.50", pair, asset.Spot), "recovery must start")
				synctest.Wait()
				first := conn.unsubscribeCalls.Load() + conn.subscribeCalls.Load()
				require.Positive(t, first, "recovery must make an attempt")
				if conn.unsubscribeCalls.Load() > 0 {
					assert.Equal(t, uint32(subscription.ResubscribingState), conn.unsubscribeState.Load(), "temporary native unsubscribe should preserve reconnect intent")
				}
				if failedEvent == unsubscribeEvent {
					assert.Equal(t, subscription.SubscribedState, subs[0].State(), "failed unsubscribe should restore its prior local state")
				}
				conn.failing.Store(false)
				synctest.Sleep(time.Minute + time.Millisecond)
				synctest.Wait()
				assert.Greater(t, conn.unsubscribeCalls.Load()+conn.subscribeCalls.Load(), first, "recovery should retry without waiting for another incoming update")
				sub := e.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ob.BTC_USDT.50", Asset: asset.Spot}})
				require.NotNil(t, sub, "failed subscribe must retain a recoverable target")
				assert.Equal(t, subscription.SubscribedState, sub.State(), "recovery should restore subscription")
				snapshot := []byte(`{"t":1757377580046,"full":true,"s":"ob.BTC_USDT.50","u":100,"b":[["100","1"]],"a":[["101","1"]]}`)
				require.NoError(t, e.processOrderbookUpdateWithSnapshot(ctx, conn, snapshot, time.Now(), asset.Spot), "snapshot must restore depth")
				synctest.Wait()
				increment := []byte(`{"t":1757377580047,"full":false,"s":"ob.BTC_USDT.50","U":101,"u":101,"b":[["100","2"]],"a":[]}`)
				require.NoError(t, e.processOrderbookUpdateWithSnapshot(ctx, conn, increment, time.Now(), asset.Spot), "next increment must apply")
				book, err := e.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
				require.NoError(t, err, "recovered book must be readable")
				assert.Equal(t, int64(101), book.LastUpdateID, "recovered book should advance")
				require.NotEmpty(t, book.Bids, "recovered book must retain bids")
				assert.Equal(t, float64(2), book.Bids[0].Amount, "recovered book should apply changed amount")
				assert.False(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "snapshot should complete recovery")
				calls := conn.unsubscribeCalls.Load() + conn.subscribeCalls.Load()
				synctest.Sleep(2 * time.Minute)
				assert.Equal(t, calls, conn.unsubscribeCalls.Load()+conn.subscribeCalls.Load(), "completed recovery should stop retries")
			})
		})
	}
}

func testResubscribeSnapshotDuringRequest(t *testing.T) {
	t.Helper()
	t.Parallel()
	for _, unsubscribed := range []bool{false, true} {
		name := "cancel queued unsubscribe"
		if unsubscribed {
			name = "finish subscribe after unsubscribe committed"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				e := new(Exchange)
				require.NoError(t, testexch.Setup(e), "exchange setup must succeed")
				e.Name = t.Name()
				e.Features.Subscriptions = subscription.List{{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50}}
				subs, err := e.Features.Subscriptions.ExpandTemplates(e)
				require.NoError(t, err, "subscriptions must expand")
				subs = slices.DeleteFunc(subs, func(s *subscription.Subscription) bool { return s.QualifiedChannel != "ob.BTC_USDT.50" })
				require.Len(t, subs, 1, "recovery fixture must select one instrument")
				base, err := e.Websocket.CreateTestConnection(asset.Spot)
				require.NoError(t, err, "connection must initialise")
				conn := &gateRecoveryConnection{Connection: base, blocked: make(chan struct{})}
				if unsubscribed {
					conn.release = make(chan struct{})
				}
				require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn), "connection must register")
				require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(conn, subs...), "subscriptions must register")
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				pair := currency.NewBTCUSDT()
				require.NoError(t, e.wsOBResubMgr.Resubscribe(ctx, e, conn, "ob.BTC_USDT.50", pair, asset.Spot), "recovery must start")
				<-conn.blocked
				e.wsOBResubMgr.CompletedResubscribe(pair, asset.Spot)
				if unsubscribed {
					close(conn.release)
				}
				synctest.Wait()
				if !unsubscribed {
					assert.True(t, conn.cancelled.Load(), "completion should cancel the queued attempt immediately")
				}
				want := int64(0)
				if unsubscribed {
					want = 1
				}
				assert.Equal(t, want, conn.subscribeCalls.Load(), "completion should restore only an already-unsubscribed feed")
				assert.False(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "completion should stop suppression")
				synctest.Sleep(2 * time.Minute)
				assert.Equal(t, int64(1), conn.unsubscribeCalls.Load(), "completion should prevent another unsubscribe")
			})
		})
	}
}

func testResubscribeRenewedDemand(t *testing.T) {
	t.Helper()
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "exchange setup must succeed")
		e.Name = t.Name()
		e.Features.Subscriptions = subscription.List{{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50}}
		subs, err := e.Features.Subscriptions.ExpandTemplates(e)
		require.NoError(t, err, "subscriptions must expand")
		subs = slices.DeleteFunc(subs, func(s *subscription.Subscription) bool { return s.QualifiedChannel != "ob.BTC_USDT.50" })
		require.Len(t, subs, 1, "recovery fixture must select one instrument")
		base, err := e.Websocket.CreateTestConnection(asset.Spot)
		require.NoError(t, err, "connection must initialise")
		conn := &gateRecoveryConnection{Connection: base, blockedEvent: subscribeEvent, blocked: make(chan struct{}), release: make(chan struct{})}
		require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn), "connection must register")
		require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(conn, subs...), "subscriptions must register")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		pair := currency.NewBTCUSDT()
		require.NoError(t, e.wsOBResubMgr.Resubscribe(ctx, e, conn, "ob.BTC_USDT.50", pair, asset.Spot), "recovery must start")
		<-conn.blocked
		e.wsOBResubMgr.CompletedResubscribe(pair, asset.Spot)
		require.NoError(t, e.wsOBResubMgr.Resubscribe(ctx, e, conn, "ob.BTC_USDT.50", pair, asset.Spot), "new gap must request recovery before prior worker retires")
		assert.True(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "renewed demand should suppress invalid increments immediately")
		close(conn.release)
		synctest.Wait()
		assert.Greater(t, conn.unsubscribeCalls.Load(), int64(1), "new recovery demand should survive completion of the previous subscribe")
		assert.True(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "new recovery should await its snapshot")
		e.wsOBResubMgr.CompletedResubscribe(pair, asset.Spot)
		synctest.Wait()
	})
}

func testResubscribeConnectionReplacement(t *testing.T) {
	t.Helper()
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "exchange setup must succeed")
		e.Name = t.Name()
		e.Features.Subscriptions = subscription.List{{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50}}
		subs, err := e.Features.Subscriptions.ExpandTemplates(e)
		require.NoError(t, err, "subscriptions must expand")
		subs = slices.DeleteFunc(subs, func(s *subscription.Subscription) bool { return s.QualifiedChannel != "ob.BTC_USDT.50" })
		require.Len(t, subs, 1, "recovery fixture must select one instrument")
		base, err := e.Websocket.CreateTestConnection(asset.Spot)
		require.NoError(t, err, "connection must initialise")
		old := &gateRecoveryConnection{Connection: base, blockedEvent: subscribeEvent, blocked: make(chan struct{})}
		require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, old), "old connection must register")
		require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(old, subs...), "old subscriptions must register")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		pair := currency.NewBTCUSDT()
		require.NoError(t, e.wsOBResubMgr.Resubscribe(ctx, e, old, "ob.BTC_USDT.50", pair, asset.Spot), "old recovery must start")
		<-old.blocked
		replacementBase, err := e.Websocket.CreateTestConnection(asset.Spot)
		require.NoError(t, err, "replacement connection must initialise")
		replacement := &gateRecoveryConnection{Connection: replacementBase}
		require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, replacement), "replacement connection must register")
		replacementSubs, err := e.Features.Subscriptions.ExpandTemplates(e)
		require.NoError(t, err, "replacement subscriptions must expand")
		replacementSubs = slices.DeleteFunc(replacementSubs, func(s *subscription.Subscription) bool { return s.QualifiedChannel != "ob.BTC_USDT.50" })
		require.Len(t, replacementSubs, 1, "replacement fixture must select one instrument")
		require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(replacement, replacementSubs...), "replacement subscriptions must register")
		require.NoError(t, e.wsOBResubMgr.Resubscribe(ctx, e, replacement, "ob.BTC_USDT.50", pair, asset.Spot), "replacement recovery must start")
		synctest.Wait()
		assert.True(t, old.cancelled.Load(), "connection replacement should cancel the old queued subscribe even while parent remains alive")
		assert.True(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "old worker cleanup should preserve replacement recovery")
		synctest.Sleep(time.Minute + time.Millisecond)
		assert.Equal(t, int64(1), old.subscribeCalls.Load(), "old connection should never retry after replacement")
		assert.Greater(t, replacement.subscribeCalls.Load(), int64(1), "replacement should continue recovery")
		e.wsOBResubMgr.CompletedResubscribe(pair, asset.Spot)
		synctest.Wait()
	})
}
