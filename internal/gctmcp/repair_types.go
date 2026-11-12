package gctmcp

import "time"

// repairRequest identifies one exact, deliberately requested maintenance operation.
type repairRequest struct {
	ID       string `json:"id"              jsonschema:"Unique 8 to 64 lowercase letters, digits or hyphens; reuse exactly after an uncertain submission."`
	Action   string `json:"action"          jsonschema:"refresh_balances or refresh_pairs."`
	Exchange string `json:"exchange"`
	Asset    string `json:"asset,omitempty"`
	Reason   string `json:"reason"          jsonschema:"stale_data or retrieval_failed."`
}

// repairRecord is an owner-only receipt; it never stores balances or raw RPC errors.
type repairRecord struct {
	Version   int           `json:"version"`
	Request   repairRequest `json:"request"`
	StartedAt time.Time     `json:"startedAt"`
	EndedAt   time.Time     `json:"endedAt,omitzero"`
	Status    string        `json:"status"`
	Failure   string        `json:"failure,omitempty"`
}

// repairStatusRequest selects an exact receipt or the bounded saved ledger.
type repairStatusRequest struct {
	ID string `json:"id,omitempty"`
}

// repairResult retains uncertainty rather than silently repeating a mutation.
type repairResult struct {
	Status  string         `json:"status"`
	Failure string         `json:"failure,omitempty"`
	Records []repairRecord `json:"records"`
	Caveat  string         `json:"caveat"`
}
