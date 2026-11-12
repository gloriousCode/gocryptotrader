package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/internal/diagnostics"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
)

const diagnosticShutdownTimeout = 5 * time.Second

// startDiagnostics starts optional journal collection with independent storage handling.
func (bot *Engine) startDiagnostics(ctx context.Context) {
	if bot == nil || bot.Config == nil || !bot.Settings.EnableDiagnostics || bot.Settings.DataDir == "" {
		return
	}
	waitCtx, cancel := context.WithTimeout(ctx, diagnosticShutdownTimeout)
	finished := bot.waitDiagnostics(waitCtx)
	cancel()
	if !finished {
		gctlog.Warnln(gctlog.Global, "Diagnostic recorder could not start because the previous recorder has not released ownership.")
		return
	}
	bot.runtimeMu.Lock()
	if bot.diagnosticsDone != nil {
		select {
		case <-bot.diagnosticsDone:
		default:
			bot.runtimeMu.Unlock()
			gctlog.Warnln(gctlog.Global, "Diagnostic recorder startup skipped because another recorder is active.")
			return
		}
	}
	done := make(chan struct{})
	bot.diagnosticsDone = done
	bot.runtimeMu.Unlock()
	settings := bot.Settings
	startedAt := bot.uptime.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	observed := &Engine{ExchangeManager: bot.ExchangeManager, currencyPairSyncer: bot.currencyPairSyncer, DatabaseManager: bot.DatabaseManager}
	go func() {
		defer close(done)
		observed.recordDiagnostics(ctx, filepath.Join(settings.DataDir, "diagnostics"), &settings, startedAt)
	}()
}

// waitDiagnostics joins recorder cleanup without making daemon shutdown unbounded.
func (bot *Engine) waitDiagnostics(ctx context.Context) bool {
	if bot == nil {
		return true
	}
	bot.runtimeMu.RLock()
	done := bot.diagnosticsDone
	bot.runtimeMu.RUnlock()
	if done == nil {
		return true
	}
	select {
	case <-done:
	default:
		select {
		case <-done:
		case <-ctx.Done():
			return false
		}
	}
	bot.runtimeMu.Lock()
	if bot.diagnosticsDone == done {
		bot.diagnosticsDone = nil
	}
	bot.runtimeMu.Unlock()
	return true
}

