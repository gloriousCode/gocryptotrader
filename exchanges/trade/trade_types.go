package trade

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	buffer []Data
	candles []kline.Candle
	processor Processor
)

// Data defines trade data
type Data struct {
	ID uuid.UUID
	Timestamp    time.Time
	Exchange     string
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Price        float64
	Amount       float64
	Side         order.Side
}

type CandleHolder struct {
	candle kline.Candle
	trades []Data
}

// Processor used for processing trade data in batches
// and saving them to the database
type Processor struct {
	mutex sync.Mutex
	shutdownC chan struct{}
	started    int32
}

type ByDate []Data

func (b ByDate) Len() int {
	return len(b)
}

func (b ByDate) Less(i, j int) bool {
	return b[i].Timestamp.Before(b[j].Timestamp)
}

func (b ByDate) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
