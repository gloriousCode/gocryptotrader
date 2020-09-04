package two_outta_three

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"sync"
	"time"
)

const Name = "TwoOuttaThree"

// TOT two outta three is an implementation of ITrader which focusses on
// having two indicators met to sell a position
// for buying, it requires three, lol
type TOT struct {
	sync.RWMutex
	currencyPair currency.Pair
	assetType asset.Item
	exchange   exchange.IBotExchange
	shutdown   chan struct{}
	TickerTime time.Duration
	Interval kline.Interval
	AvailableFunds float64
	LockedFunds float64
	InMarket bool
	Started    int32
	IsSetup    int32
}
