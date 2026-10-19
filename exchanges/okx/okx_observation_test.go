package okx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
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

// TestObservationLiquidationOrders exercises the public subscription and push through a local socket.
func TestObservationLiquidationOrders(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "setup must succeed")
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: liquidationOrders, Asset: asset.PerpetualSwap}}
	public, err := ex.generateSubscriptions(true)
	require.NoError(t, err, "public subscription expansion must succeed")
	require.Len(t, public, 1, "all enabled swap instruments must share one subscription")
	require.False(t, public[0].Authenticated, "liquidation observations must use public data")
	require.JSONEq(t, `{"channel":"liquidation-orders","instType":"SWAP"}`, public[0].QualifiedChannel, "subscription must match the documented asset scope")
	private, err := ex.generateSubscriptions(false)
	require.NoError(t, err, "private subscription expansion must succeed")
	require.Empty(t, private, "liquidation observations must not require credentials")

	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		assert.Equal(t, "/ws/v5/public", r.URL.Path, "liquidations should use the public socket")
		conn, err := (&gws.Upgrader{}).Upgrade(w, r, nil)
		if !assert.NoError(t, err, "local websocket upgrade should succeed") {
			return
		}
		defer conn.Close()
		assert.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Second)), "mock reads should be bounded")
		for _, operation := range []string{operationSubscribe, operationUnsubscribe} {
			_, payload, err := conn.ReadMessage()
			if !assert.NoError(t, err, "subscription message should arrive") {
				return
			}
			assert.JSONEq(t, fmt.Sprintf(`{"op":%q,"args":[{"channel":"liquidation-orders","instType":"SWAP"}]}`, operation), string(payload), "wire request should contain only documented arguments")
			if operation == operationSubscribe {
				assert.NoError(t, conn.WriteMessage(gws.TextMessage, []byte(`{"arg":{"channel":"liquidation-orders","instType":"SWAP"},"data":[{"instId":"BTC-USDT-SWAP","instType":"SWAP","details":[{"bkPx":"60000","posSide":"short","side":"buy","sz":"13","ts":"1700000000000"}]}]}`)), "liquidation push should write")
			}
		}
	}))
	t.Cleanup(server.Close)
	conn, err := ex.Websocket.CreateTestConnection(nil)
	require.NoError(t, err, "public connection must be available without authentication")
	require.Contains(t, conn.GetURL(), "/ws/v5/public", "default public connection must use the documented route")
	conn.SetURL("ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v5/public")
	require.NoError(t, conn.Dial(t.Context(), &gws.Dialer{}, nil, nil), "local public socket must connect")
	t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "local socket should close") })
	require.NoError(t, ex.Subscribe(t.Context(), conn, public), "asset subscription must reach the public socket")
	response := conn.ReadMessage()
	require.Equal(t, gws.TextMessage, response.Type, "public socket must return a text push")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, response.Raw), "liquidation push must decode")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "one typed envelope must be relayed")
	event, ok := (<-ex.Websocket.DataHandler.C).Data.(*WsLiquidationOrders)
	require.True(t, ok, "public liquidation envelope must preserve its type")
	require.Equal(t, "SWAP", event.Argument.InstrumentType, "push must retain instrument scope")
	require.Len(t, event.Data, 1, "push must retain the market row")
	require.Equal(t, "BTC-USDT-SWAP", event.Data[0].InstrumentID, "instrument identity must survive decoding")
	require.Len(t, event.Data[0].Details, 1, "push must retain the liquidation detail")
	assert.Equal(t, 13.0, event.Data[0].Details[0].QuantityOfLiquidation.Float64(), "contract quantity should remain unconverted")
	assert.Equal(t, int64(1700000000000), event.Data[0].Details[0].Timestamp.Time().UnixMilli(), "liquidation time should survive decoding")
	require.NoError(t, ex.Unsubscribe(t.Context(), conn, public), "asset unsubscription must reach the public socket")
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "mock subscription cycle must complete")
	}
}
