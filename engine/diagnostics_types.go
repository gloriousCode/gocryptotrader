package engine

import (
	"sync"
	"time"
)

const (
	diagnosticScopeKey     = "scope"
	diagnosticExchangeKey  = "exchange"
	diagnosticAssetKey     = "asset"
	diagnosticTruncatedKey = "truncated"
)

// diagnosticEvents retains metadata-only error history independently of MCP sessions.
type diagnosticEvents struct {
	mu      sync.Mutex
	rows    []diagnosticEvent
	evicted uint64
}

// diagnosticEvent coalesces identical metadata without retaining log arguments.
type diagnosticEvent struct {
	Subsystem string    `json:"subsystem"`
	Category  string    `json:"category"`
	FirstAt   time.Time `json:"firstAt"`
	LastAt    time.Time `json:"lastAt"`
	Count     uint64    `json:"count"`
}

// diagnosticOwner identifies the one process permitted to publish this journal.
type diagnosticOwner struct {
	Path  string
	Token string
}
