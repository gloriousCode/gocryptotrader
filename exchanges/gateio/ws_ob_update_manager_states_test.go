package gateio

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func waitForState(t *testing.T, cache *updateCache, exp state) {
	t.Helper()
	deadline := time.Now().Add(time.Second * 2)
	for {
		cache.mtx.Lock()
		cur := cache.state
		cache.mtx.Unlock()
		if cur == exp {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for state %v, last state %v", exp, cur)
		}
		time.Sleep(time.Millisecond * 20)
	}
}

func TestProcessOrderbookUpdateStates(t *testing.T) {
	processDelay := time.Millisecond
	syncDeadline := time.Second
	ctx := t.Context()
	e := new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Gateio Setup error: %s", err)
	}

	pair := currency.NewPair(currency.DOGE, currency.BABYDOGE)
	require.NoError(t, e.Base.SetPairs([]currency.Pair{pair}, asset.USDTMarginedFutures, true))

	m := newWsOBUpdateManager(processDelay, syncDeadline)

	cache, err := m.LoadCache(pair, asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.Equal(t, initialised, cache.state, "newly loaded cache should be in initialised state")

	// 1. initialised -> queuingUpdates
	updateOneID := int64(1337)
	upd1 := &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures, UpdateID: updateOneID, AllowEmpty: true, UpdateTime: time.Now()}
	require.NoError(t, m.ProcessOrderbookUpdate(ctx, e, updateOneID, upd1))

	cache, err = m.LoadCache(pair, asset.USDTMarginedFutures)
	require.NoError(t, err)

	// While still queuing, enqueue a second update (tests queueOrderbookUpdate path)
	updateTwoID := updateOneID + 1
	upd2 := &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures, UpdateID: updateTwoID, AllowEmpty: true, UpdateTime: time.Now()}
	require.NoError(t, m.ProcessOrderbookUpdate(ctx, e, updateTwoID, upd2))

	waitForState(t, cache, queuingUpdates)
	assert.Equal(t, 2, len(cache.updates))

	// 2. desync orderbook with a lame update
	upd3 := &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures, UpdateID: 2000000, AllowEmpty: true, UpdateTime: time.Now()}
	require.NoError(t, m.ProcessOrderbookUpdate(ctx, e, 2000000, upd3))
	waitForState(t, cache, desynchronised)

	// Manager should recover and re-synchronise
	require.NoError(t, m.ProcessOrderbookUpdate(ctx, e, 1337, upd3))
	waitForState(t, cache, queuingUpdates)

	m = newWsOBUpdateManager(processDelay, time.Millisecond*100)
	upd3.UpdateID = 1338
	require.NoError(t, m.ProcessOrderbookUpdate(ctx, e, 1337, upd3))
	waitForState(t, cache, queuingUpdates)
	time.Sleep(time.Second)
	upd3.UpdateID = 1339
	require.NoError(t, m.ProcessOrderbookUpdate(ctx, e, 1338, upd3))
	waitForState(t, cache, synchronised)
}
