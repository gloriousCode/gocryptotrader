package gateio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type wsOBResubManager struct {
	lookup map[key.PairAsset]*wsOBRecovery
	m      sync.RWMutex
}

type wsOBRecovery struct {
	connection websocket.Connection
	parentDone <-chan struct{}
	done       <-chan struct{}
	cancel     context.CancelFunc
	stop       context.CancelFunc
	requested  bool
}

func newWSOBResubManager() *wsOBResubManager {
	return &wsOBResubManager{lookup: make(map[key.PairAsset]*wsOBRecovery)}
}

// IsResubscribing suppresses incremental messages until a valid replacement
// snapshot arrives. Recovery retries independently, even when the feed is silent.
func (m *wsOBResubManager) IsResubscribing(pair currency.Pair, a asset.Item) bool {
	m.m.RLock()
	defer m.m.RUnlock()
	recovery := m.lookup[key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}]
	if recovery == nil {
		return false
	}
	select {
	case <-recovery.done:
		return recovery.requested
	default:
		return true
	}
}

// Resubscribe retains the original target across failed unsubscribe/subscribe
// phases; removing a failed target from the manager store must not lose recovery.
func (m *wsOBResubManager) Resubscribe(ctx context.Context, e *Exchange, conn websocket.Connection, qualifiedChannel string, pair currency.Pair, a asset.Item) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := e.Websocket.Orderbook.InvalidateOrderbook(pair, a); err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
		return err
	}
	identity := key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}
	m.m.Lock()
	if previous := m.lookup[identity]; previous != nil {
		if previous.connection == conn {
			select {
			case <-previous.parentDone:
			default:
				select {
				case <-previous.done:
					previous.requested = true
				default:
				}
				m.m.Unlock()
				return nil
			}
		}
		previous.stop()
		delete(m.lookup, identity)
	}
	m.m.Unlock()
	subKey := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: qualifiedChannel, Asset: a}}
	sub := e.Websocket.GetSubscription(subKey)
	if sub == nil && conn.Subscriptions() != nil {
		sub = conn.Subscriptions().Get(subKey)
	}
	if sub == nil {
		return fmt.Errorf("%w: %q", subscription.ErrNotFound, qualifiedChannel)
	}
	subscribe, unsubscribe := e.Subscribe, e.Unsubscribe
	switch a {
	case asset.Spot, asset.Margin:
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		subscribe, unsubscribe = e.FuturesSubscribe, e.FuturesUnsubscribe
	case asset.DeliveryFutures:
		subscribe, unsubscribe = e.DeliveryFuturesSubscribe, e.DeliveryFuturesUnsubscribe
	case asset.Options:
		subscribe, unsubscribe = e.OptionsSubscribe, e.OptionsUnsubscribe
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	workerCtx, stop := context.WithCancel(ctx)
	recoveryCtx, cancel := context.WithCancel(workerCtx)
	recovery := &wsOBRecovery{connection: conn, parentDone: workerCtx.Done(), done: recoveryCtx.Done(), cancel: cancel, stop: stop}
	m.m.Lock()
	if previous := m.lookup[identity]; previous != nil {
		select {
		case <-previous.parentDone:
		default:
			select {
			case <-previous.done:
				previous.requested = true
			default:
			}
			m.m.Unlock()
			cancel()
			stop()
			return nil
		}
	}
	m.lookup[identity] = recovery
	m.m.Unlock()
	go func() {
		defer stop()
		defer func() {
			cancel()
			m.m.Lock()
			if m.lookup[identity] == recovery {
				delete(m.lookup, identity)
			}
			m.m.Unlock()
		}()
		retry := max(time.Minute, 2*e.WebsocketResponseMaxLimit)
		timer := time.NewTimer(0)
		defer timer.Stop()
		unsubscribed := sub.State() == subscription.UnsubscribedState
		completed := recoveryCtx.Done()
		targets := subscription.List{sub}
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-completed:
				if !unsubscribed {
					m.m.Lock()
					if !recovery.requested {
						if m.lookup[identity] == recovery {
							delete(m.lookup, identity)
						}
						m.m.Unlock()
						return
					}
					renewedCtx, renewedCancel := context.WithCancel(workerCtx)
					recoveryCtx, cancel = renewedCtx, renewedCancel //nolint:fatcontext // Each replacement derives from workerCtx, never from the previous recovery.
					recovery.done, recovery.cancel, recovery.requested = recoveryCtx.Done(), cancel, false
					completed = recoveryCtx.Done()
					m.m.Unlock()
					timer.Reset(0)
					continue
				}
				// If the unsubscribe committed before the snapshot arrived, restore the
				// live subscription before finishing. Parent cancellation still wins.
				completed = nil
			case <-timer.C:
			}
			if workerCtx.Err() != nil {
				return
			}
			var err error
			if !unsubscribed {
				attempt, cancelAttempt := context.WithTimeout(recoveryCtx, retry)
				err = unsubscribe(attempt, conn, targets)
				cancelAttempt()
				unsubscribed = err == nil
			}
			if unsubscribed {
				attempt, cancelAttempt := context.WithTimeout(workerCtx, retry)
				err = subscribe(attempt, conn, targets)
				cancelAttempt()
				if err == nil {
					unsubscribed = false
					if recoveryCtx.Err() != nil {
						completed = recoveryCtx.Done()
						continue
					}
				}
			}
			if err != nil && workerCtx.Err() == nil && !errors.Is(err, context.Canceled) {
				log.Errorf(log.ExchangeSys, "Failed to recover channel %q: %v", qualifiedChannel, err)
			}
			timer.Reset(retry)
		}
	}()
	return nil
}

// CompletedResubscribe cancels queued unsubscribe attempts without discarding a
// worker which still needs to restore an already-unsubscribed feed.
func (m *wsOBResubManager) CompletedResubscribe(pair currency.Pair, a asset.Item) {
	m.m.Lock()
	defer m.m.Unlock()
	if recovery := m.lookup[key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}]; recovery != nil {
		recovery.requested = false
		recovery.cancel()
	}
}

type qualifiedChannelKey struct {
	*subscription.Subscription
}

func (k qualifiedChannelKey) Match(eachKey subscription.MatchableKey) bool {
	s := eachKey.GetSubscription()
	return k.Subscription.QualifiedChannel == s.QualifiedChannel && k.Subscription.Asset == s.Asset
}

func (k qualifiedChannelKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}
