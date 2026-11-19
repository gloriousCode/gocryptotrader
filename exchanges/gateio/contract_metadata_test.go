package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestNormaliseFuturesContracts(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.November, 16, 1, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name               string
		product            asset.Item
		base               currency.Code
		settlement         futures.ContractSettlementType
		divisor            float64
		delisted, zeroTime bool
		want               error
	}{
		{name: "linear", product: asset.USDTMarginedFutures, base: currency.BTC, settlement: futures.Linear, divisor: 1},
		{name: "bundled underlying", product: asset.USDTMarginedFutures, base: currency.NewCode("MBABYDOGE"), settlement: futures.Linear, divisor: 1e6},
		{name: "inverse", product: asset.CoinMarginedFutures, base: currency.BTC, settlement: futures.Inverse, divisor: 1},
		{name: "quanto", product: asset.CoinMarginedFutures, base: currency.ETH, settlement: futures.Quanto, divisor: 1},
		{name: "delisted at source time", product: asset.USDTMarginedFutures, base: currency.BTC, settlement: futures.Linear, divisor: 1, delisted: true},
		{name: "unsupported asset", product: asset.Spot, want: asset.ErrNotSupported},
		{name: "missing source time", product: asset.USDTMarginedFutures, zeroTime: true, want: errContractObservationTime},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := FuturesContract{Name: currency.NewPair(tc.base, currency.USDT), QuantoMultiplier: types.Number(0.001), LeverageMax: types.Number(20), FundingInterval: 28800, FundingRate: types.Number(0.0001)}
			if tc.delisted {
				row.DelistedTime = types.Time(now.Add(-time.Minute))
			}
			at := now
			if tc.zeroTime {
				at = time.Time{}
			}
			rows, err := NormaliseFuturesContracts("GateIO", tc.product, []FuturesContract{row}, at)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
				return
			}
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, tc.settlement, rows[0].SettlementType)
			require.Equal(t, tc.divisor, rows[0].PriceDivisor)
			require.Equal(t, !tc.delisted, rows[0].IsActive)
			require.Equal(t, row.Name, rows[0].Name)
			require.Equal(t, 0.001, rows[0].Multiplier)
			if tc.divisor == 1e6 {
				require.True(t, rows[0].Underlying.Base.Equal(currency.BABYDOGE))
				require.True(t, row.Name.Base.Equal(currency.NewCode("MBABYDOGE")))
			}
		})
	}
	t.Run("empty response", func(t *testing.T) {
		t.Parallel()
		rows, err := NormaliseFuturesContracts("GateIO", asset.USDTMarginedFutures, nil, now)
		require.NoError(t, err)
		require.Empty(t, rows)
	})
}
