// Command gctdiagnosticrun records bounded exit evidence for one explicitly launched process.
package main

import (
	"io"
	"time"
)

// crashRecord omits command arguments, environment values and raw output.
type crashRecord struct {
	ProcessID     int       `json:"processID,omitempty"`
	StartedAt     time.Time `json:"startedAt"`
	FinishedAt    time.Time `json:"finishedAt,omitzero"`
	ExitCode      *int      `json:"exitCode,omitempty"`
	PanicObserved bool      `json:"panicObserved"`
	Interrupted   bool      `json:"interrupted"`
	StartFailed   bool      `json:"startFailed"`
}

// panicWriter retains only a marker observation and a bounded marker-sized tail.
type panicWriter struct {
	output io.Writer
	tail   string
	seen   bool
}
