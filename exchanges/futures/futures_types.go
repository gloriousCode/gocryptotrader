package futures

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrPositionClosed returned when attempting to amend a closed position
	ErrPositionClosed = errors.New("the position is closed")
	// ErrNilPNLCalculator is raised when pnl calculation is requested for
	// an exchange, but the fields are not set properly
	ErrNilPNLCalculator = errors.New("nil pnl calculator received")
	// ErrPositionLiquidated is raised when checking PNL status only for
	// it to be liquidated
	ErrPositionLiquidated = errors.New("position liquidated")
	// ErrNotFuturesAsset returned when futures data is requested on a non-futures asset
	ErrNotFuturesAsset = errors.New("asset type is not futures")
	// ErrUSDValueRequired returned when usd value unset
	ErrUSDValueRequired = errors.New("USD value required")
	// ErrOfflineCalculationSet is raised when collateral calculation is set to be offline, yet is attempted online
	ErrOfflineCalculationSet = errors.New("offline calculation set")
	// ErrPositionNotFound is raised when a position is not found
	ErrPositionNotFound = errors.New("position not found")
	// ErrNotPerpetualFuture is returned when a currency is not a perpetual future
	ErrNotPerpetualFuture = errors.New("not a perpetual future")
	// ErrNoPositionsFound returned when there is no positions returned
	ErrNoPositionsFound = errors.New("no positions found")
	// ErrGetFundingDataRequired is returned when requesting funding rate data without the prerequisite
	ErrGetFundingDataRequired = errors.New("getfundingdata is a prerequisite")
	// ErrOrderHistoryTooLarge is returned when you lookup order history, but with too early a start date
	ErrOrderHistoryTooLarge     = errors.New("order history start date too long ago")
	ErrContractTypeNotSupported = errors.New("contract type not supported")
	ErrContractMismatch         = errors.New("contract mismatch")

	errExchangeNameMismatch           = errors.New("exchange name mismatch")
	errTimeUnset                      = errors.New("time unset")
	errMissingPNLCalculationFunctions = errors.New("futures tracker requires exchange PNL calculation functions")
	errOrderNotEqualToTracker         = errors.New("order does not match tracker data")
	errPositionDiscrepancy            = errors.New("there is a position considered open, but it is not the latest, please review")
	errAssetMismatch                  = errors.New("provided asset does not match")
	errEmptyUnderlying                = errors.New("underlying asset unset")
	errNilSetup                       = errors.New("nil setup received")
	errNilOrder                       = errors.New("nil order received")
	errNoPNLHistory                   = errors.New("no pnl history")
	errCannotCalculateUnrealisedPNL   = errors.New("cannot calculate unrealised PNL")
	errDoesntMatch                    = errors.New("doesn't match")
	errCannotTrackInvalidParams       = errors.New("parameters set incorrectly, cannot track")
)

// PNLCalculation is an interface to allow multiple
// ways of calculating PNL to be used for futures positions
type PNLCalculation interface {
	CalculatePNL(context.Context, *PNLCalculatorRequest) (*PNLResult, error)
	GetCurrencyForRealisedPNL(realisedAsset asset.Item, realisedPair currency.Pair) (currency.Code, asset.Item, error)
}

// TotalCollateralResponse holds all collateral
type TotalCollateralResponse struct {
	CollateralCurrency                          currency.Code
	TotalValueOfPositiveSpotBalances            udecimal.Decimal
	CollateralContributedByPositiveSpotBalances udecimal.Decimal
	UsedCollateral                              udecimal.Decimal
	UsedBreakdown                               *collateral.UsedBreakdown
	AvailableCollateral                         udecimal.Decimal
	AvailableMaintenanceCollateral              udecimal.Decimal
	UnrealisedPNL                               udecimal.Decimal
	BreakdownByCurrency                         []collateral.ByCurrency
	BreakdownOfPositions                        []collateral.ByPosition
}

// PositionController manages all futures orders
// across all exchanges assets and pairs
// its purpose is to handle the minutia of tracking
// and so all you need to do is send all orders to
// the position controller and its all tracked happily
type PositionController struct {
	m                     sync.Mutex
	multiPositionTrackers map[key.ExchangeAssetPair]*MultiPositionTracker
	updated               time.Time
}

// MultiPositionTracker will track the performance of
// futures positions over time. If an old position tracker
// is closed, then the position controller will create a new one
// to track the current positions
type MultiPositionTracker struct {
	m                  sync.Mutex
	exchange           string
	asset              asset.Item
	pair               currency.Pair
	underlying         currency.Code
	collateralCurrency currency.Code
	positions          []*PositionTracker
	// order positions allows for an easier time knowing which order is
	// part of which position tracker
	orderPositions             map[string]*PositionTracker
	offlinePNLCalculation      bool
	useExchangePNLCalculations bool
	exchangePNLCalculation     PNLCalculation
}

