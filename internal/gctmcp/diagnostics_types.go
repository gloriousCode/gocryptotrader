package gctmcp

import "time"

// inspectionRequest selects bounded saved observations, never a file or URL.
type inspectionRequest struct {
	Exchange string `json:"exchange,omitempty" jsonschema:"Optional native exchange filter."`
	Asset    string `json:"asset,omitempty"    jsonschema:"Optional native asset filter."`
	Base     string `json:"base,omitempty"     jsonschema:"Optional native base currency filter."`
	Quote    string `json:"quote,omitempty"    jsonschema:"Optional native quote currency, retaining contract suffixes."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"Maximum market or exchange rows, from 1 to 200; default 50."`
	Offset   int    `json:"offset,omitempty"   jsonschema:"Non-negative offset within the filtered saved rows. Each read may observe a newer snapshot."`
}

// inspectionResult distinguishes the time of reading a record from its capture time.
type inspectionResult struct {
	Source     string    `json:"source"`
	ReadAt     time.Time `json:"readAt"`
	CapturedAt time.Time `json:"capturedAt,omitzero"`
	StartedAt  time.Time `json:"startedAt,omitzero"`
	RunID      string    `json:"runId,omitempty"`
	State      string    `json:"state,omitempty"`
	Status     string    `json:"status"`
	AgeSeconds float64   `json:"ageSeconds,omitempty"`
	Failure    string    `json:"failure,omitempty"`
	Truncated  bool      `json:"truncated"`
	Data       any       `json:"data,omitempty"`
	Caveat     string    `json:"caveat"`
}
