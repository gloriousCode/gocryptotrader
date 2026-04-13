package options

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Trade is a normalised options trade payload for websocket and reconciliation flows.
type Trade struct {
	ExchangeName      string
	Pair              currency.Pair
	AssetType         asset.Item
	InstrumentID      string
	TradeID           string
	Side              order.Side
	Price             float64
	Size              float64
	ExchangeTimestamp time.Time
	ReceivedAt        time.Time
	Sequence          int64
}
