package okx

import (
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Okx is the overarching type across this package
type Okx struct {
	exchange.Base
}

const (
	okxAPIURL     = "https://www.okx.com/"
	okxAPIVersion = "v5"
	publicWsURL   = "wss://ws.okx.com:8443/ws/v5/public"
	privateWsURL  = "wss://ws.okx.com:8443/ws/v5/private"

	//trade endpoints
	epTradeSubURL          = "trade"
	epOrder                = "order"
	epBatchOrders          = "batch-orders"
	epCancelOrder          = "cancel-order"
	epCancelBatchOrders    = "cancel-batch-orders"
	epAmendOrder           = "amend-order"
	epAmendBatchOrders     = "amend-batch-orders"
	epClosePosition        = "close-position"
	epOrdersPending        = "orders-pending"
	epOrdersHistory        = "orders-history"
	epOrdersHistoryArchive = "orders-history-archive"
	epFills                = "fills"
	epFillsHistory         = "fills-history"
	epOrderAlgo            = "order-algo"
	epCancelAlgos          = "cancel-algos"
	epCancelAdvanceAlgos   = "cancel-advance-algos"
	epOrdersAlgoPending    = "orders-algo-pending"
	epOrdersAlgoHistory    = "orders-algo-history"
)

// Start implementing public and private exchange API funcs below
