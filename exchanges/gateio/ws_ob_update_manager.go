package gateio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errOrderbookSnapshotOutdated      = errors.New("orderbook snapshot is outdated")
	errPendingUpdatesNotApplied       = errors.New("pending updates not applied")
	errInvalidOrderbookUpdateInterval = errors.New("invalid orderbook update interval")
	errApplyingOrderbookUpdate        = errors.New("error applying orderbook update")

	defaultWSOrderbookUpdateDeadline  = time.Minute * 2
	defaultWsOrderbookUpdateTimeDelay = time.Second * 2
	spotOrderbookUpdateKey            = subscription.MustChannelKey(subscription.OrderbookChannel)
)

type wsOBUpdateManager struct {
	lookup   map[key.PairAsset]*updateCache
	deadline time.Duration
	delay    time.Duration
	mtx      sync.RWMutex
	obFunc   func(ctx context.Context, p currency.Pair, a asset.Item, limit uint64) (*orderbook.Book, error)
}

type updateCache struct {
	updates []pendingUpdate
	ch      chan int64 // receives newest pending update IDs while syncing snapshot
}

// state determines how orderbook updates are handled
type state uint8

const (
	initialised state = iota
	queuingUpdates
	synchronised
	desynchronised
)

type pendingUpdate struct {
	update        *orderbook.Update
	firstUpdateID int64
}

func newWsOBUpdateManager(delay, deadline time.Duration) *wsOBUpdateManager {
	return &wsOBUpdateManager{lookup: make(map[key.PairAsset]*updateCache), deadline: deadline, delay: delay}
}

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

func (m *wsOBUpdateManager) ProcessOrderbookUpdate(ctx context.Context, e *Exchange, firstUpdateID int64, update *orderbook.Update) error {
	cache, err := m.LoadCache(update.Pair, update.Asset)
	if err != nil {
		return err
	}
	cache.mtx.Lock()
	defer cache.mtx.Unlock()
	switch cache.state {
	case synchronised: // prioritised hot path
		return m.synchroniseOrderbook(ctx, e, update, firstUpdateID, cache)
	case initialised:
		return m.initialiseOrderbook(ctx, e, firstUpdateID, update, cache)
	case queuingUpdates:
		return m.queueOrderbookUpdate(cache, update, firstUpdateID)
	case desynchronised:
		return m.desynchroniseOrderbook(ctx, cache, e, update, firstUpdateID)
	default:
		return fmt.Errorf("unknown orderbook cache state %d for %v %v", cache.state, update.Pair, update.Asset)
	}
}

func (m *wsOBUpdateManager) queueOrderbookUpdate(cache *updateCache, update *orderbook.Update, firstUpdateID int64) error {
	cache.enqueueUpdate(update, firstUpdateID)
	cache.notifyLatest(update.UpdateID)
	return nil
}

func (m *wsOBUpdateManager) desynchroniseOrderbook(ctx context.Context, cache *updateCache, e *Exchange, update *orderbook.Update, firstUpdateID int64) error {
	cache.clearNoLock()
	if err := e.Websocket.Orderbook.InvalidateOrderbook(update.Pair, update.Asset); err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
		return err
	}
	cache.state = initialised
	return m.initialiseOrderbook(ctx, e, firstUpdateID, update, cache)
}

func (m *wsOBUpdateManager) initialiseOrderbook(ctx context.Context, e *Exchange, firstUpdateID int64, update *orderbook.Update, cache *updateCache) error {
	go func() {
		if err := cache.SyncOrderbook(ctx, e, update.Pair, update.Asset, m.delay, m.deadline); err != nil {
			log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", e.Name, update.Pair, update.Asset, err)
			return
		}
	}()
	cache.state = queuingUpdates
	cache.enqueueUpdate(update, firstUpdateID)
	return nil
}

func (m *wsOBUpdateManager) synchroniseOrderbook(ctx context.Context, e *Exchange, update *orderbook.Update, firstUpdateID int64, cache *updateCache) error {
	lastUpdateID, err := e.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset)
	if err != nil {
		return err
	}
	if lastUpdateID+1 != firstUpdateID {
		cache.state = desynchronised
		return m.desynchroniseOrderbook(ctx, cache, e, update, firstUpdateID)
	}
	if err := applyOrderbookUpdate(e, update); err != nil {
		cache.state = desynchronised
		return m.desynchroniseOrderbook(ctx, cache, e, update, firstUpdateID)
	}
	// Successful application while already in synchronised path keeps us synchronised
	cache.state = synchronised
	return nil
}

