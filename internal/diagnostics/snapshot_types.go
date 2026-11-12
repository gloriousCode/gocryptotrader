// Package diagnostics stores bounded, credential-free observations independently of RPC.
package diagnostics

import "time"

// Snapshot identifies saved engine observations by run and capture time.
type Snapshot struct {
	Version    int            `json:"version"`
	RunID      string         `json:"runId"`
	StartedAt  time.Time      `json:"startedAt"`
	CapturedAt time.Time      `json:"capturedAt"`
	State      string         `json:"state"`
	Views      map[string]any `json:"views"`
}
