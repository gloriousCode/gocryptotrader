package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/internal/diagnostics"
)

func TestEngineStartDiagnostics(t *testing.T) {
	// AllocsPerRun changes process-wide GOMAXPROCS and forbids parallel tests.
	t.Run("disabled and incomplete engines do not record", func(t *testing.T) {
		var absent *Engine
		absent.startDiagnostics(t.Context())
		new(Engine).startDiagnostics(t.Context())
		configuredButDisabled := &Engine{Config: &config.Config{}, Settings: Settings{DataDir: "testdata/never-created"}}
		ctx := t.Context()
		allocations := testing.AllocsPerRun(100, func() { configuredButDisabled.startDiagnostics(ctx) })
		assert.Zero(t, allocations, "disabled recording should not allocate timers, goroutines or journal state")
		_, err := os.Stat("testdata/never-created")
		assert.ErrorIs(t, err, os.ErrNotExist, "disabled diagnostics should not create files")
	})
	t.Run("enabled recorder observes runtime cancellation", func(t *testing.T) {
		directory := filepath.Join("testdata", "diagnostics_start", "diagnostics")
		t.Cleanup(func() {
			_ = os.Remove(filepath.Join(directory, "runtime.json"))
			_ = os.Remove(filepath.Join(directory, "previous.json"))
		})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		bot := &Engine{Config: &config.Config{}, Settings: Settings{DataDir: filepath.Join("testdata", "diagnostics_start"), CoreSettings: CoreSettings{EnableDiagnostics: true}}}
		bot.startDiagnostics(ctx)
		bot.runtimeMu.RLock()
		done := bot.diagnosticsDone
		bot.runtimeMu.RUnlock()
		require.NotNil(t, done, "enabled recorders must have a tracked cleanup signal")
		assert.Eventually(t, func() bool {
			snapshot, err := diagnostics.Read(filepath.Join(directory, "runtime.json"))
			if err != nil || snapshot.State != "stopping" {
				return false
			}
			_, err = os.Stat(filepath.Join(directory, "owner.lock"))
			return os.IsNotExist(err)
		}, 3*time.Second, 10*time.Millisecond, "recorder should save cancellation and release ownership")
		joinCtx, joinCancel := context.WithTimeout(t.Context(), 3*time.Second)
		defer joinCancel()
		require.True(t, bot.waitDiagnostics(joinCtx), "the recorder must finish observer and ownership cleanup")
		previous, err := diagnostics.Read(filepath.Join(directory, "runtime.json"))
		require.NoError(t, err, "the first run must remain readable before restart")
		bot.startDiagnostics(ctx)
		require.True(t, bot.waitDiagnostics(joinCtx), "a subsequent run must finish without competing with the previous owner")
		current, err := diagnostics.Read(filepath.Join(directory, "runtime.json"))
		require.NoError(t, err, "the subsequent run must publish evidence")
		assert.NotEqual(t, previous.RunID, current.RunID, "the subsequent recorder should have a fresh run identity")
		retained, err := diagnostics.Read(filepath.Join(directory, "previous.json"))
		require.NoError(t, err, "restart must archive the first completed observation")
		assert.Equal(t, previous.RunID, retained.RunID, "restart should retain the actual previous run")
	})
	t.Run("previous recorder cleanup must finish before replacement", func(t *testing.T) {
		done := make(chan struct{})
		bot := &Engine{Config: &config.Config{}, Settings: Settings{DataDir: "testdata/never-created", CoreSettings: CoreSettings{EnableDiagnostics: true}}, diagnosticsDone: done}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		bot.startDiagnostics(ctx)
		assert.Equal(t, done, bot.diagnosticsDone, "a pending cleanup should retain its original ownership signal")
		_, err := os.Stat("testdata/never-created")
		assert.ErrorIs(t, err, os.ErrNotExist, "refused replacement should not create another journal")
		close(done)
	})
}

func TestEngineWaitDiagnostics(t *testing.T) {
	t.Parallel()
	closed := make(chan struct{})
	close(closed)
	for _, tc := range []struct {
		name      string
		engine    *Engine
		cancelled bool
		finished  bool
	}{
		{name: "nil engine", finished: true},
		{name: "disabled recorder", engine: new(Engine), finished: true},
		{name: "completed cleanup", engine: &Engine{diagnosticsDone: closed}, finished: true},
		{name: "completed cleanup wins over cancellation", engine: &Engine{diagnosticsDone: closed}, cancelled: true, finished: true},
		{name: "pending cleanup respects deadline", engine: &Engine{diagnosticsDone: make(chan struct{})}, cancelled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.cancelled {
				cancel()
			}
			assert.Equal(t, tc.finished, tc.engine.waitDiagnostics(ctx), "joining should distinguish completed cleanup from a bounded timeout")
			if tc.engine != nil && tc.finished {
				assert.Nil(t, tc.engine.diagnosticsDone, "completed cleanup should release its lifecycle reference")
			}
		})
	}
	t.Run("waits for owned cleanup completion", func(t *testing.T) {
		t.Parallel()
		done := make(chan struct{})
		bot := &Engine{diagnosticsDone: done}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		result := make(chan bool, 1)
		go func() { result <- bot.waitDiagnostics(ctx) }()
		close(done)
		select {
		case finished := <-result:
			assert.True(t, finished, "completed ownership cleanup should release the join")
		case <-ctx.Done():
			require.FailNow(t, "recorder cleanup must release its waiter")
		}
	})
}

