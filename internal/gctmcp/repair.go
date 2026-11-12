package gctmcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errRepairRequest = errors.New("invalid data repair request")

const repairCaveat = "Receipts are historical evidence, not permission to trade. A matching ID is never executed twice. On an uncertain outcome inspect the receipt and actual data; never invent a new ID to retry. An unresolved repair-active.json requires operator reconciliation after confirming the old request has stopped. At most 64 receipts are admitted per directory; archive and reconcile them deliberately before rotating the directory."

// addRepairTools separates deliberate mutation from read-only receipt inspection.
func (b *Bridge) addRepairTools(server *mcp.Server) {
	if b.options.DiagnosticsDir == "" || b.authorisePermissions(0, false) != nil {
		return
	}
	p := b.options.Policy.Permissions
	if p.Diagnostics || p.AccountData || p.SystemControl {
		mcp.AddTool(server, &mcp.Tool{Name: "gct_repair_status", Description: "Read permitted repair receipts independently of RPC. Balance receipts require accountData; pair receipts require diagnostics or systemControl.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
			func(_ context.Context, _ *mcp.CallToolRequest, request repairStatusRequest) (*mcp.CallToolResult, *repairResult, error) {
				result := b.inspectRepairs(request.ID)
				boundRepairResult(result, b.options.MaxResponseBytes)
				return &mcp.CallToolResult{IsError: result.Failure != ""}, result, nil
			})
	}
	canRefresh := p.AccountData || p.SystemControl && p.OrderManagement
	if b.options.Offline || b.authorisePermissions(0, true) != nil || !canRefresh {
		return
	}
	destructive := true
	mcp.AddTool(server, &mcp.Tool{Name: "gct_repair", Description: "Perform one tracked cache refresh. refresh_balances requires accountData; refresh_pairs requires systemControl and orderManagement because subscription changes can close connections. Execution mode is required. Actions are never retried automatically.", Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: true}},
		func(ctx context.Context, _ *mcp.CallToolRequest, request repairRequest) (*mcp.CallToolResult, *repairResult, error) {
			result := b.repair(ctx, &request)
			boundRepairResult(result, b.options.MaxResponseBytes)
			return &mcp.CallToolResult{IsError: result.Failure != ""}, result, nil
		})
}

// validateRepairRequest rejects paths, arbitrary operations and ambiguous native identities.
func validateRepairRequest(request *repairRequest) error {
	if request == nil || len(request.ID) < 8 || len(request.ID) > 64 || strings.Trim(request.ID, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" || request.Exchange == "" || len(request.Exchange) > 32 || strings.Trim(request.Exchange, "abcdefghijklmnopqrstuvwxyz0123456789_-") != "" {
		return errRepairRequest
	}
	if len(request.Asset) > 32 || strings.Trim(request.Asset, "abcdefghijklmnopqrstuvwxyz0123456789_-") != "" {
		return errRepairRequest
	}
	switch request.Reason {
	case "stale_data", "retrieval_failed":
	default:
		return errRepairRequest
	}
	switch request.Action {
	case "refresh_balances":
		if request.Asset == "" {
			return errRepairRequest
		}
	case "refresh_pairs":
		if request.Asset != "" {
			return errRepairRequest
		}
	default:
		return errRepairRequest
	}
	return nil
}

// readRepair validates a bounded receipt before treating it as duplicate protection.
func readRepair(root *os.Root, id string) (*repairRecord, error) {
	if len(id) < 8 || len(id) > 64 || strings.Trim(id, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		return nil, errRepairRequest
	}
	file, err := root.Open("repair-" + id + ".json")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 16385))
	if err != nil || len(data) > 16384 {
		return nil, errRepairRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record *repairRecord
	if decoder.Decode(&record) != nil || decoder.Decode(new(any)) != io.EOF || record == nil || record.Version != 1 || record.Request.ID != id || validateRepairRequest(&record.Request) != nil || record.StartedAt.IsZero() {
		return nil, errRepairRequest
	}
	if record.Failure != "" {
		known := record.Failure == "repair_target_not_enabled" || record.Failure == "repair_permission_denied"
		for code := codes.OK; code <= codes.Unauthenticated; code++ {
			known = known || record.Failure == "repair_rpc_"+code.String()
		}
		if !known {
			return nil, errRepairRequest
		}
	}
	switch record.Status {
	case "running", statusUnknown, statusRejected, "completed_unverified":
		return record, nil
	default:
		return nil, errRepairRequest
	}
}

