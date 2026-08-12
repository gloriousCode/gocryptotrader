package options

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// OrderbookLevel is a normalised options orderbook price level.
type OrderbookLevel struct {
	Price  float64
	Amount float64
}

// Orderbook is a normalised options orderbook snapshot or delta payload.
type Orderbook struct {
	ExchangeName      string
	Pair              currency.Pair
	AssetType         asset.Item
	InstrumentID      string
	IsSnapshot        bool
	Bids              []OrderbookLevel
	Asks              []OrderbookLevel
	ExchangeTimestamp time.Time
	ReceivedAt        time.Time
	Sequence          int64
	PrevSequence      int64
}
