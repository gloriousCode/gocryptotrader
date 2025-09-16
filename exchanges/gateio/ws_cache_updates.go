package gateio

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// enqueueUpdate appends a pending update (non-concurrently safe, caller holds lock)
func (cache *updateCache) enqueueUpdate(u *orderbook.Update, firstID int64) {
	cache.updates = append(cache.updates, pendingUpdate{update: u, firstUpdateID: firstID})
}

// notifyLatest tries to send the latest update ID to any waiter (non-blocking)
func (cache *updateCache) notifyLatest(id int64) {
	select {
	case cache.ch <- id:
	default:
	}
}

// SyncOrderbook fetches and synchronises an orderbook snapshot to the limit size so that pending updates can be
// applied to the orderbook.
func (cache *updateCache) SyncOrderbook(ctx context.Context, e *Exchange, b *orderbook.Book, delay, deadline time.Duration) error {
	// REST requests can be behind websocket updates by a large margin, so we wait here to allow the cache to fill with
	// updates before we fetch the orderbook snapshot.
	select {
	case <-ctx.Done():
		cache.clearWithLock()
		return ctx.Err()
	case <-time.After(delay):
	}
	// Setting deadline to error out instead of waiting for rate limiter delay which excessively builds a backlog of
	// pending updates.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(deadline))
	defer cancel()

	if err := cache.waitForUpdate(ctx, cache.lastUpdateID+1); err != nil {
		return err
	}
	cache.mtx.Lock() // Lock here to prevent ws handle data interference with REST request above.
	defer func() {
		cache.clearNoLock()
		cache.mtx.Unlock()
	}()

	if b.Asset != asset.Spot { // Regarding Spot, Margin and Cross Margin, the asset is hard coded to `spot in the calling function
		if err := e.Websocket.Orderbook.LoadSnapshot(b); err != nil {
			cache.state = desynchronised
			return err
		}
	} else {
		// Spot, Margin, and Cross Margin books are all classified as spot
		for i := range standardMarginAssetTypes {
			if enabled, _ := e.IsPairEnabled(b.Pair, standardMarginAssetTypes[i]); !enabled {
				continue
			}
			b.Asset = standardMarginAssetTypes[i]
			if err := e.Websocket.Orderbook.LoadSnapshot(b); err != nil {
				cache.state = desynchronised
				return err
			}
		}
	}
	if err := cache.applyPendingUpdates(e); err != nil {
		return err
	}
	// If pending updates applied successfully we are now synchronised
	cache.state = synchronised
	return nil
}

// TODO: When subscription config is added for all assets update limits to use sub.Levels

// waitForUpdate waits for an update with an ID >= nextUpdateID
func (cache *updateCache) waitForUpdate(ctx context.Context, nextUpdateID int64) error {
	var updateListLastUpdateID int64
	cache.mtx.Lock()
	updateListLastUpdateID = cache.updates[len(cache.updates)-1].update.UpdateID
	cache.mtx.Unlock()

	if updateListLastUpdateID >= nextUpdateID {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case recentPendingUpdateID := <-cache.ch:
			if recentPendingUpdateID >= nextUpdateID {
				return nil
			}
		}
	}
}

// applyPendingUpdates applies all pending updates to the orderbook
// Does not lock cache
func (cache *updateCache) applyPendingUpdates(e *Exchange) error {
	updated := false
	for _, data := range cache.updates {
		bookLastUpdateID, err := e.Websocket.Orderbook.LastUpdateID(data.update.Pair, data.update.Asset)
		if err != nil {
			return err
		}

		nextUpdateID := bookLastUpdateID + 1 // From docs: `baseId+1`

		// From docs: Dump all notifications which satisfy `u` < `baseId+1`
		if data.update.UpdateID < nextUpdateID {
			continue
		}

		pendingFirstUpdateID := data.firstUpdateID // `U`
		// From docs: `baseID+1` < first notification `U` current base order book falls behind notifications
		if nextUpdateID < pendingFirstUpdateID {
			return errOrderbookSnapshotOutdated
		}

		if err := applyOrderbookUpdate(e, data.update); err != nil {
			return err
		}
		updated = true
	}
	if !updated {
		return errPendingUpdatesNotApplied
	}
	// Successful application of all pending updates should mark cache as synchronised (unless overridden later)
	cache.state = synchronised
	return nil
}

func (cache *updateCache) clearWithLock() {
	cache.mtx.Lock()
	defer cache.mtx.Unlock()
	cache.clearNoLock()
}

// clearNoLock clears the cache without locking. Caller must hold lock.
func (cache *updateCache) clearNoLock() {
	cache.updates = nil
}
