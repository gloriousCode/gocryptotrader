package holdings

import (
	"testing"
	"time"

	"github.com/quagmt/udecimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

func pair(t *testing.T) *funding.SpotPair {
	t.Helper()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, udecimal.Zero, udecimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, udecimal.MustFromFloat64(1337), udecimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	p, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func collateral(t *testing.T) *funding.CollateralPair {
	t.Helper()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, udecimal.Zero, udecimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, udecimal.MustFromFloat64(1337), udecimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	p, err := funding.CreateCollateral(b, q)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCreate(t *testing.T) {
	t.Parallel()
	_, err := Create(nil, pair(t))
	assert.ErrorIs(t, err, common.ErrNilEvent)

	_, err = Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, pair(t))
	assert.NoError(t, err)

	_, err = Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Futures},
	}, collateral(t))
	assert.NoError(t, err)
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	h, err := Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, pair(t))
	assert.NoError(t, err)

	t1 := h.Timestamp
	err = h.Update(&fill.Fill{
		Base: &event.Base{
			Time: time.Now(),
		},
	}, pair(t))
	assert.NoError(t, err)

	if t1.Equal(h.Timestamp) {
		t.Errorf("expected '%v' received '%v'", h.Timestamp, t1)
	}
}

func TestUpdateValue(t *testing.T) {
	t.Parallel()
	b := &event.Base{AssetType: asset.Spot}
	h, err := Create(&fill.Fill{
		Base: b,
	}, pair(t))
	assert.NoError(t, err)

	err = h.UpdateValue(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	h.BaseSize = udecimal.MustFromFloat64(1)
	err = h.UpdateValue(&kline.Kline{
		Base:  b,
		Close: udecimal.MustFromFloat64(1337),
	})
	assert.NoError(t, err)

	if !h.BaseValue.Equal(udecimal.MustFromFloat64(1337)) {
		t.Errorf("expected '%v' received '%v'", h.BaseSize, udecimal.MustFromFloat64(1337))
	}
}

func TestUpdateBuyStats(t *testing.T) {
	t.Parallel()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, udecimal.MustFromFloat64(1), udecimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, udecimal.MustFromFloat64(100), udecimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	p, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	h, err := Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, pair(t))
	assert.NoError(t, err)

	err = h.update(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              udecimal.MustFromFloat64(1),
		ClosePrice:          udecimal.MustFromFloat64(500),
		VolumeAdjustedPrice: udecimal.MustFromFloat64(500),
		PurchasePrice:       udecimal.MustFromFloat64(500),
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			OrderID:     "udecimal.MustFromFloat64(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewBTCUSDT(),
			Trades:      nil,
			Fee:         1,
		},
	}, p)
	assert.NoError(t, err)

	if !h.BaseSize.Equal(p.BaseAvailable()) {
		t.Errorf("expected '%v' received '%v'", 1, h.BaseSize)
	}
	if !h.BaseValue.Equal(p.BaseAvailable().Mul(udecimal.MustFromFloat64(500))) {
		t.Errorf("expected '%v' received '%v'", 500, h.BaseValue)
	}
	if !h.QuoteSize.Equal(udecimal.MustFromFloat64(100)) {
		t.Errorf("expected '%v' received '%v'", 100, h.QuoteSize)
	}
	if !h.TotalValue.Equal(udecimal.MustFromFloat64(600)) {
		t.Errorf("expected '%v' received '%v'", 999, h.TotalValue)
	}
	if !h.BoughtAmount.Equal(udecimal.MustFromFloat64(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if !h.SoldAmount.IsZero() {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if !h.TotalFees.Equal(udecimal.MustFromFloat64(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.TotalFees)
	}

	err = h.update(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              udecimal.MustFromFloat64(0.5),
		ClosePrice:          udecimal.MustFromFloat64(500),
		VolumeAdjustedPrice: udecimal.MustFromFloat64(500),
		PurchasePrice:       udecimal.MustFromFloat64(500),
		Order: &order.Detail{
			Price:       500,
			Amount:      0.5,
			Exchange:    testExchange,
			OrderID:     "udecimal.MustFromFloat64(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewBTCUSDT(),
			Trades:      nil,
			Fee:         0.5,
		},
	}, p)
	assert.NoError(t, err)

	if !h.BoughtAmount.Equal(udecimal.MustFromFloat64(1.5)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if !h.SoldAmount.IsZero() {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if !h.TotalFees.Equal(udecimal.MustFromFloat64(1.5)) {
		t.Errorf("expected '%v' received '%v'", 1.5, h.TotalFees)
	}
}

func TestUpdateSellStats(t *testing.T) {
	t.Parallel()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, udecimal.MustFromFloat64(1), udecimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, udecimal.MustFromFloat64(100), udecimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	p, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}

	h, err := Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, p)
	assert.NoError(t, err)

	err = h.update(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              udecimal.MustFromFloat64(1),
		ClosePrice:          udecimal.MustFromFloat64(500),
		VolumeAdjustedPrice: udecimal.MustFromFloat64(500),
		PurchasePrice:       udecimal.MustFromFloat64(500),
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			OrderID:     "udecimal.MustFromFloat64(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewBTCUSDT(),
			Fee:         1,
		},
	}, p)
	assert.NoError(t, err)

	if !h.BaseSize.Equal(udecimal.MustFromFloat64(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BaseSize)
	}
	if !h.BaseValue.Equal(udecimal.MustFromFloat64(500)) {
		t.Errorf("expected '%v' received '%v'", 500, h.BaseValue)
	}
	if !h.QuoteInitialFunds.Equal(udecimal.MustFromFloat64(100)) {
		t.Errorf("expected '%v' received '%v'", 100, h.QuoteInitialFunds)
	}
	if !h.QuoteSize.Equal(udecimal.MustFromFloat64(100)) {
		t.Errorf("expected '%v' received '%v'", 100, h.QuoteSize)
	}
	if !h.TotalValue.Equal(udecimal.MustFromFloat64(600)) {
		t.Errorf("expected '%v' received '%v'", 600, h.TotalValue)
	}
	if !h.BoughtAmount.Equal(udecimal.MustFromFloat64(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if !h.SoldAmount.IsZero() {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if !h.TotalFees.Equal(udecimal.MustFromFloat64(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.TotalFees)
	}

	err = h.update(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Direction:           order.Sell,
		Amount:              udecimal.MustFromFloat64(1),
		ClosePrice:          udecimal.MustFromFloat64(500),
		VolumeAdjustedPrice: udecimal.MustFromFloat64(500),
		PurchasePrice:       udecimal.MustFromFloat64(500),
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			OrderID:     "udecimal.MustFromFloat64(1337)",
			Type:        order.Limit,
			Side:        order.Sell,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewBTCUSDT(),
			Trades:      nil,
			Fee:         1,
		},
	}, p)
	assert.NoError(t, err)

	if !h.BoughtAmount.Equal(udecimal.MustFromFloat64(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if !h.SoldAmount.Equal(udecimal.MustFromFloat64(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.SoldAmount)
	}
	if !h.TotalFees.Equal(udecimal.MustFromFloat64(2)) {
		t.Errorf("expected '%v' received '%v'", 2, h.TotalFees)
	}
}
