package strategies

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/strategybase"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
)

func TestAddStrategies(t *testing.T) {
	t.Parallel()
	err := addStrategies(nil)
	if !errors.Is(err, errNoStrategies) {
		t.Error(err)
	}

	err = addStrategies([]strategybase.Handler{&dollarcostaverage.Strategy{
		Strategy: strategybase.Strategy{Name: dollarcostaverage.Name},
	}})
	if !errors.Is(err, strategies.ErrStrategyAlreadyExists) {
		t.Error(err)
	}

	err = addStrategies([]strategybase.Handler{&CustomStrategy{}})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
}

type CustomStrategy struct {
	strategybase.Strategy
}

func (s *CustomStrategy) New() strategybase.Handler {
	return &CustomStrategy{}
}

func (s *CustomStrategy) GetName() string {
	return "custom-strategy"
}

func (s *CustomStrategy) GetDescription() string {
	return "this is a demonstration of loading strategies via custom plugins"
}

func (s *CustomStrategy) SupportsSimultaneousProcessing() bool {
	return true
}

func (s *CustomStrategy) OnSignal(d data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	return s.createSignal(d)
}
func (s *CustomStrategy) OnSimultaneousSignals(d []data.Handler, f funding.IFundingTransferer, p portfolio.Handler) ([]signal.Event, error) {
	return nil, nil
}

func (s *CustomStrategy) createSignal(d data.Handler) (*signal.Signal, error) {
	return nil, nil
}

func (s *CustomStrategy) SetCustomSettings(json.RawMessage) error {
	return nil
}

func (s *CustomStrategy) SetDefaults() {}
