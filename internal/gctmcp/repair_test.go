package gctmcp

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestBridgeAddRepairTools(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		offline     bool
		execution   bool
		permissions config.MCPPermissions
		want        int
	}{
		{name: "offline", offline: true, execution: true, permissions: config.MCPPermissions{AccountData: true}, want: 1},
		{name: "read only", permissions: config.MCPPermissions{AccountData: true}, want: 1},
		{name: "account refresh execution", execution: true, permissions: config.MCPPermissions{AccountData: true}, want: 2},
		{name: "system control alone has no refresh action", execution: true, permissions: config.MCPPermissions{SystemControl: true}, want: 1},
		{name: "pair refresh execution", execution: true, permissions: config.MCPPermissions{SystemControl: true, OrderManagement: true}, want: 2},
		{name: "diagnostic receipt inspection", permissions: config.MCPPermissions{Diagnostics: true}, want: 1},
		{name: "public only", execution: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bridge := &Bridge{options: Options{Policy: testExecutionPolicy, Offline: tc.offline, DiagnosticsDir: "testdata/inspection", MaxResponseBytes: 4096}}
			bridge.options.Policy.Permissions = tc.permissions
			if !tc.execution {
				bridge.options.Policy.Mode = config.MCPReadOnly
			}
			server := mcp.NewServer(&mcp.Implementation{Name: "repair-test", Version: "1"}, nil)
			bridge.addRepairTools(server)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			t.Cleanup(cancel)
			serverTransport, clientTransport := mcp.NewInMemoryTransports()
			serverSession, err := server.Connect(ctx, serverTransport, nil)
			require.NoError(t, err, "repair MCP server must connect")
			t.Cleanup(func() { _ = serverSession.Close() })
			session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
			require.NoError(t, err, "repair MCP client must connect")
			t.Cleanup(func() { _ = session.Close() })
			tools, err := session.ListTools(ctx, nil)
			require.NoError(t, err, "repair tools must list")
			assert.Len(t, tools.Tools, tc.want, "only permitted refresh tools should be advertised")
			for _, tool := range tools.Tools {
				if tool.Name == "gct_repair_status" {
					assert.True(t, tool.Annotations.ReadOnlyHint, "receipt inspection should remain read-only")
					bridge.options.MaxResponseBytes = 100
					result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{}})
					require.NoError(t, err, "receipt callback must return a structured result")
					encoded, err := json.Marshal(result)
					require.NoError(t, err, "bounded callback result must encode")
					assert.Contains(t, string(encoded), "repair_response_size_limit", "external tool callback should enforce its response budget")
				} else {
					assert.False(t, tool.Annotations.ReadOnlyHint, "repair execution should be marked as a change")
					assert.True(t, tool.Annotations.IdempotentHint, "durable request identity should prevent repeats")
				}
			}
		})
	}
}

func TestValidateRepairRequest(t *testing.T) {
	t.Parallel()
	base := repairRequest{ID: "repair-test-001", Action: "refresh_balances", Exchange: "okx", Asset: "perpetualswap", Reason: "stale_data"}
	for _, tc := range []struct {
		name string
		edit func(*repairRequest)
		fail bool
	}{
		{name: "native asset"},
		{name: "invalid ID", edit: func(r *repairRequest) { r.ID = "../secret" }, fail: true},
		{name: "short ID", edit: func(r *repairRequest) { r.ID = "short" }, fail: true},
		{name: "unknown operation", edit: func(r *repairRequest) { r.Action = "restart" }, fail: true},
		{name: "unknown reason", edit: func(r *repairRequest) { r.Reason = "because" }, fail: true},
		{name: "URL exchange rejected", edit: func(r *repairRequest) { r.Exchange = "http://localhost" }, fail: true},
		{name: "unsafe asset rejected", edit: func(r *repairRequest) { r.Asset = "spot/secret" }, fail: true},
		{name: "balance refresh requires asset", edit: func(r *repairRequest) { r.Asset = "" }, fail: true},
		{name: "pair refresh", edit: func(r *repairRequest) { r.Action, r.Asset = "refresh_pairs", "" }},
		{name: "pair refresh cannot select asset", edit: func(r *repairRequest) { r.Action = "refresh_pairs" }, fail: true},
		{name: "history repair excluded", edit: func(r *repairRequest) { r.Action = "candle_history" }, fail: true},
		{name: "history reason excluded", edit: func(r *repairRequest) { r.Reason = "missing_history" }, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := base
			if tc.edit != nil {
				tc.edit(&request)
			}
			err := validateRepairRequest(&request)
			if tc.fail {
				assert.ErrorIs(t, err, errRepairRequest, "invalid action should retain a stable validation error")
			} else {
				assert.NoError(t, err, "exact supported action should validate")
			}
		})
	}
}

