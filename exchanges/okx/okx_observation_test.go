package okx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestObservationFundingRates(t *testing.T) {
	t.Parallel()
	for _, missing := range []bool{false, true} {
		t.Run(strconv.FormatBool(missing), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "setup must succeed")
			at := time.Now().UTC().Truncate(time.Millisecond)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v5/public/funding-rate-history":
					realised := `,"realizedRate":"0.0002"`
					if missing {
						realised = ""
					}
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"instId":"BTC-USDT-SWAP","fundingTime":"%d","fundingRate":"0.009"%s}]}`, at.Add(-time.Hour).UnixMilli(), realised)
					assert.NoError(t, err, "response should write")
				case "/api/v5/public/funding-rate":
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"instId":"BTC-USDT-SWAP","fundingTime":"%d","nextFundingTime":"%d","fundingRate":"0.0003","ts":"%d"}]}`, at.Add(time.Hour).UnixMilli(), at.Add(9*time.Hour).UnixMilli(), at.UnixMilli())
					assert.NoError(t, err, "response should write")
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
				}
			}))
			defer server.Close()
			require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/api/v5/"), "endpoint must be local")
			latest, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualSwap, Pair: perpetualSwapPair})
			require.NoError(t, err, "latest funding must load")
			require.Len(t, latest, 1, "latest must contain one market")
			assert.Equal(t, at, latest[0].LatestRate.Time.UTC(), "rate should retain observation time")
			assert.Equal(t, at.Add(time.Hour), latest[0].TimeOfNextRate.UTC(), "next settlement should be imminent")
			hist, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap, Pair: perpetualSwapPair, StartDate: at.Add(-2 * time.Hour), EndDate: at})
			if missing {
				assert.ErrorIs(t, err, common.ErrNoResponse, "missing realised rate should fail closed")
				return
			}
			require.NoError(t, err, "history must load")
			require.Len(t, hist.FundingRates, 1, "history must contain one settlement")
			assert.Equal(t, "0.0002", hist.FundingRates[0].Rate.String(), "history should use realised rate")
			assert.True(t, hist.PaymentCurrency.Equal(currency.USDT), "settlement currency should be USDT without account payments")
		})
	}
}

func TestObservationLiquidationOrders(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "setup must succeed")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "SWAP", r.URL.Query().Get("instType"), "swap request should reach transport")
		assert.Equal(t, "filled", r.URL.Query().Get("state"), "state should be transmitted")
		assert.Equal(t, "100", r.URL.Query().Get("limit"), "maximum page size should be transmitted")
		_, err := fmt.Fprint(w, `{"code":"0","data":[{"instId":"BTC-USDT-SWAP","details":[]},{"instId":"ETH-USDT-SWAP","details":[]}]}`)
		assert.NoError(t, err, "response should write")
	}))
	defer server.Close()
	require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/api/v5/"), "endpoint must be local")
	request := &LiquidationOrderRequestParams{InstrumentType: "swap", State: "filled", Limit: 100}
	result, err := ex.GetLiquidationOrders(t.Context(), request)
	require.NoError(t, err, "swap liquidations must load")
	assert.Len(t, result, 2, "all returned instruments should be preserved")
	assert.Equal(t, "swap", request.InstrumentType, "request should remain caller owned")
	_, err = ex.GetLiquidationOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "nil request should fail safely")
	_, err = ex.GetLiquidationOrders(t.Context(), &LiquidationOrderRequestParams{InstrumentType: "MARGIN"})
	assert.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired, "margin should still require identity")
}
