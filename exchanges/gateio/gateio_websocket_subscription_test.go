package gateio

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var errSubscriptionFixture = errors.New("subscription fixture failure")

func TestSubscriptionOperations(t *testing.T) {
	t.Parallel()
	t.Run("shared capacity across concurrent batches", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			e := new(Exchange)
			var calls atomic.Int32
			release := make(chan struct{})
			var wg sync.WaitGroup
			for range 4 {
				wg.Go(func() {
					subs := make(subscription.List, 12)
					for i := range subs {
						subs[i] = &subscription.Subscription{Channel: spotTickerChannel, Asset: asset.Spot}
					}
					assert.NoError(t, e.subscriptionOperations(t.Context(), subscribeEvent, subs, func(context.Context, *subscription.Subscription) error {
						calls.Add(1)
						<-release
						return nil
					}), "all native operations should complete")
				})
			}
			synctest.Wait()
			assert.Equal(t, int32(subscriptionConcurrency), calls.Load(), "all calls should share one capacity limit")
			close(release)
			wg.Wait()
			assert.Equal(t, int32(48), calls.Load(), "each native operation should run once")
		})
	})
	for _, beforeDispatch := range []bool{false, true} {
		t.Run(map[bool]string{true: "already cancelled", false: "cancelled while queued"}[beforeDispatch], func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				e := new(Exchange)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if beforeDispatch {
					cancel()
				}
				subs := make(subscription.List, 32)
				for i := range subs {
					subs[i] = &subscription.Subscription{Channel: futuresOrderbookV2, Asset: asset.USDTMarginedFutures}
				}
				var calls atomic.Int32
				completed := make(chan error, 1)
				go func() {
					completed <- e.subscriptionOperations(ctx, subscribeEvent, subs, func(ctx context.Context, _ *subscription.Subscription) error {
						calls.Add(1)
						<-ctx.Done()
						return ctx.Err()
					})
				}()
				synctest.Wait()
				want := int32(subscriptionConcurrency)
				if beforeDispatch {
					want = 0
				}
				assert.Equal(t, want, calls.Load(), "only available uncancelled operations should start")
				cancel()
				err := <-completed
				assert.ErrorIs(t, err, context.Canceled, "queued cancellation should retain its classification")
				var contextual *WebsocketSubscriptionError
				require.ErrorAs(t, err, &contextual, "caller cancellation must retain safe operation context")
				assert.Equal(t, futuresOrderbookV2, contextual.Channel, "caller cancellation should identify the configured channel")
				assert.Equal(t, subscribeEvent, contextual.Event, "caller cancellation should identify the operation")
				assert.Equal(t, asset.USDTMarginedFutures, contextual.Asset, "caller cancellation should identify the asset")
				assert.Empty(t, e.subscriptionSlots, "cancellation should release every acquired slot")
			})
		})
	}
	for _, failure := range []struct {
		name string
		err  error
	}{{"acknowledgement timeout", websocket.ErrSignatureTimeout}, {"rejected subscription", ErrSubscriptionRejected}} {
		t.Run(failure.name+" aborts queued targets", func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				e := new(Exchange)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				subs := make(subscription.List, 32)
				for i := range subs {
					subs[i] = &subscription.Subscription{Channel: spotTickerChannel, Asset: asset.Spot}
				}
				subs[0].Channel = spotTradesChannel
				fail := make(chan struct{})
				finished := make(chan error, 1)
				var calls atomic.Int32
				go func() {
					finished <- e.subscriptionOperations(ctx, subscribeEvent, subs, func(ctx context.Context, sub *subscription.Subscription) error {
						calls.Add(1)
						if sub == subs[0] {
							<-fail
							return failure.err
						}
						<-ctx.Done()
						return ctx.Err()
					})
				}()
				synctest.Wait()
				assert.Equal(t, int32(subscriptionConcurrency), calls.Load(), "in-flight requests should fill only available slots")
				close(fail)
				synctest.Wait()
				assert.Equal(t, int32(subscriptionConcurrency), calls.Load(), "no queued target should start after the first failure")
				select {
				case err := <-finished:
					assert.ErrorIs(t, err, failure.err, "batch should retain the original causal failure")
					assert.NotErrorIs(t, err, context.Canceled, "derived cancellation should not replace the causal failure")
				default:
					assert.Fail(t, "batch should cancel and drain in-flight requests promptly")
					cancel()
					<-finished
				}
				assert.Empty(t, e.subscriptionSlots, "failed batches should release every shared slot")
			})
		})
	}
	t.Run("failure retains completed in-flight outcomes", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		var calls atomic.Int32
		completed := make(chan struct{})
		err := e.subscriptionOperations(t.Context(), subscribeEvent, subscription.List{{Channel: spotTickerChannel, Asset: asset.Spot}, {Channel: spotTradesChannel, Asset: asset.Spot}}, func(_ context.Context, s *subscription.Subscription) error {
			calls.Add(1)
			if s.Channel == spotTickerChannel {
				<-completed
				return errSubscriptionFixture
			}
			close(completed)
			return nil
		})
		assert.ErrorIs(t, err, errSubscriptionFixture, "operation failure should retain its classification")
		assert.ErrorContains(t, err, "subscribe spot.tickers spot", "failure should identify the safe operation context")
		assert.Equal(t, int32(2), calls.Load(), "completed in-flight operations should retain their outcomes")
	})
}

