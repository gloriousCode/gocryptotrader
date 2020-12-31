package common

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// DecimalPlaces is a lovely little holder
	// for the amount of decimal places we want to allow
	DecimalPlaces = 8
	// DoNothing is an explicit signal for the backtester to not perform an action
	// based upon indicator results
	DoNothing order.Side = "DO NOTHING"
	// CouldNotBuy/Sell is flagged when a BUY/SELL signal is raised in the strategy/signal phase, but the
	// portfolio manager or exchange cannot place an order
	CouldNotBuy  order.Side = "COULD NOT BUY"
	CouldNotSell order.Side = "COULD NOT SELL"
	// Missing Data is signalled during the strategy/signal phase when data has been identified as missing
	// No buy or sell events can occur
	MissingData order.Side = "MISSING DATA"
	// used to identify the type of data in a config
	CandleStr = "candle"
	TradeStr  = "trade"
)

// EventHandler interface implements required GetTime() & Pair() return
type EventHandler interface {
	IsEvent() bool
	GetTime() time.Time
	Pair() currency.Pair
	GetExchange() string
	GetInterval() kline.Interval
	GetAssetType() asset.Item

	GetWhy() string
	AppendWhy(string)
}

// DataHandler interface used for loading and interacting with Data
type DataEventHandler interface {
	EventHandler
	DataType() DataType
	Price() float64
}

type DataType uint8

// Directioner dictates the side of an order
type Directioner interface {
	SetDirection(side order.Side)
	GetDirection() order.Side
}
