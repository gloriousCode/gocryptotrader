package gctmcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	errInvalidOptions = errors.New("invalid MCP bridge options")
	errInvalidRequest = errors.New("request must be a protobuf JSON object with recognised fields and valid values")
)

// New validates limits and permissions without connecting to or starting GCT.
func New(conn grpc.ClientConnInterface, options *Options) (*Bridge, error) {
	if options == nil {
		return nil, errInvalidOptions
	}
	switch {
	case !options.Offline && (conn == nil || reflect.ValueOf(conn).Kind() == reflect.Pointer && reflect.ValueOf(conn).IsNil()):
		return nil, fmt.Errorf("%w: missing RPC connection", errInvalidOptions)
	case options.Timeout <= 0 || options.Timeout > time.Minute:
		return nil, fmt.Errorf("%w: timeout must be positive and at most one minute", errInvalidOptions)
	case options.MaxMessages < 1 || options.MaxMessages > 1000:
		return nil, fmt.Errorf("%w: message limit must be between 1 and 1000", errInvalidOptions)
	case options.MaxResponseBytes < 4096 || options.MaxResponseBytes > 4<<20:
		return nil, fmt.Errorf("%w: response limit must be between 4096 and 4194304 bytes", errInvalidOptions)
	}
	policy := options.Policy
	if !policy.Enabled {
		return nil, fmt.Errorf("%w: %w: MCP is disabled in configuration", errInvalidOptions, errPermissionDenied)
	}
	if policy.Mode == "" {
		policy.Mode = config.MCPReadOnly
	}
	if policy.Mode != config.MCPReadOnly && policy.Mode != config.MCPExecution {
		return nil, fmt.Errorf("%w: MCP mode must be readonly or execution", errInvalidOptions)
	}
	configured := *options
	configured.Policy = policy
	return &Bridge{conn: conn, options: configured}, nil
}

// Server advertises only classified commands permitted by the operator.
func (b *Bridge) Server() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "gct-rpc", Version: "0.1.0"}, nil)
	b.addDiagnosticTools(server)
	b.addRepairTools(server)
	if b.options.Offline {
		return server
	}
	service := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService")
	for i := range service.Methods().Len() {
		method := service.Methods().Get(i)
		name := string(method.Name())
		policy, classified := methodPolicies[name]
		if !classified || b.authoriseMethod(name, nil) != nil || method.IsStreamingClient() {
			continue
		}
		readOnly := !policy.execution
		if name == "GetFuturesPositionsOrders" && b.authorisePermissions(accessOrderManagement, true) == nil {
			readOnly = false
		}
		destructive := !readOnly
		description := "Call GCT RPC " + name + ". Request uses protobuf JSON fields. Results are observations; retrieval time does not establish market freshness."
		if !readOnly {
			description += " This command can change state. A transport error can leave its outcome unknown; inspect state before retrying."
		}
		if method.IsStreamingServer() {
			description += " Returns a bounded sample, not a persistent subscription or necessarily complete history."
		}
		server.AddTool(&mcp.Tool{Name: "gct_" + name, Description: description, InputSchema: requestSchema(method), Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly, DestructiveHint: &destructive}},
			func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return b.call(ctx, method, request.Params.Arguments), nil
			})
	}
	return server
}

