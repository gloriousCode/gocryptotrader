package ticker

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

var errNoTickerData = errors.New("no ticker data provided")

// Data is a struct which implements the data.Streamer interface
// It holds candle data for a specified range with helper functions
type Data struct {
	data.Base
	addedTimes     map[int64]bool
	PollTime       kline.Interval
	Tickers        []ticker.Price
	UnderlyingPair currency.Pair
}
