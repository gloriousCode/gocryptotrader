package traders

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"time"
)

type ITrader interface {
	Setup(exchangeName string, timer time.Duration, cp currency.Pair, assetType asset.Item, interval kline.Interval) error
	Start() error
	Stop()
	Monitor()
	GetCapital() error
	HasCapital() bool
	DoesMarketMatchBuyConditions() (bool, error)
	DoesMarketMatchSellConditions() (bool, error)
}

