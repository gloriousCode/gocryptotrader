package gctmcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestMethodPolicies(t *testing.T) {
	t.Parallel()
	t.Run("restricted methods are excluded", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"GetAuditEvent", "WithdrawalEventByID", "WithdrawalEventsByExchange", "WithdrawalEventsByDate", "GetSavedTrades", "ConvertTradesToCandles", "FindMissingSavedCandleIntervals", "FindMissingSavedTradeIntervals", "SetExchangeTradeProcessing", "UpsertDataHistoryJob", "GetDataHistoryJobDetails", "GetActiveDataHistoryJobs", "GetDataHistoryJobsBetween", "GetDataHistoryJobSummary", "SetDataHistoryJobStatus", "UpdateDataHistoryJobPrerequisite", "GetExchangeOTPCode", "GetExchangeOTPCodes", "WebsocketSetURL", "WebsocketSetProxy"} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.NotContains(t, methodPolicies, name, "restricted methods should not be exposed")
				assert.Contains(t, excludedMethods, name, "restricted methods should have explicit exclusions")
			})
		}
	})
	methods := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods()
	require.Equal(t, methods.Len(), len(methodPolicies)+len(excludedMethods), "every RPC must be classified or explicitly excluded")
	for i := range methods.Len() {
		method := methods.Get(i)
		t.Run(string(method.Name()), func(t *testing.T) {
			t.Parallel()
			name := string(method.Name())
			policy, classified := methodPolicies[name]
			reason, excluded := excludedMethods[name]
			require.NotEqual(t, classified, excluded, "RPC must have exactly one classification or exclusion")
			if excluded {
				assert.NotEmpty(t, reason, "excluded RPC should explain its omission")
				return
			}
			assert.Zero(t, policy.permissions & ^(accessDiagnostics|accessAccountData|accessOrderManagement|accessWithdrawals|accessSystemControl), "RPC should use recognised permissions")
		})
	}
	for method := range methodPolicies {
		assert.NotNil(t, methods.ByName(protoreflect.Name(method)), "classified method should exist in the RPC service")
	}
	for method := range excludedMethods {
		assert.NotNil(t, methods.ByName(protoreflect.Name(method)), "excluded method should exist in the RPC service")
	}
}