type subscriptionResponseConnection struct {
	FixtureConnection
	response []byte
	err      error
}

func (c *subscriptionResponseConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return c.response, c.err
}

func TestSendSubscriptionRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		event    string
		response string
		sendErr  error
		expected error
	}{
		{name: "subscribe success", event: subscribeEvent, response: `{"channel":"spot.tickers","event":"subscribe","result":{"status":"success"}}`},
		{name: "unsubscribe rejected", event: unsubscribeEvent, response: `{"error":{"code":2}}`, expected: ErrSubscriptionRejected},
		{name: "unsubscribe success", event: unsubscribeEvent, response: `{"channel":"spot.tickers","event":"unsubscribe","result":{"status":"success"}}`},
		{name: "send failure", event: subscribeEvent, sendErr: errSubscriptionFixture, expected: errSubscriptionFixture},
		{name: "malformed JSON", event: subscribeEvent, response: `{`, expected: ErrSubscriptionAcknowledgement},
		{name: "missing acknowledgement", event: subscribeEvent, response: `{}`, expected: ErrSubscriptionAcknowledgement},
		{name: "missing result", event: subscribeEvent, response: `{"channel":"spot.tickers","event":"subscribe"}`, expected: ErrSubscriptionAcknowledgement},
		{name: "wrong channel", event: subscribeEvent, response: `{"channel":"spot.trades","event":"subscribe","result":{"status":"success"}}`, expected: ErrSubscriptionAcknowledgement},
		{name: "wrong event", event: subscribeEvent, response: `{"channel":"spot.tickers","event":"unsubscribe","result":{"status":"success"}}`, expected: ErrSubscriptionAcknowledgement},
		{name: "unsuccessful result", event: subscribeEvent, response: `{"channel":"spot.tickers","event":"subscribe","result":{"status":"failed"}}`, expected: ErrSubscriptionAcknowledgement},
		{name: "server error", event: subscribeEvent, response: `{"error":{"code":2,"message":"sensitive server text"}}`, expected: ErrSubscriptionRejected},
		{name: "non-null zero code error", event: subscribeEvent, response: `{"error":{"code":0,"message":"sensitive server text"}}`, expected: ErrSubscriptionRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := new(Exchange)
			require.NoError(t, testexch.Setup(e), "exchange setup must succeed")
			base, err := e.Websocket.CreateTestConnection(asset.Spot)
			require.NoError(t, err, "connection must initialise")
			conn := &subscriptionResponseConnection{response: []byte(tc.response), err: tc.sendErr}
			conn.Connection = base
			require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn), "connection must register")
			sub := &subscription.Subscription{Channel: spotTickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
			if tc.event == unsubscribeEvent {
				require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(conn, sub), "unsubscribe target must register")
			}
			err = e.sendSubscriptionRequest(t.Context(), conn, &WsInput{ID: 123, Channel: spotTickerChannel, Event: tc.event}, sub)
			if tc.expected != nil {
				assert.ErrorIs(t, err, tc.expected, "failure should retain its classification")
				assert.NotContains(t, err.Error(), "sensitive server text", "error should omit arbitrary server messages")
				if tc.event == unsubscribeEvent {
					assert.Len(t, e.Websocket.GetSubscriptions(), 1, "rejected unsubscribe should retain its target")
				} else {
					assert.Empty(t, e.Websocket.GetSubscriptions(), "rejected subscribe should not record local success")
				}
				return
			}
			require.NoError(t, err, "valid acknowledgement must succeed")
			if tc.event == unsubscribeEvent {
				assert.Empty(t, e.Websocket.GetSubscriptions(), "acknowledged unsubscribe should remove the target")
			} else {
				assert.Len(t, e.Websocket.GetSubscriptions(), 1, "acknowledged subscribe should record the target")
			}
		})
	}
}

