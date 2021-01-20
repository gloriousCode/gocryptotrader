package risk

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Handler defines what is expected to be able to assess risk of an order
type Handler interface {
	EvaluateOrder(order.Event, []holdings.Holding, compliance.Snapshot) (*order.Order, error)
}

// Risk contains all currency settings in order to evaluate potential orders
type Risk struct {
	CurrencySettings map[string]map[asset.Item]map[currency.Pair]*CurrencySettings
	CanUseLeverage   bool
	MaximumLeverage  float64
}

// CurrencySettings contains relevant limits to assess risk
type CurrencySettings struct {
	MaxLeverageRatio    float64
	MaxLeverageRate     float64
	MaximumHoldingRatio float64
}
