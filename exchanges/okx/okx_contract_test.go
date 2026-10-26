package okx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
)

func TestFuturesContractFromInstrument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		instrument *Instrument
		asset      asset.Item
		err        error
	}{
		{name: "nil", asset: asset.PerpetualSwap, err: common.ErrNilPointer},
		{name: "spot", instrument: &Instrument{}, asset: asset.Spot, err: futures.ErrNotFuturesAsset},
		{name: "linear", instrument: &Instrument{InstrumentID: currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-"), InstrumentType: "SWAP", Underlying: "BTC-USDT", SettlementCurrency: currency.USDT, ContractType: "linear", State: "live"}, asset: asset.PerpetualSwap},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := &Exchange{}
			result, err := e.FuturesContractFromInstrument(tc.instrument, tc.asset)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "conversion must preserve the expected error")
				return
			}
			require.NoError(t, err, "conversion must succeed")
			assert.Equal(t, futures.Linear, result.SettlementType, "conversion should preserve settlement")
			assert.Equal(t, tc.instrument.InstrumentID, result.Name, "conversion should preserve instrument identity")
		})
	}
}
