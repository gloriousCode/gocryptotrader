package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc"
)

func TestDummyServerContext(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		getContext func() context.Context
	}{
		{name: "default background"},
		{name: "provided context", getContext: t.Context},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stream := &dummyServer{getContext: tc.getContext}
			if tc.getContext == nil {
				assert.Equal(t, context.Background(), stream.Context(), "default stream should use a background context")
			} else {
				assert.Same(t, tc.getContext(), stream.Context(), "stream should retain its supplied context")
			}
		})
	}
}

func TestDummyServerSendMsg(t *testing.T) {
	t.Parallel()
	t.Run("default discards messages", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, new(dummyServer).SendMsg("message"), "the default stream should discard messages")
	})
	t.Run("callback receives messages and returns error", func(t *testing.T) {
		t.Parallel()
		stream := &dummyServer{send: func(message any) error {
			assert.Equal(t, "message", message, "the callback should receive the sent message")
			return errExpectedTestError
		}}
		assert.ErrorIs(t, stream.SendMsg("message"), errExpectedTestError, "the stream should preserve the callback error")
	})
}

func TestGetOrderbookStream(t *testing.T) {
	t.Parallel()
	t.Run("cancelled idle feed", func(t *testing.T) {
		t.Parallel()
		manager := NewExchangeManager()
		exch, err := manager.NewExchangeByName("binance")
		require.NoError(t, err, "the fake exchange must be created")
		exch.SetDefaults()
		base := exch.GetBase()
		base.Name = newUniqueFakeExchangeName()
		base.Enabled = true
		pair := currency.NewBTCUSDT()
		base.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled:  true,
				Available:     currency.Pairs{pair},
				Enabled:       currency.Pairs{pair},
				ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
				RequestFormat: &currency.PairFormat{Delimiter: "-"},
			},
		}
		require.NoError(t, manager.Add(exch), "the fake exchange must be registered")
		server := &RPCServer{Engine: &Engine{ExchangeManager: manager, Config: &config.Config{}}}
		depth, err := orderbook.DeployDepth(base.Name, pair, asset.Spot)
		require.NoError(t, err, "the test orderbook must be deployed")
		require.NoError(t, depth.LoadSnapshot(&orderbook.Book{
			Bids:        orderbook.Levels{{Price: 1, Amount: 1}},
			Asks:        orderbook.Levels{{Price: 2, Amount: 1}},
			LastUpdated: time.Now(),
		}), "the test orderbook must have an initial snapshot")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		initial := true
		stream := &grpc.GenericServerStream[gctrpc.GetOrderbookStreamRequest, gctrpc.OrderbookResponse]{ServerStream: &dummyServer{getContext: func() context.Context { return ctx }, send: func(any) error {
			if initial {
				initial = false
				return nil
			}
			return ctx.Err()
		}}}
		request := &gctrpc.GetOrderbookStreamRequest{Exchange: base.Name, Pair: &gctrpc.CurrencyPair{Base: pair.Base.String(), Quote: pair.Quote.String()}, AssetType: asset.Spot.String()}
		done := make(chan error, 1)
		go func() { done <- server.GetOrderbookStream(request, stream) }()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled, "the idle stream should return its context cancellation")
			return
		case <-time.After(time.Second):
			assert.Fail(t, "the idle stream should stop without waiting for another market update")
		}
		// Release the handler on failure so cleanup remains bounded.
		cleanup := time.NewTicker(10 * time.Millisecond)
		defer cleanup.Stop()
		deadline := time.NewTimer(time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-done:
				return
			case <-cleanup.C:
				depth.Alert()
			case <-deadline.C:
				require.FailNow(t, "the stream must stop during cleanup")
			}
		}
	})
}

