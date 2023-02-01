package strategies

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/technicalanalysis"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/binancecashandcarry"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/strategybase"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/top2bottom2"
	"github.com/thrasher-corp/gocryptotrader/common"
)

// LoadStrategyByName returns the strategy by its name
func LoadStrategyByName(name string, useSimultaneousProcessing bool) (strategybase.Handler, error) {
	strategies := GetSupportedStrategies()
	var strategy strategybase.Handler
	var err error
	for i := range strategies {
		strategy, err = createNewStrategy(name, useSimultaneousProcessing, strategies[i])
		if err != nil {
			if errors.Is(err, strategybase.ErrStrategyNotFound) {
				continue
			}
			return nil, err
		}
		break
	}
	return strategy, nil
}

func createNewStrategy(name string, useSimultaneousProcessing bool, h strategybase.Handler) (strategybase.Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("cannot load %v supported strategies contains %w", name, common.ErrNilPointer)
	}
	// ensure that we use a new instance of a strategy
	newStrategy := h.New()
	if newStrategy.GetName() != name {
		return nil, fmt.Errorf("%w %v", strategybase.ErrStrategyNotFound, name)
	}
	if useSimultaneousProcessing && !newStrategy.SupportsSimultaneousProcessing() {
		return nil, strategybase.ErrSimultaneousProcessingNotSupported
	}
	newStrategy.SetSimultaneousProcessing(useSimultaneousProcessing)
	return newStrategy, nil
}

// GetSupportedStrategies returns a static list of set strategies
// they must be set in here for the backtester to recognise them
func GetSupportedStrategies() StrategyHolder {
	m.Lock()
	defer m.Unlock()
	return supportedStrategies
}

// AddStrategy will add a strategy to the list of strategies
func AddStrategy(strategy strategybase.Handler) error {
	if strategy == nil {
		return fmt.Errorf("%w strategy handler", common.ErrNilPointer)
	}
	m.Lock()
	defer m.Unlock()
	for i := range supportedStrategies {
		if strings.EqualFold(supportedStrategies[i].GetName(), strategy.GetName()) {
			return fmt.Errorf("'%v' %w", strategy.GetName(), ErrStrategyAlreadyExists)
		}
	}
	supportedStrategies = append(supportedStrategies, strategy)
	return nil
}

var (
	m sync.Mutex

	supportedStrategies = StrategyHolder{
		new(dollarcostaverage.Strategy).New(),
		new(technicalanalysis.Strategy).New(),
		new(top2bottom2.Strategy).New(),
		new(binancecashandcarry.Strategy).New(),
	}
)