// MultiPositionTrackerSetup holds the parameters
// required to set up a multi position tracker
type MultiPositionTrackerSetup struct {
	Exchange                  string
	Asset                     asset.Item
	Pair                      currency.Pair
	Underlying                currency.Code
	CollateralCurrency        currency.Code
	OfflineCalculation        bool
	UseExchangePNLCalculation bool
	ExchangePNLCalculation    PNLCalculation
}

// PositionTracker tracks futures orders until the overall position is considered closed
// eg a user can open a short position, append to it via two more shorts, reduce via a small long and
// finally close off the remainder via another long. All of these actions are to be
// captured within one position tracker. It allows for a user to understand their PNL
// specifically for futures positions. Utilising spot/futures arbitrage will not be tracked
// completely within this position tracker, however, can still provide a good
// timeline of performance until the position is closed
type PositionTracker struct {
	m                         sync.Mutex
	useExchangePNLCalculation bool
	collateralCurrency        currency.Code
	offlinePNLCalculation     bool
	PNLCalculation
	exchange           string
	asset              asset.Item
	contractPair       currency.Pair
	underlying         currency.Code
	exposure           udecimal.Decimal
	openingDirection   order.Side
	openingPrice       udecimal.Decimal
	openingSize        udecimal.Decimal
	openingDate        time.Time
	latestDirection    order.Side
	latestPrice        udecimal.Decimal
	lastUpdated        time.Time
	unrealisedPNL      udecimal.Decimal
	realisedPNL        udecimal.Decimal
	status             order.Status
	closingPrice       udecimal.Decimal
	closingDate        time.Time
	shortPositions     []order.Detail
	longPositions      []order.Detail
	pnlHistory         []PNLResult
	fundingRateDetails *fundingrate.HistoricalRates
}

// PositionTrackerSetup contains all required fields to
// setup a position tracker
type PositionTrackerSetup struct {
	Exchange                  string
	Asset                     asset.Item
	Pair                      currency.Pair
	EntryPrice                udecimal.Decimal
	Underlying                currency.Code
	CollateralCurrency        currency.Code
	Side                      order.Side
	UseExchangePNLCalculation bool
	OfflineCalculation        bool
	PNLCalculator             PNLCalculation
}

// TotalCollateralCalculator holds many collateral calculators
// to calculate total collateral standing with one struct
type TotalCollateralCalculator struct {
	CollateralAssets []CollateralCalculator
	CalculateOffline bool
	FetchPositions   bool
}

// CollateralCalculator is used to determine
// the size of collateral holdings for an exchange
// eg on Bybit, the collateral is scaled depending on what
// currency it is
type CollateralCalculator struct {
	CalculateOffline   bool
	CollateralCurrency currency.Code
	Asset              asset.Item
	Side               order.Side
	USDPrice           udecimal.Decimal
	IsLiquidating      bool
	IsForNewPosition   bool
	FreeCollateral     udecimal.Decimal
	LockedCollateral   udecimal.Decimal
	UnrealisedPNL      udecimal.Decimal
}

// OpenInterest holds open interest data for an exchange pair asset
type OpenInterest struct {
	Key          key.ExchangeAssetPair
	OpenInterest float64
	LastUpdated  time.Time
}

// PNLCalculator implements the PNLCalculation interface
// to call CalculatePNL and is used when a user wishes to have a
// consistent method of calculating PNL across different exchanges
type PNLCalculator struct{}

// PNLCalculatorRequest is used to calculate PNL values
// for an open position
type PNLCalculatorRequest struct {
	Pair             currency.Pair
	CalculateOffline bool
	Underlying       currency.Code
	Asset            asset.Item
	Leverage         udecimal.Decimal
	EntryPrice       udecimal.Decimal
	EntryAmount      udecimal.Decimal
	Amount           udecimal.Decimal
	CurrentPrice     udecimal.Decimal
	PreviousPrice    udecimal.Decimal
	Time             time.Time
	OrderID          string
	Fee              udecimal.Decimal
	PNLHistory       []PNLResult
	Exposure         udecimal.Decimal
	OrderDirection   order.Side
	OpeningDirection order.Side
	CurrentDirection order.Side
}

// PNLResult stores a PNL result from a point in time
type PNLResult struct {
	Status                order.Status
	Time                  time.Time
	UnrealisedPNL         udecimal.Decimal
	RealisedPNLBeforeFees udecimal.Decimal
	RealisedPNL           udecimal.Decimal
	Price                 udecimal.Decimal
	Exposure              udecimal.Decimal
	Direction             order.Side
	Fee                   udecimal.Decimal
	IsLiquidated          bool
	// Is event is supposed to show that something has happened and it isn't just tracking in time
	IsOrder bool
}

