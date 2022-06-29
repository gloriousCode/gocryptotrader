package fundingrate

import (
	"errors"
	"time"
)

var (
	errInvalidInput = errors.New("exchange, asset, currency, start & end cannot be empty")
	errNoCandleData = errors.New("no funding rate data provided")
	// ErrNoCandleDataFound returns when no candle data is found
	ErrNoCandleDataFound = errors.New("no funding rate data found")
)

// RatesHolder generic funding rate holder for modelPSQL & modelSQLite
type RatesHolder struct {
	ID          string
	ExchangeID  string
	Asset       string
	Currency    string
	SourceJobID string
	Rates       []Rate
}

// Rate holds each funding rate
type Rate struct {
	Timestamp        time.Time
	Rate             float64
	ValidationJobID  string
	ValidationIssues string
}
