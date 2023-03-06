package risk

import (
	"errors"

	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
)

var (
	errNoCurrencySettings       = errors.New("lacking currency settings, cannot evaluate order")
	errLeverageNotAllowed       = errors.New("order is using leverage when leverage is not enabled in config")
	errCannotPlaceLeverageOrder = errors.New("cannot place leveraged order")
)

// Handler defines what is expected to be able to assess risk of an order
type Handler interface {
	EvaluateOrder(order.Event, []holdings.Holding, compliance.Snapshot) (*order.Order, error)
}

// Risk contains all currency settings in order to evaluate potential orders
type Risk struct {
	canUseLeverage  bool
	currentLeverage float64
	leverageTarget  float64
	maxLVR          decimal.Decimal
	riskFreeRate    decimal.Decimal
}

// CurrencySettings contains relevant limits to assess risk
// TODO utilise leverage at a currency level
type CurrencySettings struct {
	TargetLeverage float64
	MaxLVR         decimal.Decimal
}
