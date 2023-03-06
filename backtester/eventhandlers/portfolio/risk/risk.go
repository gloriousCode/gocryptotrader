package risk

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// EvaluateOrder goes through a standard list of evaluations to make to ensure that
// we are in a position to follow through with an order
func (r *Risk) EvaluateOrder(o order.Event, latestHoldings []holdings.Holding, s compliance.Snapshot) (*order.Order, error) {
	if o == nil || latestHoldings == nil {
		return nil, gctcommon.ErrNilPointer
	}
	retOrder, ok := o.(*order.Order)
	if !ok {
		return nil, fmt.Errorf("%w expected order event", common.ErrInvalidDataType)
	}
	ex := o.GetExchange()
	a := o.GetAssetType()
	p := o.Pair().Format(currency.EMPTYFORMAT)

	if o.IsLeveraged() && !r.canUseLeverage {
		return nil, errLeverageNotAllowed
	}
	if len(latestHoldings) > 1 {
		ratio := assessHoldingsRatio(o.Pair(), latestHoldings)
		if ratio.InexactFloat64() > r.leverageTarget {
			return nil, fmt.Errorf("order would exceed maximum holding ratio of %v to %v for %v %v %v. %w", r.leverageTarget, ratio, ex, a, p, errCannotPlaceLeverageOrder)
		}
	}
	return retOrder, nil
}

func assessHoldingsRatio(c currency.Pair, h []holdings.Holding) decimal.Decimal {
	resp := make(map[currency.Pair]decimal.Decimal)
	totalPosition := decimal.Zero
	for i := range h {
		resp[h[i].Pair] = resp[h[i].Pair].Add(h[i].BaseValue)
		totalPosition = totalPosition.Add(h[i].BaseValue)
	}

	if totalPosition.IsZero() {
		return decimal.Zero
	}
	ratio := resp[c].Div(totalPosition)

	return ratio
}
