package gctmcp

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/config"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// authorisePermissions keeps private data scopes independent of the global execution gate.
func (b *Bridge) authorisePermissions(required accessPermission, execution bool) error {
	policy := &b.options.Policy
	if !policy.Enabled || policy.Mode != "" && policy.Mode != config.MCPReadOnly && policy.Mode != config.MCPExecution {
		return errPermissionDenied
	}
	p := &policy.Permissions
	if required&accessDiagnostics != 0 && !p.Diagnostics || required&accessAccountData != 0 && !p.AccountData || required&accessOrderManagement != 0 && !p.OrderManagement || required&accessWithdrawals != 0 && !p.Withdrawals || required&accessSystemControl != 0 && !p.SystemControl || execution && policy.Mode != config.MCPExecution {
		return errPermissionDenied
	}
	return nil
}

// authoriseMethod checks both fixed authority and argument-dependent private data or writes.
func (b *Bridge) authoriseMethod(method string, request protoreflect.Message) error {
	if b.options.Offline {
		return errMethodUnavailable
	}
	policy, exists := methodPolicies[method]
	if !exists {
		return errMethodUnavailable
	}
	if err := b.authorisePermissions(policy.permissions, policy.execution); err != nil {
		return fmt.Errorf("%w: RPC %s", err, method)
	}
	if request == nil {
		return nil
	}
	if err := validateRequestPolicy(method, request); err != nil {
		return err
	}
	if (method == "EnableSubsystem" || method == "DisableSubsystem") && strings.EqualFold(request.Get(request.Descriptor().Fields().ByName("subsystem")).String(), "orders") {
		if err := b.authorisePermissions(accessOrderManagement, true); err != nil {
			return fmt.Errorf("%w: the orders subsystem requires order management access", err)
		}
	}
	if method == "WebsocketSetEnabled" && !request.Get(request.Descriptor().Fields().ByName("enable")).Bool() {
		if err := b.authorisePermissions(accessOrderManagement, true); err != nil {
			return fmt.Errorf("%w: disconnecting a websocket requires order management access", err)
		}
	}
	if method == "GetFuturesPositionsOrders" && request.Get(request.Descriptor().Fields().ByName("sync_with_order_manager")).Bool() {
		if err := b.authorisePermissions(accessOrderManagement, true); err != nil {
			return fmt.Errorf("%w: managed-order synchronisation", err)
		}
	}
	if method == "GetFundingRates" && request.Get(request.Descriptor().Fields().ByName("include_payments")).Bool() {
		if err := b.authorisePermissions(accessAccountData, false); err != nil {
			return fmt.Errorf("%w: funding payments require account data access", err)
		}
	}
	return nil
}

// authoriseRepair grants each action only its relevant data or control scope.
func (b *Bridge) authoriseRepair(request *repairRequest) error {
	if validateRepairRequest(request) != nil || b.options.Offline {
		return errRepairRequest
	}
	method := "UpdateExchangeSupportedPairs"
	if request.Action == "refresh_balances" {
		method = "UpdateAccountBalances"
	}
	return b.authoriseMethod(method, nil)
}
