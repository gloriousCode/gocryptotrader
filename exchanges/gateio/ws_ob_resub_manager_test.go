package gateio

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestNewWSOBResubManager(t *testing.T) {
	t.Parallel()

	m := newWSOBResubManager()
	require.NotNil(t, m)
	assert.NotNil(t, m.lookup)
}

func TestIsResubscribing(t *testing.T) {
	t.Parallel()

	m := newWSOBResubManager()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	m.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = &wsOBRecovery{parentDone: ctx.Done(), done: ctx.Done(), cancel: cancel, stop: cancel}
	assert.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	assert.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Futures))
}

func TestResubscribe(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"missing snapshot", testResubscribeMissingSnapshot},
		{"failure recovery", testResubscribeFailureRecovery},
		{"snapshot during request", testResubscribeSnapshotDuringRequest},
		{"renewed demand", testResubscribeRenewedDemand},
		{"connection replacement", testResubscribeConnectionReplacement},
	} {
		t.Run(tc.name, tc.run)
	}

	m := newWSOBResubManager()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e))
	e.Name = t.Name()

	baseConn, err := e.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err)
	conn := &FixtureConnection{Connection: baseConn}
	require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn))

	err = m.Resubscribe(t.Context(), e, conn, "notfound", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, subscription.ErrNotFound)
	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))

	err = e.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
		Asks:        []orderbook.Level{{Price: 50000, Amount: 0.1}},
		Bids:        []orderbook.Level{{Price: 49000, Amount: 0.2}},
		Exchange:    e.Name,
		Pair:        currency.NewBTCUSDT(),
		Asset:       asset.Spot,
		LastUpdated: time.Now(),
	})
	require.NoError(t, err)
	err = m.Resubscribe(t.Context(), e, conn, "notfound", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, subscription.ErrNotFound)

	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))

	e.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50},
	}
	expanded, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err)

	err = e.Websocket.AddSubscriptions(conn, expanded...)
	require.NoError(t, err)

	qualifiedChannel := "ob.BTC_USDT.50"
	err = e.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
		Asks:        []orderbook.Level{{Price: 50000, Amount: 0.1}},
		Bids:        []orderbook.Level{{Price: 49000, Amount: 0.2}},
		Exchange:    e.Name,
		Pair:        currency.NewBTCUSDT(),
		Asset:       asset.Spot,
		LastUpdated: time.Now(),
	})
	require.NoError(t, err)
	err = m.Resubscribe(t.Context(), e, conn, qualifiedChannel, currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err)
	assert.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot), "manager should mark the pair as resubscribing immediately")
	assert.Eventually(t,
		func() bool {
			sub := e.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: qualifiedChannel, Asset: asset.Spot}})
			return sub != nil && sub.State() == subscription.SubscribedState
		},
		time.Second,
		10*time.Millisecond,
		"subscription should be resubscribed by the background routine",
	)

	m.CompletedResubscribe(currency.NewBTCUSDT(), asset.Spot)
	assert.Eventually(t,
		func() bool {
			return !m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot)
		},
		time.Second,
		10*time.Millisecond,
		"resubscription state should clear after completion is signalled",
	)

	t.Run("parent cancellation clears background recovery", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			m := newWSOBResubManager()
			ctx, cancel := context.WithCancel(t.Context())
			require.NoError(t, m.Resubscribe(ctx, e, &FixtureConnection{}, qualifiedChannel, currency.NewBTCUSDT(), asset.Spot), "Resubscribe must dispatch before the untracked connection fails")
			cancel()
			synctest.Wait()
			assert.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot), "parent cancellation should clear the tracking entry")
		})
	})
}

