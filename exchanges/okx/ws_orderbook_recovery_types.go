package okx

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

type orderbookRecoveryKey struct {
	connection websocket.Connection
	channel    string
	instrument string
}

type orderbookRecovery struct {
	done      <-chan struct{}
	cancel    context.CancelFunc
	requested bool
}
