package ticker

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
)

// Ticker holds ticker data and an event to be processed as
// a common.DataEventHandler type
type Ticker struct {
	*event.Base
	Last        decimal.Decimal
	High        decimal.Decimal
	Low         decimal.Decimal
	Bid         decimal.Decimal
	Ask         decimal.Decimal
	Volume      decimal.Decimal
	QuoteVolume decimal.Decimal
	Open        decimal.Decimal
	Close       decimal.Decimal
}
