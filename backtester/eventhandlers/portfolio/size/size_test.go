package size

import (
	"testing"
	"time"

	"github.com/quagmt/udecimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestSizingAccuracy(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MaximumSize:  udecimal.MustFromFloat64(1),
		MaximumTotal: udecimal.MustFromFloat64(10),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := udecimal.MustFromFloat64(10)
	availableFunds := udecimal.MustFromFloat64(11)
	feeRate := udecimal.MustFromFloat64(0.02)
	buyLimit := udecimal.MustFromFloat64(1)
	amountWithoutFee, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	assert.NoError(t, err)

	totalWithFee := (price.Mul(amountWithoutFee)).Add(globalMinMax.MaximumTotal.Mul(feeRate))
	if !totalWithFee.Equal(globalMinMax.MaximumTotal) {
		t.Errorf("expected %v received %v", globalMinMax.MaximumTotal, totalWithFee)
	}
}

func TestSizingOverMaxSize(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MaximumSize:  udecimal.MustFromFloat64(0.5),
		MaximumTotal: udecimal.MustFromFloat64(1337),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := udecimal.MustFromFloat64(1338)
	availableFunds := udecimal.MustFromFloat64(1338)
	feeRate := udecimal.MustFromFloat64(0.02)
	buyLimit := udecimal.MustFromFloat64(1)
	amount, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	assert.NoError(t, err)

	if amount.GreaterThan(globalMinMax.MaximumSize) {
		t.Error("greater than max")
	}
}

func TestSizingUnderMinSize(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  udecimal.MustFromFloat64(1),
		MaximumSize:  udecimal.MustFromFloat64(2),
		MaximumTotal: udecimal.MustFromFloat64(1337),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := udecimal.MustFromFloat64(1338)
	availableFunds := udecimal.MustFromFloat64(1338)
	feeRate := udecimal.MustFromFloat64(0.02)
	buyLimit := udecimal.MustFromFloat64(1)
	_, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	assert.ErrorIs(t, err, errLessThanMinimum)
}

func TestMaximumBuySizeEqualZero(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  udecimal.MustFromFloat64(1),
		MaximumTotal: udecimal.MustFromFloat64(1437),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := udecimal.MustFromFloat64(1338)
	availableFunds := udecimal.MustFromFloat64(13380)
	feeRate := udecimal.MustFromFloat64(0.02)
	buyLimit := udecimal.MustFromFloat64(1)
	amount, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	if amount != buyLimit || err != nil {
		t.Errorf("expected: %v, received %v, err: %+v", buyLimit, amount, err)
	}
}

func TestMaximumSellSizeEqualZero(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  udecimal.MustFromFloat64(1),
		MaximumTotal: udecimal.MustFromFloat64(1437),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := udecimal.MustFromFloat64(1338)
	availableFunds := udecimal.MustFromFloat64(13380)
	feeRate := udecimal.MustFromFloat64(0.02)
	sellLimit := udecimal.MustFromFloat64(1)
	amount, _, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	if amount != sellLimit || err != nil {
		t.Errorf("expected: %v, received %v, err: %+v", sellLimit, amount, err)
	}
}

func TestSizingErrors(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  udecimal.MustFromFloat64(1),
		MaximumSize:  udecimal.MustFromFloat64(2),
		MaximumTotal: udecimal.MustFromFloat64(1337),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := udecimal.MustFromFloat64(1338)
	availableFunds := udecimal.Zero
	feeRate := udecimal.MustFromFloat64(0.02)
	buyLimit := udecimal.MustFromFloat64(1)
	_, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	assert.ErrorIs(t, err, errNoFunds)
}

func TestCalculateSellSize(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  udecimal.MustFromFloat64(1),
		MaximumSize:  udecimal.MustFromFloat64(2),
		MaximumTotal: udecimal.MustFromFloat64(1337),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := udecimal.MustFromFloat64(1338)
	availableFunds := udecimal.Zero
	feeRate := udecimal.MustFromFloat64(0.02)
	sellLimit := udecimal.MustFromFloat64(1)
	_, _, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	assert.ErrorIs(t, err, errNoFunds)

	availableFunds = udecimal.MustFromFloat64(1337)
	_, _, err = sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	assert.ErrorIs(t, err, errLessThanMinimum)

	price = udecimal.MustFromFloat64(12)
	availableFunds = udecimal.MustFromFloat64(1339)
	amount, fee, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	assert.NoError(t, err)

	if !amount.Equal(sellLimit) {
		t.Errorf("received '%v' expected '%v'", amount, sellLimit)
	}
	if !amount.Mul(price).Mul(feeRate).Equal(fee) {
		t.Errorf("received '%v' expected '%v'", amount.Mul(price).Mul(feeRate), fee)
	}
}

func TestSizeOrder(t *testing.T) {
	t.Parallel()
	s := Size{}
	_, _, err := s.SizeOrder(nil, udecimal.Zero, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	o := &order.Order{
		Base: &event.Base{
			Offset:         1,
			Exchange:       "binance",
			Time:           time.Now(),
			CurrencyPair:   currency.NewBTCUSDT(),
			UnderlyingPair: currency.NewBTCUSDT(),
			AssetType:      asset.Spot,
		},
	}
	cs := &exchange.Settings{}
	_, _, err = s.SizeOrder(o, udecimal.Zero, cs)
	assert.ErrorIs(t, err, errNoFunds)

	_, _, err = s.SizeOrder(o, udecimal.MustFromFloat64(1337), cs)
	assert.ErrorIs(t, err, errCannotAllocate)

	o.Direction = gctorder.Buy
	_, _, err = s.SizeOrder(o, udecimal.MustFromFloat64(1337), cs)
	assert.ErrorIs(t, err, errCannotAllocate)

	o.ClosePrice = udecimal.MustFromFloat64(1)
	s.BuySide.MaximumSize = udecimal.MustFromFloat64(1)
	s.BuySide.MinimumSize = udecimal.MustFromFloat64(1)
	_, _, err = s.SizeOrder(o, udecimal.MustFromFloat64(1337), cs)
	assert.NoError(t, err)

	o.Amount = udecimal.MustFromFloat64(1)
	o.Direction = gctorder.Sell
	_, _, err = s.SizeOrder(o, udecimal.MustFromFloat64(1337), cs)
	assert.NoError(t, err)

	s.SellSide.MaximumSize = udecimal.MustFromFloat64(1)
	s.SellSide.MinimumSize = udecimal.MustFromFloat64(1)
	_, _, err = s.SizeOrder(o, udecimal.MustFromFloat64(1337), cs)
	assert.NoError(t, err)

	o.Direction = gctorder.ClosePosition
	_, _, err = s.SizeOrder(o, udecimal.MustFromFloat64(1337), cs)
	assert.NoError(t, err)

	// spot futures sizing
	o.FillDependentEvent = &signal.Signal{
		Base:               o.Base,
		MatchesOrderAmount: true,
		ClosePrice:         udecimal.MustFromFloat64(1337),
	}
	exch := binance.Exchange{}
	// TODO adjust when Binance futures wrappers are implemented
	cs.Exchange = &exch
	_, _, err = s.SizeOrder(o, udecimal.MustFromFloat64(1337), cs)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	o.ClosePrice = udecimal.MustFromFloat64(1000000000)
	o.Amount = udecimal.MustFromFloat64(1000000000)
	_, _, err = s.SizeOrder(o, udecimal.MustFromFloat64(1337), cs)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)
}