// writeRepair records intent exclusively and completes it by an atomic same-root rename.
func writeRepair(root *os.Root, record *repairRecord, intent bool) error {
	if record == nil || record.Version != 1 || validateRepairRequest(&record.Request) != nil {
		return errRepairRequest
	}
	data, err := json.Marshal(record)
	if err != nil || len(data) > 16384 {
		return errRepairRequest
	}
	name := "repair-" + record.Request.ID + ".json"
	path := name
	if !intent {
		path += ".next"
	}
	file, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if !intent {
		return root.Rename(path, name)
	}
	return nil
}

// inspectRepairs exposes only receipts covered by the process's data permissions.
func (b *Bridge) inspectRepairs(id string) *repairResult {
	p := b.options.Policy.Permissions
	if b.authorisePermissions(0, false) != nil || !p.Diagnostics && !p.AccountData && !p.SystemControl {
		return &repairResult{Status: statusUnavailable, Failure: "repair_permission_denied", Records: []repairRecord{}, Caveat: repairCaveat}
	}
	result := readRepairs(b.options.DiagnosticsDir, id)
	records := make([]repairRecord, 0, len(result.Records))
	for i := range result.Records {
		record := &result.Records[i]
		if record.Request.Action == "refresh_balances" && !p.AccountData || record.Request.Action == "refresh_pairs" && !p.Diagnostics && !p.SystemControl {
			if id != "" {
				result.Status, result.Failure = statusUnavailable, "repair_receipt_unavailable_or_invalid"
			}
			continue
		}
		records = append(records, *record)
	}
	result.Records = records
	return result
}

// readRepairs bounds ledger admission independently of the caller's receipt visibility.
func readRepairs(directoryPath, id string) *repairResult {
	result := &repairResult{Status: statusHistorical, Records: []repairRecord{}, Caveat: repairCaveat}
	root, err := os.OpenRoot(directoryPath)
	if err != nil {
		result.Status, result.Failure = statusUnavailable, "repair_directory_unavailable"
		return result
	}
	defer root.Close()
	if id != "" {
		record, err := readRepair(root, id)
		if err != nil {
			result.Status, result.Failure = statusUnavailable, "repair_receipt_unavailable_or_invalid"
			return result
		}
		result.Records = append(result.Records, *record)
		return result
	}
	directory, err := root.Open(".")
	if err != nil {
		result.Status, result.Failure = statusUnavailable, "repair_directory_unavailable"
		return result
	}
	defer directory.Close()
	entries, err := directory.ReadDir(513)
	if err != nil && !errors.Is(err, io.EOF) || len(entries) > 512 {
		result.Status, result.Failure = statusUnavailable, "repair_directory_scan_limit"
		return result
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "repair-") || !strings.HasSuffix(name, ".json") || name == "repair-active.json" {
			continue
		}
		record, err := readRepair(root, strings.TrimSuffix(strings.TrimPrefix(name, "repair-"), ".json"))
		if err != nil || len(result.Records) == 64 {
			result.Status, result.Failure = statusUnavailable, "repair_ledger_unavailable_or_full"
			return result
		}
		result.Records = append(result.Records, *record)
	}
	slices.SortFunc(result.Records, func(a, b repairRecord) int { return b.StartedAt.Compare(a.StartedAt) })
	return result
}

// boundRepairResult includes both MCP representations in the response budget.
func boundRepairResult(result *repairResult, maximum int) {
	data, err := json.Marshal(result)
	if err == nil {
		data, err = json.Marshal(&mcp.CallToolResult{StructuredContent: result, Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}})
	}
	if err != nil || len(data) > maximum-256 {
		result.Status, result.Failure, result.Records = statusUnavailable, "repair_response_size_limit", nil
	}
}