func TestReadRepair(t *testing.T) {
	// The permanent receipt fixture directory is shared by filesystem tests.
	root, err := os.OpenRoot("testdata/repair")
	require.NoError(t, err, "fixture root must open")
	t.Cleanup(func() { _ = root.Close() })
	id := fmt.Sprintf("test-read-%d", os.Getpid())
	for _, tc := range []struct {
		name string
		edit func(*repairRecord)
		raw  string
		fail bool
	}{
		{name: "running receipt"},
		{name: "known RPC failure", edit: func(r *repairRecord) { r.Status, r.Failure = "unknown", "repair_rpc_Unavailable" }},
		{name: "preflight rejection", edit: func(r *repairRecord) { r.Status, r.Failure = "rejected", "repair_target_not_enabled" }},
		{name: "raw failure withheld", edit: func(r *repairRecord) { r.Failure = "private-error-detail" }, fail: true},
		{name: "unknown version", edit: func(r *repairRecord) { r.Version = 2 }, fail: true},
		{name: "unknown state", edit: func(r *repairRecord) { r.Status = "invented" }, fail: true},
		{name: "wrong request identity", edit: func(r *repairRecord) { r.Request.ID = "different-id" }, fail: true},
		{name: "missing admission time", edit: func(r *repairRecord) { r.StartedAt = time.Time{} }, fail: true},
		{name: "unknown fields", raw: `{"private":"secret"}`, fail: true},
		{name: "null record", raw: `null`, fail: true},
		{name: "oversized record", raw: strings.Repeat("x", 16385), fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "repair-" + id + ".json"
			t.Cleanup(func() { _ = root.Remove(name) })
			record := &repairRecord{Version: 1, Request: repairRequest{ID: id, Action: "refresh_pairs", Exchange: "okx", Reason: "stale_data"}, StartedAt: time.Now().UTC(), Status: "running"}
			if tc.edit != nil {
				tc.edit(record)
			}
			data := []byte(tc.raw)
			if tc.raw == "" {
				var err error
				data, err = json.Marshal(record)
				require.NoError(t, err, "test receipt must encode")
			}
			require.NoError(t, os.WriteFile(filepath.Join("testdata", "repair", name), data, 0o600), "test receipt must be written")
			read, err := readRepair(root, id)
			if tc.fail {
				assert.ErrorIs(t, err, errRepairRequest, "malformed or private receipt data should be rejected")
			} else {
				require.NoError(t, err, "valid receipt must be readable")
				assert.Equal(t, record.Request, read.Request, "receipt should preserve exact request identity")
			}
		})
	}
	_, err = readRepair(root, "../secret")
	assert.ErrorIs(t, err, errRepairRequest, "receipt lookup should reject paths")
}

func TestWriteRepair(t *testing.T) {
	// The permanent receipt fixture directory is shared by filesystem tests.
	root, err := os.OpenRoot("testdata/repair")
	require.NoError(t, err, "fixture root must open")
	t.Cleanup(func() { _ = root.Close() })
	id := fmt.Sprintf("test-write-%d", os.Getpid())
	name := "repair-" + id + ".json"
	t.Cleanup(func() { _ = root.Remove(name); _ = root.Remove(name + ".next") })
	assert.ErrorIs(t, writeRepair(root, nil, true), errRepairRequest, "missing receipt should be rejected")
	record := &repairRecord{Version: 1, Request: repairRequest{ID: id, Action: "refresh_pairs", Exchange: "okx", Reason: "stale_data"}, StartedAt: time.Now().UTC(), Status: "running"}
	require.NoError(t, writeRepair(root, record, true), "intent must be durable before execution")
	err = writeRepair(root, record, true)
	assert.True(t, os.IsExist(err), "exclusive intent should refuse duplicate creation")
	info, err := root.Stat(name)
	require.NoError(t, err, "intent metadata must be readable")
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "receipt permissions should be owner-only")
	record.Status, record.EndedAt = "completed_unverified", time.Now().UTC()
	require.NoError(t, writeRepair(root, record, false), "completion must replace intent atomically")
	read, err := readRepair(root, id)
	require.NoError(t, err, "completed receipt must be readable")
	assert.Equal(t, "completed_unverified", read.Status, "completion should be persisted")
	_, err = root.Stat(name + ".next")
	assert.True(t, os.IsNotExist(err), "completed atomic write should leave no pending record")
}

