package margin

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Margin types
const (
	UnsetMarginType   Type = 0
	SingleAssetMargin Type = 1 << iota
	MultiAssetMargin
	UnknownMarginType
)

var (
	// ErrUnknownMarginType returned when an unknown margin type is received
	ErrUnknownMarginType = errors.New("unknown margin type")
)

// Type helps identify the margin rules of a futures position
// or account, or anything.
// Different rules apply to different exchanges based on the type
type Type uint8

// RateHistoryRequest is used to request a funding rate
type RateHistoryRequest struct {
	Exchange           string
	Asset              asset.Item
	Currency           currency.Code
	StartDate          time.Time
	EndDate            time.Time
	GetPredictedRate   bool
	GetLendingPayments bool
	GetBorrowRates     bool
	GetBorrowCosts     bool

	// CalculateOffline allows for the borrow rate, lending payment amount
	// and borrow costs to be calculated offline. It requires the takerfeerate
	// and existing rates
	CalculateOffline bool
	TakeFeeRate      decimal.Decimal
	// Rates is used when calculating offline and determiningPayments
	// Each Rate must have the Rate and Size fields populated
	Rates []Rate
}

// RateHistoryResponse has the funding rate details
type RateHistoryResponse struct {
	Rates              []Rate
	SumBorrowCosts     decimal.Decimal
	AverageBorrowSize  decimal.Decimal
	SumLendingPayments decimal.Decimal
	AverageLendingSize decimal.Decimal
	PredictedRate      Rate
	TakerFeeRate       decimal.Decimal
}

// Rate has the funding rate details
// and optionally the borrow rate
type Rate struct {
	Time             time.Time
	MarketBorrowSize decimal.Decimal
	HourlyRate       decimal.Decimal
	YearlyRate       decimal.Decimal
	HourlyBorrowRate decimal.Decimal
	YearlyBorrowRate decimal.Decimal
	LendingPayment   LendingPayment
	BorrowCost       BorrowCost
}

// LendingPayment contains a lending rate payment
type LendingPayment struct {
	Payment decimal.Decimal
	Size    decimal.Decimal
}

// BorrowCost contains the borrow rate costs
type BorrowCost struct {
	Cost decimal.Decimal
	Size decimal.Decimal
}

// Requirements contains what is needed to open or hold a position
type Requirements struct {
	Exchange                     string
	Asset                        asset.Item
	Pair                         currency.Pair
	InitialMarginRequirement     decimal.Decimal
	MaintenanceMarginRequirement decimal.Decimal
	MaxLeverage                  float64
	CollateralScaling            decimal.Decimal
}
