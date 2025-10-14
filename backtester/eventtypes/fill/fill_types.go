package fill

import (
	"github.com/quagmt/udecimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Fill is an event that details the events from placing an order
type Fill struct {
	*event.Base
	Direction           order.Side       `json:"side"`
	Amount              udecimal.Decimal `json:"amount"`
	ClosePrice          udecimal.Decimal `json:"close-price"`
	VolumeAdjustedPrice udecimal.Decimal `json:"volume-adjusted-price"`
	PurchasePrice       udecimal.Decimal `json:"purchase-price"`
	Total               udecimal.Decimal `json:"total"`
	ExchangeFee         udecimal.Decimal `json:"exchange-fee"`
	Slippage            udecimal.Decimal `json:"slippage"`
	Order               *order.Detail    `json:"-"`
	FillDependentEvent  signal.Event
	Liquidated          bool
}

// Event holds all functions required to handle a fill event
type Event interface {
	common.Event
	common.Directioner

	SetAmount(udecimal.Decimal)
	GetAmount() udecimal.Decimal
	GetClosePrice() udecimal.Decimal
	GetVolumeAdjustedPrice() udecimal.Decimal
	GetSlippageRate() udecimal.Decimal
	GetPurchasePrice() udecimal.Decimal
	GetTotal() udecimal.Decimal
	GetExchangeFee() udecimal.Decimal
	SetExchangeFee(udecimal.Decimal)
	GetOrder() *order.Detail
	GetFillDependentEvent() signal.Event
	IsLiquidated() bool
}
