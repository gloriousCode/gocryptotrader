package currencypairsyncer

import (
	"sync"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"

	"github.com/thrasher-corp/gocryptotrader/config"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// CurrencyPairSyncerConfig stores the currency pair config
type CurrencyPairSyncerConfig struct {
	SyncTicker       bool
	SyncOrderbook    bool
	SyncTrades       bool
	SyncContinuously bool
	SyncTimeout      time.Duration
	NumWorkers       int
	Verbose          bool
}

// ExchangeSyncerConfig stores the exchange syncer config
type ExchangeSyncerConfig struct {
	SyncDepositAddresses bool
	SyncOrders           bool
}

type ExchangeManagerGetExchanges interface {
	GetExchanges() []exchange.IBotExchange
}

// ExchangeCurrencyPairSyncer stores the exchange currency pair syncer object
type ExchangeCurrencyPairSyncer struct {
	Cfg                      CurrencyPairSyncerConfig
	CurrencyPairs            []CurrencyPairSyncAgent
	tickerBatchLastRequested map[string]time.Time
	mux                      sync.Mutex
	initSyncWG               sync.WaitGroup

	exchangeManager       ExchangeManagerGetExchanges
	initSyncCompleted     int32
	initSyncStarted       int32
	initSyncStartTime     time.Time
	shutdown              int32
	fiatDisplayCurrency   currency.Code
	delimiter             string
	uppercase             bool
	remoteConfig          *config.RemoteControlConfig
	websocketDataReciever botWebsocketDataReceiver
}

// SyncBase stores information
type SyncBase struct {
	IsUsingWebsocket bool
	IsUsingREST      bool
	IsProcessing     bool
	LastUpdated      time.Time
	HaveData         bool
	NumErrors        int
}

// CurrencyPairSyncAgent stores the sync agent info
type CurrencyPairSyncAgent struct {
	Created   time.Time
	Exchange  string
	AssetType asset.Item
	Pair      currency.Pair
	Ticker    SyncBase
	Orderbook SyncBase
	Trade     SyncBase
}
