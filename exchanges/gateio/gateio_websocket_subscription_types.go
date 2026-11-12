package gateio

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// WebsocketSubscriptionError exposes safe subscription diagnostics separately
// from the wrapped error, which may contain arbitrary transport/server text.
type WebsocketSubscriptionError struct {
	Channel  string
	Event    string
	Asset    asset.Item
	Duration time.Duration
	Err      error
}

// Error deliberately omits the underlying error text from diagnostic logs.
func (e *WebsocketSubscriptionError) Error() string {
	return fmt.Sprintf("%s %s %s: subscription request failed", e.Event, e.Channel, e.Asset)
}

// Unwrap preserves transport, cancellation and acknowledgement classifications.
func (e *WebsocketSubscriptionError) Unwrap() error {
	return e.Err
}

// WebsocketSubscriptionDetails reports configured context without request IDs or payloads.
func (e *WebsocketSubscriptionError) WebsocketSubscriptionDetails() (channel, event string, marketAsset asset.Item, duration time.Duration) {
	return e.Channel, e.Event, e.Asset, e.Duration
}
