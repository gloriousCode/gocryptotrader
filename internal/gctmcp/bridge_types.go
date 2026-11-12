// Package gctmcp exposes explicitly classified GCT RPC commands over MCP.
package gctmcp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"google.golang.org/grpc"
)

const (
	statusRejected    = "rejected"
	statusHistorical  = "historical"
	statusUnknown     = "unknown"
	statusUnavailable = "unavailable"
)

// Options fixes permissions and resource limits for the lifetime of a bridge.
type Options struct {
	Policy           config.MCPConfig
	Timeout          time.Duration
	MaxMessages      int
	MaxResponseBytes int
	DiagnosticsDir   string
	Offline          bool
}

// Bridge connects MCP tools to one operator-selected, authenticated RPC client.
type Bridge struct {
	conn    grpc.ClientConnInterface
	options Options
}

// callInput keeps stream limits separate from protobuf request fields.
type callInput struct {
	Request     json.RawMessage `json:"request"`
	MaxMessages int             `json:"maxMessages,omitempty"`
}

// callResult preserves partial observations without implying exchange freshness.
type callResult struct {
	Method           string            `json:"method"`
	CheckedAt        time.Time         `json:"checkedAt"`
	Status           string            `json:"status"`
	Failure          string            `json:"failure,omitempty"`
	Messages         []json.RawMessage `json:"messages"`
	Truncated        bool              `json:"truncated"`
	ApplicationError bool              `json:"applicationError"`
	Caveat           string            `json:"caveat"`
}
