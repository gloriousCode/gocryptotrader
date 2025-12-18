package binance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/fees"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestUpdateFees(t *testing.T) {
	t.Parallel()
	//	for _, a := range e.GetAssetTypes(false) {
	err := e.UpdateFees(request.WithVerbose(t.Context()), asset.Spot)
	require.NoError(t, err)
	p, err := e.CurrencyPairs.GetPairs(asset.Spot, false)
	require.NoError(t, err)
	creds, err := e.GetCredentials(t.Context())
	require.NoError(t, err)
	for _, pair := range p {
		k := fees.NewKey(e.Name, asset.Spot, pair, *creds)
		t.Run(k.String(), func(t *testing.T) {
			t.Parallel()
			fee, err := e.GetFeeButts(k)
			require.NoError(t, err)
			assert.Greater(t, fee.MakerFee, float64(0))
			assert.Greater(t, fee.TakerFee, float64(0))
		})
	}
}
