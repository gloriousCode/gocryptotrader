package log

import (
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCapture(t *testing.T) {
	t.Parallel()
	for _, capacity := range []int{-1, 0, 1, MaxCaptureCapacity, MaxCaptureCapacity + 1} {
		t.Run(strconv.Itoa(capacity), func(t *testing.T) {
			t.Parallel()
			c, err := NewCapture(capacity)
			if capacity < 1 || capacity > MaxCaptureCapacity {
				require.ErrorIs(t, err, errCaptureCapacity, "invalid capacity must fail")
				return
			}
			require.NoError(t, err, "valid capacity must succeed")
			assert.Len(t, c.entries, capacity, "history should have a fixed capacity")
		})
	}
}

func TestRedactCapturedMessage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ input, expected string }{
		{" reconnect failed ", "reconnect failed"},
		{"Authorization: Basic sensitive", "[REDACTED: potentially sensitive diagnostic]"},
		{"{\"apiKey\":\"sensitive\"}", "[REDACTED: potentially sensitive diagnostic]"},
		{"PASSWORD=sensitive", "[REDACTED: potentially sensitive diagnostic]"},
		{"wss://user:sensitive@example.com", "[REDACTED: potentially sensitive diagnostic]"},
		{"-----BEGIN PRIVATE KEY-----\nsensitive", "[REDACTED: potentially sensitive diagnostic]"},
		{"bad\xff", "bad�"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, redactCapturedMessage(tc.input), "capture should sanitise sensitive content")
		})
	}
}

func TestRecord(t *testing.T) {
	t.Parallel()
	c, err := NewCapture(2)
	require.NoError(t, err, "capture must initialise")
	now := time.Now()
	c.Record(now, "error", "REQUESTER", "unlabelled request body")
	assert.Zero(t, c.Query(&CaptureQuery{}).Latest, "request diagnostics should be omitted")
	c.Record(now, "error", "WEBSOCKET", "first")
	c.Record(now, "warn", "WEBSOCKET", strings.Repeat("界", 3000))
	c.Record(now, "info", "GLOBAL", "third")
	result := c.Query(&CaptureQuery{})
	require.Len(t, result.Entries, 2, "ring must evict the oldest entry")
	assert.Equal(t, uint64(1), result.Overwritten, "evictions should be reported")
	assert.Equal(t, "WARN", result.Entries[0].Severity, "severity should be canonical")
	assert.True(t, result.Entries[0].Truncated, "long messages should be marked truncated")
	assert.LessOrEqual(t, len(result.Entries[0].Message), MaxCaptureMessageSize, "message should be bounded")
	assert.True(t, utf8.ValidString(result.Entries[0].Message), "truncation should preserve valid UTF8")
	assert.Equal(t, time.UTC, result.Entries[0].Timestamp.Location(), "timestamps should use UTC")
}

