package gateio

import "github.com/thrasher-corp/gocryptotrader/exchanges/asset"

// PrivateWebsocketEvent retains an authenticated Gate.io push with fields that
// are omitted from the exchange-neutral order, fill, account and position data.
// Payload is an owned copy of the complete exchange message.
type PrivateWebsocketEvent struct {
	Asset   asset.Item
	Payload []byte
}
