package okx

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var errRecoverySend = errors.New("offline recovery request failed")

type recoveryConnection struct {
	subscriptionRecorderConnection
	failedEvent      string
	fail             atomic.Bool
	attempts         atomic.Int64
	blockUnsubscribe bool
	releaseSubscribe chan struct{}
}

func (c *recoveryConnection) SendJSONMessage(ctx context.Context, endpoint request.EndpointLimit, payload any) error {
	c.attempts.Add(1)
	if req, ok := payload.(WSSubscriptionInformationList); ok && req.Operation == operationSubscribe && c.releaseSubscribe != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.releaseSubscribe:
		}
	}
	if req, ok := payload.(WSSubscriptionInformationList); ok && req.Operation == operationUnsubscribe && c.blockUnsubscribe {
		<-ctx.Done()
		return ctx.Err()
	}
	if req, ok := payload.(WSSubscriptionInformationList); ok && req.Operation == c.failedEvent && c.fail.Load() {
		return errRecoverySend
	}
	return c.subscriptionRecorderConnection.SendJSONMessage(ctx, endpoint, payload)
}

func TestRecoverOrderbook(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		conn websocket.Connection
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "typed nil", conn: (*subscriptionRecorderConnection)(nil), err: common.ErrNilPointer},
		{name: "untracked", conn: &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}, err: subscription.ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := &Exchange{}
			assert.ErrorIs(t, e.recoverOrderbook(t.Context(), tc.conn, "books", mainPair), tc.err, "invalid recovery input should fail closed")
		})
	}
	for _, failedEvent := range []string{"", "shared spot margin", "retained unsubscribed target", operationUnsubscribe, operationSubscribe, "missing initial snapshot", "snapshot during queued unsubscribe", "snapshot during queued subscribe", "gap after snapshot during queued subscribe", "cancelled generation"} {
		t.Run("failed_"+failedEvent, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				tracked := new(Exchange)
				require.NoError(t, testexch.Setup(tracked), "exchange setup must succeed")
				tracked.Name = t.Name()
				conn := &recoveryConnection{subscriptions: subscription.NewStore(), failedEvent: failedEvent, blockUnsubscribe: failedEvent == "snapshot during queued unsubscribe"}
				if failedEvent == "snapshot during queued subscribe" || failedEvent == "gap after snapshot during queued subscribe" {
					conn.releaseSubscribe = make(chan struct{})
				}
				conn.fail.Store(true)
				require.NoError(t, tracked.Websocket.TrackTestConnection(nil, conn), "connection must register")
				sub := &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
				require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(conn, sub), "subscription must register")
				require.NoError(t, conn.Subscriptions().Add(sub), "connection subscription must register")
				if failedEvent == "retained unsubscribed target" {
					require.NoError(t, tracked.Websocket.RemoveSubscriptions(conn, sub), "fixture must leave only the retained connection target")
				}
				targetCount := 1
				if failedEvent == "shared spot margin" {
					margin := &subscription.Subscription{Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, Channel: subscription.OrderbookChannel, QualifiedChannel: sub.QualifiedChannel}
					require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(conn, margin), "margin subscription must register")
					require.NoError(t, conn.Subscriptions().Add(margin), "margin connection target must register")
					targetCount++
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				snapshot := []byte(`{"arg":{"channel":"books","instId":"BTC-USDT","instType":"SPOT"},"action":"snapshot","data":[{"asks":[["101","1","0","1"]],"bids":[["100","1","0","1"]],"ts":"1659792392540","prevSeqId":-1,"seqId":10}]}`)
				if failedEvent != "missing initial snapshot" {
					require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, snapshot), "snapshot must load")
				}
				gap := []byte(`{"arg":{"channel":"books","instId":"BTC-USDT","instType":"SPOT"},"action":"update","data":[{"asks":[],"bids":[],"ts":"1659792392640","prevSeqId":12,"seqId":13}]}`)
				require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, gap), "sequence gap must start recovery")
				synctest.Wait()
				attempts := conn.attempts.Load()
				require.Positive(t, attempts, "recovery must attempt outbound requests")
				for range 20 {
					require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, gap), "pending updates must remain suppressed")
				}
				synctest.Wait()
				assert.Equal(t, attempts, conn.attempts.Load(), "pending updates should not cause a retry storm")
				if failedEvent == "snapshot during queued unsubscribe" || failedEvent == "snapshot during queued subscribe" || failedEvent == "gap after snapshot during queued subscribe" || failedEvent == "cancelled generation" {
					if failedEvent == "cancelled generation" {
						cancel()
					} else {
						require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, snapshot), "snapshot must finish recovery")
						if failedEvent == "snapshot during queued subscribe" || failedEvent == "gap after snapshot during queued subscribe" {
							if failedEvent == "gap after snapshot during queued subscribe" {
								require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, gap), "new gap must request recovery before old worker retires")
							}
							close(conn.releaseSubscribe)
						}
					}
					synctest.Wait()
					if failedEvent == "gap after snapshot during queued subscribe" {
						assert.Greater(t, conn.attempts.Load(), attempts, "fresh gap should renew recovery without waiting for another update")
						require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, snapshot), "replacement snapshot must finish renewed recovery")
						synctest.Wait()
						attempts = conn.attempts.Load()
					}
					tracked.orderbookRecoveryMu.Lock()
					assert.Empty(t, tracked.orderbookRecoveries, "finished generation should release recovery state")
					tracked.orderbookRecoveryMu.Unlock()
					synctest.Sleep(2 * time.Minute)
					assert.Equal(t, attempts, conn.attempts.Load(), "finished recovery should not retry")
					switch failedEvent {
					case "snapshot during queued unsubscribe":
						assert.Empty(t, conn.Requests(), "cancelled unsubscribe should never be sent")
					case "snapshot during queued subscribe", "gap after snapshot during queued subscribe":
						reqs := conn.Requests()
						require.GreaterOrEqual(t, len(reqs), 2, "successful unsubscribe must be followed by subscribe")
						assert.Equal(t, operationSubscribe, reqs[1].Operation, "recovery should restore its subscription before retiring")
						assert.NotNil(t, tracked.Websocket.GetSubscription(sub), "restored subscription should remain tracked")
					}
					return
				}
				conn.fail.Store(false)
				synctest.Sleep(time.Minute + time.Millisecond)
				require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, gap), "expired recovery must remain retryable")
				synctest.Wait()
				assert.Greater(t, conn.attempts.Load(), attempts, "recovery should retry after missing snapshot or failed request")
				assert.NotNil(t, tracked.Websocket.GetSubscription(sub), "retry should restore manager tracking")
				assert.Len(t, conn.Subscriptions().List(), targetCount, "retry should retain connection targets")
				assert.Len(t, tracked.Websocket.GetSubscriptions(), targetCount, "retry should restore all logical manager subscriptions")
				for _, req := range conn.Requests() {
					assert.Len(t, req.Arguments, 1, "each recovery phase should actually send its instrument")
				}
				require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, snapshot), "fresh snapshot must restore the book")
				update := []byte(`{"arg":{"channel":"books","instId":"BTC-USDT","instType":"SPOT"},"action":"update","data":[{"asks":[],"bids":[["100","2","0","1"]],"ts":"1659792392740","prevSeqId":10,"seqId":11}]}`)
				require.NoError(t, tracked.wsProcessOrderBooks(ctx, conn, update), "fresh incremental update must apply")
				book, err := tracked.Websocket.Orderbook.GetOrderbook(mainPair, asset.Spot)
				require.NoError(t, err, "recovered book must be readable")
				assert.Equal(t, int64(11), book.LastUpdateID, "recovered book should advance")
				assert.Equal(t, float64(2), book.Bids[0].Amount, "recovered book should update its amount")
				cancel()
				synctest.Wait()
				tracked.orderbookRecoveryMu.Lock()
				assert.Empty(t, tracked.orderbookRecoveries, "completed recovery should release its state")
				tracked.orderbookRecoveryMu.Unlock()
			})
		})
	}
}

func TestCompleteOrderbookRecovery(t *testing.T) {
	t.Parallel()
	for _, matching := range []bool{false, true} {
		name := "unmatched"
		if matching {
			name = "matching"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
			key := orderbookRecoveryKey{connection: conn, channel: "books", instrument: mainPair.String()}
			e := &Exchange{orderbookRecoveries: map[orderbookRecoveryKey]*orderbookRecovery{key: {done: ctx.Done(), cancel: cancel, requested: true}}}
			channel := "other"
			if matching {
				channel = "books"
			}
			e.completeOrderbookRecovery(conn, channel, mainPair)
			e.completeOrderbookRecovery(conn, channel, mainPair)
			if matching {
				assert.ErrorIs(t, ctx.Err(), context.Canceled, "matching snapshot should cancel recovery")
				assert.False(t, e.orderbookRecoveries[key].requested, "matching snapshot should clear renewed demand")
			} else {
				assert.NoError(t, ctx.Err(), "another channel should remain pending")
				assert.True(t, e.orderbookRecoveries[key].requested, "another channel should retain renewed demand")
			}
		})
	}
}
