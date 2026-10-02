// Package mcpserver exposes a curated read-only interface to GCT's existing RPC handlers.
package mcpserver

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/gctrpc"
)

type adapter struct {
	backend           gctrpc.GoCryptoTraderServiceServer
	timeout           time.Duration
	timeInNanoSeconds bool
}

// ToolInput supports bounded queries without exposing arbitrary RPC methods.
type ToolInput struct {
	OrderID          string `json:"order_id,omitempty"          jsonschema:"Exchange order identifier for get_order"`
	Offset           uint64 `json:"offset,omitempty"            jsonschema:"Offset into an orders or account balance snapshot; default 0"`
	UnderlyingBase   string `json:"underlying_base,omitempty"   jsonschema:"Optional underlying base currency for futures positions"`
	UnderlyingQuote  string `json:"underlying_quote,omitempty"  jsonschema:"Optional underlying quote currency for futures positions"`
	Asset            string `json:"asset,omitempty"             jsonschema:"Market asset type such as spot or futures"`
	Base             string `json:"base,omitempty"              jsonschema:"Base currency code, for example BTC"`
	Quote            string `json:"quote,omitempty"             jsonschema:"Quote currency code, for example USDT"`
	Depth            uint64 `json:"depth,omitempty"             jsonschema:"Maximum order-book levels per side, 1 to 100; default 20"`
	IncludePredicted bool   `json:"include_predicted,omitempty" jsonschema:"Include predicted funding separately from the latest rate"`
	Exchange         string `json:"exchange,omitempty"          jsonschema:"Exchange name for inspection and market queries"`
	Enabled          bool   `json:"enabled,omitempty"           jsonschema:"Only enabled exchanges when true"`
	AfterSequence    uint64 `json:"after_sequence,omitempty"    jsonschema:"Exclusive log cursor; resets on process restart"`
	Since            string `json:"since,omitempty"             jsonschema:"Inclusive RFC3339 timestamp"`
	Until            string `json:"until,omitempty"             jsonschema:"Inclusive RFC3339 timestamp"`
	Severity         string `json:"severity,omitempty"          jsonschema:"INFO WARN ERROR or DEBUG; omitted selects all"`
	Subsystem        string `json:"subsystem,omitempty"         jsonschema:"Exact subsystem name, case insensitive"`
	Contains         string `json:"contains,omitempty"          jsonschema:"Case insensitive message substring"`
	Limit            uint64 `json:"limit,omitempty"             jsonschema:"Maximum entries or summary groups, 1 to 200; default 100"`
}