func TestReadRepairs(t *testing.T) {
	// The permanent receipt fixture directory is shared by filesystem tests.
	root, err := os.OpenRoot("testdata/repair")
	require.NoError(t, err, "fixture root must open")
	t.Cleanup(func() { _ = root.Close() })
	assert.Empty(t, readRepairs("testdata/repair", "").Records, "empty ledger should remain readable")
	oldest := time.Now().Add(-time.Hour).UTC()
	ids := make([]string, 0, 2)
	for i := range 2 {
		id := fmt.Sprintf("test-inspect-%d-%d", os.Getpid(), i)
		ids = append(ids, id)
		t.Cleanup(func() { _ = root.Remove("repair-" + id + ".json") })
		require.NoError(t, writeRepair(root, &repairRecord{Version: 1, Request: repairRequest{ID: id, Action: "refresh_pairs", Exchange: "okx", Reason: "stale_data"}, StartedAt: oldest.Add(time.Duration(i) * time.Minute), Status: "completed_unverified"}, true), "test receipts must persist")
	}
	result := readRepairs("testdata/repair", "")
	require.Empty(t, result.Failure, "valid ledger must be readable")
	require.Len(t, result.Records, 2, "all retained receipts must be returned")
	assert.Equal(t, ids[1], result.Records[0].Request.ID, "latest admission should sort first")
	assert.Empty(t, readRepairs("testdata/repair", ids[0]).Failure, "exact receipt should be readable offline")
	assert.Empty(t, readRepairs("testdata/repair", ids[0]).Failure, "internal admission evidence should not depend on the external response budget")
	assert.Equal(t, "repair_receipt_unavailable_or_invalid", readRepairs("testdata/repair", "missing-receipt").Failure, "missing receipt should remain unavailable")
	assert.Equal(t, "repair_directory_unavailable", readRepairs("testdata/missing", "").Failure, "unavailable directory should be explicit")
}

func TestBridgeInspectRepairs(t *testing.T) {
	// The permanent receipt fixture directory is shared by filesystem tests.
	root, err := os.OpenRoot("testdata/repair")
	require.NoError(t, err, "receipt fixture must open")
	t.Cleanup(func() { _ = root.Close() })
	for _, action := range []string{"refresh_pairs", "refresh_balances"} {
		id := fmt.Sprintf("privacy-%s-%d", action, os.Getpid())
		id = strings.ReplaceAll(id, "_", "-")
		request := repairRequest{ID: id, Action: action, Exchange: "public-exchange", Reason: "stale_data"}
		if action == "refresh_balances" {
			request.Exchange, request.Asset = "private-exchange", "spot"
		}
		require.NoError(t, writeRepair(root, &repairRecord{Version: 1, Request: request, StartedAt: time.Now().UTC(), Status: "completed_unverified"}, true), "privacy fixture must persist")
		t.Cleanup(func() { _ = root.Remove("repair-" + id + ".json") })
	}
	for _, tc := range []struct {
		name        string
		permissions config.MCPPermissions
		wantAction  string
	}{
		{name: "diagnostics sees operational receipts", permissions: config.MCPPermissions{Diagnostics: true}, wantAction: "refresh_pairs"},
		{name: "system control sees pair receipts", permissions: config.MCPPermissions{SystemControl: true}, wantAction: "refresh_pairs"},
		{name: "account data sees balance receipts", permissions: config.MCPPermissions{AccountData: true}, wantAction: "refresh_balances"},
		{name: "order management grants no receipt access", permissions: config.MCPPermissions{OrderManagement: true}},
		{name: "public grants no receipt access"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bridge := &Bridge{options: Options{Policy: config.MCPConfig{Enabled: true, Permissions: tc.permissions}, DiagnosticsDir: "testdata/repair"}}
			result := bridge.inspectRepairs("")
			if tc.wantAction == "" {
				assert.Equal(t, "repair_permission_denied", result.Failure, "ungranted receipt access should be denied")
				assert.Empty(t, result.Records, "denied access should reveal no receipts")
				return
			}
			require.Empty(t, result.Failure, "permitted receipt inspection must succeed")
			require.Len(t, result.Records, 1, "only the permitted action's receipt must be visible")
			assert.Equal(t, tc.wantAction, result.Records[0].Request.Action, "listing should filter private receipt metadata")
			if !tc.permissions.AccountData {
				encoded, err := json.Marshal(result)
				require.NoError(t, err, "filtered receipts must encode")
				assert.NotContains(t, string(encoded), "private-exchange", "listings should withhold private account targets")
				privateID := fmt.Sprintf("privacy-refresh-balances-%d", os.Getpid())
				private := bridge.inspectRepairs(privateID)
				assert.Equal(t, "repair_receipt_unavailable_or_invalid", private.Failure, "exact private IDs should reveal no account evidence")
				assert.Empty(t, private.Records, "exact private lookups should reveal no metadata")
			}
		})
	}
}

