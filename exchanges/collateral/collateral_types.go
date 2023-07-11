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
	ScaledPricing    ScaledPricing

	MarginRequirementCurrency    currency.Code
	InitialMarginRequirement     decimal.Decimal
	MaintenanceMarginRequirement decimal.Decimal
	UnrealisedPNL                decimal.Decimal
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
	MarkPrice        decimal.Decimal
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
