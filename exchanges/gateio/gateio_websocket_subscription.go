package gateio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const subscriptionConcurrency = 8

var (
	// ErrSubscriptionPayloadCount indicates that native subscription payloads cannot be matched to their targets.
	ErrSubscriptionPayloadCount = errors.New("subscription payload count mismatch")
	// ErrSubscriptionRejected indicates that Gate rejected a subscription request.
	ErrSubscriptionRejected = errors.New("subscription rejected")
	// ErrSubscriptionAcknowledgement indicates that a reply does not confirm the requested subscription operation.
	ErrSubscriptionAcknowledgement = errors.New("invalid subscription acknowledgement")
)

// subscriptionOperations bounds request/acknowledgement work across initial
// subscriptions and concurrent recovery calls without queueing a goroutine per market.
// The first failure cancels its batch before releasing capacity, preventing later
// targets from amplifying an unhealthy stream while retaining the causal error.
func (e *Exchange) subscriptionOperations(ctx context.Context, event string, subs subscription.List, operation func(context.Context, *subscription.Subscription) error) error {
	e.subscriptionSlotsOnce.Do(func() { e.subscriptionSlots = make(chan struct{}, subscriptionConcurrency) })
	batchCtx, cancelBatch := context.WithCancel(ctx)
	defer cancelBatch()
	var workers sync.WaitGroup
	var firstFailure sync.Once
	var failure error
	var interrupted *subscription.Subscription
dispatch:
	for _, s := range subs {
		if batchCtx.Err() != nil {
			interrupted = s
			break
		}
		select {
		case e.subscriptionSlots <- struct{}{}:
		case <-batchCtx.Done():
			interrupted = s
			break dispatch
		}
		if batchCtx.Err() != nil {
			<-e.subscriptionSlots
			interrupted = s
			break
		}
		workers.Go(func() {
			defer func() { <-e.subscriptionSlots }()
			started := time.Now()
			err := batchCtx.Err()
			if err == nil {
				err = operation(batchCtx, s)
			}
			if err != nil {
				firstFailure.Do(func() {
					failure = &WebsocketSubscriptionError{Channel: s.Channel, Event: event, Asset: s.Asset, Duration: time.Since(started), Err: err}
					cancelBatch()
				})
			}
		})
	}
	workers.Wait()
	if failure != nil {
		return failure
	}
	if interrupted != nil {
		return &WebsocketSubscriptionError{Channel: interrupted.Channel, Event: event, Asset: interrupted.Asset, Err: ctx.Err()}
	}
	return nil
}

// sendSubscriptionRequest changes local subscription state only after Gate
// acknowledges the corresponding channel and event with a successful result.
func (e *Exchange) sendSubscriptionRequest(ctx context.Context, conn websocket.Connection, req *WsInput, sub *subscription.Subscription) error {
	result, err := conn.SendMessageReturnResponse(ctx, websocketRateLimitNotNeededEPL, req.ID, req)
	if err != nil {
		return err
	}
	var resp WsEventResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return ErrSubscriptionAcknowledgement
	}
	if resp.Error != nil {
		return fmt.Errorf("%w: code=%d", ErrSubscriptionRejected, resp.Error.Code)
	}
	if resp.Channel != req.Channel || resp.Event != req.Event || resp.Result == nil || resp.Result.Status != "success" {
		return ErrSubscriptionAcknowledgement
	}
	if req.Event == unsubscribeEvent {
		return e.Websocket.RemoveSubscriptions(conn, sub)
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, sub)
}
