package gctmcp

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestBridgeAddDiagnosticTools(t *testing.T) {
	t.Parallel()
	for _, configured := range []bool{false, true} {
		t.Run(fmt.Sprintf("configured_%t", configured), func(t *testing.T) {
			t.Parallel()
			bridge := &Bridge{options: Options{Policy: testExecutionPolicy, Timeout: time.Second, MaxResponseBytes: 1 << 20, Offline: true}}
			if configured {
				bridge.options.DiagnosticsDir = "testdata/inspection"
			}
			server := mcp.NewServer(&mcp.Implementation{Name: "diagnostics-test", Version: "1"}, nil)
			bridge.addDiagnosticTools(server)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			t.Cleanup(cancel)
			serverTransport, clientTransport := mcp.NewInMemoryTransports()
			serverSession, err := server.Connect(ctx, serverTransport, nil)
			require.NoError(t, err, "MCP diagnostic server must connect")
			t.Cleanup(func() { _ = serverSession.Close() })
			session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
			require.NoError(t, err, "MCP diagnostic client must connect")
			t.Cleanup(func() { _ = session.Close() })
			tools, err := session.ListTools(ctx, nil)
			require.NoError(t, err, "diagnostic tools must list without an RPC client")
			if !configured {
				require.Len(t, tools.Tools, 1, "known bridge policy must remain available without a recorder directory")
				assert.Equal(t, "gct_permissions", tools.Tools[0].Name, "only current permissions should be advertised without saved evidence")
				return
			}
			assert.Len(t, tools.Tools, len(diagnosticSources), "offline configuration should advertise all diagnostic views")
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "gct_runtime_health", Arguments: map[string]any{}})
			require.NoError(t, err, "offline runtime evidence must be readable")
			assert.False(t, result.IsError, "valid saved diagnostics should not produce a tool error")
			encoded, err := json.Marshal(result)
			require.NoError(t, err, "MCP diagnostic result must encode")
			assert.NotContains(t, string(encoded), "private-test-secret", "credential fields should not reach the MCP client")
		})
	}
}

