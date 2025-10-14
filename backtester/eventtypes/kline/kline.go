package kline

import (
	"github.com/quagmt/udecimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// GetClosePrice returns the closing price of a kline
func (k *Kline) GetClosePrice() udecimal.Decimal {
	return k.Close
}

// GetHighPrice returns the high price of a kline
func (k *Kline) GetHighPrice() udecimal.Decimal {
	return k.High
}

// GetLowPrice returns the low price of a kline
func (k *Kline) GetLowPrice() udecimal.Decimal {
	return k.Low
}

// GetOpenPrice returns the open price of a kline
func (k *Kline) GetOpenPrice() udecimal.Decimal {
	return k.Open
}

// GetVolume returns the volume of a kline
func (k *Kline) GetVolume() udecimal.Decimal {
	return k.Volume
}

// GetUnderlyingPair returns the open price of a kline
func (k *Kline) GetUnderlyingPair() currency.Pair {
	return k.UnderlyingPair
}

// IsKline is a function to help distinguish between kline.Event
// and signal.Event as signal.Event implements kline.Event definitions otherwise
// this function is not called
func (k *Kline) IsKline() bool {
	return true
}