func TestBoundRepairResult(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, 4096} {
		t.Run(fmt.Sprintf("payload_%d", size), func(t *testing.T) {
			t.Parallel()
			result := &repairResult{Status: "historical", Caveat: "public", Records: []repairRecord{{Failure: strings.Repeat("x", size)}}}
			boundRepairResult(result, 4096)
			if size > 0 {
				assert.Equal(t, "repair_response_size_limit", result.Failure, "oversized receipts should be omitted")
			}
			data, err := json.Marshal(result)
			require.NoError(t, err, "bounded receipt must encode")
			wire, err := json.Marshal(&mcp.CallToolResult{StructuredContent: result, Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}})
			require.NoError(t, err, "wire result must encode")
			assert.LessOrEqual(t, len(wire), 4096, "both MCP representations should fit the wire budget")
		})
	}
}

func TestBridgeRepair(t *testing.T) {
	// The permanent receipt fixture directory is a singleton shared by filesystem tests.
	for _, tc := range []struct {
		name       string
		mode       string
		wantStatus string
		wantGuard  bool
		wantCalls  int32
	}{
		{name: "durable success and duplicate suppression", wantStatus: "completed_unverified", wantCalls: 1},
		{name: "small response limit does not limit ledger admission", mode: "populated ledger", wantStatus: "completed_unverified", wantCalls: 1},
		{name: "pair refresh does not require unrelated account access", mode: "private ledger", wantStatus: "completed_unverified", wantCalls: 1},
		{name: "private balance receipts enforce admission backoff", mode: "private ledger recent", wantStatus: "rejected"},
		{name: "unknown mutation retains execution guard", mode: "mutation error", wantStatus: "unknown", wantGuard: true, wantCalls: 1},
		{name: "rejected target releases execution guard", mode: "disabled exchange", wantStatus: "rejected"},
		{name: "completion persistence failure retains guard", mode: "completion failure", wantStatus: "unknown", wantGuard: true, wantCalls: 1},
		{name: "existing execution guard blocks admission", mode: "guard", wantStatus: "rejected", wantGuard: true},
		{name: "offline mode cannot execute", mode: "offline", wantStatus: "rejected"},
		{name: "missing account permission cannot execute", mode: "permission", wantStatus: "rejected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := fmt.Sprintf("test-admission-%d", os.Getpid())
			root, err := os.OpenRoot("testdata/repair")
			require.NoError(t, err, "fixture root must open")
			t.Cleanup(func() { _ = root.Close() })
			for _, name := range []string{"repair-" + id + ".json", "repair-" + id + ".json.next", "repair-" + id + "-new.json", "repair-active.json"} {
				t.Cleanup(func() { _ = root.Remove(name) })
			}
			var calls atomic.Int32
			listener := bufconn.Listen(1 << 20)
			server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
				method, _ := grpc.MethodFromServerStream(stream)
				if method == "/gctrpc.GoCryptoTraderService/GetExchangeInfo" {
					if err := stream.RecvMsg(new(gctrpc.GenericExchangeNameRequest)); err != nil {
						return err
					}
					return stream.SendMsg(&gctrpc.GetExchangeInfoResponse{Enabled: tc.mode != "disabled exchange", SupportedAssets: map[string]*gctrpc.PairsSupported{"spot": {EnabledPairs: "BTC-USDT"}}})
				}
				if method == "/gctrpc.GoCryptoTraderService/UpdateAccountBalances" {
					if err := stream.RecvMsg(new(gctrpc.GetAccountBalancesRequest)); err != nil {
						return err
					}
					calls.Add(1)
					if tc.mode == "mutation error" {
						return status.Error(codes.Unavailable, "private-error")
					}
					return stream.SendMsg(new(gctrpc.GetAccountBalancesResponse))
				}
				if method == "/gctrpc.GoCryptoTraderService/UpdateExchangeSupportedPairs" {
					if err := stream.RecvMsg(new(gctrpc.UpdateExchangeSupportedPairsRequest)); err != nil {
						return err
					}
					calls.Add(1)
					return stream.SendMsg(new(gctrpc.GenericResponse))
				}
				return status.Error(codes.Unimplemented, "unexpected RPC")
			}))
			t.Cleanup(server.Stop)
			t.Cleanup(func() { _ = listener.Close() })
			go func() { _ = server.Serve(listener) }()
			conn, err := grpc.NewClient("passthrough:///admission", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDisableRetry(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
			require.NoError(t, err, "isolated RPC client must initialise")
			t.Cleanup(func() { _ = conn.Close() })
			bridge := &Bridge{conn: conn, options: Options{Policy: testExecutionPolicy, DiagnosticsDir: "testdata/repair", Timeout: time.Second, MaxResponseBytes: 1 << 20, Offline: tc.mode == "offline"}}
			if tc.mode == "permission" {
				bridge.options.Policy.Permissions.AccountData = false
			}
			request := repairRequest{ID: id, Action: "refresh_balances", Exchange: "binance", Asset: "spot", Reason: "retrieval_failed"}
			if strings.HasPrefix(tc.mode, "private ledger") {
				bridge.options.Policy.Permissions = config.MCPPermissions{SystemControl: true, OrderManagement: true}
				request.Action, request.Asset = "refresh_pairs", ""
				oldID := fmt.Sprintf("test-private-%d", os.Getpid())
				t.Cleanup(func() { _ = root.Remove("repair-" + oldID + ".json") })
				old := &repairRecord{Version: 1, Request: repairRequest{ID: oldID, Action: "refresh_balances", Exchange: "private-exchange", Asset: "spot", Reason: "stale_data"}, StartedAt: time.Now().Add(-time.Hour).UTC(), Status: "completed_unverified"}
				if tc.mode == "private ledger recent" {
					old.StartedAt = time.Now().UTC()
				}
				require.NoError(t, writeRepair(root, old, true), "private ledger evidence must persist")
			}
			if tc.mode == "populated ledger" {
				bridge.options.MaxResponseBytes = 4096
				for i := range 8 {
					oldID := fmt.Sprintf("test-old-%d-%d", os.Getpid(), i)
					t.Cleanup(func() { _ = root.Remove("repair-" + oldID + ".json") })
					old := &repairRecord{Version: 1, Request: repairRequest{ID: oldID, Action: "refresh_pairs", Exchange: "okx", Reason: "stale_data"}, StartedAt: time.Now().Add(-time.Hour).UTC(), Status: "completed_unverified"}
					require.NoError(t, writeRepair(root, old, true), "older valid receipts must be persisted")
				}
			}
			if tc.mode == "guard" || tc.mode == "completion failure" {
				name := "repair-active.json"
				if tc.mode == "completion failure" {
					name = "repair-" + id + ".json.next"
				}
				file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
				require.NoError(t, err, "failure fixture must be created")
				require.NoError(t, file.Close(), "failure fixture must close")
			}
			result := bridge.repair(t.Context(), &request)
			assert.Equal(t, tc.wantStatus, result.Status, "admission should preserve known rejection and uncertain execution")
			assert.Equal(t, tc.wantCalls, calls.Load(), "mutation should execute at most once")
			if strings.HasPrefix(tc.mode, "private ledger") {
				encoded, err := json.Marshal(result)
				require.NoError(t, err, "refresh result must encode")
				assert.NotContains(t, string(encoded), "private-exchange", "admission evidence should not expose unrelated account metadata")
				if tc.mode == "private ledger recent" {
					assert.Equal(t, "repair_admission_backoff", result.Failure, "hidden receipts should still enforce admission timing")
				}
			}
			_, err = root.Stat("repair-active.json")
			if tc.wantGuard {
				assert.NoError(t, err, "uncertain execution should keep its durable guard")
			} else {
				assert.True(t, os.IsNotExist(err), "finished or rejected execution should not retain a guard")
			}
			if tc.wantCalls > 0 || tc.mode == "disabled exchange" {
				duplicate := bridge.repair(t.Context(), &request)
				assert.Equal(t, "historical", duplicate.Status, "duplicate request should return its existing receipt")
				assert.Equal(t, tc.wantCalls, calls.Load(), "duplicate receipt should prevent repeating RPC execution")
				changed := request
				changed.Reason = "stale_data"
				assert.Equal(t, "repair_id_parameter_conflict", bridge.repair(t.Context(), &changed).Failure, "same ID with changed parameters should be rejected")
			}
			if tc.mode == "" {
				request.ID += "-new"
				assert.Equal(t, "repair_admission_backoff", bridge.repair(t.Context(), &request).Failure, "new admissions should respect the minute backoff")
				assert.Equal(t, int32(1), calls.Load(), "backoff should not execute another mutation")
			}
		})
	}
	t.Run("concurrent admissions do not overlap and retain backoff", func(t *testing.T) {
		// The permanent receipt fixture directory is a singleton shared by filesystem tests.
		root, err := os.OpenRoot("testdata/repair")
		require.NoError(t, err, "fixture root must open")
		t.Cleanup(func() { _ = root.Close() })
		t.Cleanup(func() { _ = root.Remove("repair-active.json") })
		for i := range 18 {
			id := fmt.Sprintf("test-concurrent-%d-%d", os.Getpid(), i)
			t.Cleanup(func() {
				_ = root.Remove("repair-" + id + ".json")
				_ = root.Remove("repair-" + id + ".json.next")
			})
		}
		var calls atomic.Int32
		entered := make(chan struct{})
		release := make(chan struct{})
		listener := bufconn.Listen(1 << 20)
		server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
			method, _ := grpc.MethodFromServerStream(stream)
			if method == "/gctrpc.GoCryptoTraderService/GetExchangeInfo" {
				if err := stream.RecvMsg(new(gctrpc.GenericExchangeNameRequest)); err != nil {
					return err
				}
				return stream.SendMsg(&gctrpc.GetExchangeInfoResponse{Enabled: true, SupportedAssets: map[string]*gctrpc.PairsSupported{"spot": {}}})
			}
			if method != "/gctrpc.GoCryptoTraderService/UpdateAccountBalances" {
				return status.Error(codes.Unimplemented, "unexpected RPC")
			}
			if err := stream.RecvMsg(new(gctrpc.GetAccountBalancesRequest)); err != nil {
				return err
			}
			if calls.Add(1) == 1 {
				close(entered)
			}
			select {
			case <-release:
				return stream.SendMsg(new(gctrpc.GetAccountBalancesResponse))
			case <-stream.Context().Done():
				return stream.Context().Err()
			}
		}))
		t.Cleanup(server.Stop)
		t.Cleanup(func() { _ = listener.Close() })
		go func() { _ = server.Serve(listener) }()
		conn, err := grpc.NewClient("passthrough:///concurrent-repair", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDisableRetry(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
		require.NoError(t, err, "isolated concurrent RPC client must initialise")
		t.Cleanup(func() { _ = conn.Close() })
		bridge := &Bridge{conn: conn, options: Options{Policy: testExecutionPolicy, DiagnosticsDir: "testdata/repair", Timeout: 5 * time.Second, MaxResponseBytes: 1 << 20}}
		request := repairRequest{ID: fmt.Sprintf("test-concurrent-%d-0", os.Getpid()), Action: "refresh_balances", Exchange: "binance", Asset: "spot", Reason: "stale_data"}
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		first := make(chan *repairResult, 1)
		go func() { first <- bridge.repair(ctx, &request) }()
		select {
		case <-entered:
		case <-ctx.Done():
			require.FailNow(t, "the first repair must enter its bounded mutation")
		}
		results := make(chan *repairResult, 16)
		for i := range 16 {
			competing := request
			competing.ID = fmt.Sprintf("test-concurrent-%d-%d", os.Getpid(), i+1)
			go func() { results <- bridge.repair(ctx, &competing) }()
		}
		for range 16 {
			select {
			case result := <-results:
				assert.Contains(t, []string{"repair_execution_guard_exists", "repair_admission_backoff"}, result.Failure, "competing admissions should be rejected by ownership or backoff")
			case <-ctx.Done():
				require.FailNow(t, "all competing admissions must finish before the deadline")
			}
		}
		assert.Equal(t, int32(1), calls.Load(), "competing repairs should never overlap execution")
		close(release)
		select {
		case result := <-first:
			assert.Equal(t, "completed_unverified", result.Status, "the owner should finish its admitted operation")
		case <-ctx.Done():
			require.FailNow(t, "the admitted owner must finish after its RPC is released")
		}
		request.ID = fmt.Sprintf("test-concurrent-%d-17", os.Getpid())
		assert.Equal(t, "repair_admission_backoff", bridge.repair(ctx, &request).Failure, "post-completion admission should reread the current ledger")
		assert.Equal(t, int32(1), calls.Load(), "post-completion backoff should prevent a second mutation")
	})
}

