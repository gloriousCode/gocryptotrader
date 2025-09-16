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
}

type updateCache struct {
	updates              []pendingUpdate
	mtx                  sync.Mutex
	state                state
	ch                   chan int64 // receives newest pending update IDs while syncing snapshot
	snapshotLastUpdateID int64
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

func (m *wsOBUpdateManager) ProcessOrderbookUpdate(ctx context.Context, e *Exchange, firstUpdateID int64, update *orderbook.Update) error {
	cache, err := m.LoadCache(update.Pair, update.Asset)
	if err != nil {
		return err
	}
	cache.mtx.Lock()
	defer cache.mtx.Unlock()
	switch cache.state {
	case synchronised: // prioritised hot path
		return m.updateSynchronisedOrderbook(ctx, e, update, firstUpdateID, cache)
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
		if err := m.setupOrderbookSnapshot(e, update, ctx, cache); err != nil {
			log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to set orderbook limits: %v", e.Name, err)
		}
	}()
	cache.state = queuingUpdates
	cache.enqueueUpdate(update, firstUpdateID)
	return nil
}

func (m *wsOBUpdateManager) setupOrderbookSnapshot(e *Exchange, update *orderbook.Update, ctx context.Context, cache *updateCache) error {
	var lim uint64
	lim = e.wsOrderbookLimits.get(update.Asset)
	if lim == 0 {
		if err := e.setWSOrderbookLimits(); err != nil {
			return err
		}
	}
	b, err := e.fetchOrderbook(ctx, update.Pair, update.Asset, lim)
	if err != nil {
		return err
	}
	cache.snapshotLastUpdateID = b.LastUpdateID
	return cache.SyncOrderbookSnapshotToUpdates(ctx, e, b, m.delay, m.deadline)
}

func (m *wsOBUpdateManager) updateSynchronisedOrderbook(ctx context.Context, e *Exchange, update *orderbook.Update, firstUpdateID int64, cache *updateCache) error {
	var err error
	cache.snapshotLastUpdateID, err = e.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset)
	if err != nil {
		return err
	}
	if cache.snapshotLastUpdateID+1 != firstUpdateID {
		cache.state = desynchronised
		return m.desynchroniseOrderbook(ctx, cache, e, update, firstUpdateID)
	}
	if err := applyOrderbookUpdate(e, update); err != nil {
		cache.state = desynchronised
		return m.desynchroniseOrderbook(ctx, cache, e, update, firstUpdateID)
	}
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

func (e *Exchange) setWSOrderbookLimits() error {
	e.wsOrderbookLimits.m.Lock()
	defer e.wsOrderbookLimits.m.Unlock()
	if len(e.wsOrderbookLimits.l) > 0 {
		return nil
	}
	e.wsOrderbookLimits.l = make(map[asset.Item]uint64)
	for _, a := range e.GetAssetTypes(false) {
		switch a {
		case asset.Margin, asset.CrossMargin, asset.Spot: // Regarding Spot, Margin and Cross Margin, the asset is hard coded to `spot` in the calling function
			sub := e.Websocket.GetSubscription(spotOrderbookUpdateKey)
			if sub == nil {
				return fmt.Errorf("%w for %q", subscription.ErrNotFound, spotOrderbookUpdateKey)
			}
			// There is no way to set levels when we subscribe for this specific channel
			// Extract limit from interval e.g. 20ms == 20 limit book and 100ms == 100 limit book.
			lim := uint64(sub.Interval.Duration().Milliseconds()) //nolint:gosec // No overflow risk
			if lim != 20 && lim != 100 {
				return fmt.Errorf("%w: %d. Valid limits are 20 and 100", errInvalidOrderbookUpdateInterval, lim)
			}
			e.wsOrderbookLimits.l[a] = lim
		case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
			e.wsOrderbookLimits.l[a] = futuresOrderbookUpdateLimit
		case asset.DeliveryFutures:
			e.wsOrderbookLimits.l[a] = deliveryFuturesUpdateLimit
		case asset.Options:
			e.wsOrderbookLimits.l[a] = optionOrderbookUpdateLimit
		default:
			return fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
		}
	}
	return nil
}

func (l *wsOrderbookLimits) get(a asset.Item) uint64 {
	l.m.Lock()
	defer l.m.Unlock()
	return l.l[a]
}