func TestEngineRecordDiagnostics(t *testing.T) {
	t.Parallel()
	t.Run("retains previous run", func(t *testing.T) {
		t.Parallel()
		directory := filepath.Join("testdata", "diagnostics_record")
		path := filepath.Join(directory, "runtime.json")
		t.Cleanup(func() {
			_ = os.Remove(path)
			_ = os.Remove(filepath.Join(directory, "previous.json"))
		})
		now := time.Now().UTC()
		previous := &diagnostics.Snapshot{Version: 1, RunID: "previous-test", StartedAt: now.Add(-time.Hour), CapturedAt: now.Add(-time.Minute), State: "running", Views: map[string]any{}}
		require.NoError(t, diagnostics.Write(path, previous), "a previous run must be saved before replacement")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		bot := &Engine{Config: &config.Config{}}
		bot.recordDiagnostics(ctx, directory, &Settings{}, now)
		current, err := diagnostics.Read(path)
		require.NoError(t, err, "the cancelled run must save its final recorder observation")
		assert.Equal(t, "stopping", current.State, "cancellation should not claim completed engine shutdown")
		assert.NotEqual(t, previous.RunID, current.RunID, "new runs should have distinct identities")
		retained, err := diagnostics.Read(filepath.Join(directory, "previous.json"))
		require.NoError(t, err, "the previous run must remain readable offline")
		assert.Equal(t, previous, retained, "previous evidence should retain its original timestamps and state")
		for _, source := range []string{"runtime_health", "identity", "performance", "configuration", "permissions", "market_health", "market_snapshots", "instrument_metadata", "storage", "websocket_health", "events", "run_report"} {
			assert.Contains(t, current.Views, source, "every diagnostic source should retain evidence")
		}
		_, err = os.Stat(filepath.Join(directory, "owner.lock"))
		assert.ErrorIs(t, err, os.ErrNotExist, "completed recorder should release its claim")
	})
	t.Run("archive failure preserves previous runtime evidence", func(t *testing.T) {
		t.Parallel()
		directory := filepath.Join("testdata", "diagnostics_archive")
		path := filepath.Join(directory, "runtime.json")
		t.Cleanup(func() { _ = os.Remove(path) })
		now := time.Now().UTC()
		previous := &diagnostics.Snapshot{Version: 1, RunID: "archive-failure", StartedAt: now.Add(-time.Hour), CapturedAt: now.Add(-time.Minute), State: "running", Views: map[string]any{}}
		require.NoError(t, diagnostics.Write(path, previous), "the previous runtime record must exist before archive failure")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		new(Engine).recordDiagnostics(ctx, directory, &Settings{}, now)
		retained, err := diagnostics.Read(path)
		require.NoError(t, err, "failed archiving must leave previous evidence readable")
		assert.Equal(t, previous, retained, "failed archiving should not replace the only previous-run evidence")
		_, err = os.Stat(filepath.Join(directory, "owner.lock"))
		assert.ErrorIs(t, err, os.ErrNotExist, "failed recording should release its ownership claim")
	})
}

func TestDiagnosticEventsObserve(t *testing.T) {
	t.Parallel()
	events := new(diagnosticEvents)
	events.observe("INFO", "worker")
	assert.Empty(t, events.rows, "ordinary logs should not consume warning history")
	events.observe("ERROR", "worker")
	events.observe("ERROR", "worker")
	require.Len(t, events.rows, 1, "identical metadata must coalesce")
	assert.Equal(t, uint64(2), events.rows[0].Count, "repeated errors should retain their occurrence count")
	events.observe("WARN", "private@secret")
	assert.Equal(t, "other", events.rows[1].Subsystem, "unrecognised subsystem text should not be retained")
	for i := range 130 {
		events.observe("WARN", fmt.Sprintf("worker-%d", i))
	}
	assert.Len(t, events.rows, 128, "retained groups should be bounded")
	assert.Equal(t, uint64(4), events.evicted, "eviction should count lost groups")
	assert.Equal(t, "worker-129", events.rows[len(events.rows)-1].Subsystem, "the newest group should be retained")
}

