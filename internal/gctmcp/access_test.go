package gctmcp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

var testExecutionPolicy = config.MCPConfig{Enabled: true, Mode: config.MCPExecution, Permissions: config.MCPPermissions{Diagnostics: true, AccountData: true, OrderManagement: true, Withdrawals: true, SystemControl: true}}

func TestBridgeAuthorisePermissions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		required    accessPermission
		permissions config.MCPPermissions
	}{
		{name: "diagnostics", required: accessDiagnostics, permissions: config.MCPPermissions{Diagnostics: true}},
		{name: "account data", required: accessAccountData, permissions: config.MCPPermissions{AccountData: true}},
		{name: "order management", required: accessOrderManagement, permissions: config.MCPPermissions{OrderManagement: true}},
		{name: "withdrawals", required: accessWithdrawals, permissions: config.MCPPermissions{Withdrawals: true}},
		{name: "system control", required: accessSystemControl, permissions: config.MCPPermissions{SystemControl: true}},
		{name: "public data"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := &Bridge{options: Options{Policy: config.MCPConfig{Enabled: true, Permissions: tc.permissions}}}
			assert.NoError(t, b.authorisePermissions(tc.required, false), "granted reads should be allowed in readonly mode")
			assert.ErrorIs(t, b.authorisePermissions(tc.required, true), errPermissionDenied, "readonly mode should prevent every mutation")
			b.options.Policy.Mode = config.MCPExecution
			assert.NoError(t, b.authorisePermissions(tc.required, true), "execution mode should allow explicitly granted mutations")
			b.options.Policy.Permissions = config.MCPPermissions{}
			if tc.required != 0 {
				assert.ErrorIs(t, b.authorisePermissions(tc.required, false), errPermissionDenied, "execution should not grant private permissions")
			}
			b.options.Policy.Enabled = false
			assert.ErrorIs(t, b.authorisePermissions(tc.required, false), errPermissionDenied, "disabled MCP should deny public and private operations")
			b.options.Policy.Enabled = true
			b.options.Policy.Mode = "future"
			assert.ErrorIs(t, b.authorisePermissions(tc.required, false), errPermissionDenied, "unknown modes should fail closed")
		})
	}
}

