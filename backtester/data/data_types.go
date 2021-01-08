package data

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	CandleType common.DataType = iota
)

type HandlerPerCurrency struct {
	data map[string]map[asset.Item]map[currency.Pair]Handler
}

type Holder interface {
	Setup()
	SetDataForCurrency(string, asset.Item, currency.Pair, Handler)
	GetAllData() map[string]map[asset.Item]map[currency.Pair]Handler
	GetDataForCurrency(string, asset.Item, currency.Pair) Handler
	Reset()
}

type Data struct {
	latest common.DataEventHandler
	stream []common.DataEventHandler
	offset int
}

// Handler interface for Loading and Streaming data
type Handler interface {
	Loader
	Streamer
	Reset()
}

// Loader interface for Loading data into backtest supported format
type Loader interface {
	Load() error
}

// Streamer interface handles loading, parsing, distributing BackTest data
type Streamer interface {
	Next() (common.DataEventHandler, bool)
	GetStream() []common.DataEventHandler
	History() []common.DataEventHandler
	Latest() common.DataEventHandler
	List() []common.DataEventHandler
	Offset() int

	StreamOpen() []float64
	StreamHigh() []float64
	StreamLow() []float64
	StreamClose() []float64
	StreamVol() []float64

	HasDataAtTime(time.Time) bool
}
