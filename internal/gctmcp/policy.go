package gctmcp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	fieldPair               = "pair"
	destinationExcluded     = "Connection destinations are configured by the operator outside MCP."
	otpExcluded             = "OTP generation is unavailable through MCP."
	databaseHistoryExcluded = "Database history operations are outside the MCP server scope."
)

var (
	errPermissionDenied  = errors.New("MCP permission is denied")
	errMethodUnavailable = errors.New("RPC method is unavailable")
	errPolicyRequest     = errors.New("RPC request does not match the classified method")
	errDatabaseHistory   = errors.New("database history operations are unavailable through MCP")
)

var excludedMethods = map[string]string{
	"WebsocketSetURL":                  destinationExcluded,
	"WebsocketSetProxy":                destinationExcluded,
	"GetExchangeOTPCode":               otpExcluded,
	"GetExchangeOTPCodes":              otpExcluded,
	"GetAuditEvent":                    databaseHistoryExcluded,
	"WithdrawalEventByID":              databaseHistoryExcluded,
	"WithdrawalEventsByExchange":       databaseHistoryExcluded,
	"WithdrawalEventsByDate":           databaseHistoryExcluded,
	"GetSavedTrades":                   databaseHistoryExcluded,
	"ConvertTradesToCandles":           databaseHistoryExcluded,
	"FindMissingSavedCandleIntervals":  databaseHistoryExcluded,
	"FindMissingSavedTradeIntervals":   databaseHistoryExcluded,
	"SetExchangeTradeProcessing":       databaseHistoryExcluded,
	"UpsertDataHistoryJob":             databaseHistoryExcluded,
	"GetDataHistoryJobDetails":         databaseHistoryExcluded,
	"GetActiveDataHistoryJobs":         databaseHistoryExcluded,
	"GetDataHistoryJobsBetween":        databaseHistoryExcluded,
	"GetDataHistoryJobSummary":         databaseHistoryExcluded,
	"SetDataHistoryJobStatus":          databaseHistoryExcluded,
	"UpdateDataHistoryJobPrerequisite": databaseHistoryExcluded,

	"GetConfig": "The RPC server does not implement GetConfig.",
	"GetEvents": "The RPC server does not implement GetEvents.",
}

// These RPC handlers dereference nested messages without checking their presence.
var requiredMessageFields = map[string][]protoreflect.Name{
	"GetTicker":                   {fieldPair},
	"GetOrderbook":                {fieldPair},
	"CancelBatchOrders":           {fieldPair},
	"ModifyOrder":                 {fieldPair},
	"AddEvent":                    {fieldPair, "condition_params"},
	"GetOrderbookStream":          {fieldPair},
	"GetTickerStream":             {fieldPair},
	"GetFundingRates":             {fieldPair},
	"GetLatestFundingRate":        {fieldPair},
	"GetTechnicalAnalysis":        {fieldPair},
	"GetOrderbookMovement":        {fieldPair},
	"GetOrderbookAmountByNominal": {fieldPair},
	"GetOrderbookAmountByImpact":  {fieldPair},
}

// validateRequestPolicy checks the actual decoded request, including protobuf JSON aliases.
func validateRequestPolicy(method string, request protoreflect.Message) error {
	if _, ok := methodPolicies[method]; !ok {
		return fmt.Errorf("%w: %s", errMethodUnavailable, method)
	}
	descriptor := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods().ByName(protoreflect.Name(method))
	if request == nil || !request.IsValid() || request.Descriptor().FullName() != descriptor.Input().FullName() {
		return fmt.Errorf("%w: %s", errPolicyRequest, method)
	}
	if method == "GetHistoricCandles" {
		for _, name := range []protoreflect.Name{"use_db", "sync", "fill_missing_with_trades", "force"} {
			if request.Get(request.Descriptor().Fields().ByName(name)).Bool() {
				return fmt.Errorf("%w: %s.%s", errDatabaseHistory, method, name)
			}
		}
	}
	if method == "EnableSubsystem" || method == "DisableSubsystem" {
		subsystem := strings.ToLower(request.Get(request.Descriptor().Fields().ByName("subsystem")).String())
		if subsystem == "database" || subsystem == "data_history_manager" {
			return fmt.Errorf("%w: %s", errDatabaseHistory, method)
		}
	}
	for _, name := range requiredMessageFields[method] {
		if !request.Has(request.Descriptor().Fields().ByName(name)) {
			return fmt.Errorf("%w: %s requires %s", errPolicyRequest, method, name)
		}
	}
	if method == "GetTechnicalAnalysis" && strings.EqualFold(request.Get(request.Descriptor().Fields().ByName("algorithm_type")).String(), "COCO") && !request.Has(request.Descriptor().Fields().ByName("other_pair")) {
		return fmt.Errorf("%w: %s requires other_pair for COCO", errPolicyRequest, method)
	}
	if method == "GetOpenInterest" {
		data := request.Get(request.Descriptor().Fields().ByName("data")).List()
		for i := range data.Len() {
			item := data.Get(i).Message()
			if !item.Has(item.Descriptor().Fields().ByName(fieldPair)) {
				return fmt.Errorf("%w: %s requires data[%d].pair", errPolicyRequest, method, i)
			}
		}
	}
	return nil
}