func TestBridgeAuthoriseMethod(t *testing.T) {
	t.Parallel()
	t.Run("connection destinations remain operator managed", func(t *testing.T) {
		t.Parallel()
		for grants := range 32 {
			for _, mode := range []string{"", config.MCPReadOnly, config.MCPExecution} {
				t.Run(fmt.Sprintf("permissions %05b mode %s", grants, mode), func(t *testing.T) {
					t.Parallel()
					bridge := &Bridge{options: Options{Policy: config.MCPConfig{Enabled: true, Mode: mode, Permissions: config.MCPPermissions{
						Diagnostics:     grants&1 != 0,
						AccountData:     grants&2 != 0,
						OrderManagement: grants&4 != 0,
						Withdrawals:     grants&8 != 0,
						SystemControl:   grants&16 != 0,
					}}}}
					for _, request := range []struct {
						method  string
						message protoreflect.Message
					}{
						{method: "WebsocketSetURL", message: (&gctrpc.WebsocketSetURLRequest{Exchange: "coinbase", Url: "wss://unapproved.example/socket"}).ProtoReflect()},
						{method: "WebsocketSetProxy", message: (&gctrpc.WebsocketSetProxyRequest{Exchange: "coinbase", Proxy: "http://unapproved.example:8080"}).ProtoReflect()},
					} {
						assert.ErrorIs(t, bridge.authoriseMethod(request.method, nil), errMethodUnavailable, "destination controls should be absent from the catalogue for every permission combination")
						assert.ErrorIs(t, bridge.authoriseMethod(request.method, request.message), errMethodUnavailable, "destination requests should remain unavailable for every permission combination")
					}
				})
			}
		}
	})
	t.Run("connection changes require cancellation authority", func(t *testing.T) {
		t.Parallel()
		for _, method := range []string{"DisableExchange", "SetExchangePair", "SetAllExchangePairs", "SetExchangeAsset", "UpdateExchangeSupportedPairs", "Shutdown"} {
			t.Run(method, func(t *testing.T) {
				t.Parallel()
				bridge := &Bridge{options: Options{Policy: config.MCPConfig{Enabled: true, Mode: config.MCPExecution, Permissions: config.MCPPermissions{SystemControl: true}}}}
				assert.ErrorIs(t, bridge.authoriseMethod(method, nil), errPermissionDenied, "connection changes should require potential cancellation authority")
				bridge.options.Policy.Permissions.OrderManagement = true
				assert.NoError(t, bridge.authoriseMethod(method, nil), "system control and order management should permit connection changes")
			})
		}
	})
	for _, tc := range []struct {
		name, method, input string
		permissions         config.MCPPermissions
		execution, want     bool
	}{
		{name: "public ticker", method: "GetTicker", want: true},
		{name: "public reader cannot read balances", method: "GetAccountBalances"},
		{name: "account balances", method: "GetAccountBalances", permissions: config.MCPPermissions{AccountData: true}, want: true},
		{name: "account orders", method: "GetOrders", permissions: config.MCPPermissions{AccountData: true}, want: true},
		{name: "order lookup requires execution", method: "GetOrder", permissions: config.MCPPermissions{AccountData: true, OrderManagement: true}},
		{name: "order lookup requires management authority", method: "GetOrder", permissions: config.MCPPermissions{AccountData: true}, execution: true},
		{name: "order lookup requires account data", method: "GetOrder", permissions: config.MCPPermissions{OrderManagement: true}, execution: true},
		{name: "order lookup synchronises managed orders", method: "GetOrder", permissions: config.MCPPermissions{AccountData: true, OrderManagement: true}, execution: true, want: true},
		{name: "orders permission does not grant account reads", method: "GetOrders", permissions: config.MCPPermissions{OrderManagement: true}, execution: true},
		{name: "withdrawal permission does not grant account reads", method: "GetAccountBalances", permissions: config.MCPPermissions{Withdrawals: true}, execution: true},
		{name: "readonly order execution", method: "SubmitOrder", permissions: config.MCPPermissions{OrderManagement: true}},
		{name: "order execution", method: "SubmitOrder", permissions: config.MCPPermissions{OrderManagement: true}, execution: true, want: true},
		{name: "order permission does not grant withdrawals", method: "WithdrawCryptocurrencyFunds", permissions: config.MCPPermissions{OrderManagement: true}, execution: true},
		{name: "withdrawal execution", method: "WithdrawCryptocurrencyFunds", permissions: config.MCPPermissions{Withdrawals: true}, execution: true, want: true},
		{name: "public funding", method: "GetFundingRates", want: true},
		{name: "funding payments require account data", method: "GetFundingRates", input: `{"includePayments":true}`},
		{name: "funding payments", method: "GetFundingRates", input: `{"include_payments":true}`, permissions: config.MCPPermissions{AccountData: true}, want: true},
		{name: "exchange candles", method: "GetHistoricCandles", input: `{"useDb":false,"sync":false}`, want: true},
		{name: "managed position read", method: "GetFuturesPositionsOrders", permissions: config.MCPPermissions{AccountData: true}, want: true},
		{name: "position sync requires orders", method: "GetFuturesPositionsOrders", input: `{"syncWithOrderManager":true}`, permissions: config.MCPPermissions{AccountData: true}, execution: true},
		{name: "position sync requires execution", method: "GetFuturesPositionsOrders", input: `{"sync_with_order_manager":true}`, permissions: config.MCPPermissions{AccountData: true, OrderManagement: true}},
		{name: "position sync", method: "GetFuturesPositionsOrders", input: `{"syncWithOrderManager":true}`, permissions: config.MCPPermissions{AccountData: true, OrderManagement: true}, execution: true, want: true},
		{name: "orders subsystem requires orders", method: "DisableSubsystem", input: `{"subsystem":"Orders"}`, permissions: config.MCPPermissions{SystemControl: true}, execution: true},
		{name: "orders subsystem", method: "DisableSubsystem", input: `{"subsystem":"orders"}`, permissions: config.MCPPermissions{SystemControl: true, OrderManagement: true}, execution: true, want: true},
		{name: "nontrading subsystem", method: "DisableSubsystem", input: `{"subsystem":"internet_monitor"}`, permissions: config.MCPPermissions{SystemControl: true}, execution: true, want: true},
		{name: "shutdown requires order cancellation authority", method: "Shutdown", permissions: config.MCPPermissions{SystemControl: true}, execution: true},
		{name: "shutdown", method: "Shutdown", permissions: config.MCPPermissions{SystemControl: true, OrderManagement: true}, execution: true, want: true},
		{name: "balance refresh requires execution", method: "UpdateAccountBalances", permissions: config.MCPPermissions{AccountData: true}},
		{name: "balance refresh", method: "UpdateAccountBalances", permissions: config.MCPPermissions{AccountData: true}, execution: true, want: true},
		{name: "pair refresh requires cancellation authority", method: "UpdateExchangeSupportedPairs", permissions: config.MCPPermissions{SystemControl: true}, execution: true},
		{name: "pair refresh", method: "UpdateExchangeSupportedPairs", permissions: config.MCPPermissions{SystemControl: true, OrderManagement: true}, execution: true, want: true},
		{name: "portfolio editing needs system control", method: "AddPortfolioAddress", permissions: config.MCPPermissions{AccountData: true}, execution: true},
		{name: "portfolio editing", method: "AddPortfolioAddress", permissions: config.MCPPermissions{AccountData: true, SystemControl: true}, execution: true, want: true},
		{name: "portfolio removal needs system control", method: "RemovePortfolioAddress", permissions: config.MCPPermissions{AccountData: true}, execution: true},
		{name: "public withdrawal status", method: "CurrencyStateWithdraw", want: true},
		{name: "public deposit status", method: "CurrencyStateDeposit", want: true},
		{name: "public trading status", method: "CurrencyStateTrading", want: true},
		{name: "public pair status", method: "CurrencyStateTradingPair", want: true},
		{name: "disconnect exchange requires orders", method: "DisableExchange", permissions: config.MCPPermissions{SystemControl: true}, execution: true},
		{name: "disconnect exchange", method: "DisableExchange", permissions: config.MCPPermissions{SystemControl: true, OrderManagement: true}, execution: true, want: true},
		{name: "websocket disable requires orders", method: "WebsocketSetEnabled", input: `{"enable":false}`, permissions: config.MCPPermissions{SystemControl: true}, execution: true},
		{name: "websocket enable", method: "WebsocketSetEnabled", input: `{"enable":true}`, permissions: config.MCPPermissions{SystemControl: true}, execution: true, want: true},
		{name: "websocket disable", method: "WebsocketSetEnabled", input: `{"enable":false}`, permissions: config.MCPPermissions{SystemControl: true, OrderManagement: true}, execution: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			method := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods().ByName(protoreflect.Name(tc.method))
			request := dynamicpb.NewMessage(method.Input())
			if tc.input != "" {
				require.NoError(t, protojson.Unmarshal([]byte(tc.input), request), "request must decode")
			}
			for _, field := range requiredMessageFields[tc.method] {
				request.Mutable(request.Descriptor().Fields().ByName(field))
			}
			mode := config.MCPReadOnly
			if tc.execution {
				mode = config.MCPExecution
			}
			b := &Bridge{options: Options{Policy: config.MCPConfig{Enabled: true, Mode: mode, Permissions: tc.permissions}}}
			err := b.authoriseMethod(tc.method, request)
			if tc.want {
				assert.NoError(t, err, "configured authority should permit the operation")
			} else {
				assert.ErrorIs(t, err, errPermissionDenied, "missing data or execution authority should deny the operation")
			}
		})
	}
	t.Run("unknown excluded and offline methods", func(t *testing.T) {
		t.Parallel()
		b := &Bridge{options: Options{Policy: testExecutionPolicy}}
		assert.ErrorIs(t, b.authoriseMethod("FutureMethod", nil), errMethodUnavailable, "unknown methods should fail closed")
		for name := range excludedMethods {
			assert.ErrorIs(t, b.authoriseMethod(name, nil), errMethodUnavailable, "excluded methods should be hidden")
			method := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods().ByName(protoreflect.Name(name))
			assert.ErrorIs(t, b.authoriseMethod(name, dynamicpb.NewMessage(method.Input())), errMethodUnavailable, "excluded methods should fail at dispatch")
		}
		b.options.Offline = true
		assert.ErrorIs(t, b.authoriseMethod("GetTicker", nil), errMethodUnavailable, "offline mode should deny RPC calls")
	})
}