// call never retries an RPC and retains already received messages on failure.
func (b *Bridge) call(ctx context.Context, method protoreflect.MethodDescriptor, arguments json.RawMessage) *mcp.CallToolResult {
	name := string(method.Name())
	result := callResult{Method: name, CheckedAt: time.Now().UTC(), Status: "complete", Messages: []json.RawMessage{}, Caveat: "RPC observations are not proof of fresh exchange data or completed fills. Source timestamps retain RPC units (some depend on daemon settings). Streams are bounded samples. API keys, connection URLs and raw error detail are withheld. No historical cache is substituted and no failed command is retried."}
	var input callInput
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	request := dynamicpb.NewMessage(method.Input())
	if len(arguments) > 1<<20 || decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF || len(input.Request) == 0 || !strings.HasPrefix(strings.TrimSpace(string(input.Request)), "{") || protojson.Unmarshal(input.Request, request) != nil || input.MaxMessages < 0 || input.MaxMessages > b.options.MaxMessages || !method.IsStreamingServer() && input.MaxMessages != 0 {
		result.Status, result.Failure = statusRejected, errInvalidRequest.Error()
		return encodeResult(&result)
	}
	if err := b.authoriseMethod(name, request.ProtoReflect()); err != nil {
		result.Status, result.Failure = statusRejected, err.Error()
		return encodeResult(&result)
	}
	ctx, cancel := context.WithTimeout(ctx, b.options.Timeout)
	defer cancel()
	path := "/" + string(method.Parent().FullName()) + "/" + name
	var rpcErr error
	if !method.IsStreamingServer() {
		response := dynamicpb.NewMessage(method.Output())
		rpcErr = b.conn.Invoke(ctx, path, request, response, grpc.MaxCallRecvMsgSize(b.options.MaxResponseBytes))
		if rpcErr == nil {
			b.appendResponse(&result, response)
		}
	} else {
		var stream grpc.ClientStream
		stream, rpcErr = b.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, path, grpc.MaxCallRecvMsgSize(b.options.MaxResponseBytes))
		if rpcErr == nil {
			rpcErr = stream.SendMsg(request)
		}
		if rpcErr == nil {
			rpcErr = stream.CloseSend()
		}
		limit := input.MaxMessages
		if limit == 0 {
			limit = b.options.MaxMessages
		}
		for rpcErr == nil && len(result.Messages) < limit {
			response := dynamicpb.NewMessage(method.Output())
			rpcErr = stream.RecvMsg(response)
			if rpcErr == nil && !b.appendResponse(&result, response) {
				break
			}
		}
		if rpcErr == nil {
			result.Status, result.Truncated = "partial", true
		}
	}
	if rpcErr != nil && !errors.Is(rpcErr, io.EOF) {
		result.Status, result.Failure = statusUnknown, "rpc_"+status.Code(rpcErr).String()
		if len(result.Messages) > 0 {
			result.Status = "partial"
		}
	}
	return encodeResult(&result)
}

// appendResponse limits the aggregate JSON payload after removing sensitive fields.
func (b *Bridge) appendResponse(result *callResult, response *dynamicpb.Message) bool {
	data, err := protojson.MarshalOptions{UseProtoNames: true, EmitDefaultValues: true}.Marshal(response)
	if err != nil {
		result.Status, result.Failure = statusUnknown, "invalid_rpc_response"
		return false
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		result.Status, result.Failure = statusUnknown, "invalid_rpc_json"
		return false
	}
	sanitise(value, &result.ApplicationError)

	data, err = json.Marshal(value)
	if err != nil {
		result.Status, result.Failure = statusUnknown, "invalid_sanitised_response"
		return false
	}
	candidate := *result
	candidate.Messages = append(candidate.Messages, data)
	encoded, err := json.Marshal(encodeResult(&candidate))
	if err != nil {
		result.Status, result.Failure = statusUnknown, "invalid_sanitised_response"
		return false
	}
	// MCP includes both text and structured JSON. Leave space for final status changes.
	if len(encoded) > b.options.MaxResponseBytes-256 {
		result.Status, result.Failure, result.Truncated = "partial", "response_size_limit", true
		return false
	}
	result.Messages = candidate.Messages
	return true
}

// sanitise removes credential-bearing fields and marks in-message RPC errors.
func sanitise(value any, applicationError *bool) {
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			switch key {
			case "api_key", "http_proxy", "proxy_address", "running_url", "params", "otp_code", "otp_codes":
				delete(item, key)
				continue
			case "order_status":
				statuses, ok := child.(map[string]any)
				if !ok {
					item[key] = "cancellation_unconfirmed"
					*applicationError = true
					continue
				}
				for id, value := range statuses {
					reportedStatus, ok := value.(string)
					if ok {
						switch strings.ToUpper(strings.TrimSpace(reportedStatus)) {
						case "CANCELLED", "CANCELED", "SUCCESS", "OK", "TRUE":
							statuses[id] = "cancelled"
							continue
						}
					}
					statuses[id] = "cancellation_unconfirmed"
					*applicationError = true
				}
				continue
			case "error":
				if detail, ok := child.(string); ok && detail != "" {
					*applicationError = true
					item[key] = "RPC application error; detail withheld"
				}
			case "message":
				if detail, ok := child.(string); ok && detail != "" {
					item[key] = "RPC message withheld"
				}
			}
			sanitise(child, applicationError)
		}
	case []any:
		for _, child := range item {
			sanitise(child, applicationError)
		}
	}
}

// encodeResult supplies both MCP text and structured content from the same result.
func encodeResult(result *callResult) *mcp.CallToolResult {
	data, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Cannot encode RPC result"}}}
	}
	return &mcp.CallToolResult{IsError: result.Failure != "" || result.ApplicationError, StructuredContent: result, Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}
}
