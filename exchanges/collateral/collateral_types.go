package collateral

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	// PortfolioMode has collateral allocated across account
	PortfolioMode
	// UnknownMode has collateral allocated in an unknown manner at present, but is not unset
	UnknownMode
	// SpotFuturesMode has collateral allocated across spot and futures accounts
	SpotFuturesMode
)

const (
	unsetCollateralStr       = "unset"
	singleCollateralStr      = "single"
	multiCollateralStr       = "multi"
	portfolioCollateralStr   = "portfolio"
	spotFuturesCollateralStr = "spot_futures"
	unknownCollateralStr     = "unknown"
)

// ErrInvalidCollateralMode is returned when converting invalid string to collateral mode
var ErrInvalidCollateralMode = errors.New("invalid collateral mode")

var supportedCollateralModes = SingleMode | MultiMode | PortfolioMode | SpotFuturesMode

// ByPosition shows how much collateral is used
// from positions
type ByPosition struct {
	PositionCurrency currency.Pair
	Size             udecimal.Decimal
	OpenOrderSize    udecimal.Decimal
	PositionSize     udecimal.Decimal
	MarkPrice        udecimal.Decimal
	RequiredMargin   udecimal.Decimal
	CollateralUsed   udecimal.Decimal
}

// ByCurrency individual collateral contribution
// along with what the potentially scaled collateral
// currency it is represented as
// eg in Bybit ScaledCurrency is USDC
type ByCurrency struct {
	Currency                    currency.Code
	SkipContribution            bool
	TotalFunds                  udecimal.Decimal
	AvailableForUseAsCollateral udecimal.Decimal
	CollateralContribution      udecimal.Decimal
	AdditionalCollateralUsed    udecimal.Decimal
	FairMarketValue             udecimal.Decimal
	Weighting                   udecimal.Decimal
	ScaledCurrency              currency.Code
	UnrealisedPNL               udecimal.Decimal
	ScaledUsed                  udecimal.Decimal
	ScaledUsedBreakdown         *UsedBreakdown
	Error                       error
}

// UsedBreakdown provides a detailed
// breakdown of where collateral is currently being allocated
type UsedBreakdown struct {
	LockedInStakes                  udecimal.Decimal
	LockedInNFTBids                 udecimal.Decimal
	LockedInFeeVoucher              udecimal.Decimal
	LockedInSpotMarginFundingOffers udecimal.Decimal
	LockedInSpotOrders              udecimal.Decimal
	LockedAsCollateral              udecimal.Decimal
	UsedInPositions                 udecimal.Decimal
	UsedInSpotMarginBorrows         udecimal.Decimal
}
