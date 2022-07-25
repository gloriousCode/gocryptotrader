package ticker

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// GetClosePrice returns the closing price of a kline
func (t *Ticker) GetClosePrice() decimal.Decimal {
	if t.Close.IsZero() {
		return t.Last
	}
	return t.Close
}

// GetHighPrice returns the high price of a kline
func (t *Ticker) GetHighPrice() decimal.Decimal {
	return t.High
}

// GetLowPrice returns the low price of a kline
func (t *Ticker) GetLowPrice() decimal.Decimal {
	return t.Low
}

// GetOpenPrice returns the open price of a kline
func (t *Ticker) GetOpenPrice() decimal.Decimal {
	return t.Open
}

// GetLastPrice returns the last price
func (t *Ticker) GetLastPrice() decimal.Decimal {
	return t.Last
}

// GetUnderlyingPair returns the open price of a kline
func (t *Ticker) GetUnderlyingPair() currency.Pair {
	return t.UnderlyingPair
}
