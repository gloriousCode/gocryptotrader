package okx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// recoverOrderbook keeps invalid books unavailable while retrying a replacement snapshot at a bounded cadence.
// The reader's context owns the recovery lifetime, so disconnected generations cannot retain recovery workers.
func (e *Exchange) recoverOrderbook(ctx context.Context, conn websocket.Connection, channel string, pair currency.Pair) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	key := orderbookRecoveryKey{connection: conn, channel: channel, instrument: pair.String()}
	var subs subscription.List
	for _, sub := range conn.Subscriptions().List() {
		if channelName(sub) == channel && sub.Pairs.Contains(pair, true) {
			subs = append(subs, sub)
		}
	}
	if len(subs) == 0 {
		return fmt.Errorf("%w: %s %s", subscription.ErrNotFound, channel, pair)
	}
	e.orderbookRecoveryMu.Lock()
	if recovery := e.orderbookRecoveries[key]; recovery != nil {
		select {
		case <-recovery.done:
			recovery.requested = true
		default:
		}
		e.orderbookRecoveryMu.Unlock()
		return nil
	}
	if e.orderbookRecoveries == nil {
		e.orderbookRecoveries = make(map[orderbookRecoveryKey]*orderbookRecovery)
	}
	recoveryCtx, cancelRecovery := context.WithCancel(ctx)
	recovery := &orderbookRecovery{done: recoveryCtx.Done(), cancel: cancelRecovery}
	e.orderbookRecoveries[key] = recovery
	e.orderbookRecoveryMu.Unlock()
	go func() {
		defer func() {
			e.orderbookRecoveryMu.Lock()
			recovery.cancel()
			if e.orderbookRecoveries[key] == recovery {
				delete(e.orderbookRecoveries, key)
			}
			e.orderbookRecoveryMu.Unlock()
		}()
		timer := time.NewTimer(0)
		defer timer.Stop()
		unsubscribed := true
		for _, sub := range subs {
			if sub.State() != subscription.UnsubscribedState {
				unsubscribed = false
				break
			}
		}
		completed := recovery.done
		for {
			select {
			case <-ctx.Done():
				return
			case <-completed:
				if !unsubscribed {
					e.orderbookRecoveryMu.Lock()
					if !recovery.requested {
						delete(e.orderbookRecoveries, key)
						e.orderbookRecoveryMu.Unlock()
						return
					}
					// A new gap arrived while the previous recovery was finishing. Renew this worker atomically
					// so the new demand cannot disappear between snapshot completion and worker retirement.
					renewedCtx, renewedCancel := context.WithCancel(ctx)
					recoveryCtx = renewedCtx //nolint:fatcontext // Each renewal derives from the unchanged reader context, never the previous recovery.
					recovery.done, recovery.cancel, recovery.requested = renewedCtx.Done(), renewedCancel, false
					completed = recovery.done
					e.orderbookRecoveryMu.Unlock()
					timer.Reset(0)
					continue
				}
				// A snapshot may race the unsubscribe. Finish restoring the subscription before retiring this worker.
				completed = nil
			case <-timer.C:
			}
			if ctx.Err() != nil {
				return
			}
			var err error
			if !unsubscribed {
				attemptCtx, cancel := context.WithTimeout(recoveryCtx, time.Minute)
				err = e.Unsubscribe(attemptCtx, conn, subs)
				cancel()
				unsubscribed = err == nil
			}
			if unsubscribed {
				// Retain the reader lifetime here: cancelling recovery after a fresh snapshot must not leave an
				// already-unsubscribed feed disabled. Shutdown still cancels the parent and releases this request.
				attemptCtx, cancel := context.WithTimeout(ctx, time.Minute)
				err = e.Subscribe(attemptCtx, conn, subs)
				cancel()
				if err == nil {
					unsubscribed = false
					if recoveryCtx.Err() != nil {
						completed = recoveryCtx.Done()
						continue
					}
				}
			}
			if err != nil && ctx.Err() == nil && !errors.Is(err, context.Canceled) {
				log.Errorf(log.ExchangeSys, "Failed to recover %s %s orderbook: %v", channel, pair, err)
			}
			timer.Reset(time.Minute)
		}
	}()
	return nil
}

// completeOrderbookRecovery releases recovery state only after a replacement snapshot has loaded successfully.
func (e *Exchange) completeOrderbookRecovery(conn websocket.Connection, channel string, pair currency.Pair) {
	key := orderbookRecoveryKey{connection: conn, channel: channel, instrument: pair.String()}
	e.orderbookRecoveryMu.Lock()
	defer e.orderbookRecoveryMu.Unlock()
	if recovery := e.orderbookRecoveries[key]; recovery != nil {
		recovery.requested = false
		recovery.cancel()
	}
}
