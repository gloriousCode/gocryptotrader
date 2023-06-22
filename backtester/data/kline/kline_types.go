package kline

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var errNoCandleData = errors.New("no candle data provided")

// CandleEvents is a struct which implements the data.Streamer interface
// It holds candle data for a specified range with helper functions
type CandleEvents struct {
	*data.Base
	Item         *gctkline.Item
	RangeHolder  *gctkline.IntervalRangeHolder
	FundingRates *fundingrate.Rates
}