// Position is a basic holder for position information
type Position struct {
	Exchange           string
	Asset              asset.Item
	Pair               currency.Pair
	Underlying         currency.Code
	CollateralCurrency currency.Code
	RealisedPNL        udecimal.Decimal
	UnrealisedPNL      udecimal.Decimal
	Status             order.Status
	OpeningDate        time.Time
	OpeningPrice       udecimal.Decimal
	OpeningSize        udecimal.Decimal
	OpeningDirection   order.Side
	LatestPrice        udecimal.Decimal
	LatestSize         udecimal.Decimal
	LatestDirection    order.Side
	LastUpdated        time.Time
	CloseDate          time.Time
	Orders             []order.Detail
	PNLHistory         []PNLResult
	FundingRates       fundingrate.HistoricalRates
}

// PositionSummaryRequest is used to request a summary of an open position
type PositionSummaryRequest struct {
	Asset asset.Item
	Pair  currency.Pair
	// UnderlyingPair is optional if the exchange requires it for a contract like BTCUSDT-13333337
	UnderlyingPair currency.Pair

	// offline calculation requirements below
	CalculateOffline          bool
	Direction                 order.Side
	FreeCollateral            udecimal.Decimal
	TotalCollateral           udecimal.Decimal
	CurrentPrice              udecimal.Decimal
	CurrentSize               udecimal.Decimal
	CollateralUsed            udecimal.Decimal
	NotionalPrice             udecimal.Decimal
	MaxLeverageForAccount     udecimal.Decimal
	TotalOpenPositionNotional udecimal.Decimal
	// EstimatePosition if enabled, can be used to calculate a new position
	EstimatePosition bool
	// These fields are also used for offline calculation
	OpeningPrice      udecimal.Decimal
	OpeningSize       udecimal.Decimal
	Leverage          udecimal.Decimal
	TotalAccountValue udecimal.Decimal
}

// PositionDetails are used to track open positions
// in the order manager
type PositionDetails struct {
	Exchange string
	Asset    asset.Item
	Pair     currency.Pair
	Orders   []order.Detail
}

// PositionsRequest defines the request to
// retrieve futures position data
type PositionsRequest struct {
	Asset     asset.Item
	Pairs     currency.Pairs
	StartDate time.Time
	EndDate   time.Time
	// RespectOrderHistoryLimits is designed for the order manager
	// it allows for orders to be tracked if the start date in the config is
	// beyond the allowable limits by the API, rather than returning an error
	RespectOrderHistoryLimits bool
}

// PositionResponse are used to track open positions
// in the order manager
type PositionResponse struct {
	Pair                   currency.Pair
	Asset                  asset.Item
	ContractSettlementType ContractSettlementType
	Orders                 []order.Detail
}

// PositionSummary returns basic details on an open position
type PositionSummary struct {
	Pair           currency.Pair
	Asset          asset.Item
	MarginType     margin.Type
	CollateralMode collateral.Mode
	// The currency in which the values are quoted against. Isn't always pair.Quote
	// eg BTC-USDC-230929's quote in GCT is 230929, but the currency should be USDC
	Currency  currency.Code
	StartDate time.Time

	AvailableEquity     udecimal.Decimal
	CashBalance         udecimal.Decimal
	DiscountEquity      udecimal.Decimal
	EquityUSD           udecimal.Decimal
	IsolatedEquity      udecimal.Decimal
	IsolatedLiabilities udecimal.Decimal
	IsolatedUPL         udecimal.Decimal
	NotionalLeverage    udecimal.Decimal
	TotalEquity         udecimal.Decimal
	StrategyEquity      udecimal.Decimal
	MarginBalance       udecimal.Decimal

	IsolatedMargin               udecimal.Decimal
	NotionalSize                 udecimal.Decimal
	Leverage                     udecimal.Decimal
	MaintenanceMarginRequirement udecimal.Decimal
	InitialMarginRequirement     udecimal.Decimal
	EstimatedLiquidationPrice    udecimal.Decimal
	CollateralUsed               udecimal.Decimal
	MarkPrice                    udecimal.Decimal
	CurrentSize                  udecimal.Decimal
	ContractSize                 udecimal.Decimal
	ContractMultiplier           udecimal.Decimal
	ContractSettlementType       ContractSettlementType
	AverageOpenPrice             udecimal.Decimal
	UnrealisedPNL                udecimal.Decimal
	RealisedPNL                  udecimal.Decimal
	MaintenanceMarginFraction    udecimal.Decimal
	FreeCollateral               udecimal.Decimal
	TotalCollateral              udecimal.Decimal
	FrozenBalance                udecimal.Decimal
	EquityOfCurrency             udecimal.Decimal
}
