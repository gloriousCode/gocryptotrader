package kline

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
)

// Kline holds kline data and an event to be processed as
// a common.Event type
type Kline struct {
	*event.Base
	Open             udecimal.Decimal
	Close            udecimal.Decimal
	Low              udecimal.Decimal
	High             udecimal.Decimal
	Volume           udecimal.Decimal
	ValidationIssues string
}

// Event is a kline data event
type Event interface {
	data.Event
	IsKline() bool
}
