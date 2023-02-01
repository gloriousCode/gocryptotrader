package strategies

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/strategybase"
)

// ErrStrategyAlreadyExists returned when a strategy matches the same name
var ErrStrategyAlreadyExists = errors.New("strategy already exists")

// StrategyHolder holds strategies
type StrategyHolder []strategybase.Handler