func TestGetExchangeOrderbookStream(t *testing.T) {
	// The dispatcher is a process-wide singleton; these stream cases run serially.
	t.Run("cancelled idle feed", func(t *testing.T) {
		wasRunning := dispatch.IsRunning()
		require.NoError(t, dispatch.EnsureRunning(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit), "the test dispatcher must start")
		t.Cleanup(func() {
			if !wasRunning {
				assert.NoError(t, dispatch.Stop(), "the test dispatcher should stop")
			}
		})
		manager := NewExchangeManager()
		exch, err := manager.NewExchangeByName("binance")
		require.NoError(t, err, "the fake exchange must be created")
		exch.SetDefaults()
		base := exch.GetBase()
		base.Name = newUniqueFakeExchangeName()
		base.Enabled = true
		pair := currency.NewBTCUSDT()
		base.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled:  true,
				Available:     currency.Pairs{pair},
				Enabled:       currency.Pairs{pair},
				ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
				RequestFormat: &currency.PairFormat{Delimiter: "-"},
			},
		}
		require.NoError(t, manager.Add(exch), "the fake exchange must be registered")
		server := &RPCServer{Engine: &Engine{ExchangeManager: manager, Config: &config.Config{}}}
		depth, err := orderbook.DeployDepth(base.Name, pair, asset.Spot)
		require.NoError(t, err, "the test orderbook must be deployed")
		require.NoError(t, depth.LoadSnapshot(&orderbook.Book{
			Bids:        orderbook.Levels{{Price: 1, Amount: 1}},
			Asks:        orderbook.Levels{{Price: 2, Amount: 1}},
			LastUpdated: time.Now(),
		}), "the test orderbook must have an initial snapshot")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		stream := &grpc.GenericServerStream[gctrpc.GetExchangeOrderbookStreamRequest, gctrpc.OrderbookResponse]{ServerStream: &dummyServer{getContext: func() context.Context { return ctx }, send: func(any) error { return ctx.Err() }}}
		request := &gctrpc.GetExchangeOrderbookStreamRequest{Exchange: base.Name}
		done := make(chan error, 1)
		go func() { done <- server.GetExchangeOrderbookStream(request, stream) }()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled, "the idle stream should return its context cancellation")
			return
		case <-time.After(time.Second):
			assert.Fail(t, "the idle stream should stop without waiting for another market update")
		}
		// Release the handler on failure so cleanup remains bounded.
		cleanup := time.NewTicker(10 * time.Millisecond)
		defer cleanup.Stop()
		deadline := time.NewTimer(time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-done:
				return
			case <-cleanup.C:
				depth.Publish()
			case <-deadline.C:
				require.FailNow(t, "the stream must stop during cleanup")
			}
		}
	})
}

func TestGetTickerStream(t *testing.T) {
	// The dispatcher is a process-wide singleton; these stream cases run serially.
	t.Run("cancelled idle feed", func(t *testing.T) {
		wasRunning := dispatch.IsRunning()
		require.NoError(t, dispatch.EnsureRunning(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit), "the test dispatcher must start")
		t.Cleanup(func() {
			if !wasRunning {
				assert.NoError(t, dispatch.Stop(), "the test dispatcher should stop")
			}
		})
		manager := NewExchangeManager()
		exch, err := manager.NewExchangeByName("binance")
		require.NoError(t, err, "the fake exchange must be created")
		exch.SetDefaults()
		base := exch.GetBase()
		base.Name = newUniqueFakeExchangeName()
		base.Enabled = true
		pair := currency.NewBTCUSDT()
		base.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled:  true,
				Available:     currency.Pairs{pair},
				Enabled:       currency.Pairs{pair},
				ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
				RequestFormat: &currency.PairFormat{Delimiter: "-"},
			},
		}
		require.NoError(t, manager.Add(exch), "the fake exchange must be registered")
		server := &RPCServer{Engine: &Engine{ExchangeManager: manager, Config: &config.Config{}}}
		price := &ticker.Price{ExchangeName: base.Name, Pair: pair, AssetType: asset.Spot, Bid: 1, Ask: 2, Last: 1.5}
		require.NoError(t, ticker.ProcessTicker(price), "the test ticker must be registered")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		stream := &grpc.GenericServerStream[gctrpc.GetTickerStreamRequest, gctrpc.TickerResponse]{ServerStream: &dummyServer{getContext: func() context.Context { return ctx }, send: func(any) error { return ctx.Err() }}}
		request := &gctrpc.GetTickerStreamRequest{Exchange: base.Name, Pair: &gctrpc.CurrencyPair{Base: pair.Base.String(), Quote: pair.Quote.String()}, AssetType: asset.Spot.String()}
		done := make(chan error, 1)
		go func() { done <- server.GetTickerStream(request, stream) }()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled, "the idle stream should return its context cancellation")
			return
		case <-time.After(time.Second):
			assert.Fail(t, "the idle stream should stop without waiting for another market update")
		}
		// Release the handler on failure so cleanup remains bounded.
		cleanup := time.NewTicker(10 * time.Millisecond)
		defer cleanup.Stop()
		deadline := time.NewTimer(time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-done:
				return
			case <-cleanup.C:
				assert.NoError(t, ticker.ProcessTicker(price), "cleanup should publish a ticker update")
			case <-deadline.C:
				require.FailNow(t, "the stream must stop during cleanup")
			}
		}
	})
}

