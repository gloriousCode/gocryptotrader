package gateio

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type futuresDialCaptureConnection struct {
	websocket.Connection
	header http.Header
}

// TestProcessFuturesTickers verifies Gate mark and index prices survive websocket normalisation.
func TestProcessFuturesTickers(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	require.NoError(t, ex.processFuturesTickers(t.Context(), []byte(`{
		"time":1800000000,
		"channel":"futures.tickers",
		"event":"update",
		"result":[{"contract":"BTC_USDT","last":"100.5","mark_price":"101","index_price":"100","volume_24h_quote":"2000000"}]
	}`), asset.USDTMarginedFutures))

	select {
	case event := <-ex.Websocket.DataHandler.C:
		prices, ok := event.Data.([]ticker.Price)
		require.True(t, ok)
		require.Len(t, prices, 1)
		assert.Equal(t, 101.0, prices[0].MarkPrice)
		assert.Equal(t, 100.0, prices[0].IndexPrice)
		assert.Equal(t, time.Unix(1_800_000_000, 0), prices[0].LastUpdated)
	default:
		require.Fail(t, "expected a futures ticker event")
	}
}

func (c *futuresDialCaptureConnection) Dial(_ context.Context, _ *gws.Dialer, header http.Header, _ url.Values) error {
	c.header = header.Clone()
	return nil
}

func (c *futuresDialCaptureConnection) GetURL() string { return usdtFuturesWebsocketURL }

func (c *futuresDialCaptureConnection) SetupPingHandler(request.EndpointLimit, websocket.PingHandler) {
}

func TestSetFuturesWebsocketUserID(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, (*Exchange)(nil).SetFuturesWebsocketUserID(12345), common.ErrNilPointer)
	ex := new(Exchange)
	require.ErrorIs(t, ex.SetFuturesWebsocketUserID(0), errFuturesWebsocketUserIDNotSet)
	require.NoError(t, ex.SetFuturesWebsocketUserID(12345))
	require.Equal(t, int64(12345), ex.futuresWebsocketUserID.Load())
}

func TestWsFuturesConnect(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	conn := new(futuresDialCaptureConnection)
	require.NoError(t, ex.WsFuturesConnect(t.Context(), conn))
	require.Equal(t, "1", conn.header.Get("X-Gate-Size-Decimal"))
}

