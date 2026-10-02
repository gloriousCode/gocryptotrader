package log

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

var (
	errCaptureCapacity  = errors.New("capture capacity must be between 1 and 10000")
	sensitiveLogContent = []string{"authorization", "password", "passwd", "secret", "token", "api_key", "apikey", "api-key", "api key", "signature", "cookie", "private key", "private_key", "privatekey", "pemkey", "otp", "credential", "bearer ", "-----begin", "://", "request body", "response body"}
)

// NewCapture creates a history with a fixed maximum memory footprint.
func NewCapture(capacity int) (*Capture, error) {
	if capacity < 1 || capacity > MaxCaptureCapacity {
		return nil, errCaptureCapacity
	}
	return &Capture{entries: make([]CapturedEntry, capacity)}, nil
}

// redactCapturedMessage suppresses credential-labelled messages and HTTP dumps.
// Unlabelled secrets cannot be recognised reliably; callers must never log them.
func redactCapturedMessage(message string) string {
	lower := strings.ToLower(message)
	for _, marker := range sensitiveLogContent {
		if strings.Contains(lower, marker) {
			return "[REDACTED: potentially sensitive diagnostic]"
		}
	}
	return strings.ToValidUTF8(strings.TrimSpace(message), "�")
}

// Record stores a sanitised message without retaining arbitrary structured fields.
// Request logs are excluded because HTTP payloads may contain unlabelled secrets.
func (c *Capture) Record(timestamp time.Time, severity, subsystem, message string) {
	if strings.EqualFold(subsystem, "REQUESTER") {
		return
	}
	message = redactCapturedMessage(message)
	truncated := len(message) > MaxCaptureMessageSize
	if truncated {
		message = strings.ToValidUTF8(message[:MaxCaptureMessageSize-3], "") + "..."
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.next++
	c.entries[(c.next-1)%uint64(len(c.entries))] = CapturedEntry{Sequence: c.next, Timestamp: timestamp.UTC(), Severity: strings.ToUpper(severity), Subsystem: subsystem, Message: message, Truncated: truncated}
	if c.size < uint64(len(c.entries)) {
		c.size++
	}
}

// Query filters the retained window and reports pagination and eviction evidence.
// Summaries aggregate all matching retained entries before limiting groups.
func (c *Capture) Query(q *CaptureQuery) CaptureResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := CaptureResponse{Enabled: true, Latest: c.next, Next: q.After}
	if c.size == 0 {
		return result
	}
	result.Oldest = c.next - c.size + 1
	result.Overwritten = result.Oldest - 1
	limit := q.Limit
	if limit == 0 {
		limit = 100
	}
	limit = min(limit, 200)
	groups := make(map[string]CaptureGroup)
	for seq := result.Oldest; seq <= c.next; seq++ {
		e := c.entries[(seq-1)%uint64(len(c.entries))]
		if seq <= q.After || (!q.Since.IsZero() && e.Timestamp.Before(q.Since)) || (!q.Until.IsZero() && e.Timestamp.After(q.Until)) || (q.Severity != "" && !strings.EqualFold(e.Severity, q.Severity)) || (q.Subsystem != "" && !strings.EqualFold(e.Subsystem, q.Subsystem)) || (q.Contains != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(q.Contains))) {
			continue
		}
		result.Matched++
		if q.Summary {
			key := fmt.Sprintf("%s\x00%s\x00%s", e.Severity, e.Subsystem, e.Message)
			g, exists := groups[key]
			if !exists {
				g = CaptureGroup{Severity: e.Severity, Subsystem: e.Subsystem, Message: e.Message, First: e.Timestamp}
			}
			g.Count++
			g.Last = e.Timestamp
			groups[key] = g
		} else if uint64(len(result.Entries)) < limit {
			result.Entries = append(result.Entries, e)
			result.Next = seq
		}
	}
	if q.Summary {
		result.Next = result.Latest
		for _, g := range groups {
			result.Groups = append(result.Groups, g)
		}
		slices.SortFunc(result.Groups, func(a, b CaptureGroup) int {
			if n := cmp.Compare(b.Count, a.Count); n != 0 {
				return n
			}
			if n := a.Last.Compare(b.Last); n != 0 {
				return -n
			}
			if n := cmp.Compare(a.Subsystem, b.Subsystem); n != 0 {
				return n
			}
			if n := cmp.Compare(a.Severity, b.Severity); n != 0 {
				return n
			}
			return cmp.Compare(a.Message, b.Message)
		})
		result.HasMore = uint64(len(result.Groups)) > limit
		result.Groups = result.Groups[:min(uint64(len(result.Groups)), limit)]
	} else {
		result.HasMore = result.Matched > uint64(len(result.Entries))
		if !result.HasMore {
			result.Next = max(result.Next, result.Latest)
		}
	}
	return result
}
