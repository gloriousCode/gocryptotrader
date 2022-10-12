package size

import (
	"errors"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
)

var (
	// ErrCantUseLeverageAndMatchOrderAmount returned when requesting leverage AND matching order size with spot
	ErrCantUseLeverageAndMatchOrderAmount = errors.New("asd")

	errNoFunds         = errors.New("no funds available")
	errLessThanMinimum = errors.New("sized amount less than minimum")
	errCannotAllocate  = errors.New("portfolio manager cannot allocate funds for an order")
)

// Size contains buy and sell side rules
type Size struct {
	BuySide  exchange.MinMax
	SellSide exchange.MinMax
}

// Request is the request to size an order
type Request struct {
	OrderEvent      order.Event
	AmountAvailable decimal.Decimal
	Settings        *exchange.Settings
	CanUseLeverage  bool
	Leverage        float64
}

// Response is the response to size an order
type Response struct {
	Order    *order.Order
	Fee      decimal.Decimal
	Leverage float64
}
