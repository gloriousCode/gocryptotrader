package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// IsEvent returns whether the event is an event
func (b *Base) IsEvent() bool {
	return true
}

// GetTime returns the time
func (b *Base) GetTime() time.Time {
	return b.Time
}

// Pair returns the currency pair
func (b *Base) Pair() currency.Pair {
	return b.CurrencyPair
}

// GetExchange returns the exchange
func (b *Base) GetExchange() string {
	return b.Exchange
}

// GetAssetType returns the asset type
func (b *Base) GetAssetType() asset.Item {
	return b.AssetType
}

// GetInterval returns the interval
func (b *Base) GetInterval() kline.Interval {
	return b.Interval
}

// AppendReason adds reasoning for a decision being made
func (b *Base) AppendReason(y string) {
	if b.Why == "" {
		b.Why = y
	} else {
		b.Why = y + ". " + b.Why
	}
}

// GetReason returns the why
func (b *Base) GetReason() string {
	return b.Why
}