func TestBridgeInspectDiagnostic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		dir       string
		execution bool
	}{
		{name: "no daemon directory"},
		{name: "no daemon record", dir: "testdata/missing"},
		{name: "daemon view not recorded", dir: "testdata/inspection"},
		{name: "execution authority", execution: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy := config.MCPConfig{Enabled: true, Permissions: config.MCPPermissions{Diagnostics: true}}
			if tc.execution {
				policy = testExecutionPolicy
			}
			bridge, err := New(nil, &Options{Policy: policy, Offline: true, DiagnosticsDir: tc.dir, Timeout: time.Second, MaxMessages: 1, MaxResponseBytes: 4096})
			require.NoError(t, err, "readonly diagnostic policy must initialise without an RPC client")
			result := bridge.inspectDiagnostic(t.Context(), "permissions", &inspectionRequest{})
			require.Empty(t, result.Failure, "known bridge authority must not depend on a runtime journal")
			assert.Equal(t, "current_policy", result.Status, "current bridge policy should be distinguished from saved daemon data")
			data, ok := result.Data.(map[string]any)
			require.True(t, ok, "permission sources must be separated")
			current, ok := data["bridge"].(map[string]any)
			require.True(t, ok, "current authority must identify the bridge")
			assert.Equal(t, bridge.options.Policy, current["policy"], "effective policy should be the bridge's immutable copy")
			assert.Equal(t, true, current["offline"], "offline restrictions should remain explicit")
			if !tc.execution {
				assert.Equal(t, config.MCPReadOnly, bridge.options.Policy.Mode, "omitted modes should resolve to readonly")
			}
			daemon, ok := data["daemon"].(map[string]any)
			require.True(t, ok, "missing daemon evidence must remain separate")
			assert.Equal(t, statusUnavailable, daemon["status"], "missing recorder evidence should remain unavailable")
		})
	}
	for _, tc := range []struct {
		name    string
		source  string
		request inspectionRequest
		path    string
		maximum int
		failure string
		runID   string
	}{
		{name: "offline heartbeat", source: "runtime_health", runID: "test-inspection"},
		{name: "missing view", source: "storage", failure: "diagnostic_view_not_recorded"},
		{name: "unknown source", source: "unknown", failure: "invalid_inspection_request"},
		{name: "excessive limit", source: "runtime_health", request: inspectionRequest{Limit: 201}, failure: "invalid_inspection_request"},
		{name: "negative offset", source: "runtime_health", request: inspectionRequest{Offset: -1}, failure: "invalid_inspection_request"},
		{name: "missing record", source: "runtime_health", path: "testdata/missing", failure: "diagnostic_record_unavailable_or_invalid"},
		{name: "previous crash fallback", source: "crash_report", runID: "test-previous"},
		{name: "wire size limit", source: "runtime_health", maximum: 100, failure: "diagnostic_response_size_limit"},
		{name: "native market filter", source: "market_health", request: inspectionRequest{Exchange: "OKX", Asset: "perpetualswap", Base: "BTC", Quote: "USDT-SWAP"}, runID: "test-inspection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, maximum := tc.path, tc.maximum
			if path == "" {
				path = "testdata/inspection"
			}
			if maximum == 0 {
				maximum = 1 << 20
			}
			bridge := &Bridge{options: Options{Policy: testExecutionPolicy, DiagnosticsDir: path, MaxResponseBytes: maximum}}
			result := bridge.inspectDiagnostic(t.Context(), tc.source, &tc.request)
			assert.Equal(t, tc.failure, result.Failure, "inspection should distinguish missing, invalid and bounded evidence")
			if tc.runID != "" {
				assert.Equal(t, tc.runID, result.RunID, "identity should come from the saved run")
				assert.Equal(t, "historical", result.Status, "saved evidence should remain explicitly historical")
			}
			encoded, err := json.Marshal(result)
			require.NoError(t, err, "inspection result must encode")
			assert.NotContains(t, string(encoded), "private-test-secret", "raw credential fields should be withheld")
			if tc.source == "crash_report" {
				assert.Contains(t, string(encoded), `"exitOutcome":"unknown"`, "unclean previous state should not be claimed as a known crash")
			}
			if tc.name == "native market filter" {
				assert.Contains(t, string(encoded), "USDT-SWAP", "native suffix should remain intact")
				assert.NotContains(t, string(encoded), "binance", "nonmatching exchange rows should be omitted")
			}
		})
	}
	t.Run("cancelled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result := (&Bridge{options: Options{Policy: testExecutionPolicy}}).inspectDiagnostic(ctx, "runtime_health", &inspectionRequest{})
		assert.Equal(t, "invalid_inspection_request", result.Failure, "cancelled calls should not read files")
	})
}

func TestBoundInspection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		data    any
		maximum int
		failure string
	}{
		{name: "fits", data: map[string]any{"observed": true}, maximum: 4096},
		{name: "oversized", data: strings.Repeat("a", 4096), maximum: 4096, failure: "diagnostic_response_size_limit"},
		{name: "invalid output", data: func() {}, maximum: 4096, failure: "diagnostic_response_size_limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := &inspectionResult{Status: "historical", Data: tc.data}
			boundInspection(result, tc.maximum)
			assert.Equal(t, tc.failure, result.Failure, "inspection output should respect its actual MCP wire budget")
			if tc.failure != "" {
				assert.Nil(t, result.Data, "unencodable or oversized data should be withheld")
			}
		})
	}
}

