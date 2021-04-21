package websocketroutinemanager

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// Manager is used to process websocket updates from a unified location
type Manager struct {
	started         int32
	verbose         bool
	exchangeManager iExchangeManager
	orderManager    iOrderManager
	syncer          iCurrencyPairSyncer
	currencyConfig  *config.CurrencyConfig
	shutdown        chan struct{}
	wg              sync.WaitGroup
}

// iExchangeManager defines a limited scoped exchange manager
type iExchangeManager interface {
	GetExchanges() []exchange.IBotExchange
}

// iCurrencyPairSyncer defines a limited scoped currency pair syncer
type iCurrencyPairSyncer interface {
	IsRunning() bool
	PrintTickerSummary(*ticker.Price, string, error)
	PrintOrderbookSummary(*orderbook.Base, string, error)
	Update(string, currency.Pair, asset.Item, int, error) error
}

// iOrderManager defines a limited scoped order manager
type iOrderManager interface {
	Exists(*order.Detail) bool
	Add(*order.Detail) error
	Cancel(*order.Cancel) error
	GetByExchangeAndID(string, string) (*order.Detail, error)
	UpdateExistingOrder(*order.Detail) error
}

var (
	errNilExchangeManager    = errors.New("nil exchange manager received")
	errNilOrderManager       = errors.New("nil order manager received")
	errNilCurrencyPairSyncer = errors.New("nil currency pair syncer received")
	errNilCurrencyConfig     = errors.New("nil currency config received")
	errNilCurrencyPairFormat = errors.New("nil currency pair format received")
)
