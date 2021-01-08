package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
)

// CandleEvent for OHLCV tick data
type CandleEvent interface {
	common.DataEventHandler
}
