package holdings

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ErrInitialFundsZero is an error when initial funds are zero or less
var ErrInitialFundsZero = errors.New("initial funds <= 0")

// Holding contains pricing statistics for a given time
// for a given exchange asset pair
type Holding struct {
	Offset            int64
	Item              currency.Code
	Pair              currency.Pair
	Asset             asset.Item       `json:"asset"`
	Exchange          string           `json:"exchange"`
	Timestamp         time.Time        `json:"timestamp"`
	BaseInitialFunds  udecimal.Decimal `json:"base-initial-funds"`
	BaseSize          udecimal.Decimal `json:"base-size"`
	BaseValue         udecimal.Decimal `json:"base-value"`
	QuoteInitialFunds udecimal.Decimal `json:"quote-initial-funds"`
	TotalInitialValue udecimal.Decimal `json:"total-initial-value"`
	QuoteSize         udecimal.Decimal `json:"quote-size"`
	SoldAmount        udecimal.Decimal `json:"sold-amount"`
	SoldValue         udecimal.Decimal `json:"sold-value"`
	BoughtAmount      udecimal.Decimal `json:"bought-amount"`
	CommittedFunds    udecimal.Decimal `json:"committed-funds"`

	IsLiquidated bool

	TotalValueDifference      udecimal.Decimal
	ChangeInTotalValuePercent udecimal.Decimal
	PositionsValueDifference  udecimal.Decimal

	TotalValue                   udecimal.Decimal `json:"total-value"`
	TotalFees                    udecimal.Decimal `json:"total-fees"`
	TotalValueLostToVolumeSizing udecimal.Decimal `json:"total-value-lost-to-volume-sizing"`
	TotalValueLostToSlippage     udecimal.Decimal `json:"total-value-lost-to-slippage"`
	TotalValueLost               udecimal.Decimal `json:"total-value-lost"`
}

// ClosePriceReader is used for holdings calculations
// without needing to consider event types
type ClosePriceReader interface {
	common.Event
	GetClosePrice() udecimal.Decimal
}