func TestHandleSubscription(t *testing.T) {
	t.Parallel()
	for _, count := range []int{0, 2} {
		t.Run(map[int]string{0: "too few payloads", 2: "too many payloads"}[count], func(t *testing.T) {
			t.Parallel()
			e := new(Exchange)
			err := e.handleSubscription(t.Context(), nil, subscribeEvent, subscription.List{{Channel: spotTickerChannel}}, func(context.Context, string, subscription.List) ([]WsInput, error) {
				return make([]WsInput, count), nil
			})
			assert.ErrorIs(t, err, ErrSubscriptionPayloadCount, "mismatched payloads should fail before sending")
		})
	}
	t.Run("payload generation fails", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		err := e.handleSubscription(t.Context(), nil, subscribeEvent, nil, func(context.Context, string, subscription.List) ([]WsInput, error) {
			return nil, errSubscriptionFixture
		})
		assert.ErrorIs(t, err, errSubscriptionFixture, "payload generation should retain its error")
	})
}

func TestWebsocketSubscriptionErrorError(t *testing.T) {
	t.Parallel()
	err := &WebsocketSubscriptionError{Channel: futuresOrderbookV2, Event: subscribeEvent, Asset: asset.USDTMarginedFutures, Err: errSubscriptionFixture}
	assert.Equal(t, "subscribe futures.obu usdtmarginedfutures: subscription request failed", err.Error(), "error should show safe configured context")
	assert.NotContains(t, err.Error(), errSubscriptionFixture.Error(), "error should omit arbitrary wrapped text")
}

func TestWebsocketSubscriptionErrorUnwrap(t *testing.T) {
	t.Parallel()
	err := &WebsocketSubscriptionError{Err: errSubscriptionFixture}
	assert.ErrorIs(t, err, errSubscriptionFixture, "wrapped errors should retain their classification")
	assert.Same(t, errSubscriptionFixture, err.Unwrap(), "unwrap should return the original error")
}

func TestWebsocketSubscriptionErrorWebsocketSubscriptionDetails(t *testing.T) {
	t.Parallel()
	err := &WebsocketSubscriptionError{Channel: futuresOrderbookV2, Event: subscribeEvent, Asset: asset.USDTMarginedFutures, Duration: time.Second}
	channel, event, a, duration := err.WebsocketSubscriptionDetails()
	assert.Equal(t, futuresOrderbookV2, channel, "details should expose the configured channel")
	assert.Equal(t, subscribeEvent, event, "details should expose the requested event")
	assert.Equal(t, asset.USDTMarginedFutures, a, "details should expose the requested asset")
	assert.Equal(t, time.Second, duration, "details should expose the operation duration")
}

func TestManageSubs(t *testing.T) {
	t.Parallel()
	for _, event := range []string{subscribeEvent, unsubscribeEvent} {
		t.Run(event, func(t *testing.T) {
			t.Parallel()
			e := new(Exchange)
			require.NoError(t, testexch.Setup(e), "exchange setup must succeed")
			base, err := e.Websocket.CreateTestConnection(asset.Spot)
			require.NoError(t, err, "connection must initialise")
			conn := &FixtureConnection{Connection: base}
			require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn), "connection must register")
			subs := subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}
			if event == unsubscribeEvent {
				expanded, err := subs.ExpandTemplates(e)
				require.NoError(t, err, "unsubscribe target must expand")
				require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(conn, expanded...), "unsubscribe target must register")
			}
			require.NoError(t, e.manageSubs(t.Context(), event, conn, subs), "native subscription operation must succeed")
			if event == unsubscribeEvent {
				assert.Empty(t, e.Websocket.GetSubscriptions(), "unsubscribe should remove the target")
			} else {
				assert.Len(t, e.Websocket.GetSubscriptions(), 1, "subscribe should record the target")
			}
		})
	}
}
