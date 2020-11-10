package signal

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (s *Signal) IsSignal() bool {
	return true
}

func (s *Signal) SetDirection(st order.Side) {
	s.Direction = st
}

func (s *Signal) GetDirection() order.Side {
	return s.Direction
}

func (s *Signal) Pair() currency.Pair {
	return s.CurrencyPair
}

func (s *Signal) SetAmount(f float64) {
	s.Amount = f
}

func (s *Signal) GetAmount() float64 {
	return s.Amount
}

func (s *Signal) GetPrice() float64 {
	return s.Price
}

func (s *Signal) SetPrice(f float64) {
	s.Price = f
}

func (s *Signal) GetWhy() string {
	return s.Why
}

func (s *Signal) SetWhy(y string) {
	s.Why = y
}