func TestValidateRequestPolicy(t *testing.T) {
	t.Parallel()
	methods := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods()
	t.Run("database history options remain unavailable with execution permissions", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			method string
			input  string
		}{
			{method: "GetHistoricCandles", input: `{"useDb":true}`},
			{method: "GetHistoricCandles", input: `{"use_db":true}`},
			{method: "GetHistoricCandles", input: `{"sync":true}`},
			{method: "GetHistoricCandles", input: `{"fillMissingWithTrades":true}`},
			{method: "GetHistoricCandles", input: `{"fill_missing_with_trades":true}`},
			{method: "GetHistoricCandles", input: `{"force":true}`},
			{method: "EnableSubsystem", input: `{"subsystem":"database"}`},
			{method: "DisableSubsystem", input: `{"subsystem":"DATABASE"}`},
			{method: "EnableSubsystem", input: `{"subsystem":"data_history_manager"}`},
			{method: "DisableSubsystem", input: `{"subsystem":"Data_History_Manager"}`},
		} {
			t.Run(tc.method+tc.input, func(t *testing.T) {
				t.Parallel()
				request := dynamicpb.NewMessage(methods.ByName(protoreflect.Name(tc.method)).Input())
				require.NoError(t, protojson.Unmarshal([]byte(tc.input), request), "request must decode")
				assert.ErrorIs(t, validateRequestPolicy(tc.method, request), errDatabaseHistory, "database history options should remain unavailable even with execution permissions")
			})
		}
	})
	t.Run("all excluded methods reject dispatch", func(t *testing.T) {
		t.Parallel()
		for name := range excludedMethods {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				request := dynamicpb.NewMessage(methods.ByName(protoreflect.Name(name)).Input())
				assert.ErrorIs(t, validateRequestPolicy(name, request), errMethodUnavailable, "excluded methods should remain unavailable at dispatch")
			})
		}
	})
	for method := range methodPolicies {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			request := dynamicpb.NewMessage(methods.ByName(protoreflect.Name(method)).Input())
			for _, field := range requiredMessageFields[method] {
				request.Mutable(request.Descriptor().Fields().ByName(field))
			}
			assert.NoError(t, validateRequestPolicy(method, request), "valid message structure should pass policy validation")
		})
	}
	for method, fields := range requiredMessageFields {
		for _, field := range fields {
			t.Run(method+" requires "+string(field), func(t *testing.T) {
				t.Parallel()
				request := dynamicpb.NewMessage(methods.ByName(protoreflect.Name(method)).Input())
				for _, required := range fields {
					request.Mutable(request.Descriptor().Fields().ByName(required))
				}
				assert.NoError(t, validateRequestPolicy(method, request), "present nested messages should pass the policy")
				request.Clear(request.Descriptor().Fields().ByName(field))
				err := validateRequestPolicy(method, request)
				assert.ErrorIs(t, err, errPolicyRequest, "missing nested messages should be rejected before invoking RPC")
				assert.ErrorContains(t, err, string(field), "the rejection should identify the missing field")
			})
		}
	}
	for _, tc := range []struct {
		name    string
		method  string
		input   string
		wantErr error
	}{
		{name: "interest entry missing pair", method: "GetOpenInterest", input: `{"data":[{"asset":"perpetualswap"}]}`, wantErr: errPolicyRequest},
		{name: "later interest entry missing pair", method: "GetOpenInterest", input: `{"data":[{"pair":{}},{"asset":"perpetualswap"}]}`, wantErr: errPolicyRequest},
		{name: "interest entry null pair", method: "GetOpenInterest", input: `{"data":[{"pair":null}]}`, wantErr: errPolicyRequest},
		{name: "interest entry pair present", method: "GetOpenInterest", input: `{"data":[{"pair":{}}]}`},
		{name: "interest empty list", method: "GetOpenInterest", input: `{}`},
		{name: "correlation missing other pair", method: "GetTechnicalAnalysis", input: `{"pair":{},"algorithmType":"COCO"}`, wantErr: errPolicyRequest},
		{name: "mixed case correlation missing other pair", method: "GetTechnicalAnalysis", input: `{"pair":{},"algorithm_type":"cOcO"}`, wantErr: errPolicyRequest},
		{name: "correlation other pair present", method: "GetTechnicalAnalysis", input: `{"pair":{},"otherPair":{},"algorithmType":"COCO"}`},
		{name: "other analysis leaves other pair optional", method: "GetTechnicalAnalysis", input: `{"pair":{},"algorithmType":"TWAP"}`},
		{name: "futures underlying pair optional", method: "GetFuturesPositionsSummary", input: `{"pair":{}}`},
		{name: "order pair backend validation preserved", method: "GetOrders", input: `{}`},
		{name: "exchange candles without database options", method: "GetHistoricCandles", input: `{}`},
		{name: "exchange candles explicitly disable database options", method: "GetHistoricCandles", input: `{"useDb":false,"sync":false,"fillMissingWithTrades":false,"force":false}`},
		{name: "exchange candles explicitly disable protobuf database options", method: "GetHistoricCandles", input: `{"use_db":false,"sync":false,"fill_missing_with_trades":false,"force":false,"ex_request":true}`},
		{name: "exchange historic trades remain available", method: "GetHistoricTrades", input: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := dynamicpb.NewMessage(methods.ByName(protoreflect.Name(tc.method)).Input())
			require.NoError(t, protojson.Unmarshal([]byte(tc.input), request), "request must decode")
			err := validateRequestPolicy(tc.method, request)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "unsafe nested message input should be rejected")
			} else {
				assert.NoError(t, err, "safe optional fields should preserve RPC behaviour")
			}
		})
	}
	for _, tc := range []struct {
		name    string
		method  string
		request protoreflect.Message
		wantErr error
	}{
		{name: "unknown method", method: "FutureRPC", request: (&gctrpc.GetInfoRequest{}).ProtoReflect(), wantErr: errMethodUnavailable},
		{name: "excluded method", method: "GetConfig", request: (&gctrpc.GetConfigRequest{}).ProtoReflect(), wantErr: errMethodUnavailable},
		{name: "excluded events", method: "GetEvents", request: (&gctrpc.GetEventsRequest{}).ProtoReflect(), wantErr: errMethodUnavailable},
		{name: "nil message", method: "GetInfo", wantErr: errPolicyRequest},
		{name: "invalid message", method: "GetInfo", request: (*gctrpc.GetInfoRequest)(nil).ProtoReflect(), wantErr: errPolicyRequest},
		{name: "mismatched message", method: "GetInfo", request: (&gctrpc.SubmitOrderRequest{}).ProtoReflect(), wantErr: errPolicyRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateRequestPolicy(tc.method, tc.request), tc.wantErr, "invalid policy input should return its sentinel error")
		})
	}
}
