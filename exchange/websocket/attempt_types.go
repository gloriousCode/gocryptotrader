package websocket

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ConnectionAttempt is one completed connect-and-subscribe attempt. It contains
// no URL, raw error, credentials or account payload. SelectionID hashes public
// channel, product, pair, interval and depth selection, not private account identity.
type ConnectionAttempt struct {
	Exchange    string
	Asset       asset.Item
	SelectionID string
	StartedAt   time.Time
	CompletedAt time.Time
	Outcome     string
}

// ConnectionAttemptObserver must return promptly and must not call manager methods.
// Initial setup may hold the manager lifecycle lock; recovery invokes it after installation.
type ConnectionAttemptObserver func(ConnectionAttempt)
