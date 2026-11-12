package engine

import (
	"slices"
	"strings"
	"time"
)

// diagnosticProgress reports cached sync attempts with their observation times.
func (m *SyncManager) diagnosticProgress(now time.Time) map[string]any {
	if m == nil {
		return map[string]any{"status": "unavailable", diagnosticScopeKey: "No sync manager was captured when this recorder started; other worker progress is unknown."}
	}
	if !m.mux.TryLock() {
		return map[string]any{"status": "busy", diagnosticScopeKey: "The startup sync-manager instance is busy. Native-market coverage and attempt progress are unknown until its lock is available."}
	}
	agents := make([]*currencyPairSyncAgent, 0, len(m.currencyPairs))
	for _, agent := range m.currencyPairs {
		agents = append(agents, agent)
	}
	m.mux.Unlock()
	slices.SortFunc(agents, func(a, b *currencyPairSyncAgent) int {
		return strings.Compare(a.Key.Exchange+":"+a.Key.Asset.String()+":"+a.Pair.String(), b.Key.Exchange+":"+b.Key.Asset.String()+":"+b.Pair.String())
	})
	rows := make([]map[string]any, 0, min(len(agents)*3, 200))
	total := 0
	threshold := max(time.Minute, 2*max(m.config.TimeoutREST, m.config.TimeoutWebsocket))
	for _, agent := range agents {
		for topic := SyncItemTicker; topic <= SyncItemTrade; topic++ {
			if !agent.locks[topic].TryLock() {
				total++
				if len(rows) < 200 {
					rows = append(rows, map[string]any{diagnosticExchangeKey: agent.Key.Exchange, diagnosticAssetKey: agent.Key.Asset.String(), "base": agent.Pair.Base.String(), "quote": agent.Pair.Quote.String(), "topic": topic.String(), "status": "busy", diagnosticScopeKey: "The tracker lock is busy. Attempt timestamps, error totals and whether a request is in flight are unknown."})
				}
				continue
			}
			tracker := agent.trackers[topic]
			if tracker == nil {
				agent.locks[topic].Unlock()
				continue
			}
			observation := *tracker
			agent.locks[topic].Unlock()
			total++
			if len(rows) == 200 {
				continue
			}
			protocol := "unknown"
			if observation.IsUsingWebsocket {
				protocol = "websocket"
			} else if observation.IsUsingREST {
				protocol = "rest"
			}
			row := map[string]any{diagnosticExchangeKey: agent.Key.Exchange, diagnosticAssetKey: agent.Key.Asset.String(), "base": agent.Pair.Base.String(), "quote": agent.Pair.Quote.String(), "topic": topic.String(), "status": "observed", "haveAttempted": observation.HaveData, "errorCount": observation.NumErrors, "protocol": protocol, "needsInspection": !observation.LastUpdated.IsZero() && now.Sub(observation.LastUpdated) > threshold && m.started.Load()}
			if !observation.LastUpdated.IsZero() {
				row["lastAttemptAt"] = observation.LastUpdated.UTC()
			}
			rows = append(rows, row)
		}
	}
	return map[string]any{"status": "observed", "running": m.started.Load(), "initialSyncStarted": m.initSyncStarted.Load(), "initialSyncCompleted": m.initSyncCompleted.Load(), "attempts": rows, "total": total, diagnosticTruncatedKey: total > len(rows), "inspectionThresholdSeconds": threshold.Seconds(), diagnosticScopeKey: "Startup sync-manager instance only; later replacements are unknown. Timestamps include failed attempts; errors are cumulative. Quote health and completed requests are not inferred. Overdue status is advisory."}
}
