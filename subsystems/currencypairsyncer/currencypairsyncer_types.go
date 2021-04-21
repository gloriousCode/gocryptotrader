package currencypairsyncer

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// iWebsocketDataReceiver limits exposure of accessible functions to websocket data receiver
type iWebsocketDataReceiver interface {
	IsRunning() bool
	WebsocketDataReceiver(ws *stream.Websocket)
	WebsocketDataHandler(string, interface{}) error
}

// iExchangeManager limits exposure of accessible functions to exchange manager
type iExchangeManager interface {
	GetExchanges() []exchange.IBotExchange
	GetExchangeByName(string) exchange.IBotExchange
}

// syncBase stores information
type syncBase struct {
	IsUsingWebsocket bool
	IsUsingREST      bool
	IsProcessing     bool
	LastUpdated      time.Time
	HaveData         bool
	NumErrors        int
}

// currencyPairSyncAgent stores the sync agent info
type currencyPairSyncAgent struct {
	Created   time.Time
	Exchange  string
	AssetType asset.Item
	Pair      currency.Pair
	Ticker    syncBase
	Orderbook syncBase
	Trade     syncBase
}

// Config stores the currency pair config
type Config struct {
	SyncTicker       bool
	SyncOrderbook    bool
	SyncTrades       bool
	SyncContinuously bool
	SyncTimeout      time.Duration
	NumWorkers       int
	Verbose          bool
}

// Manager stores the exchange currency pair syncer object
type Manager struct {
	initSyncCompleted   int32
	initSyncStarted     int32
	started             int32
	delimiter           string
	uppercase           bool
	initSyncStartTime   time.Time
	fiatDisplayCurrency currency.Code
	mux                 sync.Mutex
	initSyncWG          sync.WaitGroup

	currencyPairs            []currencyPairSyncAgent
	tickerBatchLastRequested map[string]time.Time

	remoteConfig          *config.RemoteControlConfig
	config                Config
	exchangeManager       iExchangeManager
	websocketDataReceiver iWebsocketDataReceiver
}
