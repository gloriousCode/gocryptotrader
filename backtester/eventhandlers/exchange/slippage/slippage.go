package slippage

import (
	"math/rand"

	"github.com/quagmt/udecimal"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// EstimateSlippagePercentage takes in an int range of numbers
// turns it into a percentage
func EstimateSlippagePercentage(maximumSlippageRate, minimumSlippageRate udecimal.Decimal) udecimal.Decimal {
	if minimumSlippageRate.LessThan(udecimal.MustFromInt64(1, 0)) || minimumSlippageRate.GreaterThan(udecimal.MustFromInt64(100, 0)) {
		return udecimal.MustFromInt64(1, 0)
	}
	if maximumSlippageRate.LessThan(udecimal.MustFromInt64(1, 0)) || maximumSlippageRate.GreaterThan(udecimal.MustFromInt64(100, 0)) {
		return udecimal.MustFromInt64(1, 0)
	}

	// the language here is confusing. The maximum slippage rate is the lower bounds of the number,
	// eg 80 means for every dollar, keep 80%
	minInt, _ := minimumSlippageRate.Int64()
	maxInt, _ := maximumSlippageRate.Int64()
	randSeed := int(minInt) - int(maxInt)
	if randSeed > 0 {
		result := int64(rand.Intn(randSeed)) //nolint:gosec // basic number generation required, no need for crypto/rand

		resultDec := udecimal.MustFromInt64(result, 0)
		divResult, _ := maximumSlippageRate.Add(resultDec).Div(udecimal.MustFromInt64(100, 0))
		return divResult
	}
	return udecimal.MustFromInt64(1, 0)
}

// CalculateSlippageByOrderbook returns the price slippage for an order
func CalculateSlippageByOrderbook(ob *orderbook.Book, side gctorder.Side, allocatedFunds, feeRate udecimal.Decimal) (price, amount udecimal.Decimal, err error) {
	var result *orderbook.WhaleBombResult
	result, err = ob.SimulateOrder(allocatedFunds.InexactFloat64(), side == gctorder.Buy)
	if err != nil {
		return
	}
	rate := (result.MinimumPrice - result.MaximumPrice) / result.MaximumPrice
	price = udecimal.MustFromFloat64(result.MinimumPrice * (rate + 1))
	amount = udecimal.MustFromFloat64(result.Amount * (1 - feeRate.InexactFloat64()))
	return
}
