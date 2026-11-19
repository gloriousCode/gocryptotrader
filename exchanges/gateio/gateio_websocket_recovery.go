package gateio

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// onWebsocketConnectionFault retires only depth owned by the failed connection
// before its replacement can publish, including downstream detached book copies.
func (e *Exchange) onWebsocketConnectionFault(conn websocket.Connection, subs subscription.List) error {
	if e.wsOBResubMgr != nil {
		e.wsOBResubMgr.m.Lock()
		for identity, recovery := range e.wsOBResubMgr.lookup {
			if recovery.connection == conn {
				recovery.stop()
				delete(e.wsOBResubMgr.lookup, identity)
			}
		}
		e.wsOBResubMgr.m.Unlock()
	}
	var invalidation websocket.OrderbookInvalidation
	seen := make(map[key.PairAsset]struct{})
	var errs error
	for _, sub := range subs {
		if sub == nil || !isSingleOrderbookChannel(channelName(sub)) {
			continue
		}
		for _, pair := range sub.Pairs {
			if pair.IsEmpty() {
				continue
			}
			identity := key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: sub.Asset}
			if _, ok := seen[identity]; ok {
				continue
			}
			seen[identity] = struct{}{}
			if err := e.wsOBUpdateMgr.InvalidateOrderbook(pair, sub.Asset); err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
				errs = common.AppendError(errs, err)
			}
			invalidation.Pairs = append(invalidation.Pairs, identity)
		}
	}
	if len(invalidation.Pairs) == 0 {
		return errs
	}
	invalidation.At = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, invalidation))
}
