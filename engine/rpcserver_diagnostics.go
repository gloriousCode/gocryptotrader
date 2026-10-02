package engine

import (
	"context"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetDiagnosticLogs exposes only the opt-in retained history, never arbitrary log files.
func (*RPCServer) GetDiagnosticLogs(_ context.Context, r *gctrpc.GetDiagnosticLogsRequest) (*gctrpc.GetDiagnosticLogsResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	q := &log.CaptureQuery{After: r.AfterSequence, Severity: r.Severity, Subsystem: r.Subsystem, Contains: r.Contains, Limit: r.Limit, Summary: r.Summary}
	if r.Since != nil {
		if err := r.Since.CheckValid(); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid since timestamp")
		}
		q.Since = r.Since.AsTime()
	}
	if r.Until != nil {
		if err := r.Until.CheckValid(); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid until timestamp")
		}
		q.Until = r.Until.AsTime()
	}
	if (!q.Since.IsZero() && !q.Until.IsZero() && q.Since.After(q.Until)) || r.Limit > 200 || len(r.Contains) > 256 || len(r.Subsystem) > 128 {
		return nil, status.Error(codes.InvalidArgument, "invalid diagnostic query bounds")
	}
	switch strings.ToUpper(r.Severity) {
	case "", "INFO", "WARN", "ERROR", "DEBUG":
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid severity")
	}
	capture := log.DiagnosticCapture.Load()
	if capture == nil {
		return &gctrpc.GetDiagnosticLogsResponse{}, nil
	}
	result := capture.Query(q)
	response := &gctrpc.GetDiagnosticLogsResponse{Enabled: result.Enabled, Matched: result.Matched, NextSequence: result.Next, OldestSequence: result.Oldest, LatestSequence: result.Latest, Overwritten: result.Overwritten, HasMore: result.HasMore}
	for _, e := range result.Entries {
		response.Entries = append(response.Entries, &gctrpc.DiagnosticLogEntry{Sequence: e.Sequence, Timestamp: timestamppb.New(e.Timestamp), Severity: e.Severity, Subsystem: e.Subsystem, Message: e.Message, Truncated: e.Truncated})
	}
	for _, g := range result.Groups {
		response.Groups = append(response.Groups, &gctrpc.DiagnosticLogGroup{Severity: g.Severity, Subsystem: g.Subsystem, Message: g.Message, Count: g.Count, First: timestamppb.New(g.First), Last: timestamppb.New(g.Last)})
	}
	return response, nil
}