// recordDiagnostics has one writer and releases its log observer when its runtime ends.
func (bot *Engine) recordDiagnostics(ctx context.Context, directory string, settings *Settings, startedAt time.Time) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		gctlog.Warnln(gctlog.Global, "Diagnostic recorder could not establish its run identity; recording is unavailable.")
		return
	}
	runID := hex.EncodeToString(random[:])
	owner, err := acquireDiagnosticOwner(directory, runID)
	if err != nil {
		gctlog.Warnln(gctlog.Global, "Diagnostic recorder could not claim its journal; recording is unavailable.")
		return
	}
	defer owner.release()
	path := filepath.Join(directory, "runtime.json")
	if previous, err := diagnostics.Read(path); err == nil {
		if err := diagnostics.Write(filepath.Join(directory, "previous.json"), previous); err != nil {
			gctlog.Warnln(gctlog.Global, "Diagnostic recorder could not retain previous-run evidence; the existing journal has been preserved.")
			return
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		gctlog.Warnln(gctlog.Global, "Diagnostic recorder could not validate previous-run evidence; the existing journal has been preserved.")
		return
	}
	events := new(diagnosticEvents)
	id := gctlog.AddObserver(events.observe)
	defer gctlog.RemoveObserver(id)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	startupFingerprint := ""
	writeFailed := false
	var stackLocations []string
	var stackCapturedAt time.Time
	var stackTruncated bool
	for {
		state := "running"
		if ctx.Err() != nil {
			state = "stopping"
		}
		began := time.Now().UTC()
		if stackCapturedAt.IsZero() || began.Sub(stackCapturedAt) >= time.Minute {
			stackLocations, stackTruncated = diagnosticStackLocations()
			stackCapturedAt = began
		}
		snapshot := bot.diagnosticSnapshot(runID, startedAt, began, state, settings, events)
		configuration, _ := snapshot.Views["configuration"].(map[string]any)
		if startupFingerprint == "" {
			startupFingerprint, _ = configuration["currentFingerprint"].(string)
		}
		configuration["startupFingerprint"] = startupFingerprint
		snapshot.Views["runtime_health"] = map[string]any{"heartbeatAt": began, "collectionDurationSeconds": time.Since(began).Seconds(), "previousWriteFailed": writeFailed, diagnosticScopeKey: "Recorder and sync-attempt progress only; other workers are unknown. Stack locations are bounded, sampled at most once per minute and independently timestamped. Stopping means runtime cancellation, not completed shutdown.", "stackLocations": stackLocations, "stackCapturedAt": stackCapturedAt, "stackTruncated": stackTruncated, "workerProgress": bot.currencyPairSyncer.diagnosticProgress(began)}
		failed := diagnostics.Write(path, snapshot) != nil
		if failed && !writeFailed {
			gctlog.Warnln(gctlog.Global, "Diagnostic recorder could not publish its journal; saved observations may be stale.")
		} else if !failed && writeFailed {
			gctlog.Infoln(gctlog.Global, "Diagnostic journal publication resumed.")
		}
		writeFailed = failed
		if state == "stopping" {
			return
		}
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
}

// observe classifies headers only; raw messages never enter the diagnostic journal.
func (events *diagnosticEvents) observe(header, subsystem string) {
	var category string
	switch upper := strings.ToUpper(header); {
	case strings.Contains(upper, "ERROR"), strings.Contains(upper, "FATAL"):
		category = "error_observed"
	case strings.Contains(upper, "WARN"):
		category = "warning_observed"
	default:
		return
	}
	if len(subsystem) > 48 || strings.Trim(subsystem, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 _-") != "" {
		subsystem = "other"
	}
	now := time.Now().UTC()
	events.mu.Lock()
	defer events.mu.Unlock()
	for i := range events.rows {
		if events.rows[i].Subsystem == subsystem && events.rows[i].Category == category {
			events.rows[i].LastAt = now
			events.rows[i].Count++
			return
		}
	}
	if len(events.rows) == 128 {
		events.rows = events.rows[1:]
		events.evicted++
	}
	events.rows = append(events.rows, diagnosticEvent{Subsystem: subsystem, Category: category, FirstAt: now, LastAt: now, Count: 1})
}

// diagnosticStackLocations removes argument values and filesystem directories from stacks.
func diagnosticStackLocations() ([]string, bool) {
	buffer := make([]byte, 64<<10)
	n := runtime.Stack(buffer, true)
	truncated := n == len(buffer)
	locations := make([]string, 0, 64)
	for line := range strings.SplitSeq(string(buffer[:n]), "\n") {
		if len(locations) == 64 {
			truncated = true
			break
		}
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ".go:") {
			continue
		}
		location, _, _ := strings.Cut(line, " ")
		locations = append(locations, filepath.Base(location))
	}
	return locations, truncated
}

// diagnosticPerformance uses runtime counters without starting a profiling session.
func diagnosticPerformance() map[string]any {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	samples := []metrics.Sample{{Name: "/cpu/classes/total:cpu-seconds"}}
	metrics.Read(samples)
	performance := map[string]any{diagnosticScopeKey: "Runtime CPU classes estimate available CPU budget, not OS CPU usage.", "goroutines": runtime.NumGoroutine(), "heapBytes": memory.HeapAlloc, "heapObjects": memory.HeapObjects, "gcCycles": memory.NumGC, "gcPauseTotalSeconds": float64(memory.PauseTotalNs) / float64(time.Second), "logicalCPUs": runtime.NumCPU()}
	if samples[0].Value.Kind() == metrics.KindFloat64 {
		performance["availableRuntimeCPUSeconds"] = samples[0].Value.Float64()
	}
	return performance
}

// diagnosticBuildIdentity reports embedded executable metadata, never the current checkout.
func diagnosticBuildIdentity() map[string]any {
	identity := map[string]any{"goVersion": runtime.Version(), "revision": "unknown", "dirty": "unknown"}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				identity["revision"] = setting.Value
			case "vcs.modified":
				identity["dirty"] = setting.Value
			}
		}
	}
	return identity
}
