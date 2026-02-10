package engine

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
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

type benchSyncer struct {
	running bool
}

var benchErrSink uint64

func (b benchSyncer) IsRunning() bool { return b.running }

func (b benchSyncer) PrintTickerSummary(_ *ticker.Price, _ string, _ error) {}

func (b benchSyncer) PrintWebsocketOrderbookSummary(*orderbook.Depth) {}

func (b benchSyncer) WebsocketUpdate(_ string, _ currency.Pair, _ asset.Item, _ syncItemType, _ error) error {
	return nil
}

type benchOrderManager struct {
	running bool
}

func (b benchOrderManager) IsRunning() bool { return b.running }

func (b benchOrderManager) DataHandlerUpsert(_ *order.Detail) error { return nil }

type handlerBenchmarkCase struct {
	name          string
	data          any
	syncerRunning bool
	orderRunning  bool
}

func newBenchHandlerManager(syncerRunning, orderRunning bool) *WebsocketRoutineManager {
	return &WebsocketRoutineManager{
		syncer:       benchSyncer{running: syncerRunning},
		orderManager: benchOrderManager{running: orderRunning},
	}
}

func BenchmarkWebsocketDataHandlerTypes(b *testing.B) {
	pair := currency.NewPair(currency.BTC, currency.USD)
	now := time.Unix(0, 0)
	baseTicker := &ticker.Price{
		ExchangeName: "bench",
		Pair:         pair,
		AssetType:    asset.Spot,
		LastUpdated:  now,
	}
	baseOrder := &order.Detail{
		Exchange:  "bench",
		Pair:      pair,
		AssetType: asset.Spot,
		OrderID:   "1",
	}
	baseDepth := orderbook.NewDepth(uuid.Must(uuid.NewV4()))
	baseFunding := websocket.FundingData{
		Timestamp:    now,
		CurrencyPair: pair,
		AssetType:    asset.Spot,
		Exchange:     "bench",
		Amount:       1,
		Rate:         0.01,
		Period:       8,
		Side:         order.Buy,
	}
	baseKline := websocket.KlineData{
		Timestamp: now,
		Pair:      pair,
		AssetType: asset.Spot,
		Exchange:  "bench",
		Interval:  "1m",
	}
	baseAccount := accounts.Change{AssetType: asset.Spot}
	baseTrade := trade.Data{
		Exchange:     "bench",
		AssetType:    asset.Spot,
		CurrencyPair: pair,
		Timestamp:    now,
		Price:        1,
		Amount:       1,
	}
	baseFill := fill.Data{
		Exchange:     "bench",
		AssetType:    asset.Spot,
		CurrencyPair: pair,
		Price:        1,
		Amount:       1,
	}

	cases := []handlerBenchmarkCase{
		{name: "string", data: "ok"},
		{name: "error", data: errors.New("bench")},
		{name: "funding_data", data: baseFunding},
		{name: "ticker_ptr", data: baseTicker, syncerRunning: true},
		{name: "ticker_slice", data: []ticker.Price{*baseTicker}, syncerRunning: true},
		{name: "err_use_pointer_ticker", data: ticker.Price{ExchangeName: "bench"}},
		{name: "err_use_pointer_order", data: order.Detail{Exchange: "bench"}},
		{name: "err_use_pointer_orderbook", data: orderbook.Depth{}},
		{name: "kline_data", data: baseKline},
		{name: "kline_slice", data: []websocket.KlineData{baseKline}},
		{name: "orderbook_ptr", data: baseDepth, syncerRunning: true},
		{name: "order_detail_ptr", data: baseOrder, orderRunning: true},
		{name: "order_detail_slice", data: []order.Detail{*baseOrder}, orderRunning: true},
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
			m := newBenchHandlerManager(benchCase.syncerRunning, benchCase.orderRunning)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := m.websocketDataHandler("bench", benchCase.data); err != nil {
					atomic.AddUint64(&benchErrSink, 1)
				}
			}
		})
	}
}
