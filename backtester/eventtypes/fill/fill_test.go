package fill

import (
	"testing"

	"github.com/quagmt/udecimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestSetDirection(t *testing.T) {
	t.Parallel()
	f := Fill{
		Direction: gctorder.Sell,
	}
	f.SetDirection(gctorder.Buy)
	if f.GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetAmount(t *testing.T) {
	t.Parallel()
	f := Fill{
		Amount: udecimal.MustFromFloat64(1),
	}
	f.SetAmount(udecimal.MustFromFloat64(1337))
	if !f.GetAmount().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestGetClosePrice(t *testing.T) {
	t.Parallel()
	f := Fill{
		ClosePrice: udecimal.MustFromFloat64(1337),
	}
	if !f.GetClosePrice().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestGetVolumeAdjustedPrice(t *testing.T) {
	t.Parallel()
	f := Fill{
		VolumeAdjustedPrice: udecimal.MustFromFloat64(1337),
	}
	if !f.GetVolumeAdjustedPrice().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestGetPurchasePrice(t *testing.T) {
	t.Parallel()
	f := Fill{
		PurchasePrice: udecimal.MustFromFloat64(1337),
	}
	if !f.GetPurchasePrice().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestSetExchangeFee(t *testing.T) {
	t.Parallel()
	f := Fill{
		ExchangeFee: udecimal.MustFromFloat64(1),
	}
	f.SetExchangeFee(udecimal.MustFromFloat64(1337))
	if !f.GetExchangeFee().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected udecimal.MustFromFloat64(1337)")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	f := Fill{
		Order: &gctorder.Detail{},
	}
	if f.GetOrder() == nil {
		t.Error("expected not nil")
	}
}

func TestGetSlippageRate(t *testing.T) {
	t.Parallel()
	f := Fill{
		Slippage: udecimal.MustFromFloat64(1),
	}
	if !f.GetSlippageRate().Equal(udecimal.MustFromFloat64(1)) {
		t.Error("expected 1")
	}
}

func TestGetTotal(t *testing.T) {
	t.Parallel()
	f := Fill{}
	f.Total = udecimal.MustFromFloat64(1337)
	e := f.GetTotal()
	if !e.Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected 1337")
	}
}

func TestGetFillDependentEvent(t *testing.T) {
	t.Parallel()
	f := Fill{}
	if f.GetFillDependentEvent() != nil {
		t.Error("expected nil")
	}
	f.FillDependentEvent = &signal.Signal{
		Amount: udecimal.MustFromFloat64(1337),
	}
	e := f.GetFillDependentEvent()
	if !e.GetAmount().Equal(udecimal.MustFromFloat64(1337)) {
		t.Error("expected 1337")
	}
}

func TestIsLiquidated(t *testing.T) {
	t.Parallel()
	f := Fill{}
	if f.IsLiquidated() {
		t.Error("expected false")
	}
	f.Liquidated = true
	if !f.IsLiquidated() {
		t.Error("expected true")
	}
}
