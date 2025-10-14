package kline

import (
	"testing"

	"github.com/quagmt/udecimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestClose(t *testing.T) {
	t.Parallel()
	k := Kline{
		Close: udecimal.MustFromFloat64(1337),
	}
	if !k.GetClosePrice().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestHigh(t *testing.T) {
	t.Parallel()
	k := Kline{
		High: udecimal.MustFromFloat64(1337),
	}
	if !k.GetHighPrice().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestLow(t *testing.T) {
	t.Parallel()
	k := Kline{
		Low: udecimal.MustFromFloat64(1337),
	}
	if !k.GetLowPrice().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()
	k := Kline{
		Open: udecimal.MustFromFloat64(1337),
	}
	if !k.GetOpenPrice().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestGetUnderlyingPair(t *testing.T) {
	t.Parallel()
	k := Kline{
		Base: &event.Base{
			UnderlyingPair: currency.NewPair(currency.USD, currency.DOGE),
		},
	}
	if !k.GetUnderlyingPair().Equal(k.Base.UnderlyingPair) {
		t.Errorf("expected '%v'", k.Base.UnderlyingPair)
	}
}
