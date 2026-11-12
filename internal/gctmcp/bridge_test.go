package gctmcp

import (
	"context"
	"net"
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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestNew(t *testing.T) {
	t.Parallel()
	conn, err := grpc.NewClient("passthrough:///unused", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "test connection must configure")
	t.Cleanup(func() { _ = conn.Close() })
	for _, tc := range []struct {
		name     string
		conn     grpc.ClientConnInterface
		timeout  time.Duration
		messages int
		maximum  int
		policy   config.MCPConfig
		offline  bool
		invalid  bool
	}{
		{name: "valid", conn: conn, timeout: time.Second, messages: 1, maximum: 4096, policy: testExecutionPolicy},
		{name: "nil connection", timeout: time.Second, messages: 1, maximum: 4096, policy: testExecutionPolicy, invalid: true},
		{name: "typed nil connection", conn: (*grpc.ClientConn)(nil), timeout: time.Second, messages: 1, maximum: 4096, policy: testExecutionPolicy, invalid: true},
		{name: "zero timeout", conn: conn, messages: 1, maximum: 4096, policy: testExecutionPolicy, invalid: true},
		{name: "long timeout", conn: conn, timeout: time.Minute + 1, messages: 1, maximum: 4096, policy: testExecutionPolicy, invalid: true},
		{name: "zero messages", conn: conn, timeout: time.Second, maximum: 4096, policy: testExecutionPolicy, invalid: true},
		{name: "excessive messages", conn: conn, timeout: time.Second, messages: 1001, maximum: 4096, policy: testExecutionPolicy, invalid: true},
		{name: "small response", conn: conn, timeout: time.Second, messages: 1, maximum: 4095, policy: testExecutionPolicy, invalid: true},
		{name: "large response", conn: conn, timeout: time.Second, messages: 1, maximum: 4<<20 + 1, policy: testExecutionPolicy, invalid: true},
		{name: "disabled", offline: true, timeout: time.Second, messages: 1, maximum: 4096, invalid: true},
		{name: "invalid mode", offline: true, timeout: time.Second, messages: 1, maximum: 4096, policy: config.MCPConfig{Enabled: true, Mode: "future"}, invalid: true},
		{name: "readonly default", offline: true, timeout: time.Second, messages: 1, maximum: 4096, policy: config.MCPConfig{Enabled: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			options := &Options{Policy: tc.policy, Offline: tc.offline, Timeout: tc.timeout, MaxMessages: tc.messages, MaxResponseBytes: tc.maximum}
			bridge, err := New(tc.conn, options)
			if tc.invalid {
				require.ErrorIs(t, err, errInvalidOptions, "invalid settings must fail")
				assert.Nil(t, bridge, "invalid settings should not create a bridge")
				return
			}
			require.NoError(t, err, "valid settings must initialise")
			if tc.policy.Mode == "" {
				assert.Equal(t, config.MCPReadOnly, bridge.options.Policy.Mode, "omitted mode should resolve readonly")
			}
			options.Policy.Permissions.AccountData = !options.Policy.Permissions.AccountData
			assert.Equal(t, tc.policy.Permissions.AccountData, bridge.options.Policy.Permissions.AccountData, "the bridge should retain an immutable configuration copy")
		})
	}
	t.Run("nil options", func(t *testing.T) {
		t.Parallel()
		_, err := New(conn, nil)
		assert.ErrorIs(t, err, errInvalidOptions, "missing options should fail")
	})
}

func TestBridgeServer(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		permissions config.MCPPermissions
		execution   bool
		visible     []string
		hidden      []string
	}{
		{name: "public readonly", visible: []string{"gct_GetTicker", "gct_GetHistoricCandles", "gct_GetHistoricTrades"}, hidden: []string{"gct_GetAccountBalances", "gct_GetOrders", "gct_SubmitOrder", "gct_WithdrawCryptocurrencyFunds", "gct_Shutdown"}},
		{name: "account readonly", permissions: config.MCPPermissions{AccountData: true}, visible: []string{"gct_GetAccountBalances", "gct_GetOrders", "gct_GetFuturesPositionsOrders"}, hidden: []string{"gct_UpdateAccountBalances", "gct_SubmitOrder", "gct_GetOrder"}},
		{name: "order execution independent of account data", permissions: config.MCPPermissions{OrderManagement: true}, execution: true, visible: []string{"gct_SubmitOrder", "gct_CancelOrder"}, hidden: []string{"gct_GetOrders", "gct_GetAccountBalances", "gct_WithdrawCryptocurrencyFunds"}},
		{name: "withdrawals independent of orders", permissions: config.MCPPermissions{Withdrawals: true}, execution: true, visible: []string{"gct_WithdrawCryptocurrencyFunds"}, hidden: []string{"gct_SubmitOrder", "gct_GetOrders", "gct_GetAccountBalances"}},
		{name: "system control cannot cancel orders", permissions: config.MCPPermissions{SystemControl: true}, execution: true, visible: []string{"gct_EnableSubsystem", "gct_SetLoggerDetails"}, hidden: []string{"gct_Shutdown", "gct_SubmitOrder"}},
		{name: "all execution", permissions: testExecutionPolicy.Permissions, execution: true, visible: []string{"gct_Shutdown", "gct_UpdateAccountBalances", "gct_SubmitOrder", "gct_GetOrder"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			t.Cleanup(cancel)
			conn, err := grpc.NewClient("passthrough:///unused", grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err, "connection must configure")
			t.Cleanup(func() { _ = conn.Close() })
			mode := config.MCPReadOnly
			if tc.execution {
				mode = config.MCPExecution
			}
			bridge, err := New(conn, &Options{Policy: config.MCPConfig{Enabled: true, Mode: mode, Permissions: tc.permissions}, Timeout: time.Second, MaxMessages: 1, MaxResponseBytes: 4096})
			require.NoError(t, err, "policy must initialise")
			st, ct := mcp.NewInMemoryTransports()
			server, err := bridge.Server().Connect(ctx, st, nil)
			require.NoError(t, err, "server must connect")
			t.Cleanup(func() { _ = server.Close() })
			client, err := mcp.NewClient(&mcp.Implementation{Name: "policy-test", Version: "1"}, nil).Connect(ctx, ct, nil)
			require.NoError(t, err, "client must connect")
			t.Cleanup(func() { _ = client.Close() })
			seen := map[string]*mcp.Tool{}
			for tool, err := range client.Tools(ctx, nil) {
				require.NoError(t, err, "tools must list")
				seen[tool.Name] = tool
			}
			for _, name := range tc.visible {
				assert.Contains(t, seen, name, "permitted tools should be advertised")
			}
			for _, name := range tc.hidden {
				assert.NotContains(t, seen, name, "ungranted tools should remain hidden")
			}
			for _, name := range []string{"gct_WebsocketSetURL", "gct_WebsocketSetProxy"} {
				assert.NotContains(t, seen, name, "connection destinations should stay operator managed even with all permissions enabled")
			}
			for name := range excludedMethods {
				assert.NotContains(t, seen, "gct_"+name, "excluded tools should remain unavailable")
			}
			for name, tool := range seen {
				policy, ok := methodPolicies[strings.TrimPrefix(name, "gct_")]
				if !ok {
					continue
				}
				readOnly := !policy.execution
				if name == "gct_GetFuturesPositionsOrders" && tc.execution && tc.permissions.OrderManagement {
					readOnly = false
				}
				assert.Equal(t, readOnly, tool.Annotations.ReadOnlyHint, "hints should reflect effective mutation authority")
			}
		})
	}
}

