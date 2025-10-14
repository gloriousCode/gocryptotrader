package signal

import (
	"github.com/quagmt/udecimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// IsSignal returns whether the event is a signal type
func (s *Signal) IsSignal() bool {
	return true
}

// SetDirection sets the direction
func (s *Signal) SetDirection(st order.Side) {
	s.Direction = st
}

// GetDirection returns the direction
func (s *Signal) GetDirection() order.Side {
	return s.Direction
}

// SetBuyLimit sets the buy limit
func (s *Signal) SetBuyLimit(f udecimal.Decimal) {
	s.BuyLimit = f
}

// GetBuyLimit returns the buy limit
func (s *Signal) GetBuyLimit() udecimal.Decimal {
	return s.BuyLimit
}

// SetSellLimit sets the sell limit
func (s *Signal) SetSellLimit(f udecimal.Decimal) {
	s.SellLimit = f
}

// GetSellLimit returns the sell limit
func (s *Signal) GetSellLimit() udecimal.Decimal {
	return s.SellLimit
}

// Pair returns the currency pair
func (s *Signal) Pair() currency.Pair {
	return s.CurrencyPair
}

// GetClosePrice returns the price
func (s *Signal) GetClosePrice() udecimal.Decimal {
	return s.ClosePrice
}

// GetHighPrice returns the high price of a signal
func (s *Signal) GetHighPrice() udecimal.Decimal {
	return s.HighPrice
}

// GetLowPrice returns the low price of a signal
func (s *Signal) GetLowPrice() udecimal.Decimal {
	return s.LowPrice
}

// GetOpenPrice returns the open price of a signal
func (s *Signal) GetOpenPrice() udecimal.Decimal {
	return s.OpenPrice
}

// GetVolume returns the volume of a signal
func (s *Signal) GetVolume() udecimal.Decimal {
	return s.Volume
}

// SetPrice sets the price
func (s *Signal) SetPrice(f udecimal.Decimal) {
	s.ClosePrice = f
}

// GetAmount retrieves the order amount
func (s *Signal) GetAmount() udecimal.Decimal {
	return s.Amount
}

// SetAmount sets the order amount
func (s *Signal) SetAmount(d udecimal.Decimal) {
	s.Amount = d
}

// GetUnderlyingPair returns the underlying currency pair
func (s *Signal) GetUnderlyingPair() currency.Pair {
	return s.UnderlyingPair
}

// GetFillDependentEvent returns the fill dependent event
// so it can be added to the event queue
func (s *Signal) GetFillDependentEvent() Event {
	return s.FillDependentEvent
}

// GetCollateralCurrency returns the collateral currency
func (s *Signal) GetCollateralCurrency() currency.Code {
	return s.CollateralCurrency
}

// IsNil says if the event is nil
func (s *Signal) IsNil() bool {
	return s == nil
}

// MatchOrderAmount ensures an order must match
// its set amount or fail
func (s *Signal) MatchOrderAmount() bool {
	return s.MatchesOrderAmount
}

// ToKline is used to convert a signal event
// to a data event for the purpose of closing all positions
// function CloseAllPositions is builds signal data, but
// data event data must still be populated
func (s *Signal) ToKline() kline.Event {
	return &kline.Kline{
		Base:   s.Base,
		Open:   s.OpenPrice,
		Close:  s.ClosePrice,
		Low:    s.LowPrice,
		High:   s.HighPrice,
		Volume: s.Volume,
	}
}
