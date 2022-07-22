package ticker

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestClose(t *testing.T) {
	t.Parallel()
	k := Ticker{
		Close: decimal.NewFromInt(1337),
	}
	if !k.GetClosePrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestHigh(t *testing.T) {
	t.Parallel()
	k := Ticker{
		High: decimal.NewFromInt(1337),
	}
	if !k.GetHighPrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestLow(t *testing.T) {
	t.Parallel()
	k := Ticker{
		Low: decimal.NewFromInt(1337),
	}
	if !k.GetLowPrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()
	k := Ticker{
		Open: decimal.NewFromInt(1337),
	}
	if !k.GetOpenPrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestLast(t *testing.T) {
	t.Parallel()
	k := Ticker{
		Last: decimal.NewFromInt(1337),
	}
	if !k.GetLastPrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestGetUnderlyingPair(t *testing.T) {
	t.Parallel()
	k := Ticker{
		Base: &event.Base{
			UnderlyingPair: currency.NewPair(currency.USD, currency.DOGE),
		},
	}
	if !k.GetUnderlyingPair().Equal(k.Base.UnderlyingPair) {
		t.Errorf("expected '%v'", k.Base.UnderlyingPair)
	}
}
