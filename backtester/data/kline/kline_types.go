package kline

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

var errNoCandleData = errors.New("no candle data provided")

// PriceData is a struct which implements the data.Streamer interface
// It holds candle data for a specified range with helper functions
type PriceData struct {
	data.Base
	addedTimes  map[int64]bool
	KLine       gctkline.Item
	TickerStuff []ticker.Price
	RangeHolder *gctkline.IntervalRangeHolder
}
