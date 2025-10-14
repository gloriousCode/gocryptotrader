package order

import (
	"github.com/quagmt/udecimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Order contains all details for an order event
type Order struct {
	*event.Base
	ID                  string
	Direction           order.Side
	Status              order.Status
	ClosePrice          udecimal.Decimal
	Amount              udecimal.Decimal
	OrderType           order.Type
	Leverage            udecimal.Decimal
	AllocatedFunds      udecimal.Decimal
	BuyLimit            udecimal.Decimal
	SellLimit           udecimal.Decimal
	FillDependentEvent  signal.Event
	ClosingPosition     bool
	LiquidatingPosition bool
}

// Event inherits common event interfaces along with extra functions related to handling orders
type Event interface {
	common.Event
	common.Directioner
	GetClosePrice() udecimal.Decimal
	GetBuyLimit() udecimal.Decimal
	GetSellLimit() udecimal.Decimal
	SetAmount(udecimal.Decimal)
	GetAmount() udecimal.Decimal
	IsOrder() bool
	GetStatus() order.Status
	SetID(id string)
	GetID() string
	IsLeveraged() bool
	GetAllocatedFunds() udecimal.Decimal
	GetFillDependentEvent() signal.Event
	IsClosingPosition() bool
	IsLiquidating() bool
}