func TestGetExchangeTickerStream(t *testing.T) {
	// The dispatcher is a process-wide singleton; these stream cases run serially.
	t.Run("cancelled idle feed", func(t *testing.T) {
		wasRunning := dispatch.IsRunning()
		require.NoError(t, dispatch.EnsureRunning(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit), "the test dispatcher must start")
		t.Cleanup(func() {
			if !wasRunning {
				assert.NoError(t, dispatch.Stop(), "the test dispatcher should stop")
			}
		})
		manager := NewExchangeManager()
		exch, err := manager.NewExchangeByName("binance")
		require.NoError(t, err, "the fake exchange must be created")
		exch.SetDefaults()
		base := exch.GetBase()
		base.Name = newUniqueFakeExchangeName()
		base.Enabled = true
		pair := currency.NewBTCUSDT()
		base.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled:  true,
				Available:     currency.Pairs{pair},
				Enabled:       currency.Pairs{pair},
				ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
				RequestFormat: &currency.PairFormat{Delimiter: "-"},
			},
		}
		require.NoError(t, manager.Add(exch), "the fake exchange must be registered")
		server := &RPCServer{Engine: &Engine{ExchangeManager: manager, Config: &config.Config{}}}
		price := &ticker.Price{ExchangeName: base.Name, Pair: pair, AssetType: asset.Spot, Bid: 1, Ask: 2, Last: 1.5}
		require.NoError(t, ticker.ProcessTicker(price), "the test ticker must be registered")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		stream := &grpc.GenericServerStream[gctrpc.GetExchangeTickerStreamRequest, gctrpc.TickerResponse]{ServerStream: &dummyServer{getContext: func() context.Context { return ctx }, send: func(any) error { return ctx.Err() }}}
		request := &gctrpc.GetExchangeTickerStreamRequest{Exchange: base.Name}
		done := make(chan error, 1)
		go func() { done <- server.GetExchangeTickerStream(request, stream) }()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled, "the idle stream should return its context cancellation")
			return
		case <-time.After(time.Second):
			assert.Fail(t, "the idle stream should stop without waiting for another market update")
		}
		// Release the handler on failure so cleanup remains bounded.
		cleanup := time.NewTicker(10 * time.Millisecond)
		defer cleanup.Stop()
		deadline := time.NewTimer(time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-done:
				return
			case <-cleanup.C:
				assert.NoError(t, ticker.ProcessTicker(price), "cleanup should publish a ticker update")
			case <-deadline.C:
				require.FailNow(t, "the stream must stop during cleanup")
			}
		}
	})
}
