package strategies

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

type StrategyHandler interface {
	Name() string
	OnSignal(interfaces.DataHandler, portfolio.PortfolioHandler) (signal.SignalEvent, error)
}

const errNotFound = "strategy %v not found"
