package log

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	// MaxCaptureCapacity bounds memory retained for diagnostic queries.
	MaxCaptureCapacity = 10000
	// MaxCaptureMessageSize bounds each retained message in bytes.
	MaxCaptureMessageSize = 2048
)

// DiagnosticCapture is nil unless an explicitly enabled MCP endpoint owns a log history.
var DiagnosticCapture atomic.Pointer[Capture]

// Capture retains a bounded, process-local history independently of log output.
type Capture struct {
	mu      sync.RWMutex
	entries []CapturedEntry
	next    uint64
	size    uint64
}

// CapturedEntry identifies a log message in capture order.
type CapturedEntry struct {
	Sequence  uint64
	Timestamp time.Time
	Severity  string
	Subsystem string
	Message   string
	Truncated bool
}

// CaptureQuery selects retained messages; sequence cursors are exclusive.
type CaptureQuery struct {
	After     uint64
	Since     time.Time
	Until     time.Time
	Severity  string
	Subsystem string
	Contains  string
	Limit     uint64
	Summary   bool
}

// CaptureGroup counts identical messages within the complete retained query window.
type CaptureGroup struct {
	Severity  string
	Subsystem string
	Message   string
	Count     uint64
	First     time.Time
	Last      time.Time
}

// CaptureResponse reports retention gaps so consumers cannot mistake a partial history for a complete one.
type CaptureResponse struct {
	Enabled     bool
	Entries     []CapturedEntry
	Groups      []CaptureGroup
	Matched     uint64
	Next        uint64
	Oldest      uint64
	Latest      uint64
	Overwritten uint64
	HasMore     bool
}
