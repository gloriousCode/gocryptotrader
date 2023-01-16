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
	currencySettings, ok := r.CurrencySettings[ex][a][p.Base.Item][p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("%v %v %v %w", ex, a, p, errNoCurrencySettings)
	}

	if o.IsLeveraged() {
		if !r.CanUseLeverage {
			return nil, errLeverageNotAllowed
		}
		ratio := existingLeverageRatio(s)
		if ratio.InexactFloat64() > currencySettings.MaximumLeverage {
			return nil, fmt.Errorf("order would exceed maximum holding ratio of %v to %v for %v %v %v. %w", currencySettings.MaximumLeverage, ratio, ex, a, p, errCannotPlaceLeverageOrder)
		}
	}
	if len(latestHoldings) > 1 {
		ratio := assessHoldingsRatio(o.Pair(), latestHoldings)
		if ratio.InexactFloat64() > currencySettings.MaximumLeverage {
			return nil, fmt.Errorf("order would exceed maximum holding ratio of %v to %v for %v %v %v. %w", currencySettings.MaximumLeverage, ratio, ex, a, p, errCannotPlaceLeverageOrder)
		}
	}
	return retOrder, nil
}

// existingLeverageRatio compares orders with leverage to the total number of orders
// a proof of concept to demonstrate risk manager's ability to prevent an order from being placed
// when an order exceeds a config setting
func existingLeverageRatio(s compliance.Snapshot) decimal.Decimal {
	if len(s.Orders) == 0 {
		return decimal.Zero
	}
	var ordersWithLeverage decimal.Decimal
	for o := range s.Orders {
		if s.Orders[o].Order.Leverage != 0 {
			ordersWithLeverage = ordersWithLeverage.Add(decimal.NewFromInt(1))
		}
	}
	return ordersWithLeverage.Div(decimal.NewFromInt(int64(len(s.Orders))))
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