func TestBridgeAuthoriseRepair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		action      string
		permissions config.MCPPermissions
		want        bool
	}{
		{action: "refresh_balances"},
		{action: "refresh_balances", permissions: config.MCPPermissions{AccountData: true}, want: true},
		{action: "refresh_balances", permissions: config.MCPPermissions{SystemControl: true}},
		{action: "refresh_pairs"},
		{action: "refresh_pairs", permissions: config.MCPPermissions{SystemControl: true}},
		{action: "refresh_pairs", permissions: config.MCPPermissions{SystemControl: true, OrderManagement: true}, want: true},
		{action: "refresh_pairs", permissions: config.MCPPermissions{AccountData: true}},
	} {
		t.Run(fmt.Sprintf("%s account_%t system_%t orders_%t", tc.action, tc.permissions.AccountData, tc.permissions.SystemControl, tc.permissions.OrderManagement), func(t *testing.T) {
			t.Parallel()
			request := &repairRequest{ID: "repair-access-001", Action: tc.action, Exchange: "binance", Reason: "stale_data"}
			if tc.action == "refresh_balances" {
				request.Asset = "spot"
			}
			b := &Bridge{options: Options{Policy: config.MCPConfig{Enabled: true, Mode: config.MCPExecution, Permissions: tc.permissions}}}
			err := b.authoriseRepair(request)
			if tc.want {
				assert.NoError(t, err, "the relevant scope should allow refreshes without unrelated permissions")
			} else {
				assert.ErrorIs(t, err, errPermissionDenied, "missing scope should deny refreshes")
			}
			b.options.Policy.Mode = config.MCPReadOnly
			assert.ErrorIs(t, b.authoriseRepair(request), errPermissionDenied, "readonly mode should deny refreshes")
			b.options.Offline = true
			assert.ErrorIs(t, b.authoriseRepair(request), errRepairRequest, "offline mode should deny refreshes")
		})
	}
	t.Run("invalid request", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, new(Bridge).authoriseRepair(nil), errRepairRequest, "invalid requests should fail")
	})
}