func testResubscribeMissingSnapshot(t *testing.T) {
	t.Helper()
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "setup must succeed")
		e.Name = t.Name()
		e.Features.Subscriptions = subscription.List{{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50}}
		subs, err := e.Features.Subscriptions.ExpandTemplates(e)
		require.NoError(t, err, "subscriptions must expand")
		base, err := e.Websocket.CreateTestConnection(asset.Spot)
		require.NoError(t, err, "test connection must initialise")
		conn := &FixtureConnection{Connection: base}
		require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn), "test connection must register")
		require.NoError(t, e.Websocket.AddSubscriptions(conn, subs...), "subscriptions must register")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		pair := currency.NewBTCUSDT()
		const channel = "ob.BTC_USDT.50"
		require.NoError(t, e.wsOBResubMgr.Resubscribe(ctx, e, conn, channel, pair, asset.Spot), "initial recovery must start")
		synctest.Wait()
		require.True(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "successful acknowledgement must still await the snapshot")
		synctest.Sleep(time.Minute)
		assert.True(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "automatic recovery should keep invalid increments suppressed")
		require.NoError(t, e.wsOBResubMgr.Resubscribe(ctx, e, conn, channel, pair, asset.Spot), "expired recovery must be retryable")
		synctest.Wait()
		require.True(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "retry must await its snapshot")
		snapshot := []byte(`{"t":1757377580046,"full":true,"s":"ob.BTC_USDT.50","u":100,"b":[["100","1"]],"a":[["101","1"]]}`)
		require.NoError(t, e.processOrderbookUpdateWithSnapshot(t.Context(), conn, snapshot, time.Now(), asset.Spot), "replacement snapshot must load")
		assert.False(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "replacement snapshot should finish recovery")
		update := []byte(`{"t":1757377580047,"s":"ob.BTC_USDT.50","U":101,"u":102,"b":[["100","2"]],"a":[]}`)
		require.NoError(t, e.processOrderbookUpdateWithSnapshot(t.Context(), conn, update, time.Now(), asset.Spot), "incremental processing must resume")
		book, err := e.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
		require.NoError(t, err, "recovered book must be readable")
		assert.Equal(t, int64(102), book.LastUpdateID, "recovered book should advance")
		assert.Equal(t, float64(2), book.Bids[0].Amount, "recovered book should apply the new amount")
	})
}

func TestResubscribeWithoutOrderbook(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "test instance setup must not error")
	e.Name = t.Name()
	e.Features.Subscriptions = subscription.List{{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50}}
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	baseConn, err := e.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err, "test connection creation must succeed")
	conn := &FixtureConnection{Connection: baseConn}
	require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn), "fixture connection registration must succeed")
	require.NoError(t, e.Websocket.AddSubscriptions(conn, subs...), "subscriptions must register")
	require.NoError(t, e.wsOBResubMgr.Resubscribe(t.Context(), e, conn, "ob.BTC_USDT.50", currency.NewBTCUSDT(), asset.Spot), "Resubscribe must not require an existing orderbook")
	assert.True(t, e.wsOBResubMgr.IsResubscribing(currency.NewBTCUSDT(), asset.Spot), "a pair whose first snapshot was rejected should be able to resubscribe")
	assert.Eventually(t,
		func() bool {
			sub := e.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ob.BTC_USDT.50", Asset: asset.Spot}})
			return sub != nil && sub.State() == subscription.SubscribedState
		},
		time.Second,
		10*time.Millisecond,
		"subscription should be resubscribed by the background routine",
	)
}

