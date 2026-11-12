package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestSyncManagerDiagnosticProgress(t *testing.T) {
	t.Parallel()
	t.Run("missing manager", func(t *testing.T) {
		t.Parallel()
		var manager *SyncManager
		assert.Equal(t, "unavailable", manager.diagnosticProgress(time.Now())["status"], "missing workers should remain unknown")
	})
	t.Run("attempts distinguish error totals from successful data", func(t *testing.T) {
		t.Parallel()
		manager := &SyncManager{config: config.SyncManagerConfig{SynchronizeTicker: true, SynchronizeOrderbook: true, TimeoutREST: time.Second, TimeoutWebsocket: time.Second}, currencyPairs: make(map[key.ExchangeAssetPair]*currencyPairSyncAgent)}
		manager.started.Store(true)
		now := time.Now().UTC()
		agent := manager.add(key.NewExchangeAssetPair("fake", asset.Spot, currency.NewBTCUSDT()), syncBase{LastUpdated: now.Add(-2 * time.Minute), HaveData: true, NumErrors: 2, IsUsingREST: true})
		agent.trackers[SyncItemOrderbook].LastUpdated = now
		agent.trackers[SyncItemOrderbook].IsUsingREST = false
		agent.trackers[SyncItemOrderbook].IsUsingWebsocket = true
		result := manager.diagnosticProgress(now)
		assert.Equal(t, 2, result["total"], "configured tracker coverage should be retained")
		rows, ok := result["attempts"].([]map[string]any)
		require.True(t, ok, "attempt evidence must have a known shape")
		require.Len(t, rows, 2, "both configured topics must be retained")
		assert.Equal(t, "rest", rows[0]["protocol"], "REST receipt provenance should be retained")
		assert.Equal(t, 2, rows[0]["errorCount"], "failed attempts should retain their error counts")
		assert.Equal(t, true, rows[0]["needsInspection"], "overdue attempts should be flagged for inspection")
		assert.Equal(t, "websocket", rows[1]["protocol"], "websocket receipt provenance should be retained")
		assert.Equal(t, false, rows[1]["needsInspection"], "recent attempts should not be marked overdue")
	})
	t.Run("busy network tracker does not block diagnostic capture", func(t *testing.T) {
		t.Parallel()
		manager := &SyncManager{config: config.SyncManagerConfig{SynchronizeTicker: true}, currencyPairs: make(map[key.ExchangeAssetPair]*currencyPairSyncAgent)}
		agent := manager.add(key.NewExchangeAssetPair("fake", asset.Spot, currency.NewBTCUSDT()), syncBase{HaveData: true, NumErrors: 2})
		agent.locks[SyncItemTicker].Lock()
		defer agent.locks[SyncItemTicker].Unlock()
		done := make(chan map[string]any, 1)
		go func() { done <- manager.diagnosticProgress(time.Now().UTC()) }()
		select {
		case result := <-done:
			rows, ok := result["attempts"].([]map[string]any)
			require.True(t, ok, "busy tracker evidence must retain its known shape")
			require.Len(t, rows, 1, "the busy tracker must remain represented")
			assert.Equal(t, "busy", rows[0]["status"], "mutex contention should remain explicit")
			assert.NotContains(t, rows[0], "errorCount", "busy tracker totals should stay unknown")
			assert.NotContains(t, rows[0], "lastAttemptAt", "busy attempt timestamps should stay unknown")
		case <-time.After(250 * time.Millisecond):
			require.FailNow(t, "diagnostic capture must not wait for an in-flight network operation")
		}
	})
	t.Run("busy batch request does not block diagnostic capture", func(t *testing.T) {
		t.Parallel()
		manager := new(SyncManager)
		manager.mux.Lock()
		defer manager.mux.Unlock()
		done := make(chan map[string]any, 1)
		go func() { done <- manager.diagnosticProgress(time.Now().UTC()) }()
		select {
		case result := <-done:
			assert.Equal(t, "busy", result["status"], "manager contention should remain explicit")
			assert.NotContains(t, result, "attempts", "busy manager coverage should stay unknown")
		case <-time.After(250 * time.Millisecond):
			require.FailNow(t, "diagnostic capture must not wait for a batched network operation")
		}
	})
}