func TestBridgeExecuteRepair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		action     string
		mode       string
		wantStatus string
		wantCalls  int32
	}{
		{name: "disabled exchange rejected", action: "refresh_balances", mode: "disabled exchange", wantStatus: "rejected"},
		{name: "unsupported asset rejected", action: "refresh_balances", mode: "unsupported asset", wantStatus: "rejected"},
		{name: "balance refresh does not claim verification", action: "refresh_balances", wantStatus: "completed_unverified", wantCalls: 1},
		{name: "pair refresh does not claim verification", action: "refresh_pairs", wantStatus: "completed_unverified", wantCalls: 1},
		{name: "mutation failure has unknown outcome", action: "refresh_balances", mode: "mutation error", wantStatus: "unknown", wantCalls: 1},
		{name: "read failure before mutation is rejected", action: "refresh_pairs", mode: "info error", wantStatus: "rejected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			listener := bufconn.Listen(1 << 20)
			server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
				method, _ := grpc.MethodFromServerStream(stream)
				switch method {
				case "/gctrpc.GoCryptoTraderService/GetExchangeInfo":
					if err := stream.RecvMsg(new(gctrpc.GenericExchangeNameRequest)); err != nil {
						return err
					}
					if tc.mode == "info error" {
						return status.Error(codes.Unavailable, "private-error")
					}
					assets := map[string]*gctrpc.PairsSupported{"spot": {}}
					if tc.mode == "unsupported asset" {
						assets = nil
					}
					return stream.SendMsg(&gctrpc.GetExchangeInfoResponse{Enabled: tc.mode != "disabled exchange", SupportedAssets: assets})
				case "/gctrpc.GoCryptoTraderService/UpdateAccountBalances":
					if err := stream.RecvMsg(new(gctrpc.GetAccountBalancesRequest)); err != nil {
						return err
					}
					calls.Add(1)
					if tc.mode == "mutation error" {
						return status.Error(codes.Unavailable, "private-error")
					}
					return stream.SendMsg(new(gctrpc.GetAccountBalancesResponse))
				case "/gctrpc.GoCryptoTraderService/UpdateExchangeSupportedPairs":
					if err := stream.RecvMsg(new(gctrpc.UpdateExchangeSupportedPairsRequest)); err != nil {
						return err
					}
					calls.Add(1)
					return stream.SendMsg(new(gctrpc.GenericResponse))
				default:
					return status.Error(codes.Unimplemented, "unexpected RPC")
				}
			}))
			t.Cleanup(server.Stop)
			t.Cleanup(func() { _ = listener.Close() })
			go func() { _ = server.Serve(listener) }()
			conn, err := grpc.NewClient("passthrough:///repair", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDisableRetry(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
			require.NoError(t, err, "isolated RPC client must initialise")
			t.Cleanup(func() { _ = conn.Close() })
			bridge := &Bridge{conn: conn, options: Options{Policy: testExecutionPolicy, MaxResponseBytes: 1 << 20}}
			request := repairRequest{ID: "test-execute-repair", Action: tc.action, Exchange: "binance", Asset: "spot", Reason: "stale_data"}
			if tc.action == "refresh_pairs" {
				request.Asset = ""
			}
			record := &repairRecord{Request: request}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			bridge.executeRepair(ctx, record)
			assert.Equal(t, tc.wantStatus, record.Status, "repair status should reflect execution without claiming verification")
			assert.Equal(t, tc.wantCalls, calls.Load(), "only the fixed necessary RPCs should execute")
			encoded, err := json.Marshal(record)
			require.NoError(t, err, "safe receipt must encode")
			assert.NotContains(t, string(encoded), "private-error", "raw RPC details should not persist")
		})
	}
}
