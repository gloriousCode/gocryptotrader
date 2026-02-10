package engine

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

var benchErrSink uint64

type handlerBenchmarkCase struct {
	name          string
	data          any
	syncerRunning bool
	orderRunning  bool
}

func newBenchHandlerManager(syncerRunning, orderRunning bool) (*WebsocketRoutineManager, func(), error) {
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		return nil, nil, err
	}
	exch.SetDefaults()
	if err := em.Add(exch); err != nil {
		return nil, nil, err
	}

	syncer, err := SetupSyncManager(&config.SyncManagerConfig{
		SynchronizeTicker:       true,
		SynchronizeOrderbook:    true,
		SynchronizeTrades:       true,
		SynchronizeContinuously: true,
		FiatDisplayCurrency:     currency.USD,
		PairFormatDisplay:       &currency.EMPTYFORMAT,
		LogSyncUpdateEvents:     false,
	}, em, &config.RemoteControlConfig{}, false)
	if err != nil {
		return nil, nil, err
	}
	if syncerRunning {
		if err := syncer.Start(); err != nil {
			return nil, nil, err
		}
	}

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{})
	if err != nil {
		return nil, nil, err
	}
	if orderRunning {
		if err := om.Start(); err != nil {
			return nil, nil, err
		}
	}

	cleanup := func() {
		if orderRunning && om.IsRunning() {
			_ = om.Stop()
		}
		if syncerRunning && syncer.IsRunning() {
			_ = syncer.Stop()
		}
	}

	return &WebsocketRoutineManager{
		syncer:       syncer,
		orderManager: om,
	}, cleanup, nil
}

func BenchmarkWebsocketDataHandlerTypes(b *testing.B) {
	exchangeName := "Bitstamp"
	pair := currency.NewPair(currency.BTC, currency.USD)
	now := time.Unix(0, 0)
	baseTicker := &ticker.Price{
		ExchangeName: exchangeName,
		Pair:         pair,
		AssetType:    asset.Spot,
		LastUpdated:  now,
	}
	baseOrder := &order.Detail{
		Exchange:  exchangeName,
		Pair:      pair,
		AssetType: asset.Spot,
		OrderID:   "1",
	}
	baseDepth := orderbook.NewDepth(uuid.Must(uuid.NewV4()))
	baseFunding := websocket.FundingData{
		Timestamp:    now,
		CurrencyPair: pair,
		AssetType:    asset.Spot,
		Exchange:     exchangeName,
		Amount:       1,
		Rate:         0.01,
		Period:       8,
		Side:         order.Buy,
	}
	baseKline := websocket.KlineData{
		Timestamp: now,
		Pair:      pair,
		AssetType: asset.Spot,
		Exchange:  exchangeName,
		Interval:  "1m",
	}
	baseAccount := accounts.Change{AssetType: asset.Spot}
	baseTrade := trade.Data{
		Exchange:     exchangeName,
		AssetType:    asset.Spot,
		CurrencyPair: pair,
		Timestamp:    now,
		Price:        1,
		Amount:       1,
	}
	baseFill := fill.Data{
		Exchange:     exchangeName,
		AssetType:    asset.Spot,
		CurrencyPair: pair,
		Price:        1,
		Amount:       1,
	}

	cases := []handlerBenchmarkCase{
		{name: "string", data: "ok"},
		{name: "error", data: errors.New("bench")},
		{name: "funding_data", data: baseFunding},
		{name: "ticker_ptr-w-syncer", data: baseTicker, syncerRunning: true},
		{name: "ticker_slice-w-syncer", data: []ticker.Price{*baseTicker}, syncerRunning: true},
		{name: "ticker_ptr-no-syncer", data: baseTicker, syncerRunning: false},
		{name: "ticker_slice-no-syncer", data: []ticker.Price{*baseTicker}, syncerRunning: false},
		{name: "err_use_pointer_ticker", data: ticker.Price{ExchangeName: exchangeName}},
		{name: "err_use_pointer_order", data: order.Detail{Exchange: exchangeName}},
		{name: "err_use_pointer_orderbook", data: orderbook.Depth{}},
		{name: "kline_data", data: baseKline},
		{name: "kline_slice", data: []websocket.KlineData{baseKline}},
		{name: "orderbook_ptr-w-syncer", data: baseDepth, syncerRunning: true},
		{name: "order_detail_ptr-w-syncer", data: baseOrder, orderRunning: true},
		{name: "order_detail_slice-w-syncer", data: []order.Detail{*baseOrder}, orderRunning: true},
		{name: "orderbook_ptr-no-syncer", data: baseDepth, syncerRunning: false},
		{name: "order_detail_ptr-no-syncer", data: baseOrder, orderRunning: false},
		{name: "order_detail_slice-no-syncer", data: []order.Detail{*baseOrder}, orderRunning: false},
		{name: "unhandled_warning", data: websocket.UnhandledMessageWarning{Message: "bench"}},
		{name: "accounts_change", data: baseAccount},
		{name: "accounts_change_slice", data: []accounts.Change{baseAccount}},
		{name: "trade_data", data: baseTrade},
		{name: "trade_data_slice", data: []trade.Data{baseTrade}},
		{name: "fill_slice", data: []fill.Data{baseFill}},
		{name: "default_type", data: struct{ Name string }{Name: "bench"}},
	}

	for _, benchCase := range cases {
		b.Run(benchCase.name, func(b *testing.B) {
			m, cleanup, err := newBenchHandlerManager(benchCase.syncerRunning, benchCase.orderRunning)
			if err != nil {
				b.Fatalf("failed to setup managers: %v", err)
			}
			defer cleanup()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := m.websocketDataHandler(exchangeName, benchCase.data); err != nil {
					atomic.AddUint64(&benchErrSink, 1)
				}
			}
		})
	}
}
