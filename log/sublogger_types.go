package log

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

// Global vars related to the logger package
var (
	subLoggers map[string]*SubLogger
)
