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
	Holdings

	UnrealisedPNL        decimal.Decimal
	BreakdownByAsset     map[asset.Item]ByAsset
	BreakdownOfPositions []ByPosition
}

// Asset details collateral amounts for a currency
type Holdings struct {
	Currency      currency.Code
	Total         decimal.Decimal
	Available     decimal.Decimal
	Used          decimal.Decimal
	UsedBreakdown []UsedBreakdown
}

// ByPosition shows how much collateral is used
// from positions
type ByPosition struct {
	PositionCurrency  currency.Pair
	Asset             asset.Item
	Size              decimal.Decimal
	IndexPrice        decimal.Decimal
	MarkPrice         decimal.Decimal
	MarkIndexCurrency currency.Code
	PositionValue     decimal.Decimal
	RequiredMargin    decimal.Decimal
	CollateralUsed    decimal.Decimal
	MarginType        margin.Type
	UnrealisedPNL     decimal.Decimal
}

type ByAsset struct {
	Holdings
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
	Holdings         Holdings
	HoldingsScaled   HoldingsScaled
	HoldingsUSDEquiv HoldingsUSDEquiv
}

type AssetScaledPricing struct {
	Scale decimal.Decimal
	Price decimal.Decimal
	// ScalingBreakdown pricing can have tiers where they only scale in the ranges affected
	ScalingBreakdown []ScalingBreakdown
}

// HoldingsScaled includes extra details on how the scaling
// impacts the pricing
type HoldingsScaled struct {
	Holdings
	AssetScaledPricing
}

type HoldingsUSDEquiv struct {
	Holdings
	Price decimal.Decimal
}

// ScalingBreakdown holds tiered scaling information
type ScalingBreakdown struct {
	Level         int64
	StartingRange float64
	EndingRange   float64
	IsUsed        bool
	Amount        decimal.Decimal
	Scale         decimal.Decimal
	ScaledAmount  decimal.Decimal
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