func TestQuery(t *testing.T) {
	t.Parallel()
	base := time.Unix(1000, 0).UTC()
	for _, tc := range []struct {
		name    string
		query   CaptureQuery
		matched uint64
		entries int
		groups  int
		next    uint64
		more    bool
	}{
		{name: "empty filter", matched: 4, entries: 4, next: 4},
		{name: "cursor", query: CaptureQuery{After: 2}, matched: 2, entries: 2, next: 4},
		{name: "page", query: CaptureQuery{Limit: 1}, matched: 4, entries: 1, next: 1, more: true},
		{name: "severity", query: CaptureQuery{Severity: "error"}, matched: 3, entries: 3, next: 4},
		{name: "subsystem", query: CaptureQuery{Subsystem: "global"}, matched: 1, entries: 1, next: 4},
		{name: "contains", query: CaptureQuery{Contains: "FAIL"}, matched: 3, entries: 3, next: 4},
		{name: "time bounds", query: CaptureQuery{Since: base.Add(time.Second), Until: base.Add(2 * time.Second)}, matched: 2, entries: 2, next: 4},
		{name: "summary whole window", query: CaptureQuery{Summary: true, Limit: 1}, matched: 4, groups: 1, next: 4, more: true},
		{name: "summary all groups", query: CaptureQuery{Summary: true}, matched: 4, groups: 2, next: 4},
		{name: "no matches", query: CaptureQuery{Contains: "missing"}, next: 4},
		{name: "future cursor", query: CaptureQuery{After: 9}, next: 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := NewCapture(4)
			require.NoError(t, err, "capture must initialise")
			for i := range 4 {
				severity, subsystem, message := "error", "WEBSOCKET", "failed"
				if i == 3 {
					severity, subsystem, message = "info", "GLOBAL", "recovered"
				}
				c.Record(base.Add(time.Duration(i)*time.Second), severity, subsystem, message)
			}
			result := c.Query(&tc.query)
			assert.Equal(t, tc.matched, result.Matched, "query should report all matches")
			assert.Len(t, result.Entries, tc.entries, "entries should respect filters and page size")
			assert.Len(t, result.Groups, tc.groups, "groups should respect limits")
			assert.Equal(t, tc.next, result.Next, "cursor should allow incremental reads")
			assert.Equal(t, tc.more, result.HasMore, "truncation should be explicit")
			if tc.groups > 0 {
				assert.Equal(t, uint64(3), result.Groups[0].Count, "summary should count the full window")
				assert.Equal(t, base, result.Groups[0].First, "summary should retain first occurrence")
				assert.Equal(t, base.Add(2*time.Second), result.Groups[0].Last, "summary should retain last occurrence")
			}
		})
	}
	t.Run("empty history", func(t *testing.T) {
		t.Parallel()
		c, err := NewCapture(1)
		require.NoError(t, err, "capture must initialise")
		assert.Zero(t, c.Query(&CaptureQuery{}).Matched, "empty history should have no matches")
	})
	t.Run("concurrent writers", func(t *testing.T) {
		t.Parallel()
		c, err := NewCapture(10)
		require.NoError(t, err, "capture must initialise")
		var wg sync.WaitGroup
		for range 4 {
			wg.Go(func() {
				for range 100 {
					c.Record(base, "info", "GLOBAL", "concurrent")
					_ = c.Query(&CaptureQuery{Summary: true})
				}
			})
		}
		wg.Wait()
		result := c.Query(&CaptureQuery{})
		assert.Equal(t, uint64(400), result.Latest, "concurrent capture should preserve sequence counts")
		assert.Equal(t, uint64(390), result.Overwritten, "concurrent capture should remain bounded")
	})
}

func TestDiagnosticCapture(t *testing.T) {
	original := DiagnosticCapture.Load()
	t.Cleanup(func() { DiagnosticCapture.Store(original) })
	DiagnosticCapture.Store(nil)
	assert.Nil(t, DiagnosticCapture.Load(), "disabled capture should return nil")
	c, err := NewCapture(1)
	require.NoError(t, err, "capture must initialise")
	DiagnosticCapture.Store(c)
	assert.Same(t, c, DiagnosticCapture.Load(), "capture should be installed")
	DiagnosticCapture.Store(nil)
	assert.Nil(t, DiagnosticCapture.Load(), "disabled capture should release history")
}

func TestLoggerWorkerCapturesEmittedMessages(t *testing.T) {
	original := DiagnosticCapture.Load()
	t.Cleanup(func() { DiagnosticCapture.Store(original) })
	c, err := NewCapture(2)
	require.NoError(t, err, "capture must initialise")
	sl, cleanup := benchmarkLoggerState(true, Levels{Info: true, Error: true}, nil, io.Discard)
	defer cleanup()
	DiagnosticCapture.Store(c)
	Infof(sl, "capture %s", "message")
	Errorln(sl, "password=sensitive")
	ch := make(chan struct{})
	jobsChannel <- &job{Passback: ch}
	<-ch
	result := c.Query(&CaptureQuery{})
	require.Len(t, result.Entries, 2, "worker must capture emitted messages")
	assert.Equal(t, "capture message", result.Entries[0].Message, "worker should preserve formatted message")
	assert.Equal(t, "[REDACTED: potentially sensitive diagnostic]", result.Entries[1].Message, "worker should redact before storage")
}
