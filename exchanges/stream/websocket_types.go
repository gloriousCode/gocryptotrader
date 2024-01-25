package stream

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// Websocket functionality list and state consts
const (
	// WebsocketNotEnabled alerts of a disabled websocket
	WebsocketNotEnabled                = "exchange_websocket_not_enabled"
	WebsocketNotAuthenticatedUsingRest = "%v - Websocket not authenticated, using REST\n"
	Ping                               = "ping"
	Pong                               = "pong"
	UnhandledMessage                   = " - Unhandled websocket message: "
)

var (
	ErrWebsocketNotFound = errors.New("websocket not found")
	ErrKeyInUse          = errors.New("key already in use")
)

type WebsocketByKey struct {
	mutex      sync.RWMutex
	websockets map[any]*Websocket
}

func (w *WebsocketByKey) Shutdown() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for _, ws := range w.websockets {
		err := ws.Shutdown()
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WebsocketByKey) GetByKey(key any) (*Websocket, error) {
	if key == nil {
		return nil, fmt.Errorf("%w %v", common.ErrNilPointer, "key")
	}
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	if ws, ok := w.websockets[key]; ok {
		return ws, nil
	}
	return nil, fmt.Errorf("%w %v", ErrWebsocketNotFound, key)
}

func (w *WebsocketByKey) Add(key any, ws *Websocket) error {
	if key == nil {
		return fmt.Errorf("%w %v", common.ErrNilPointer, "key")
	}
	if ws == nil {
		return fmt.Errorf("%w %v", common.ErrNilPointer, "ws")
	}
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if _, ok := w.websockets[key]; ok {
		return fmt.Errorf("%w %v", ErrKeyInUse, key)
	}
	w.websockets[key] = ws
	return nil
}

func (w *WebsocketByKey) Remove(key any) error {
	if key == nil {
		return fmt.Errorf("%w %v", common.ErrNilPointer, "key")
	}
	w.mutex.Lock()
	defer w.mutex.Unlock()
	resp, ok := w.websockets[key]
	if !ok {
		return fmt.Errorf("%w %v", ErrWebsocketNotFound, key)
	}
	err := resp.Shutdown()
	if err != nil {
		return err
	}
	delete(w.websockets, key)
	return nil
}

type subscriptionMap map[any]*ChannelSubscription
type subscriptionMap map[any]*subscription.Subscription

// Websocket defines a return type for websocket connections via the interface
// wrapper for routine processing
type Websocket struct {
	canUseAuthenticatedEndpoints bool
	enabled                      bool
	Init                         bool
	connected                    bool
	connecting                   bool
	verbose                      bool
	connectionMonitorRunning     bool
	trafficMonitorRunning        bool
	dataMonitorRunning           bool
	trafficTimeout               time.Duration
	connectionMonitorDelay       time.Duration
	proxyAddr                    string
	defaultURL                   string
	defaultURLAuth               string
	runningURL                   string
	runningURLAuth               string
	exchangeName                 string
	m                            sync.Mutex
	fieldMutex                   sync.RWMutex
	connector                    func() error

	subscriptionMutex sync.RWMutex
	subscriptions     subscriptionMap
	Subscribe         chan []subscription.Subscription
	Unsubscribe       chan []subscription.Subscription

	// Subscriber function for package defined websocket subscriber
	// functionality
	Subscriber func([]subscription.Subscription) error
	// Unsubscriber function for packaged defined websocket unsubscriber
	// functionality
	Unsubscriber func([]subscription.Subscription) error
	// GenerateSubs function for package defined websocket generate
	// subscriptions functionality
	GenerateSubs func() ([]subscription.Subscription, error)

	DataHandler chan interface{}
	ToRoutine   chan interface{}

	Match *Match

	// shutdown synchronises shutdown event across routines
	ShutdownC chan struct{}
	Wg        *sync.WaitGroup

	// Orderbook is a local buffer of orderbooks
	Orderbook buffer.Orderbook

	// Trade is a notifier of occurring trades
	Trade trade.Trade

	// Fills is a notifier of occurring fills
	Fills fill.Fills

	// trafficAlert monitors if there is a halt in traffic throughput
	TrafficAlert chan struct{}
	// ReadMessageErrors will received all errors from ws.ReadMessage() and
	// verify if its a disconnection
	ReadMessageErrors chan error
	features          *protocol.Features

	// Standard stream connection
	Conn Connection
	// Authenticated stream connection
	AuthConn Connection

	// Latency reporter
	ExchangeLevelReporter Reporter

	// MaxSubScriptionsPerConnection defines the maximum number of
	// subscriptions per connection that is allowed by the exchange.
	MaxSubscriptionsPerConnection int
}

// WebsocketSetup defines variables for setting up a websocket connection
type WebsocketSetup struct {
	ExchangeConfig        *config.Exchange
	DefaultURL            string
	RunningURL            string
	RunningURLAuth        string
	Connector             func() error
	Subscriber            func([]subscription.Subscription) error
	Unsubscriber          func([]subscription.Subscription) error
	GenerateSubscriptions func() ([]subscription.Subscription, error)
	Features              *protocol.Features

	// Local orderbook buffer config values
	OrderbookBufferConfig buffer.Config

	TradeFeed bool

	// Fill data config values
	FillsFeed bool

	// MaxWebsocketSubscriptionsPerConnection defines the maximum number of
	// subscriptions per connection that is allowed by the exchange.
	MaxWebsocketSubscriptionsPerConnection int
}

// WebsocketConnection contains all the data needed to send a message to a WS
// connection
type WebsocketConnection struct {
	Verbose   bool
	connected int32

	// Gorilla websocket does not allow more than one goroutine to utilise
	// writes methods
	writeControl sync.Mutex

	RateLimit    int64
	ExchangeName string
	URL          string
	ProxyURL     string
	Wg           *sync.WaitGroup
	Connection   *websocket.Conn
	ShutdownC    chan struct{}

	Match             *Match
	ResponseMaxLimit  time.Duration
	Traffic           chan struct{}
	readMessageErrors chan error

	Reporter Reporter
}
