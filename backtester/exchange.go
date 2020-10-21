package backtest

import (
	"math"
)

func (e *Exchange) ExecuteOrder(o OrderEvent, data DataHandler) (*Fill, error) {
	f := &Fill{
		Event: Event{
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
		},
		Amount: o.GetAmount(),
		Price:  data.Latest().LatestPrice(),
	}

	f.Direction = o.GetDirection()
	f.Commission = e.calculateCommission(f.Amount, f.Price)
	f.ExchangeFee = e.calculateExchangeFee()
	f.Cost = e.calculateCost(f.Commission, f.ExchangeFee)

	return f, nil
}

func (e *Exchange) calculateCommission(amount, price float64) float64 {
	return math.Floor(amount*price*e.CommissionRate*10000) / 10000
}

func (e *Exchange) calculateExchangeFee() float64 {
	return e.ExchangeFee
}

func (e *Exchange) calculateCost(commission, fee float64) float64 {
	return commission + fee
}
