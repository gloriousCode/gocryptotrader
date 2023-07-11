package collateral

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
)

// Mode defines the different collateral types supported by exchanges
// For example, FTX had a global collateral pool
// Binance has either singular position collateral calculation
// or cross aka asset level collateral calculation
type Mode uint8

const (
	// UnsetMode is the default value
	UnsetMode Mode = 0
	// SingleMode has allocated collateral per position
	SingleMode Mode = 1 << (iota - 1)
	// MultiMode has collateral allocated across the whole asset
	MultiMode
	// GlobalMode has collateral allocated across account
	GlobalMode
	// UnknownMode has collateral allocated in an unknown manner at present, but is not unset
	UnknownMode
)

const (
	unsetCollateralStr   = "unset"
	singleCollateralStr  = "single"
	multiCollateralStr   = "multi"
	globalCollateralStr  = "global"
	unknownCollateralStr = "unknown"
)

// ErrInvalidCollateralMode is returned when converting invalid string to collateral mode
var ErrInvalidCollateralMode = errors.New("invalid collateral mode")

var supportedCollateralModes = SingleMode | MultiMode | GlobalMode

// TotalCollateralCalculator holds many collateral calculators
// to calculate total collateral standing with one struct
type TotalCollateralCalculator struct {
	FetchPositions bool
	// Offline settings
	CalculateOffline bool
	CollateralAssets []Calculator
}

// Calculator is used to determine
// the size of collateral holdings for an exchange
// eg on Bybit, the collateral is scaled depending on what
// currency it is
type Calculator struct {
	CalculateOffline   bool
	CollateralCurrency currency.Code
	Asset              asset.Item
	USDPrice           decimal.Decimal
	IsLiquidating      bool
	IsForNewPosition   bool
	FreeCollateral     decimal.Decimal
	LockedCollateral   decimal.Decimal
	UnrealisedPNL      decimal.Decimal
}

// TotalCollateralResponse holds all collateral
type TotalCollateralResponse struct {
	ScaledPricing
	BreakdownByAsset     map[asset.Item]ByAsset
	BreakdownOfPositions []ByPosition
}

// ByPosition shows how much collateral is used
// from positions
type ByPosition struct {
	PositionCurrency  currency.Pair
	Asset             asset.Item
	Size              decimal.Decimal
	MarkPrice         decimal.Decimal
	MarkPriceCurrency currency.Code
	PositionValue     decimal.Decimal
	RequiredMargin    decimal.Decimal
	CollateralUsed    decimal.Decimal
	Mode              Mode
	MarginType        margin.Type
	UnrealisedPNL     decimal.Decimal
}

type ByAsset struct {
	ScaledPricing
	Asset      asset.Item
	ByCurrency []ByCurrency
}

// ByCurrency individual collateral contribution
// along with what the potentially scaled collateral
// currency it is represented as
// eg in Bybit ScaledCurrency is USDC
type ByCurrency struct {
	Currency         currency.Code
	Asset            asset.Item
	SkipContribution bool
	Pricing          Pricing
	PricingUSDEquiv  ScaledPricing
	PricingScaled    ScaledPricing

	MarginRequirementCurrency    currency.Code
	InitialMarginRequirement     decimal.Decimal
	MaintenanceMarginRequirement decimal.Decimal
}

// Pricing details collateral amounts for a currency
type Pricing struct {
	Currency      currency.Code
	Total         decimal.Decimal
	Available     decimal.Decimal
	Used          decimal.Decimal
	UsedBreakdown []UsedBreakdown
}

// ScaledPricing includes extra details on how the scaling
// impacts the pricing
type ScaledPricing struct {
	Pricing
	MarkPrice decimal.Decimal
	// GlobalScale is used when an exchange only has one form of scaling
	GlobalScale decimal.Decimal
	// ScalingBreakdown is used when an exchange has tiers
	// to their collateral scaling
	ScalingBreakdown []ScalingBreakdown
}

// ScalingBreakdown holds tiered scaling information
type ScalingBreakdown struct {
	Level         int64
	StartingRange float64
	EndingRange   float64

	Amount       decimal.Decimal
	Scale        decimal.Decimal
	ScaledAmount decimal.Decimal
}

// UsedType details areas where collateral can be used
type UsedType uint8

const (
	LockedInStakes UsedType = iota
	LockedInNFTBids
	LockedInFeeVoucher
	LockedInSpotMarginFundingOffers
	LockedInSpotOrders
	LockedAsCollateral
	UsedInPositions
	UsedInSpotMarginBorrows
)

// UsedBreakdown details how collateral is being used
// if the exchange provides the information
type UsedBreakdown struct {
	UsedType UsedType
	Amount   decimal.Decimal
}