// repair admits one operation only after durable intent and an exclusive execution guard.
func (b *Bridge) repair(ctx context.Context, request *repairRequest) *repairResult {
	result := &repairResult{Status: statusRejected, Records: []repairRecord{}, Caveat: repairCaveat}
	if b.authoriseRepair(request) != nil {
		result.Failure = "repair_request_or_permission_invalid"
		return result
	}
	root, err := os.OpenRoot(b.options.DiagnosticsDir)
	if err != nil {
		result.Failure = "repair_directory_unavailable"
		return result
	}
	defer root.Close()
	if previous, err := readRepair(root, request.ID); err == nil {
		if previous.Request != *request {
			result.Failure = "repair_id_parameter_conflict"
			return result
		}
		result.Status, result.Records = statusHistorical, []repairRecord{*previous}
		return result
	} else if !os.IsNotExist(err) {
		result.Failure = "repair_receipt_unavailable_or_invalid"
		return result
	}
	guard, err := root.OpenFile("repair-active.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		result.Failure = "repair_execution_guard_exists"
		return result
	}
	keepGuard := false
	defer func() {
		if !keepGuard {
			if err := root.Remove("repair-active.json"); err != nil {
				result.Failure = "repair_guard_release_failed"
			}
		}
	}()
	if err := guard.Close(); err != nil {
		result.Failure = "repair_execution_guard_failed"
		return result
	}
	// Admission evidence must be read while holding the guard: another bridge may
	// have finished after our first duplicate check and before we claimed it.
	if previous, err := readRepair(root, request.ID); err == nil {
		if previous.Request != *request {
			result.Failure = "repair_id_parameter_conflict"
			return result
		}
		result.Status, result.Records = statusHistorical, []repairRecord{*previous}
		return result
	} else if !errors.Is(err, os.ErrNotExist) {
		result.Failure = "repair_receipt_unavailable_or_invalid"
		return result
	}
	ledger := readRepairs(b.options.DiagnosticsDir, "")
	if ledger.Failure != "" || len(ledger.Records) >= 64 {
		result.Failure = "repair_ledger_unavailable_or_full"
		return result
	}
	if len(ledger.Records) > 0 && time.Since(ledger.Records[0].StartedAt) < time.Minute {
		result.Failure = "repair_admission_backoff"
		return result
	}
	record := &repairRecord{Version: 1, Request: *request, StartedAt: time.Now().UTC(), Status: "running"}
	if err := writeRepair(root, record, true); err != nil {
		result.Failure = "repair_intent_persistence_failed"
		return result
	}
	keepGuard = true
	ctx, cancel := context.WithTimeout(ctx, b.options.Timeout)
	defer cancel()
	b.executeRepair(ctx, record)
	if err := writeRepair(root, record, false); err != nil {
		result.Failure = "repair_completion_persistence_failed"
		result.Status, result.Records = statusUnknown, []repairRecord{*record}
		return result
	}
	if record.Status != statusUnknown {
		keepGuard = false
	}
	result.Status, result.Records = record.Status, []repairRecord{*record}
	if record.Failure != "" {
		result.Failure = record.Failure
	}
	return result
}

// executeRepair uses fixed RPCs without claiming account or market reconciliation.
func (b *Bridge) executeRepair(ctx context.Context, record *repairRecord) {
	if b.authoriseRepair(&record.Request) != nil {
		record.Status, record.Failure, record.EndedAt = statusRejected, "repair_permission_denied", time.Now().UTC()
		return
	}
	client := gctrpc.NewGoCryptoTraderServiceClient(b.conn)
	options := []grpc.CallOption{grpc.MaxCallRecvMsgSize(b.options.MaxResponseBytes)}
	info, err := client.GetExchangeInfo(ctx, &gctrpc.GenericExchangeNameRequest{Exchange: record.Request.Exchange}, options...)
	attempted := false
	if err == nil && (info == nil || !info.Enabled) {
		err = errRepairRequest
	}
	if err == nil {
		switch record.Request.Action {
		case "refresh_balances":
			if info.SupportedAssets[record.Request.Asset] == nil {
				err = errRepairRequest
				break
			}
			attempted = true
			_, err = client.UpdateAccountBalances(ctx, &gctrpc.GetAccountBalancesRequest{Exchange: record.Request.Exchange, AssetType: record.Request.Asset}, options...)
		case "refresh_pairs":
			attempted = true
			_, err = client.UpdateExchangeSupportedPairs(ctx, &gctrpc.UpdateExchangeSupportedPairsRequest{Exchange: record.Request.Exchange}, options...)
		}
	}
	record.EndedAt = time.Now().UTC()
	if err != nil {
		record.Status, record.Failure = statusUnknown, "repair_rpc_"+status.Code(err).String()
		if !attempted {
			record.Status = statusRejected
		}
		if errors.Is(err, errRepairRequest) {
			record.Failure = "repair_target_not_enabled"
		}
		return
	}
	record.Status = "completed_unverified"
}
