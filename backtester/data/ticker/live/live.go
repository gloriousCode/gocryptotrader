package live

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// LoadData retrieves data from a GoCryptoTrader exchange wrapper which calls the exchange's API for the latest interval
func LoadData(ctx context.Context, exch exchange.IBotExchange, fPair currency.Pair, a asset.Item) (*ticker.Price, error) {
	return exch.FetchTicker(ctx, fPair, a)
}
