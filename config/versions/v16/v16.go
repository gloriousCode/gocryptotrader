// Package v16 adds optional websocket metrics logging to exchange configurations.
package v16

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version implements ExchangeVersion to add websocket metrics logging.
type Version struct{}

// Exchanges returns all exchanges.
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange adds websocket metrics logging when it is not configured.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	if len(exchange) == 0 {
		return exchange, nil
	}
	if _, _, _, err := jsonparser.Get(exchange, "websocketMetricsLogging"); err == nil {
		return exchange, nil
	}
	return jsonparser.Set(exchange, []byte("false"), "websocketMetricsLogging")
}

// DowngradeExchange removes websocket metrics logging.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return jsonparser.Delete(exchange, "websocketMetricsLogging"), nil
}
