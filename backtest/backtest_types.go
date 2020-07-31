package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/data"
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/backtest/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtest/strategy"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Backtest struct {
	Pair currency.Pair

	Data       data.Handler
	portfolio  portfolio.Handler
	strategy   strategy.Handler
	eventQueue []event.Handler
}