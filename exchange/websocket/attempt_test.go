package websocket

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestSetConnectionAttemptObserver(t *testing.T) {
	t.Parallel()
	manager := NewManager()
	calls := 0
	manager.SetConnectionAttemptObserver(func(ConnectionAttempt) { calls++ })
	manager.completeConnectionAttempt(&ConnectionAttempt{}, nil)
	require.Equal(t, 1, calls, "installed observer must receive a completion")
	manager.SetConnectionAttemptObserver(nil)
	manager.completeConnectionAttempt(&ConnectionAttempt{}, nil)
	assert.Equal(t, 1, calls, "cleared observer should receive no completion")
}

func TestNewConnectionAttempt(t *testing.T) {
	t.Parallel()
	manager := NewManager()
	manager.exchangeName = "example"
	first := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, Levels: 20}
	second := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
	original := manager.newConnectionAttempt(subscription.List{first, second, nil}, false)
	require.Len(t, original.SelectionID, 64, "public selection must produce a bounded opaque identity")
	assert.Equal(t, asset.Spot, original.Asset, "single product should remain explicit")
	assert.Equal(t, "example", original.Exchange, "exchange should remain explicit")
	assert.False(t, original.StartedAt.IsZero(), "actual attempt should retain start time")
	for _, tc := range []struct {
		name          string
		subs          subscription.List
		authenticated bool
		same          bool
		product       asset.Item
	}{
		{name: "order invariant", subs: subscription.List{second, first}, same: true, product: asset.Spot},
		{name: "different product", subs: subscription.List{first, {Channel: subscription.TickerChannel, Asset: asset.PerpetualSwap}}, product: asset.Empty},
		{name: "authentication scope", subs: subscription.List{first, second}, authenticated: true, product: asset.Spot},
		{name: "empty subscription scope", product: asset.Empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := manager.newConnectionAttempt(tc.subs, tc.authenticated)
			assert.Equal(t, tc.same, got.SelectionID == original.SelectionID, "identity should follow public request selection")
			assert.Equal(t, tc.product, got.Asset, "mixed or unknown product should remain unspecified")
		})
	}
}

func TestCompleteConnectionAttempt(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		err     error
		outcome string
	}{
		{name: "success", outcome: "succeeded"},
		{name: "cancelled", err: context.Canceled, outcome: "cancelled"},
		{name: "deadline", err: context.DeadlineExceeded, outcome: "timeout"},
		{name: "network timeout", err: &net.DNSError{IsTimeout: true}, outcome: "timeout"},
		{name: "signature timeout", err: ErrSignatureTimeout, outcome: "timeout"},
		{name: "subscription", err: ErrSubscriptionFailure, outcome: "subscription_failed"},
		{name: "missing acknowledgement", err: ErrSubscriptionsNotAdded, outcome: "subscription_failed"},
		{name: "unsupported subscription", err: subscription.ErrNotSupported, outcome: "unsupported"},
		{name: "authentication", err: errFailedToAuthenticate, outcome: "authentication_failed"},
		{name: "private error omitted", err: fmt.Errorf("wss://private?token=secret: %w", errDastardlyReason), outcome: "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manager := NewManager()
			var got ConnectionAttempt
			manager.SetConnectionAttemptObserver(func(attempt ConnectionAttempt) { got = attempt })
			start := time.Now().UTC()
			manager.completeConnectionAttempt(&ConnectionAttempt{Exchange: "example", StartedAt: start}, tc.err)
			assert.Equal(t, tc.outcome, got.Outcome, "completion should retain fixed outcome")
			assert.False(t, got.CompletedAt.Before(start), "completion should follow start")
			assert.NotContains(t, fmt.Sprintf("%+v", got), "secret", "callback should omit raw errors and URLs")
		})
	}
}