func TestGenerateFuturesPayload(t *testing.T) {
	t.Parallel()

	t.Run("empty channels", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, nil)
		require.ErrorIs(t, err, errNoChannelsSupplied)
	})

	t.Run("not single pair", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresTickersChannel, Pairs: nil},
		})
		require.ErrorIs(t, err, subscription.ErrNotSinglePair)
	})

	t.Run("frequency invalid interval", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresOrderbookUpdateChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"frequency": kline.Interval(time.Duration(-1))},
			},
		})
		require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	})

	t.Run("candlestick interval invalid", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresCandlesticksChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": kline.Interval(time.Duration(-1))},
			},
		})
		require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	})

	t.Run("orderbook update with snapshot missing level", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookV2, Pairs: currency.Pairs{BTCUSDT}, Params: map[string]any{}},
		})
		require.ErrorIs(t, err, common.ErrParameterRequired)
	})

	t.Run("orderbook update with snapshot bad level type", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookV2, Pairs: currency.Pairs{BTCUSDT}, Params: map[string]any{"level": 50}},
		})
		require.ErrorIs(t, err, common.ErrTypeAssertFailure)
	})

	t.Run("orderbook update with snapshot empty pair", func(t *testing.T) {
		t.Parallel()

		_, err := e.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrderbookV2, Pairs: currency.Pairs{currency.EMPTYPAIR}, Params: map[string]any{"level": uint64(50)}},
		})
		require.ErrorIs(t, err, common.ErrParameterRequired)
	})

	t.Run("happy path unauthenticated - params", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadTest"
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)

		got, err := ex.generateFuturesPayload(context.Background(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresOrderbookUpdateChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params: map[string]any{
					"frequency": kline.TwentyMilliseconds,
					"level":     "20",
					limitKey:    100,
					"accuracy":  "0",
				},
			},
			&subscription.Subscription{
				Channel: futuresCandlesticksChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": kline.FiveMin},
			},
			&subscription.Subscription{
				Channel: futuresOrderbookChannel,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"interval": "0", limitKey: 100},
			},
			&subscription.Subscription{
				Channel: futuresOrderbookV2,
				Pairs:   currency.Pairs{BTCUSDT},
				Params:  map[string]any{"level": uint64(50)},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 4)

		for i := range got {
			require.NotZero(t, got[i].ID)
			require.Equal(t, subscribeEvent, got[i].Event)
			require.NotEmpty(t, got[i].Channel)
			require.NotZero(t, got[i].Time)
			require.Nil(t, got[i].Auth, "Auth must be nil when unauthenticated")
			require.NotEmpty(t, got[i].Payload, "Payload must not be empty")
		}

		require.Equal(t, []string{BTCUSDT.String(), "20ms", "20", "100", "0"}, got[0].Payload)
		require.Equal(t, []string{"5m", BTCUSDT.String()}, got[1].Payload)
		require.Equal(t, []string{BTCUSDT.String(), "100", "0"}, got[2].Payload)
		require.Equal(t, []string{"ob." + BTCUSDT.String() + ".50"}, got[3].Payload)
	})

	t.Run("authenticated channel - missing credentials fails closed", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadAuthDisableTest"

		// Force path into GetCredentials() by allowing authenticated endpoints.
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

		got, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresBalancesChannel,
				Params:  map[string]any{"user": testFuturesUserID},
			},
		})
		require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
		require.Nil(t, got)
		require.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "credential failure must not silently downgrade websocket authentication")
	})

	t.Run("private channel requires authenticated websocket support", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		_, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrdersChannel, Params: map[string]any{"user": testFuturesUserID, allContractsKey: true}},
		})
		require.ErrorIs(t, err, errAuthenticatedWebsocketRequired)
	})

	t.Run("private channel requires user ID", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		_, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{Channel: futuresOrdersChannel, Params: map[string]any{allContractsKey: true}},
		})
		require.ErrorIs(t, err, errFuturesWebsocketUserIDNotSet)
	})

	t.Run("authenticated channel - user param inserted + signature", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = "generateFuturesPayloadAuthTest"
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})

		got, err := ex.generateFuturesPayload(t.Context(), subscribeEvent, subscription.List{
			&subscription.Subscription{
				Channel: futuresBalancesChannel,
				Params:  map[string]any{"user": testFuturesUserID},
			},
			&subscription.Subscription{
				Channel: futuresOrdersChannel,
				Params:  map[string]any{"user": testFuturesUserID, allContractsKey: true},
			},
		})
		require.NoError(t, err, "generateFuturesPayload must not error")
		require.Len(t, got, 2)

		require.NotNil(t, got[0].Auth, "Auth must not be nil when authenticated")
		require.Equal(t, "api_key", got[0].Auth.Method)
		require.Equal(t, "key", got[0].Auth.Key)
		require.NotEmpty(t, got[0].Auth.Sign)

		require.Equal(t, []string{testFuturesUserID}, got[0].Payload)
		require.Equal(t, []string{testFuturesUserID, futuresAllContracts}, got[1].Payload)

		sig, err := ex.generateWsSignature("secret", subscribeEvent, futuresBalancesChannel, got[0].Time)
		require.NoError(t, err)
		require.Equal(t, sig, got[0].Auth.Sign)
	})
}

func TestGenerateFuturesDefaultSubscriptionsAuthenticated(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	subs, err := ex.GenerateFuturesDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err)

	private := map[string]*subscription.Subscription{}
	for _, sub := range subs {
		switch sub.Channel {
		case futuresOrdersChannel, futuresUserTradesChannel, futuresBalancesChannel, futuresPositionsChannel:
			private[sub.Channel] = sub
		}
	}
	require.Len(t, private, 4)
	for _, channel := range []string{futuresOrdersChannel, futuresUserTradesChannel, futuresPositionsChannel} {
		require.Empty(t, private[channel].Pairs)
		require.Equal(t, map[string]any{allContractsKey: true}, private[channel].Params)
	}
	require.Empty(t, private[futuresBalancesChannel].Pairs)
	require.Empty(t, private[futuresBalancesChannel].Params)

	coinMSubs, err := ex.GenerateFuturesDefaultSubscriptions(asset.CoinMarginedFutures)
	require.NoError(t, err)
	for _, sub := range coinMSubs {
		require.NotContains(t, []string{futuresOrdersChannel, futuresUserTradesChannel, futuresBalancesChannel, futuresPositionsChannel}, sub.Channel)
	}

	deliverySubs, err := ex.GenerateDeliveryFuturesDefaultSubscriptions()
	require.NoError(t, err)
	for _, sub := range deliverySubs {
		require.NotContains(t, sub.Params, "user")
	}
}

