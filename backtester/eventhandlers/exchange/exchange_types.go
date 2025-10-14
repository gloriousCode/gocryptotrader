package exchange

import (
	"errors"

	"github.com/quagmt/udecimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrCannotTransact returns when its an issue to do nothing for an event
	ErrCannotTransact = errors.New("cannot transact")

	errExceededPortfolioLimit  = errors.New("exceeded portfolio limit")
	errNilCurrencySettings     = errors.New("received nil currency settings")
	errInvalidDirection        = errors.New("received invalid order direction")
	errNoCurrencySettingsFound = errors.New("no currency settings found")
)

// ExecutionHandler interface dictates what functions are required to submit an order
type ExecutionHandler interface {
	SetExchangeAssetCurrencySettings(asset.Item, currency.Pair, *Settings)
	GetCurrencySettings(string, asset.Item, currency.Pair) (Settings, error)
	ExecuteOrder(order.Event, data.Handler, *engine.OrderManager, funding.IFundReleaser) (fill.Event, error)
	Reset() error
}

// Exchange contains all the currency settings
type Exchange struct {
	CurrencySettings []Settings
}

// Settings allow the eventhandler to size an order within the limitations set by the config file
type Settings struct {
	Exchange      exchange.IBotExchange
	UseRealOrders bool

	Pair  currency.Pair
	Asset asset.Item

	MakerFee udecimal.Decimal
	TakerFee udecimal.Decimal

	BuySide  MinMax
	SellSide MinMax

	Leverage Leverage

	MinimumSlippageRate udecimal.Decimal
	MaximumSlippageRate udecimal.Decimal

	CanUseExchangeLimits    bool
	Limits                  limits.MinMaxLevel
	SkipCandleVolumeFitting bool

	UseExchangePNLCalculation bool
}

// MinMax are the rules which limit the placement of orders.
type MinMax struct {
	MinimumSize  udecimal.Decimal
	MaximumSize  udecimal.Decimal
	MaximumTotal udecimal.Decimal
}

// Leverage rules are used to allow or limit the use of leverage in orders
// when supported
type Leverage struct {
	CanUseLeverage                 bool
	MaximumOrdersWithLeverageRatio udecimal.Decimal
	MaximumLeverageRate            udecimal.Decimal
}
