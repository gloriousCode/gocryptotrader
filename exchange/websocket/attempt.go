package websocket

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// SetConnectionAttemptObserver replaces an optional observation sink atomically.
// A nil sink disables observation. It must not block or call manager methods.
func (m *Manager) SetConnectionAttemptObserver(observer ConnectionAttemptObserver) {
	if observer == nil {
		m.attemptObserver.Store(nil)
		return
	}
	m.attemptObserver.Store(&observer)
}

// newConnectionAttempt freezes public request identity before subscription state can change.
func (m *Manager) newConnectionAttempt(subs subscription.List, authenticated bool) ConnectionAttempt {
	result := ConnectionAttempt{Exchange: m.exchangeName, StartedAt: time.Now().UTC()}
	identities := make([]string, 0, len(subs)+1)
	identities = append(identities, fmt.Sprintf("authenticated:%t", authenticated))
	var products []asset.Item
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		pairs := make([]string, 0, len(sub.Pairs))
		for _, pair := range sub.Pairs {
			pairs = append(pairs, pair.Base.String()+"/"+pair.Quote.String())
		}
		slices.Sort(pairs)
		identities = append(identities, fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d\x00%t", sub.Channel, sub.Asset, strings.Join(pairs, "\x00"), sub.Interval, sub.Levels, sub.Authenticated))
		if !slices.Contains(products, sub.Asset) {
			products = append(products, sub.Asset)
		}
	}
	if len(products) == 1 {
		result.Asset = products[0]
	}
	slices.Sort(identities)
	result.SelectionID = fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(identities, "\n"))))
	return result
}

// completeConnectionAttempt emits fixed outcome categories, never the underlying error text.
func (m *Manager) completeConnectionAttempt(attempt *ConnectionAttempt, err error) {
	observer := m.attemptObserver.Load()
	if observer == nil {
		return
	}
	attempt.CompletedAt = time.Now().UTC()
	var networkError net.Error
	switch {
	case err == nil:
		attempt.Outcome = "succeeded"
	case errors.Is(err, context.Canceled):
		attempt.Outcome = "cancelled"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, ErrSignatureTimeout), errors.As(err, &networkError) && networkError.Timeout():
		attempt.Outcome = "timeout"
	case errors.Is(err, subscription.ErrNotSupported):
		attempt.Outcome = "unsupported"
	case errors.Is(err, ErrSubscriptionFailure), errors.Is(err, ErrSubscriptionsNotAdded):
		attempt.Outcome = "subscription_failed"
	case errors.Is(err, errFailedToAuthenticate):
		attempt.Outcome = "authentication_failed"
	default:
		attempt.Outcome = "failed"
	}
	(*observer)(*attempt)
}
