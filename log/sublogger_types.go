package log

// Global sub logger strings
const (
	Global           = "LOG"
	ConnectionMgr    = "CONNECTION"
	CommunicationMgr = "COMMS"
	ConfigMgr        = "CONFIG"
	DatabaseMgr      = "DATABASE"
	OrderMgr         = "ORDER"
	PortfolioMgr     = "PORTFOLIO"
	SyncMgr          = "SYNC"
	TimeMgr          = "TIMEKEEPER"
	WebsocketMgr     = "WEBSOCKET"
	EventMgr         = "EVENT"
	DispatchMgr      = "DISPATCH"
	GCTScriptMgr     = "GCTSCRIPT"
	RequestSys       = "REQUESTER"
	ExchangeSys      = "EXCHANGE"
	GRPCSys          = "GRPC"
	RESTSys          = "REST"
	Ticker           = "TICKER"
	OrderBook        = "ORDERBOOK"
)

var (
	subLoggers = make(map[string]*subLoggerDetails)
)