func TestProjectDiagnostic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		value     any
		request   inspectionRequest
		depth     int
		want      any
		truncated bool
	}{
		{name: "redaction recursively", value: map[string]any{"api-key": "private", "raw_error": "private", "name": "public", "nested": map[string]any{"Authorization": "private", "known": true}}, want: map[string]any{"name": "public", "nested": map[string]any{"known": true}}},
		{name: "identity filtering and offset", value: []any{map[string]any{"exchange": "binance", "asset": "spot", "base": "BTC", "quote": "USDT", "id": "first"}, map[string]any{"exchange": "binance", "asset": "spot", "base": "BTC", "quote": "USDT", "id": "second"}, map[string]any{"exchange": "okx", "asset": "spot"}}, request: inspectionRequest{Exchange: "BINANCE", Asset: "SPOT", Base: "btc", Quote: "usdt", Offset: 1, Limit: 1}, want: []any{map[string]any{"exchange": "binance", "asset": "spot", "base": "BTC", "quote": "USDT", "id": "second"}}},
		{name: "market page limit does not truncate unrelated observations", value: []any{float64(1), float64(2), float64(3)}, request: inspectionRequest{Limit: 2}, want: []any{float64(1), float64(2), float64(3)}},
		{name: "exchange rows ignore unavailable asset filters", value: []any{map[string]any{"exchange": "binance", "connected": true}, map[string]any{"exchange": "okx", "connected": false}}, request: inspectionRequest{Exchange: "BINANCE", Asset: "spot"}, want: []any{map[string]any{"exchange": "binance", "connected": true}}},
		{name: "configuration filters nested assets without dropping exchange", value: []any{map[string]any{"exchange": "binance", "assets": []any{map[string]any{"asset": "spot"}, map[string]any{"asset": "futures"}}}}, request: inspectionRequest{Exchange: "BINANCE", Asset: "SPOT"}, want: []any{map[string]any{"exchange": "binance", "assets": []any{map[string]any{"asset": "spot"}}}}},
		{name: "exchange offset is not applied again to nested assets", value: []any{map[string]any{"exchange": "binance", "assets": []any{map[string]any{"asset": "spot"}}}, map[string]any{"exchange": "okx", "assets": []any{map[string]any{"asset": "spot"}, map[string]any{"asset": "futures"}}}}, request: inspectionRequest{Offset: 1, Limit: 1}, want: []any{map[string]any{"exchange": "okx", "assets": []any{map[string]any{"asset": "spot"}, map[string]any{"asset": "futures"}}}}},
		{name: "depth bounded", value: "public", depth: 13, truncated: true},
		{name: "overlong text bounded", value: strings.Repeat("a", 257), want: "overlong_diagnostic_value_withheld", truncated: true},
		{name: "unknown values excluded", value: make(chan int), truncated: true},
		{name: "null remains unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var truncated bool
			assert.Equal(t, tc.want, projectDiagnostic(tc.value, &tc.request, tc.depth, &truncated), "projection should preserve only bounded observations")
			assert.Equal(t, tc.truncated, truncated, "lost observations should be disclosed")
		})
	}
	t.Run("market page limit retains twenty book levels", func(t *testing.T) {
		t.Parallel()
		levels := make([]any, 20)
		for i := range levels {
			levels[i] = map[string]any{"price": float64(i + 1), "amount": float64(1)}
		}
		value := []any{map[string]any{"exchange": "binance", "asset": "spot", "bids": levels, "asks": levels}, map[string]any{"exchange": "okx", "asset": "spot", "bids": levels}}
		var truncated bool
		projected := projectDiagnostic(value, &inspectionRequest{Limit: 1}, 0, &truncated)
		rows, ok := projected.([]any)
		require.True(t, ok, "market output must remain an array")
		require.Len(t, rows, 1, "market pagination must retain one market")
		row, ok := rows[0].(map[string]any)
		require.True(t, ok, "market row must retain fields")
		assert.Len(t, row["bids"], 20, "a one-market page should retain all twenty bid levels")
		assert.Len(t, row["asks"], 20, "a one-market page should retain all twenty ask levels")
		assert.True(t, truncated, "omitted markets should remain explicitly truncated")
	})
	t.Run("nonidentity collections retain an independent hard bound", func(t *testing.T) {
		t.Parallel()
		var truncated bool
		values := make([]any, 201)
		assert.Len(t, projectDiagnostic(values, &inspectionRequest{Limit: 1}, 0, &truncated), 200, "nonmarket collections should retain their independent bound")
		assert.True(t, truncated, "omitted nonmarket observations should be disclosed")
	})
}
