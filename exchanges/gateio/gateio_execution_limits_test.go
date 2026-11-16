package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestNormaliseSpotExecutionLimits(t *testing.T) {
	t.Parallel()
	for _, minimum := range []float64{0, .25} {
		t.Run(map[bool]string{true: "precision fallback", false: "explicit minimum"}[minimum == 0], func(t *testing.T) {
			t.Parallel()
			at := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
			input := []CurrencyPairDetail{{Base: currency.BTC, Quote: currency.USDT, MinBaseAmount: types.Number(minimum), MinQuoteAmount: 5, MaxBaseAmount: 10, MaxQuoteAmount: 1000, PricePrecision: 2, AmountPrecision: 3, SellStart: types.Time(at.Add(-time.Hour)), BuyStart: types.Time(at.Add(time.Hour))}, {TradeStatus: "untradable"}}
			levels := NormaliseSpotExecutionLimits("GateIO", input, at)
			require.Len(t, levels, 1)
			require.Equal(t, currency.BTC.Item, levels[0].Key.Base)
			require.Equal(t, asset.Spot, levels[0].Key.Asset)
			require.Equal(t, .01, levels[0].PriceStepIncrementSize)
			require.Equal(t, .001, levels[0].AmountStepIncrementSize)
			require.Equal(t, at.Add(-time.Hour), levels[0].Listed)
			require.Equal(t, float64(5), levels[0].MinimumQuoteAmount)
			require.Equal(t, float64(10), levels[0].MaximumBaseAmount)
			require.Equal(t, float64(1000), levels[0].MaximumQuoteAmount)
			if minimum == 0 {
				require.Equal(t, .001, levels[0].MinimumBaseAmount)
			} else {
				require.Equal(t, minimum, levels[0].MinimumBaseAmount)
			}
			require.Equal(t, types.Number(minimum), input[0].MinBaseAmount, "normalisation does not mutate source")
		})
	}
}

func TestNormaliseFuturesExecutionLimits(t *testing.T) {
	t.Parallel()
	for _, product := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Spot} {
		t.Run(product.String(), func(t *testing.T) {
			t.Parallel()
			rows, err := NormaliseFuturesExecutionLimits("GateIO", product, []FuturesContract{{Name: currency.NewPair(currency.NewCode("MBABYDOGE"), currency.USDT), OrderSizeMin: 1, OrderSizeMax: 100, OrderPriceRound: .01, QuantoMultiplier: .001}})
			if product == asset.Spot {
				require.ErrorIs(t, err, asset.ErrNotSupported)
				require.Empty(t, rows)
				return
			}
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, product, rows[0].Key.Asset)
			require.Equal(t, float64(1), rows[0].MinimumBaseAmount)
			require.Equal(t, float64(100), rows[0].MaximumBaseAmount)
			require.Equal(t, .01, rows[0].PriceStepIncrementSize)
			require.Equal(t, float64(1), rows[0].AmountStepIncrementSize)
			require.Equal(t, .001, rows[0].MultiplierDecimal)
			require.Equal(t, 1e6, rows[0].PriceDivisor)
		})
	}
}

func TestNormaliseDeliveryExecutionLimits(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"BTC_USDT_20261225", "BTC_USD_20261225", ""} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			expiry := time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC)
			rows, err := NormaliseDeliveryExecutionLimits("GateIO", []DeliveryContract{{Name: name, OrderSizeMin: 1, OrderSizeMax: 100, OrderPriceRound: .1, QuantoMultiplier: .001, ExpireTime: types.Time(expiry)}})
			if name == "" {
				require.Error(t, err)
				require.Empty(t, rows)
				return
			}
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, asset.DeliveryFutures, rows[0].Key.Asset)
			require.Equal(t, currency.BTC.Item, rows[0].Key.Base)
			require.Equal(t, float64(1), rows[0].MinimumBaseAmount)
			require.Equal(t, float64(100), rows[0].MaximumBaseAmount)
			require.Equal(t, .1, rows[0].PriceStepIncrementSize)
			require.Equal(t, float64(1), rows[0].AmountStepIncrementSize)
			require.Equal(t, .001, rows[0].MultiplierDecimal)
			require.Equal(t, expiry, rows[0].Expiry)
		})
	}
}