func TestFuturesV2GapRecovery(t *testing.T) {
	t.Parallel()
	exchange := new(Exchange)
	require.NoError(t, testexch.Setup(exchange), "test exchange setup must succeed")
	exchange.Name = t.Name()
	subs, err := exchange.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "futures subscriptions must generate")
	var futures *subscription.Subscription
	for _, sub := range subs {
		if sub.Channel == futuresOrderbookV2 {
			futures = sub
			break
		}
	}
	require.NotNil(t, futures, "defaults must include a futures V2 subscription")
	pair := futures.Pairs[0]
	qualifiedChannel := "ob." + pair.String() + ".50"
	spot := &subscription.Subscription{Channel: spotOrderbookV2, Asset: asset.Spot, Pairs: currency.Pairs{pair}, Levels: 50, QualifiedChannel: qualifiedChannel}
	spotBase, err := exchange.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err, "spot connection must be created")
	spotConn := &FixtureConnection{Connection: spotBase}
	require.NoError(t, exchange.Websocket.TrackTestConnection(asset.Spot, spotConn), "spot connection must be tracked")
	require.NoError(t, exchange.Websocket.AddSuccessfulSubscriptions(spotConn, spot), "spot subscription must register")
	baseConn, err := exchange.Websocket.CreateTestConnection(asset.USDTMarginedFutures)
	require.NoError(t, err, "futures connection must be created")
	conn := &FixtureConnection{Connection: baseConn}
	require.NoError(t, exchange.Websocket.TrackTestConnection(asset.USDTMarginedFutures, conn), "futures connection must be tracked")
	require.NoError(t, exchange.Websocket.AddSubscriptions(conn, futures), "generated futures subscription must register")
	for _, assetType := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
		snapshot := fmt.Appendf(nil, `{"t":1757377580046,"full":true,"s":%q,"u":100,"b":[["100","1"]],"a":[["101","1"]]}`, qualifiedChannel)
		require.NoError(t, exchange.processOrderbookUpdateWithSnapshot(t.Context(), conn, snapshot, time.Now(), assetType), "initial snapshot must load")
	}
	gap := fmt.Appendf(nil, `{"time":1757377580,"channel":"futures.obu","event":"update","result":{"t":1757377580073,"s":%q,"U":102,"u":103}}`, qualifiedChannel)
	require.NoError(t, exchange.WsHandleFuturesData(t.Context(), conn, gap, asset.USDTMarginedFutures), "futures gap must find its generated subscription")
	assert.True(t, exchange.wsOBResubMgr.IsResubscribing(pair, asset.USDTMarginedFutures), "futures should await a replacement snapshot")
	assert.False(t, exchange.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "spot should not enter recovery")
	assert.Equal(t, subscription.SubscribedState, spot.State(), "spot subscription should remain subscribed")
	_, err = exchange.Websocket.Orderbook.GetOrderbook(pair, asset.USDTMarginedFutures)
	require.Error(t, err, "gapped futures book must be unavailable")
	spotBook, err := exchange.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	require.NoError(t, err, "spot book must remain available")
	assert.Equal(t, int64(100), spotBook.LastUpdateID, "spot sequence should remain unchanged")
	require.Eventually(t, func() bool {
		return futures.State() == subscription.SubscribedState
	}, time.Second, time.Millisecond, "futures subscription must complete unsubscribe and resubscribe")
	fresh := fmt.Appendf(nil, `{"t":1757377580080,"full":true,"s":%q,"u":200,"b":[["100","2"]],"a":[["101","2"]]}`, qualifiedChannel)
	require.NoError(t, exchange.processOrderbookUpdateWithSnapshot(t.Context(), conn, fresh, time.Now(), asset.USDTMarginedFutures), "replacement snapshot must load")
	assert.False(t, exchange.wsOBResubMgr.IsResubscribing(pair, asset.USDTMarginedFutures), "replacement snapshot should clear recovery")
	update := fmt.Appendf(nil, `{"t":1757377580090,"s":%q,"U":201,"u":201,"b":[["100","3"]]}`, qualifiedChannel)
	require.NoError(t, exchange.processOrderbookUpdateWithSnapshot(t.Context(), conn, update, time.Now(), asset.USDTMarginedFutures), "incremental processing must resume")
	book, err := exchange.Websocket.Orderbook.GetOrderbook(pair, asset.USDTMarginedFutures)
	require.NoError(t, err, "recovered futures book must be available")
	assert.Equal(t, int64(201), book.LastUpdateID, "recovered book should advance its sequence")
	require.NotEmpty(t, book.Bids, "recovered book must contain bids")
	assert.Equal(t, float64(3), book.Bids[0].Amount, "recovered book should apply incremental quantities")
}

func TestCompletedResubscribe(t *testing.T) {
	t.Parallel()

	m := newWSOBResubManager()
	m.CompletedResubscribe(currency.NewBTCUSDT(), asset.Spot) // no-op
	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	m.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = &wsOBRecovery{parentDone: ctx.Done(), done: ctx.Done(), cancel: cancel, stop: cancel}
	require.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	m.CompletedResubscribe(currency.NewBTCUSDT(), asset.Spot)
	assert.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
}

func TestQualifiedChannelKey_Match(t *testing.T) {
	t.Parallel()

	require.Implements(t, (*subscription.MatchableKey)(nil), new(qualifiedChannelKey))

	k := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "test.channel", Asset: asset.Spot}}
	require.True(t, k.Match(k))
	require.False(t, k.Match(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "TEST.channel", Asset: asset.Spot}}))
	require.False(t, k.Match(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "test.channel", Asset: asset.Futures}}))
	assert.NotNil(t, k.GetSubscription())
}