func TestDiagnosticStackLocations(t *testing.T) {
	t.Parallel()
	locations, truncated := diagnosticStackLocations()
	assert.NotEmpty(t, locations, "stack observations should retain code locations")
	assert.LessOrEqual(t, len(locations), 64, "stack observations should be bounded")
	if len(locations) == 64 {
		assert.True(t, truncated, "a full retained sample should disclose possible truncation")
	}
	for _, location := range locations {
		assert.NotContains(t, location, string(filepath.Separator), "stack observations should omit filesystem directories")
		assert.NotContains(t, location, "(", "stack observations should omit argument values")
		assert.Contains(t, location, ".go:", "stack observations should identify source locations")
	}
}

func TestDiagnosticPerformance(t *testing.T) {
	t.Parallel()
	result := diagnosticPerformance()
	assert.Positive(t, result["goroutines"], "current goroutine count should be available")
	assert.Positive(t, result["heapBytes"], "current heap allocation should be available")
	assert.Positive(t, result["logicalCPUs"], "logical CPU count should be available")
}

func TestDiagnosticBuildIdentity(t *testing.T) {
	t.Parallel()
	result := diagnosticBuildIdentity()
	assert.NotEmpty(t, result["goVersion"], "the actual runtime version should be available")
	assert.Contains(t, result, "revision", "missing VCS metadata should remain explicit")
	assert.Contains(t, result, "dirty", "missing VCS state should remain explicit")
}

func TestEngineDiagnosticSnapshot(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	bot := &Engine{}
	events := new(diagnosticEvents)
	events.observe("ERROR", "worker")
	snapshot := bot.diagnosticSnapshot("test-snapshot", now, now, "running", &Settings{EnableDryRun: true, EnableGRPCShutdown: true}, events)
	assert.Equal(t, "test-snapshot", snapshot.RunID, "run identity should be retained")
	configuration, ok := snapshot.Views["configuration"].(map[string]any)
	require.True(t, ok, "configuration observations must have a known shape")
	assert.NotEmpty(t, configuration["currentFingerprint"], "sanitised configuration should have a fingerprint")
	encoded, err := json.Marshal(snapshot)
	require.NoError(t, err, "diagnostic evidence must serialise")
	assert.NotContains(t, strings.ToLower(string(encoded)), "password", "diagnostic evidence should not expose credential fields")
	assert.Contains(t, string(encoded), "error_observed", "observed warning history should be retained")
}

func TestEngineDiagnosticMarketViews(t *testing.T) {
	t.Parallel()
	for _, count := range []int{0, 1, 199, 200, 201} {
		t.Run(fmt.Sprintf("%d enabled markets", count), func(t *testing.T) {
			t.Parallel()
			manager := NewExchangeManager()
			exch, err := manager.NewExchangeByName("binance")
			require.NoError(t, err, "the fake exchange must be created")
			exch.SetDefaults()
			base := exch.GetBase()
			base.Name = newUniqueFakeExchangeName()
			base.Enabled = true
			pairs := make(currency.Pairs, 0, count)
			for i := range count {
				pairs = append(pairs, currency.NewPair(currency.NewCode(fmt.Sprintf("DIAG%03d", i)), currency.USDT))
			}
			base.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{asset.Spot: {AssetEnabled: true, Available: pairs, Enabled: pairs, ConfigFormat: &currency.PairFormat{Delimiter: "-", Uppercase: true}, RequestFormat: &currency.PairFormat{Delimiter: "-", Uppercase: true}}}
			require.NoError(t, manager.Add(exch), "fake exchange must be registered")
			if count > 0 {
				pair := pairs[0]
				require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: base.Name, Pair: pair, AssetType: asset.Spot, Last: 1.5, Bid: 1, Ask: 2, LastUpdated: time.Now()}), "a cached ticker must be published")
				depth, err := orderbook.DeployDepth(base.Name, pair, asset.Spot)
				require.NoError(t, err, "the fake orderbook must be deployed")
				require.NoError(t, depth.LoadSnapshot(&orderbook.Book{Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now()}), "the fake orderbook must load")
			}
			views := (&Engine{ExchangeManager: manager}).diagnosticMarketViews()
			for _, source := range []string{"market_health", "market_snapshots", "instrument_metadata"} {
				view, ok := views[source].(map[string]any)
				require.True(t, ok, "market view must have a known shape")
				assert.Equal(t, count, view["total"], "coverage should describe the complete configured universe")
				assert.Equal(t, count > 200, view["truncated"], "partial coverage should be explicit")
				rows, ok := view["markets"].([]map[string]any)
				require.True(t, ok, "market rows must retain native identities")
				assert.Len(t, rows, min(count, 200), "each source should bound market rows")
				if count > 0 {
					assert.Equal(t, "DIAG000", rows[0]["base"], "native identities should remain sorted")
					assert.Equal(t, "USDT", rows[0]["quote"], "native quote codes should be retained")
					assert.Equal(t, fmt.Sprintf("DIAG%03d", min(count, 200)-1), rows[len(rows)-1]["base"], "the final retained market should match the coverage bound")
				}
			}
		})
	}
}