func TestBridgeCall(t *testing.T) {
	t.Parallel()
	t.Run("indirect cancellation denied before RPC invocation", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ method, request string }{
			{method: "Shutdown", request: `{}`},
			{method: "DisableExchange", request: `{}`},
			{method: "WebsocketSetEnabled", request: `{"enable":false}`},
			{method: "DisableSubsystem", request: `{"subsystem":"orders"}`},
			{method: "EnableSubsystem", request: `{"subsystem":"ORDERS"}`},
			{method: "SetExchangePair", request: `{}`},
			{method: "SetExchangeAsset", request: `{}`},
			{method: "SetAllExchangePairs", request: `{}`},
			{method: "UpdateExchangeSupportedPairs", request: `{}`},
		} {
			t.Run(tc.method, func(t *testing.T) {
				t.Parallel()
				bridge := &Bridge{options: Options{Policy: config.MCPConfig{Enabled: true, Mode: config.MCPExecution, Permissions: config.MCPPermissions{SystemControl: true}}, Timeout: time.Second, MaxMessages: 1, MaxResponseBytes: 4096}}
				method := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods().ByName(protoreflect.Name(tc.method))
				response := bridge.call(t.Context(), method, json.RawMessage(`{"request":`+tc.request+`}`))
				require.True(t, response.IsError, "missing cancellation authority must fail before using the absent RPC client")
				result, ok := response.StructuredContent.(*callResult)
				require.True(t, ok, "permission failure must remain structured")
				assert.Contains(t, result.Failure, errPermissionDenied.Error(), "indirect order effects should require order management")
			})
		}
	})
	t.Run("restricted commands rejected before RPC invocation", func(t *testing.T) {
		t.Parallel()
		bridge := &Bridge{options: Options{Policy: testExecutionPolicy, Timeout: time.Second, MaxMessages: 1, MaxResponseBytes: 4096}}
		methods := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods()
		for name := range excludedMethods {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				response := bridge.call(t.Context(), methods.ByName(protoreflect.Name(name)), json.RawMessage(`{"request":{}}`))
				require.True(t, response.IsError, "restricted commands must fail before using the absent RPC connection")
				result, ok := response.StructuredContent.(*callResult)
				require.True(t, ok, "rejection must retain a structured result")
				assert.Equal(t, errMethodUnavailable.Error(), result.Failure, "excluded commands should fail closed before dispatch")
			})
		}
		for _, input := range []string{`{"useDb":true}`, `{"use_db":true}`, `{"sync":true}`, `{"fillMissingWithTrades":true}`, `{"fill_missing_with_trades":true}`, `{"force":true}`} {
			t.Run(input, func(t *testing.T) {
				t.Parallel()
				response := bridge.call(t.Context(), methods.ByName("GetHistoricCandles"), json.RawMessage(`{"request":`+input+`}`))
				require.True(t, response.IsError, "candle database options must fail before using the absent RPC connection")
				result, ok := response.StructuredContent.(*callResult)
				require.True(t, ok, "rejection must retain a structured result")
				assert.Contains(t, result.Failure, errDatabaseHistory.Error(), "database options should remain unavailable in execution mode")
			})
		}
	})
	t.Run("configuration denial happens before any RPC invocation", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			method    string
			arguments string
		}{
			{method: "GetAccountBalances", arguments: `{"request":{}}`},
			{method: "GetOrders", arguments: `{"request":{}}`},
			{method: "WithdrawalEventByID", arguments: `{"request":{}}`},
			{method: "SubmitOrder", arguments: `{"request":{}}`},
			{method: "GetFundingRates", arguments: `{"request":{"pair":{},"includePayments":true}}`},
		} {
			t.Run(tc.method, func(t *testing.T) {
				t.Parallel()
				bridge := &Bridge{options: Options{Policy: config.MCPConfig{Enabled: true, Mode: config.MCPReadOnly, Permissions: config.MCPPermissions{}}, Timeout: time.Second, MaxMessages: 1, MaxResponseBytes: 4096}}
				method := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods().ByName(protoreflect.Name(tc.method))
				response := bridge.call(t.Context(), method, json.RawMessage(tc.arguments))
				require.True(t, response.IsError, "ungranted private or execution requests must be rejected before using the absent RPC client")
				result, ok := response.StructuredContent.(*callResult)
				require.True(t, ok, "rejection must retain a structured result")
				assert.Equal(t, "rejected", result.Status, "permission denial should be explicit")
			})
		}
	})
	for _, tc := range []struct {
		name          string
		method        string
		arguments     string
		execution     bool
		mode          string
		wantStatus    string
		wantCount     int
		wantCalls     int32
		wantError     bool
		wantTruncated bool
		wantFailure   error
	}{
		{name: "unary success", method: "GetInfo", arguments: `{"request":{}}`, wantStatus: "complete", wantCount: 1, wantCalls: 1},
		{name: "invalid JSON", method: "GetInfo", arguments: `{`, wantStatus: "rejected", wantError: true},
		{name: "trailing JSON", method: "GetInfo", arguments: `{"request":{}} {}`, wantStatus: "rejected", wantError: true},
		{name: "missing request", method: "GetInfo", arguments: `{}`, wantStatus: "rejected", wantError: true},
		{name: "null request", method: "GetInfo", arguments: `{"request":null}`, wantStatus: "rejected", wantError: true},
		{name: "unknown envelope field", method: "GetInfo", arguments: `{"request":{},"secret":"private"}`, wantStatus: "rejected", wantError: true},
		{name: "unknown protobuf field", method: "GetInfo", arguments: `{"request":{"secret":"private"}}`, wantStatus: "rejected", wantError: true},
		{name: "unary stream limit", method: "GetInfo", arguments: `{"request":{},"maxMessages":1}`, wantStatus: "rejected", wantError: true},
		{name: "negative stream limit", method: "GetTickerStream", arguments: `{"request":{},"maxMessages":-1}`, wantStatus: "rejected", wantError: true},
		{name: "operator stream limit", method: "GetTickerStream", arguments: `{"request":{},"maxMessages":3}`, wantStatus: "rejected", wantError: true},
		{name: "readonly order execution denied", method: "SubmitOrder", arguments: `{"request":{}}`, wantStatus: "rejected", wantError: true},
		{name: "order lookup requires execution", method: "GetOrder", arguments: `{"request":{"exchange":"coinbase","orderId":"test-order","pair":{"base":"BTC","quote":"USD"},"asset":"spot"}}`, wantStatus: "rejected", wantError: true, wantFailure: errPermissionDenied},
		{name: "order lookup with management authority", method: "GetOrder", arguments: `{"request":{"exchange":"coinbase","orderId":"test-order","pair":{"base":"BTC","quote":"USD"},"asset":"spot"}}`, execution: true, wantStatus: "complete", wantCount: 1, wantCalls: 1},
		{name: "websocket URL is operator managed", method: "WebsocketSetURL", arguments: `{"request":{"exchange":"coinbase","url":"wss://unapproved.example/socket"}}`, execution: true, wantStatus: "rejected", wantError: true, wantFailure: errMethodUnavailable},
		{name: "plaintext websocket URL is operator managed", method: "WebsocketSetURL", arguments: `{"request":{"exchange":"coinbase","url":"ws://unapproved.example/socket"}}`, execution: true, wantStatus: "rejected", wantError: true, wantFailure: errMethodUnavailable},
		{name: "default websocket URL is operator managed", method: "WebsocketSetURL", arguments: `{"request":{"exchange":"coinbase","url":""}}`, execution: true, wantStatus: "rejected", wantError: true, wantFailure: errMethodUnavailable},
		{name: "websocket proxy is operator managed", method: "WebsocketSetProxy", arguments: `{"request":{"exchange":"coinbase","proxy":"http://unapproved.example:8080"}}`, execution: true, wantStatus: "rejected", wantError: true, wantFailure: errMethodUnavailable},
		{name: "SOCKS websocket proxy is operator managed", method: "WebsocketSetProxy", arguments: `{"request":{"exchange":"coinbase","proxy":"socks5://unapproved.example:1080"}}`, execution: true, wantStatus: "rejected", wantError: true, wantFailure: errMethodUnavailable},
		{name: "default websocket proxy is operator managed", method: "WebsocketSetProxy", arguments: `{"request":{"exchange":"coinbase","proxy":""}}`, execution: true, wantStatus: "rejected", wantError: true, wantFailure: errMethodUnavailable},
		{name: "sync snake alias", method: "GetFuturesPositionsOrders", arguments: `{"request":{"sync_with_order_manager":true}}`, wantStatus: "rejected", wantError: true},
		{name: "write success", method: "SubmitOrder", arguments: `{"request":{"pair":{"base":"BTC","quote":"USDT"}}}`, execution: true, wantStatus: "complete", wantCount: 1, wantCalls: 1},
		{name: "uncertain write no retry", method: "SubmitOrder", arguments: `{"request":{"pair":{"base":"BTC","quote":"USDT"}}}`, execution: true, mode: "error", wantStatus: "unknown", wantError: true, wantCalls: 1},
		{name: "server error is withheld", method: "GetInfo", arguments: `{"request":{}}`, mode: "error", wantStatus: "unknown", wantError: true, wantCalls: 1},
		{name: "deadline no response", method: "GetInfo", arguments: `{"request":{}}`, mode: "wait", wantStatus: "unknown", wantError: true, wantCalls: 1},
		{name: "finite stream EOF", method: "GetOrderbookStream", arguments: `{"request":{"pair":{"base":"BTC","quote":"USDT"}}}`, mode: "finite", wantStatus: "complete", wantCount: 1, wantCalls: 1},
		{name: "stream sample bounded", method: "GetOrderbookStream", arguments: `{"request":{"pair":{"base":"BTC","quote":"USDT"}},"maxMessages":1}`, mode: "stream", wantStatus: "partial", wantCount: 1, wantCalls: 1, wantTruncated: true},
		{name: "default stream sample bounded", method: "GetOrderbookStream", arguments: `{"request":{"pair":{"base":"BTC","quote":"USDT"}}}`, mode: "stream", wantStatus: "partial", wantCount: 2, wantCalls: 1, wantTruncated: true},
		{name: "stream preserves prior success", method: "GetOrderbookStream", arguments: `{"request":{"pair":{"base":"BTC","quote":"USDT"}}}`, mode: "stream error", wantStatus: "partial", wantCount: 1, wantCalls: 1, wantError: true},
		{name: "in-message error and subsequent success", method: "GetOrderbookStream", arguments: `{"request":{"pair":{"base":"BTC","quote":"USDT"}}}`, mode: "application error", wantStatus: "partial", wantCount: 2, wantCalls: 1, wantError: true, wantTruncated: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			method := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods().ByName(protoreflect.Name(tc.method))
			listener := bufconn.Listen(1 << 20)
			server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
				calls.Add(1)
				path, ok := grpc.MethodFromServerStream(stream)
				if !ok || path != "/gctrpc.GoCryptoTraderService/"+tc.method {
					return status.Error(codes.Unimplemented, "unexpected RPC path")
				}
				request := dynamicpb.NewMessage(method.Input())
				if err := stream.RecvMsg(request); err != nil {
					return err
				}
				if tc.mode == "error" {
					return status.Error(codes.Unavailable, "private-secret")
				}
				if tc.mode == "wait" {
					<-stream.Context().Done()
					return stream.Context().Err()
				}
				count := 1
				if tc.mode == "stream" || tc.mode == "application error" {
					count = 3
				}
				for i := range count {
					response := dynamicpb.NewMessage(method.Output())
					if tc.mode == "application error" && i == 0 {
						response.Set(response.Descriptor().Fields().ByName("error"), protoreflect.ValueOfString("private-secret"))
					}
					if err := stream.SendMsg(response); err != nil {
						return err
					}
				}
				if tc.mode == "stream error" {
					return status.Error(codes.Internal, "private-secret")
				}
				return nil
			}))
			t.Cleanup(server.Stop)
			t.Cleanup(func() { _ = listener.Close() })
			go func() { _ = server.Serve(listener) }()
			conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDisableRetry(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
			require.NoError(t, err, "test client must initialise")
			t.Cleanup(func() { _ = conn.Close() })
			policy := testExecutionPolicy
			if !tc.execution {
				policy.Mode = config.MCPReadOnly
			}
			bridge, err := New(conn, &Options{Policy: policy, Timeout: 100 * time.Millisecond, MaxMessages: 2, MaxResponseBytes: 4096})
			require.NoError(t, err, "test bridge must initialise")
			response := bridge.call(t.Context(), method, json.RawMessage(tc.arguments))
			result, ok := response.StructuredContent.(*callResult)
			require.True(t, ok, "structured result must retain its type")
			assert.Equal(t, tc.wantStatus, result.Status, "outcome should describe observed completion")
			assert.Len(t, result.Messages, tc.wantCount, "received messages should be preserved")
			assert.Equal(t, tc.wantError, response.IsError, "tool error should reflect RPC or policy failure")
			assert.Equal(t, tc.wantTruncated, result.Truncated, "sample limits should be explicit")
			if tc.wantFailure != nil {
				assert.Contains(t, result.Failure, tc.wantFailure.Error(), "rejection should identify the intended permission boundary")
			}
			assert.Equal(t, tc.wantCalls, calls.Load(), "RPC should execute at most once after validation")
			encoded, err := json.Marshal(response)
			require.NoError(t, err, "result must encode")
			assert.NotContains(t, string(encoded), "private-secret", "raw error detail should be withheld")
		})
	}
}

func TestBridgeAppendResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                 string
		method               string
		response             proto.Message
		existing             []json.RawMessage
		want                 bool
		wantCount            int
		wantApplicationError bool
		wantJSON             string
	}{
		{name: "small", response: &gctrpc.GenericResponse{Status: "success"}, want: true, wantCount: 1},
		{name: "within wire budget", response: &gctrpc.GenericResponse{Data: strings.Repeat("a", 1000)}, want: true, wantCount: 1},
		{name: "over wire budget", response: &gctrpc.GenericResponse{Data: strings.Repeat("a", 4000)}},
		{name: "aggregate budget", response: &gctrpc.GenericResponse{Data: strings.Repeat("a", 1030)}, existing: []json.RawMessage{json.RawMessage(`"` + strings.Repeat("a", 1020) + `"`)}, wantCount: 1},
		{
			name: "partial batch cancellation", method: "CancelBatchOrders", want: true, wantCount: 1, wantApplicationError: true,
			response: &gctrpc.CancelBatchOrdersResponse{Orders: []*gctrpc.Orders{{Exchange: "btcmarkets", OrderStatus: map[string]string{"cancelled-id": "success", "failed-id": "ErrorCode - private-secret"}}}},
			wantJSON: `{"orders":[{"exchange":"btcmarkets","order_status":{"cancelled-id":"cancelled","failed-id":"cancellation_unconfirmed"}}]}`,
		},
		{
			name: "partial cancel all", method: "CancelAllOrders", want: true, wantCount: 1, wantApplicationError: true,
			response: &gctrpc.CancelAllOrdersResponse{Orders: []*gctrpc.Orders{{Exchange: "yobit", OrderStatus: map[string]string{"cancelled-id": "Cancelled", "failed-id": "private-secret"}}}, Count: 2},
			wantJSON: `{"orders":[{"exchange":"yobit","order_status":{"cancelled-id":"cancelled","failed-id":"cancellation_unconfirmed"}}],"count":"2"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			response := dynamicpb.NewMessage(tc.response.ProtoReflect().Descriptor())
			proto.Merge(response, tc.response)
			bridge := &Bridge{options: Options{Policy: testExecutionPolicy, MaxResponseBytes: 4096}}
			result := callResult{Method: tc.method, Messages: tc.existing}
			assert.Equal(t, tc.want, bridge.appendResponse(&result, response), "append should respect aggregate budget")
			assert.Len(t, result.Messages, tc.wantCount, "only complete fitting messages should be retained")
			assert.Equal(t, tc.wantApplicationError, result.ApplicationError, "unconfirmed cancellation should mark an application error")
			if tc.wantJSON != "" {
				require.Len(t, result.Messages, 1, "cancellation result must retain its per-order observations")
				assert.JSONEq(t, tc.wantJSON, string(result.Messages[0]), "cancellation IDs and confirmed outcomes should be retained without raw error details")
				assert.True(t, encodeResult(&result).IsError, "partial cancellation should be an MCP tool error")
			}
			encoded, err := json.Marshal(encodeResult(&result))
			require.NoError(t, err, "result must encode")
			assert.LessOrEqual(t, len(encoded), 4096, "entire MCP response should fit the wire budget")
			assert.NotContains(t, string(encoded), "private-secret", "RPC results should not expose credentials or raw error detail")
		})
	}
}

func TestSanitise(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{
			name:      "nested credentials diagnostic messages and OTP redaction",
			input:     `{"forex":[{"api_key":"private","enabled":true}],"http_proxy":"private","proxy_address":"private","running_url":"private","params":"private","error":"private","message":"private","logger":{"error":true},"otp_code":"123456"}`,
			want:      `{"forex":[{"enabled":true}],"error":"RPC application error; detail withheld","message":"RPC message withheld","logger":{"error":true}}`,
			wantError: true,
		},
		{
			name:  "known cancellation success meanings",
			input: `{"orders":[{"order_status":{"a":"CANCELLED","b":"canceled","c":"Success","d":"OK","e":"true","error":" Cancelled "}}]}`,
			want:  `{"orders":[{"order_status":{"a":"cancelled","b":"cancelled","c":"cancelled","d":"cancelled","e":"cancelled","error":"cancelled"}}]}`,
		},
		{
			name:      "partial cancellation raw errors",
			input:     `{"orders":[{"order_status":{"a":"success","b":"private-secret","c":"","d":"unknown","e":"false"}}]}`,
			want:      `{"orders":[{"order_status":{"a":"cancelled","b":"cancellation_unconfirmed","c":"cancellation_unconfirmed","d":"cancellation_unconfirmed","e":"cancellation_unconfirmed"}}]}`,
			wantError: true,
		},
		{
			name:      "invalid cancellation status value",
			input:     `{"order_status":{"order-id":false}}`,
			want:      `{"order_status":{"order-id":"cancellation_unconfirmed"}}`,
			wantError: true,
		},
		{
			name:      "invalid cancellation status map",
			input:     `{"order_status":"private-secret"}`,
			want:      `{"order_status":"cancellation_unconfirmed"}`,
			wantError: true,
		},
		{name: "empty cancellation map", input: `{"order_status":{}}`, want: `{"order_status":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var value any
			require.NoError(t, json.Unmarshal([]byte(tc.input), &value), "sanitisation input must decode")
			var applicationError bool
			sanitise(value, &applicationError)
			assert.Equal(t, tc.wantError, applicationError, "unconfirmed cancellation and string errors should mark application failures")
			encoded, err := json.Marshal(value)
			require.NoError(t, err, "sanitised object must encode")
			assert.JSONEq(t, tc.want, string(encoded), "sanitised result should preserve safe observations")
			assert.NotContains(t, string(encoded), "private", "credentials and raw diagnostics should be removed recursively")
		})
	}
}

func TestEncodeResult(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		result callResult
		error  bool
	}{
		{name: "success", result: callResult{Messages: []json.RawMessage{json.RawMessage(`{}`)}}},
		{name: "RPC failure", result: callResult{Failure: "rpc_Unavailable"}, error: true},
		{name: "application error", result: callResult{ApplicationError: true}, error: true},
		{name: "invalid JSON", result: callResult{Messages: []json.RawMessage{json.RawMessage(`{`)}}, error: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			response := encodeResult(&tc.result)
			assert.Equal(t, tc.error, response.IsError, "tool should expose failure")
			require.Len(t, response.Content, 1, "result must include text for MCP clients")
			if tc.name != "invalid JSON" {
				text, ok := response.Content[0].(*mcp.TextContent)
				require.True(t, ok, "content must be text")
				var decoded callResult
				require.NoError(t, json.Unmarshal([]byte(text.Text), &decoded), "text must be structured JSON")
				assert.Equal(t, tc.result, decoded, "text and structured result should agree")
			}
		})
	}
}