func TestFuturesSubscribe(t *testing.T) {
	t.Parallel()

	newAuthenticatedExchange := func(t *testing.T) *Exchange {
		t.Helper()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex))
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
		return ex
	}

	t.Run("public subscription does not discover user ID", func(t *testing.T) {
		t.Parallel()

		ex := newAuthenticatedExchange(t)
		sub := &subscription.Subscription{
			Channel: futuresTickersChannel,
			Pairs:   currency.Pairs{BTCUSDT},
			Asset:   asset.USDTMarginedFutures,
		}
		require.NoError(t, ex.FuturesSubscribe(t.Context(), new(FixtureConnection), subscription.List{sub}))
		require.Zero(t, ex.futuresWebsocketUserID.Load())
		require.NotContains(t, sub.Params, "user")
	})

	t.Run("private subscription requires authentication", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex))
		err := ex.FuturesSubscribe(t.Context(), new(FixtureConnection), subscription.List{
			&subscription.Subscription{Channel: futuresBalancesChannel, Asset: asset.USDTMarginedFutures},
		})
		require.ErrorIs(t, err, errAuthenticatedWebsocketRequired)
	})

	t.Run("manual override", func(t *testing.T) {
		t.Parallel()

		ex := newAuthenticatedExchange(t)
		require.NoError(t, ex.SetFuturesWebsocketUserID(54321))
		sub := &subscription.Subscription{
			Channel: futuresOrdersChannel,
			Params:  map[string]any{allContractsKey: true},
			Asset:   asset.USDTMarginedFutures,
		}
		require.NoError(t, ex.FuturesSubscribe(t.Context(), new(FixtureConnection), subscription.List{sub}))
		require.Equal(t, "54321", sub.Params["user"])
	})

	t.Run("subscription user ID", func(t *testing.T) {
		t.Parallel()

		ex := newAuthenticatedExchange(t)
		sub := &subscription.Subscription{
			Channel: futuresBalancesChannel,
			Params:  map[string]any{"user": "67890"},
			Asset:   asset.USDTMarginedFutures,
		}
		require.NoError(t, ex.FuturesSubscribe(t.Context(), new(FixtureConnection), subscription.List{sub}))
		require.Zero(t, ex.futuresWebsocketUserID.Load())
		require.Equal(t, "67890", sub.Params["user"])
	})

	t.Run("unsupported private subscription asset", func(t *testing.T) {
		t.Parallel()

		ex := newAuthenticatedExchange(t)
		err := ex.FuturesSubscribe(t.Context(), new(FixtureConnection), subscription.List{
			&subscription.Subscription{Channel: futuresBalancesChannel, Asset: asset.Spot},
		})
		require.ErrorIs(t, err, asset.ErrNotSupported)
	})

	t.Run("automatic discovery is retained for unsubscribe", func(t *testing.T) {
		t.Parallel()

		ex := newAuthenticatedExchange(t)
		require.NoError(t, testexch.MockHTTPInstance(ex, "/"))
		sub := &subscription.Subscription{
			Channel: futuresBalancesChannel,
			Asset:   asset.USDTMarginedFutures,
		}
		conn := new(FixtureConnection)
		require.NoError(t, ex.FuturesSubscribe(t.Context(), conn, subscription.List{sub}))
		require.Equal(t, int64(12345), ex.futuresWebsocketUserID.Load())
		require.Equal(t, "12345", sub.Params["user"])

		ex.futuresWebsocketUserID.Store(0)
		require.NoError(t, ex.FuturesUnsubscribe(t.Context(), conn, subscription.List{sub}))
		require.Equal(t, "12345", sub.Params["user"])
	})

	t.Run("discovery error", func(t *testing.T) {
		t.Parallel()

		ex := newAuthenticatedExchange(t)
		require.NoError(t, testexch.MockHTTPInstance(ex, "/"))
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		sub := &subscription.Subscription{Channel: futuresBalancesChannel, Asset: asset.USDTMarginedFutures}
		err := ex.FuturesSubscribe(ctx, new(FixtureConnection), subscription.List{sub})
		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, ex.futuresWebsocketUserID.Load())
		require.NotContains(t, sub.Params, "user")
	})
}
