package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/internal/diagnostics"
)

// diagnosticSnapshot only reads cached public market data and thread-safe manager getters.
func (bot *Engine) diagnosticSnapshot(runID string, startedAt, capturedAt time.Time, state string, settings *Settings, events *diagnosticEvents) *diagnostics.Snapshot {
	views := bot.diagnosticMarketViews()
	configuration, _ := views["configuration"].(map[string]any)
	configuration["startupSettings"] = map[string]any{"grpc": settings.EnableGRPC, "grpcProxy": settings.EnableGRPCProxy, "orderManager": settings.EnableOrderManager, "historyManager": settings.EnableDataHistoryManager, "websocketManager": settings.EnableWebsocketRoutine, "tickerSync": settings.EnableTickerSyncing, "orderbookSync": settings.EnableOrderbookSyncing, "tradeSync": settings.EnableTradeSyncing, "dryRun": settings.EnableDryRun}
	configuration[diagnosticScopeKey] = "Startup settings and current thread-safe exchange/pair observations. Fingerprints cover this sanitised subset, not credentials or every configuration value."
	if encoded, err := json.Marshal(configuration); err == nil {
		hash := sha256.Sum256(encoded)
		configuration["currentFingerprint"] = hex.EncodeToString(hash[:])
	}
	identity := diagnosticBuildIdentity()
	identity["processId"] = os.Getpid()
	identity["startedAt"] = startedAt
	views["identity"] = identity
	views["performance"] = diagnosticPerformance()
	views["storage"] = map[string]any{"databaseConfiguredAtStartup": settings.EnableDatabaseManager, "databaseConnected": bot.DatabaseManager.IsConnected(), "journalRetention": "latest and previous engine run", "eventGroupLimit": 128, "marketLimit": 200, "bookDepthLimit": 20, "credentialsIncluded": false, diagnosticScopeKey: "The database manager instance captured at recorder startup; later replacements are unknown."}
	views["permissions"] = map[string]any{"observedAtStartup": startedAt, "grpcEnabledAtStartup": settings.EnableGRPC, "grpcShutdownAllowedAtStartup": settings.EnableGRPCShutdown, "orderManagerConfiguredAtStartup": settings.EnableOrderManager, "dryRunAtStartup": settings.EnableDryRun, "tradeAuthorisation": "unknown", diagnosticScopeKey: "These settings do not authorise orders, prove exchange permissions or describe later changes by other account users."}
	events.mu.Lock()
	rows := slices.Clone(events.rows)
	evicted := events.evicted
	events.mu.Unlock()
	views["events"] = map[string]any{"groups": rows, "evictedGroups": evicted, diagnosticScopeKey: "Metadata-only warning/error observations emitted through enabled GCT loggers. Counts are per retained group since its first observation, not a complete error ledger or exact time-window totals."}
	views["run_report"] = map[string]any{"startedAt": startedAt, "capturedAt": capturedAt, "state": state, "eventGroups": len(rows), "evictedEventGroups": evicted, "marketCoverage": views["market_health"], "configurationFingerprint": configuration["currentFingerprint"], "workerProgress": bot.currencyPairSyncer.diagnosticProgress(capturedAt), diagnosticScopeKey: "Saved recorder observations. Healthy recorder progress does not establish healthy workers, complete exchange coverage or trading readiness."}
	return &diagnostics.Snapshot{Version: 1, RunID: runID, StartedAt: startedAt, CapturedAt: capturedAt, State: state, Views: views}
}