// LoadCache loads the cache for the given pair and asset. If the cache does not exist, it creates a new one.
func (m *wsOBUpdateManager) LoadCache(p currency.Pair, a asset.Item) (*updateCache, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, fmt.Errorf("%w: %q", asset.ErrInvalidAsset, a)
	}
	m.mtx.RLock()
	cache, ok := m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	m.mtx.RUnlock()
	if !ok {
		cache = &updateCache{ch: make(chan int64, 1)} // small buffer to retain latest ID without blocking
		m.mtx.Lock()
		m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}] = cache
		m.mtx.Unlock()
	}
	return cache, nil
}

// SyncOrderbook fetches and synchronises an orderbook snapshot to the limit size so that pending updates can be
// applied to the orderbook.
func (cache *updateCache) SyncOrderbook(ctx context.Context, e *Exchange, pair currency.Pair, a asset.Item, delay, deadline time.Duration) error {
	limit, err := cache.extractOrderbookLimit(e, a)
	if err != nil {
		cache.clearWithLock()
		return err
	}

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

	book, err := e.fetchOrderbook(ctx, pair, a, limit)
	if err != nil {
		cache.clearWithLock()
		return err
	}

	if err := cache.waitForUpdate(ctx, book.LastUpdateID+1); err != nil {
		return err
	}

	cache.mtx.Lock() // Lock here to prevent ws handle data interference with REST request above.
	defer func() {
		cache.clearNoLock()
		cache.mtx.Unlock()
	}()

	if a != asset.Spot { // Regarding Spot, Margin and Cross Margin, the asset is hard coded to `spot in the calling function
		if err := e.Websocket.Orderbook.LoadSnapshot(book); err != nil {
			cache.state = desynchronised
			return err
		}
	} else {
		// Spot, Margin, and Cross Margin books are all classified as spot
		for i := range standardMarginAssetTypes {
			if enabled, _ := e.IsPairEnabled(pair, standardMarginAssetTypes[i]); !enabled {
				continue
			}
			book.Asset = standardMarginAssetTypes[i]
			if err := e.Websocket.Orderbook.LoadSnapshot(book); err != nil {
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
func (cache *updateCache) extractOrderbookLimit(e *Exchange, a asset.Item) (uint64, error) {
	switch a {
	case asset.Spot: // Regarding Spot, Margin and Cross Margin, the asset is hard coded to `spot` in the calling function
		sub := e.Websocket.GetSubscription(spotOrderbookUpdateKey)
		if sub == nil {
			return 0, fmt.Errorf("%w for %q", subscription.ErrNotFound, spotOrderbookUpdateKey)
		}
		// There is no way to set levels when we subscribe for this specific channel
		// Extract limit from interval e.g. 20ms == 20 limit book and 100ms == 100 limit book.
		lim := uint64(sub.Interval.Duration().Milliseconds()) //nolint:gosec // No overflow risk
		if lim != 20 && lim != 100 {
			return 0, fmt.Errorf("%w: %d. Valid limits are 20 and 100", errInvalidOrderbookUpdateInterval, lim)
		}
		return lim, nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		return futuresOrderbookUpdateLimit, nil
	case asset.DeliveryFutures:
		return deliveryFuturesUpdateLimit, nil
	case asset.Options:
		return optionOrderbookUpdateLimit, nil
	default:
		return 0, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
}

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

// applyOrderbookUpdate applies an orderbook update to the orderbook
func applyOrderbookUpdate(g *Exchange, update *orderbook.Update) error {
	if update.Asset != asset.Spot {
		return g.Websocket.Orderbook.Update(update)
	}

	var updated bool
	for i := range standardMarginAssetTypes {
		if enabled, _ := g.IsPairEnabled(update.Pair, standardMarginAssetTypes[i]); !enabled {
			continue
		}
		update.Asset = standardMarginAssetTypes[i]
		if err := g.Websocket.Orderbook.Update(update); err != nil {
			return err
		}
		updated = true
	}

	if !updated {
		return fmt.Errorf("%w: %q %q %w", errApplyingOrderbookUpdate, update.Pair, update.Asset, currency.ErrPairNotEnabled)
	}

	return nil
}
