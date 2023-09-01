package fundingrate

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrFundingRateOutsideLimits is returned when a funding rate is outside the allowed date range
	ErrFundingRateOutsideLimits = errors.New("funding rate outside limits")
	// ErrExchangeNameUnset is returned when an exchange name is not set
	ErrExchangeNameUnset = errors.New("funding rate exchange name not set")

	errPairNotSet           = errors.New("funding rate currency pair not set")
	errAssetTypeNotSet      = errors.New("funding rate asset type not set")
	errInvalidFundingRate   = errors.New("invalid funding rate")
	errFundingRateTimeUnset = errors.New("funding rate time unset")
	errFundingRateNotFound  = errors.New("funding rate not found")
)

var (
	// service stores the latest and upcoming funding rates
	service *Service
)

// Service holds fundingRate information for each individual exchange
type Service struct {
	FundingRates map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*LatestRateWithDispatchIDs
	Exchange     map[string]uuid.UUID
	mux          *dispatch.Mux
	mu           sync.Mutex
	// alerter is used to alert other systems of a funding rate change
	// it is used to alert when any funding rate is changed rather than
	// an individual funding rate subscription
	alerter alert.Notice
}

// LatestRateWithDispatchIDs struct holds the fundingRate information for a currency pair and type
type LatestRateWithDispatchIDs struct {
	LatestRateResponse
	Main  uuid.UUID
	Assoc []uuid.UUID
}

// HistoricalRatesRequest is used to request funding rate details for a position
type HistoricalRatesRequest struct {
	Asset asset.Item
	Pair  currency.Pair
	// PaymentCurrency is an optional parameter depending on exchange API
	// if you are paid in a currency that isn't easily inferred from the Pair,
	// eg BTCUSD-PERP use this field
	PaymentCurrency      currency.Code
	StartDate            time.Time
	EndDate              time.Time
	IncludePayments      bool
	IncludePredictedRate bool
	// RespectHistoryLimits if an exchange has a limit on rate history lookup
	// and your start date is beyond that time, this will set your start date
	// to the maximum allowed date rather than give you errors
	RespectHistoryLimits bool
}

// HistoricalRates is used to return funding rate details for a position
type HistoricalRates struct {
	Exchange              string
	Asset                 asset.Item
	Pair                  currency.Pair
	StartDate             time.Time
	EndDate               time.Time
	LatestRate            Rate
	PredictedUpcomingRate Rate
	FundingRates          []Rate
	PaymentSum            decimal.Decimal
	PaymentCurrency       currency.Code
	TimeOfNextRate        time.Time
}

// LatestRateRequest is used to request the latest funding rate
type LatestRateRequest struct {
	Asset                asset.Item
	Pair                 currency.Pair
	IncludePredictedRate bool
}

// LatestRateResponse for when you just want the latest rate
type LatestRateResponse struct {
	Exchange              string
	Asset                 asset.Item
	Pair                  currency.Pair
	LatestRate            Rate
	PredictedUpcomingRate Rate
	TimeOfNextRate        time.Time
}

// Rate holds details for an individual funding rate
type Rate struct {
	Time    time.Time
	Rate    decimal.Decimal
	Payment decimal.Decimal
}
