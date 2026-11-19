package gateio

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGetLiquidationHistoryRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		pair currency.Pair
		want string
	}{
		{name: "all contracts", pair: currency.EMPTYPAIR},
		{name: "specific contract", pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), want: "BTC_USDT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex))
			from := time.Unix(1700000000, 0)
			to := from.Add(5 * time.Minute)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v4/futures/usdt/liq_orders", r.URL.Path)
				assert.Equal(t, tc.want, r.URL.Query().Get("contract"))
				assert.Equal(t, tc.want != "", r.URL.Query().Has("contract"))
				assert.Equal(t, "1700000000", r.URL.Query().Get("from"))
				assert.Equal(t, "1700000300", r.URL.Query().Get("to"))
				assert.Equal(t, "1000", r.URL.Query().Get("limit"))
				_, err := w.Write([]byte("[]"))
				assert.NoError(t, err)
			}))
			defer server.Close()
			require.NoError(t, ex.SetHTTPClient(server.Client()))
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api/v4/"))
			_, err := ex.GetLiquidationHistory(t.Context(), currency.USDT, tc.pair, from, to, 1000)
			require.NoError(t, err)
		})
	}
}
