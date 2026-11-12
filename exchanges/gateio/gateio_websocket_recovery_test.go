package gateio

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestOnWebsocketConnectionFault(t *testing.T) {
	t.Parallel()
	newExchange := func(t *testing.T) *Exchange {
		t.Helper()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "exchange setup must succeed")
		// testexch.Setup replaces the websocket manager after SetDefaults.
		e.wsOBUpdateMgr = orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
			FetchDelay:         orderbookmanager.DefaultWSOrderbookUpdateTimeDelay,
			FetchDeadline:      orderbookmanager.DefaultWSOrderbookUpdateDeadline,
			FetchOrderbook:     e.fetchWSOrderbookSnapshot,
			CheckPendingUpdate: checkPendingUpdate,
			Orderbook:          &e.Websocket.Orderbook,
		})
		return e
	}

	t.Run("deduplicated native ownership and sibling isolation", func(t *testing.T) {
		t.Parallel()
		e := newExchange(t)
		e.Name = t.Name()
		pair := currency.NewBTCUSDT()
		siblingPair := currency.NewPair(currency.ETH, currency.USDT)
		for _, item := range []struct {
			pair  currency.Pair
			asset asset.Item
		}{{pair, asset.Spot}, {pair, asset.USDTMarginedFutures}, {siblingPair, asset.Spot}} {
			require.NoError(t, e.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
				Pair: item.pair, Asset: item.asset, Exchange: e.Name, LastUpdated: time.Now(),
				Bids: []orderbook.Level{{Price: 100, Amount: 1}}, Asks: []orderbook.Level{{Price: 101, Amount: 1}},
			}), "initial snapshot must load")
		}
		e.Websocket.DataHandler = stream.NewRelay(1)
		conn := &FixtureConnection{}
		sibling := &FixtureConnection{}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		siblingCtx, siblingCancel := context.WithCancel(t.Context())
		defer siblingCancel()
		identity := key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: asset.Spot}
		siblingIdentity := key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: asset.USDTMarginedFutures}
		e.wsOBResubMgr.lookup[identity] = &wsOBRecovery{connection: conn, stop: cancel}
		e.wsOBResubMgr.lookup[siblingIdentity] = &wsOBRecovery{connection: sibling, stop: siblingCancel}
		before := time.Now()
		require.NoError(t, e.onWebsocketConnectionFault(conn, subscription.List{
			{Channel: spotOrderbookV2, Asset: asset.Spot, Pairs: currency.Pairs{pair, pair, currency.EMPTYPAIR}},
			{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{pair}},
			{Channel: spotTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{siblingPair}},
			nil,
		}), "failed connection depth must retire")
		assert.ErrorIs(t, ctx.Err(), context.Canceled, "failed connection recovery work should cancel")
		assert.NoError(t, siblingCtx.Err(), "sibling recovery work should remain active")
		assert.NotContains(t, e.wsOBResubMgr.lookup, identity, "failed worker should no longer suppress replacement data")
		assert.Contains(t, e.wsOBResubMgr.lookup, siblingIdentity, "sibling worker should retain ownership")
		_, err := e.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
		assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "failed connection book should become invalid")
		_, err = e.Websocket.Orderbook.GetOrderbook(pair, asset.USDTMarginedFutures)
		assert.NoError(t, err, "same pair on a sibling asset should remain valid")
		_, err = e.Websocket.Orderbook.GetOrderbook(siblingPair, asset.Spot)
		assert.NoError(t, err, "same asset on a sibling connection should remain valid")
		select {
		case payload := <-e.Websocket.DataHandler.C:
			notification, ok := payload.Data.(websocket.OrderbookInvalidation)
			require.True(t, ok, "callback must notify downstream copied-book stores")
			assert.Equal(t, []key.PairAsset{identity}, notification.Pairs, "duplicate channels and pairs should yield one identity")
			assert.False(t, notification.At.Before(before), "invalidation boundary should precede replacement data")
		default:
			t.Fatal("callback must emit a downstream notification")
		}
	})
	t.Run("boundary follows old snapshot publication", func(t *testing.T) {
		t.Parallel()
		e := newExchange(t)
		e.Name = t.Name()
		pair := currency.NewBTCUSDT()
		firstPair := currency.NewPair(currency.ETH, currency.USDT)
		require.NoError(t, e.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{Pair: firstPair, Asset: asset.Spot, Exchange: e.Name, LastUpdated: time.Now(), Bids: []orderbook.Level{{Price: 100, Amount: 1}}, Asks: []orderbook.Level{{Price: 101, Amount: 1}}}), "first retirement fixture must load")
		entered, release := make(chan struct{}), make(chan struct{})
		published := make(chan time.Time, 1)
		notifications := make(chan websocket.OrderbookInvalidation, 1)
		var first atomic.Bool
		e.Websocket.DataHandler.SetDirectHandler(func(_ context.Context, data any) (bool, error) {
			switch event := data.(type) {
			case *orderbook.Depth:
				if first.CompareAndSwap(false, true) {
					close(entered)
					<-release
					published <- time.Now()
				}
			case websocket.OrderbookInvalidation:
				notifications <- event
			}
			return true, nil
		})
		e.wsOBUpdateMgr = orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
			FetchDeadline: time.Minute,
			FetchOrderbook: func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
				return &orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: e.Name, LastUpdated: time.Now(), LastUpdateID: 100, Bids: []orderbook.Level{{Price: 100, Amount: 1}}, Asks: []orderbook.Level{{Price: 101, Amount: 1}}}, nil
			},
			CheckPendingUpdate: func(int64, int64, *orderbook.Update) (bool, error) { return false, nil },
			Orderbook:          &e.Websocket.Orderbook,
		})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		require.NoError(t, e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, 101, &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 101, UpdateTime: time.Now(), AllowEmpty: true}), "old snapshot worker must start")
		require.Eventually(t, func() bool {
			select {
			case <-entered:
				return true
			default:
				return false
			}
		}, 2*time.Second, time.Millisecond, "old snapshot publication must start")
		cancel() // The worker has already passed its final cancellation check.
		finished := make(chan error, 1)
		go func() {
			finished <- e.onWebsocketConnectionFault(&FixtureConnection{}, subscription.List{{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot, Pairs: currency.Pairs{firstPair, pair}}})
		}()
		require.Eventually(t, func() bool {
			_, err := e.Websocket.Orderbook.GetOrderbook(firstPair, asset.Spot)
			return errors.Is(err, orderbook.ErrOrderbookInvalid)
		}, 2*time.Second, time.Millisecond, "callback must retire the first book before waiting for the publishing worker")
		close(release)
		require.Eventually(t, func() bool { return len(finished) == 1 }, 2*time.Second, time.Millisecond, "retirement must complete")
		require.NoError(t, <-finished, "retirement must finish after the old publication drains")
		notification := <-notifications
		assert.False(t, notification.At.Before(<-published), "invalidation boundary should cover every old connection publication")
	})
	t.Run("missing depth still retires copied books", func(t *testing.T) {
		t.Parallel()
		e := newExchange(t)
		e.Websocket.DataHandler = stream.NewRelay(1)
		pair := currency.NewBTCUSDT()
		require.NoError(t, e.onWebsocketConnectionFault(&FixtureConnection{}, subscription.List{{Channel: futuresOrderbookV2, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{pair}}}), "missing local books must not block recovery")
		payload := <-e.Websocket.DataHandler.C
		notification, ok := payload.Data.(websocket.OrderbookInvalidation)
		require.True(t, ok, "callback must notify detached-book consumers")
		assert.Equal(t, []key.PairAsset{{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: asset.USDTMarginedFutures}}, notification.Pairs, "notification should preserve native asset identity")
	})
	t.Run("non-book subscriptions need no notification", func(t *testing.T) {
		t.Parallel()
		e := newExchange(t)
		e.Websocket.DataHandler = stream.NewRelay(1)
		require.NoError(t, e.onWebsocketConnectionFault(&FixtureConnection{}, subscription.List{{Channel: spotTickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}, {Channel: spotOrderbookV2, Asset: asset.Spot}}), "empty/non-book targets must be harmless")
		assert.Empty(t, e.Websocket.DataHandler.C, "callback should omit empty notifications")
	})
	t.Run("notification cancellation is bounded", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			e := newExchange(t)
			e.Websocket.DataHandler.SetDirectHandler(func(ctx context.Context, _ any) (bool, error) {
				<-ctx.Done()
				return true, ctx.Err()
			})
			started := time.Now()
			err := e.onWebsocketConnectionFault(&FixtureConnection{}, subscription.List{{Channel: spotOrderbookV2, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}})
			assert.ErrorIs(t, err, context.DeadlineExceeded, "blocked notification should return its cancellation classification")
			assert.Equal(t, 2*time.Second, time.Since(started), "notification should use a bounded independent context")
		})
	})
	t.Run("notification failure prevents silent recovery", func(t *testing.T) {
		t.Parallel()
		e := newExchange(t)
		e.Websocket.DataHandler.SetDirectHandler(func(context.Context, any) (bool, error) { return true, errSubscriptionFixture })
		err := e.onWebsocketConnectionFault(&FixtureConnection{}, subscription.List{{Channel: spotOrderbookV2, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}})
		assert.ErrorIs(t, err, errSubscriptionFixture, "notification failure should remain retryable by connection recovery")
	})
}
