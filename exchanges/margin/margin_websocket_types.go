package margin

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// WebsocketRateUpdate is a normalized margin lending/borrowing rate update.
// This is intentionally separate from perpetual futures funding rates.
type WebsocketRateUpdate struct {
	Exchange string
	Asset    asset.Item
	Symbol   string
	Pair     currency.Pair
	Currency currency.Code

	BorrowRate decimal.Decimal
	LendRate   decimal.Decimal

	BorrowPeriod float64
	LendPeriod   float64

	Time time.Time
}