// diagnosticMarketViews bounds the retained universe without making exchange API requests.
func (bot *Engine) diagnosticMarketViews() map[string]any {
	var exchanges []exchange.IBotExchange
	if bot != nil && bot.ExchangeManager != nil {
		exchanges, _ = bot.ExchangeManager.GetExchanges()
	}
	slices.SortFunc(exchanges, func(a, b exchange.IBotExchange) int { return strings.Compare(a.GetName(), b.GetName()) })
	configurations := make([]map[string]any, 0, min(len(exchanges), 100))
	connections := make([]map[string]any, 0, min(len(exchanges), 100))
	health := make([]map[string]any, 0, 200)
	snapshots := make([]map[string]any, 0, 200)
	metadata := make([]map[string]any, 0, 200)
	totalMarkets, totalMetadata := 0, 0
	for _, exch := range exchanges {
		name := exch.GetName()
		assets := exch.GetAssetTypes(false)
		slices.Sort(assets)
		assetRows := make([]map[string]any, 0, len(assets))
		for _, a := range assets {
			available, availableErr := exch.GetAvailablePairs(a)
			enabled, enabledErr := exch.GetEnabledPairs(a)
			slices.SortFunc(enabled, func(a, b currency.Pair) int { return strings.Compare(a.String(), b.String()) })
			assetRows = append(assetRows, map[string]any{diagnosticAssetKey: a.String(), "availablePairs": len(available), "availableKnown": availableErr == nil, "enabledPairs": len(enabled), "enabledKnown": enabledErr == nil})
			totalMetadata += len(available)
			enabledPairs := make(map[[2]*currency.Item]bool, len(enabled))
			for _, pair := range enabled {
				enabledPairs[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}] = true
			}
			slices.SortFunc(available, func(a, b currency.Pair) int { return strings.Compare(a.String(), b.String()) })
			for _, pair := range available {
				if len(metadata) == 200 {
					break
				}
				metadata = append(metadata, map[string]any{diagnosticExchangeKey: name, diagnosticAssetKey: a.String(), "base": pair.Base.String(), "quote": pair.Quote.String(), "delimiter": pair.Delimiter, "enabled": enabledPairs[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}], "contractMetadataStatus": "unavailable"})
			}
			totalMarkets += len(enabled)
			for _, pair := range enabled {
				if len(health) == 200 {
					break
				}
				summary := map[string]any{diagnosticExchangeKey: name, diagnosticAssetKey: a.String(), "base": pair.Base.String(), "quote": pair.Quote.String(), "tickerAvailable": false, "orderbookAvailable": false}
				frame := map[string]any{diagnosticExchangeKey: name, diagnosticAssetKey: a.String(), "base": pair.Base.String(), "quote": pair.Quote.String()}
				if price, err := ticker.GetTicker(name, pair, a); err == nil {
					summary["tickerAvailable"], summary["tickerAt"] = true, price.LastUpdated.UTC()
					frame["tickerAt"], frame["last"], frame["bid"], frame["ask"] = price.LastUpdated.UTC(), price.Last, price.Bid, price.Ask
					frame["supplementaryValuesStatus"] = "positive_cached_values_only"
					if price.MarkPrice > 0 {
						frame["markPrice"] = price.MarkPrice
					}
					if price.IndexPrice > 0 {
						frame["indexPrice"] = price.IndexPrice
					}
					if price.OpenInterest > 0 {
						frame["openInterest"], frame["openInterestUnit"] = price.OpenInterest, "native_unknown"
					}
				}
				if depth, err := orderbook.GetDepth(name, pair, a); err == nil {
					if book, err := depth.RetrieveLimited(20); err == nil {
						summary["orderbookAvailable"], summary["orderbookAt"], summary["receivedAt"] = true, book.LastUpdated.UTC(), book.InsertedAt.UTC()
						summary["bidLevels"], _ = depth.GetBidLength()
						summary["askLevels"], _ = depth.GetAskLength()
						summary["restSnapshot"] = book.RestSnapshot
						frame["orderbookAt"], frame["receivedAt"] = book.LastUpdated.UTC(), book.InsertedAt.UTC()
						for _, side := range []struct {
							name   string
							levels orderbook.Levels
						}{{name: "bids", levels: book.Bids}, {name: "asks", levels: book.Asks}} {
							levels := make([]map[string]float64, 0, len(side.levels))
							for _, level := range side.levels {
								levels = append(levels, map[string]float64{"price": level.Price, "amount": level.Amount})
							}
							frame[side.name] = levels
						}
					}
				}
				health, snapshots = append(health, summary), append(snapshots, frame)
			}
		}
		if len(configurations) < 100 {
			configurations = append(configurations, map[string]any{diagnosticExchangeKey: name, "enabled": exch.IsEnabled(), "assets": assetRows})
		}
		if len(connections) < 100 {
			connection := map[string]any{diagnosticExchangeKey: name, "supported": false}
			if websocket, err := exch.GetWebsocket(); err == nil && websocket != nil {
				connection["supported"], connection["enabled"], connection["connected"], connection["connecting"], connection["subscriptions"] = true, websocket.IsEnabled(), websocket.IsConnected(), websocket.IsConnecting(), len(websocket.GetSubscriptions())
			}
			connections = append(connections, connection)
		}
	}
	return map[string]any{
		"configuration":       map[string]any{"exchanges": configurations, "totalExchanges": len(exchanges), diagnosticTruncatedKey: len(exchanges) > len(configurations)},
		"websocket_health":    map[string]any{"exchanges": connections, "totalExchanges": len(exchanges), diagnosticTruncatedKey: len(exchanges) > len(connections)},
		"market_health":       map[string]any{"markets": health, "total": totalMarkets, diagnosticTruncatedKey: totalMarkets > len(health), diagnosticScopeKey: "Enabled native pairs and cached source timestamps. Missing data stays unavailable; an old timestamp alone does not identify a disconnected exchange."},
		"market_snapshots":    map[string]any{"markets": snapshots, "total": totalMarkets, diagnosticTruncatedKey: totalMarkets > len(snapshots), "depthLimit": 20, diagnosticScopeKey: "Saved native public observations, not fills or executable capacity. Missing mark/open-interest values and native unit conversions are not inferred."},
		"instrument_metadata": map[string]any{"markets": metadata, "total": totalMetadata, diagnosticTruncatedKey: totalMetadata > len(metadata), diagnosticScopeKey: "Supported native identifiers and enabled state. Contract multipliers, settlement rules and quantity rules are unavailable in this general cache and remain unknown."},
	}
}
