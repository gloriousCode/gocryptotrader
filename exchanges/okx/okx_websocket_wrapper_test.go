package okx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func connectOKXWithMockedWebsocket(t *testing.T, handlers ...mockws.WsMockFunc) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	for instrumentType, code := range map[string]types.Number{
		instTypeSpot:    101,
		instTypeFutures: 102,
		instTypeOption:  103,
	} {
		ex.instrumentsInfoMap[instrumentType] = []Instrument{{
			InstrumentID:     mainPair,
			InstrumentIDCode: code,
		}}
	}
	handler := okxOrderWSMock
	if len(handlers) > 0 {
		handler = handlers[0]
	}
	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, handler))
	t.Cleanup(server.Close)
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ex.Websocket = websocket.NewManager()
	exchangeConfig := ex.Config
	require.NotNil(t, exchangeConfig, "exchange config must be available")
	exchangeConfig.Features.Subscriptions = subscription.List{}
	require.NoError(t, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exchangeConfig,
		Features:                     &ex.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
	}), "websocket manager setup must not error")
	require.NoError(t, ex.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  websocketURL,
		ResponseCheckTimeout: exchangeConfig.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exchangeConfig.WebsocketResponseMaxLimit,
		Connector: func(ctx context.Context, conn websocket.Connection) error {
			return conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil)
		},
		Subscriber:            func(context.Context, websocket.Connection, subscription.List) error { return nil },
		Unsubscriber:          func(context.Context, websocket.Connection, subscription.List) error { return nil },
		GenerateSubscriptions: func() (subscription.List, error) { return subscription.List{}, nil },
		Handler: func(_ context.Context, conn websocket.Connection, incoming []byte) error {
			var message struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(incoming, &message); err != nil {
				return err
			}
			if message.ID != "" {
				return conn.RequireMatchWithData(message.ID, incoming)
			}
			return nil
		},
		MessageFilter: privateConnection,
	}), "private websocket setup must not error")
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(websocketURL), "websocket URLs must update")
	require.NoError(t, ex.Websocket.Connect(t.Context()), "websocket connection must start")
	require.Eventually(t, func() bool {
		_, err := ex.Websocket.GetConnection(privateConnection)
		return err == nil
	}, time.Second, 10*time.Millisecond, "private websocket connection must become ready")
	t.Cleanup(func() {
		require.NoError(t, ex.Websocket.Disable(), "websocket manager must disable")
		require.NoError(t, ex.Websocket.Shutdown(), "websocket manager must shut down")
	})
	return ex
}

func okxOrderWSMock(tb testing.TB, payload []byte, connection *gws.Conn) error {
	tb.Helper()
	var request struct {
		ID        string `json:"id"`
		Operation string `json:"op"`
		Arguments []struct {
			InstrumentIDCode int64 `json:"instIdCode"`
		} `json:"args"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return err
	}
	if request.ID == "" {
		request.ID = "mock-id"
	}
	if request.Operation == "order" || request.Operation == "amend-order" || request.Operation == "cancel-order" || request.Operation == "batch-cancel-orders" {
		require.NotEmpty(tb, request.Arguments, "standard order request must include arguments")
		for i := range request.Arguments {
			require.Positive(tb, request.Arguments[i].InstrumentIDCode, "standard order request must include an instrument ID code")
		}
	}
	var data string
	switch request.Operation {
	case "order":
		data = `[{"ordId":"submit-order","clOrdId":"client-order","sCode":"0","sMsg":"","ts":"1694153250532"}]`
	case "amend-order":
		data = `[{"ordId":"amended-order","sCode":"0","sMsg":""}]`
	case "cancel-order", "batch-cancel-orders":
		data = `[{"ordId":"cancelled-order","sCode":"0","sMsg":""}]`
	case "sprd-order":
		data = `[{"ordId":"spread-order","clOrdId":"client-order","sCode":"0","sMsg":""}]`
	case "sprd-amend-order":
		data = `[{"ordId":"amended-spread","sCode":"0","sMsg":""}]`
	case "sprd-cancel-order":
		data = `[{"ordId":"cancelled-spread","sCode":"0","sMsg":""}]`
	case "sprd-mass-cancel":
		data = `[{"result":true,"sCode":"0","sMsg":""}]`
	default:
		data = `[{"sCode":"51000","sMsg":"failed"}]`
	}
	response := `{"id":"` + request.ID + `","op":"` + request.Operation + `","code":"0","msg":"","data":` + data + `}`
	return connection.WriteMessage(gws.TextMessage, []byte(response))
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).WebsocketSubmitOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrSubmissionIsNil, "nil submission must return the validation error")
	})
	t.Run("standard", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		response, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: mainPair, AssetType: asset.Options, Side: order.Long, Type: order.Limit, Amount: 1, Price: 1})
		require.NoError(t, err, "WebsocketSubmitOrder must not error")
		assert.Equal(t, "submit-order", response.OrderID, "order ID should match")
		assert.Equal(t, "client-order", response.ClientOrderID, "client order ID should match")
		assert.Equal(t, time.UnixMilli(1694153250532), response.Date, "order date should use the exchange timestamp")
	})
	t.Run("spread", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		response, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, Amount: 0.0001, Price: 1})
		require.NoError(t, err, "spread submission must not error")
		assert.Equal(t, "spread-order", response.OrderID, "spread order ID should match")
	})
	t.Run("missing instrument code", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		ex.instrumentsInfoMapLock.Lock()
		delete(ex.instrumentsInfoMap, instTypeOption)
		ex.instrumentsInfoMapLock.Unlock()
		_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: mainPair, AssetType: asset.Options, Side: order.Long, Type: order.Limit, Amount: 1, Price: 1})
		require.ErrorIs(t, err, errMissingInstrumentIDCode, "missing instrument code must return the expected error")
	})
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).WebsocketModifyOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "nil modification must return the expected error")
	})
	t.Run("standard", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		response, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{OrderID: "order-1", AssetType: asset.Options, Pair: mainPair, Amount: 1, Price: 1})
		require.NoError(t, err, "WebsocketModifyOrder must not error")
		assert.Equal(t, "order-1", response.OrderID, "order ID should match")
	})
	t.Run("spread", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{OrderID: "order-1", AssetType: asset.Spread, Pair: spreadPair, Amount: 0.0001, Price: 1})
		require.NoError(t, err, "spread modification must not error")
	})
	t.Run("unsupported algorithmic order", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{OrderID: "order-1", AssetType: asset.Options, Pair: mainPair, Type: order.Trigger, Amount: 1, Price: 1})
		require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "algorithmic modification must return the expected error")
	})
	t.Run("missing instrument code", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		ex.instrumentsInfoMapLock.Lock()
		delete(ex.instrumentsInfoMap, instTypeOption)
		ex.instrumentsInfoMapLock.Unlock()
		_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{OrderID: "order-1", AssetType: asset.Options, Pair: mainPair, Amount: 1, Price: 1})
		require.ErrorIs(t, err, errMissingInstrumentIDCode, "missing instrument code must return the expected error")
	})
	t.Run("fractional contract amount", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{OrderID: "order-1", AssetType: asset.Options, Pair: mainPair, Amount: 1.5, Price: 1})
		require.ErrorIs(t, err, errContractAmountCanNotBeDecimal, "fractional contract amount must return the expected error")
	})
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		err := new(Exchange).WebsocketCancelOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "nil cancellation must return the expected error")
	})
	t.Run("standard", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		require.NoError(t, ex.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "1", AssetType: asset.Options, Pair: mainPair}), "WebsocketCancelOrder must not error")
	})
	t.Run("spread", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		require.NoError(t, ex.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "1", AssetType: asset.Spread}), "spread cancellation must not error")
	})
	t.Run("unsupported algorithmic order", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "1", AssetType: asset.Options, Pair: mainPair, Type: order.Trigger})
		require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "algorithmic cancellation must return the expected error")
	})
	t.Run("missing instrument code", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		ex.instrumentsInfoMapLock.Lock()
		delete(ex.instrumentsInfoMap, instTypeOption)
		ex.instrumentsInfoMapLock.Unlock()
		err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "1", AssetType: asset.Options, Pair: mainPair})
		require.ErrorIs(t, err, errMissingInstrumentIDCode, "missing instrument code must return the expected error")
	})
}

func TestWebsocketCancelBatchOrders(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).WebsocketCancelBatchOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "empty batch must return the expected error")
	})
	t.Run("too many", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).WebsocketCancelBatchOrders(t.Context(), make([]order.Cancel, 21))
		require.ErrorIs(t, err, errExceedLimit, "oversized batch must return the expected error")
	})
	t.Run("standard", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		response, err := ex.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AssetType: asset.Options, Pair: mainPair}})
		require.NoError(t, err, "WebsocketCancelBatchOrders must not error")
		assert.Equal(t, order.Cancelled.String(), response.Status["cancelled-order"], "cancel status should match")
	})
	t.Run("unsupported algorithmic order", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		_, err := ex.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AssetType: asset.Options, Pair: mainPair, Type: order.Trigger}})
		require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "algorithmic batch cancellation must return the expected error")
	})
	t.Run("unsupported spread", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		_, err := ex.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AssetType: asset.Spread, Pair: spreadPair}})
		require.ErrorIs(t, err, asset.ErrNotSupported, "spread batch cancellation must return the expected error")
	})
	t.Run("missing instrument code", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		ex.instrumentsInfoMapLock.Lock()
		delete(ex.instrumentsInfoMap, instTypeOption)
		ex.instrumentsInfoMapLock.Unlock()
		_, err := ex.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AssetType: asset.Options, Pair: mainPair}})
		require.ErrorIs(t, err, errMissingInstrumentIDCode, "missing instrument code must return the expected error")
	})
}

func TestWebsocketCancelAllOrders(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).WebsocketCancelAllOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "nil cancellation must return the expected error")
	})
	t.Run("standard", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "client"})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v5/trade/orders-pending", r.URL.Path, "order discovery path should match")
			_, err := w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"BTC-USDT","instType":"OPTION","ordId":"1","side":"buy"}]}`))
			assert.NoError(t, err, "mock response should write")
		}))
		t.Cleanup(server.Close)
		require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must update")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api/v5/"), "REST endpoint must update")
		response, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options, Pair: mainPair})
		require.NoError(t, err, "WebsocketCancelAllOrders must not error")
		assert.Equal(t, order.Cancelled.String(), response.Status["cancelled-order"], "cancel status should match")
	})
	t.Run("spread", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		response, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spread, OrderID: "spread-id"})
		require.NoError(t, err, "spread cancel-all must not error")
		assert.Equal(t, "true", response.Status["spread-id"], "spread cancel-all status should match")
	})
	t.Run("unsupported algorithmic order", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		_, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options, Type: order.Trigger})
		require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "algorithmic cancel-all must return the expected error")
	})
	t.Run("missing asset", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		_, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{})
		require.ErrorIs(t, err, asset.ErrNotSupported, "missing asset must return the expected error")
	})
	t.Run("no matching orders", func(t *testing.T) {
		t.Parallel()
		ex := connectOKXWithMockedWebsocket(t)
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "client"})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
			assert.NoError(t, err, "mock response should write")
		}))
		t.Cleanup(server.Close)
		require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must update")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api/v5/"), "REST endpoint must update")
		response, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
		require.NoError(t, err, "empty cancel-all must not error")
		assert.Empty(t, response.Status, "empty cancel-all should have no statuses")
	})
	t.Run("batches more than 20 orders", func(t *testing.T) {
		t.Parallel()
		var websocketRequests atomic.Int64
		ex := connectOKXWithMockedWebsocket(t, func(tb testing.TB, payload []byte, connection *gws.Conn) error {
			tb.Helper()
			var request struct {
				Operation string `json:"op"`
			}
			if err := json.Unmarshal(payload, &request); err != nil {
				return err
			}
			if request.Operation == "batch-cancel-orders" {
				websocketRequests.Add(1)
			}
			return okxOrderWSMock(tb, payload, connection)
		})
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "client"})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			orders := make([]map[string]string, 21)
			for i := range orders {
				orders[i] = map[string]string{"instId": mainPair.String(), "ordId": strconv.Itoa(i), "side": order.Buy.Lower()}
			}
			encoded, err := json.Marshal(struct {
				Code string              `json:"code"`
				Data []map[string]string `json:"data"`
			}{Code: "0", Data: orders})
			assert.NoError(t, err, "mock response should marshal")
			_, err = w.Write(encoded)
			assert.NoError(t, err, "mock response should write")
		}))
		t.Cleanup(server.Close)
		require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must update")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api/v5/"), "REST endpoint must update")
		_, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
		require.NoError(t, err, "batched cancel-all must not error")
		assert.Equal(t, int64(2), websocketRequests.Load(), "cancel-all should split requests into two websocket batches")
	})
}

func TestGenericOrderWrappersUseREST(t *testing.T) {
	t.Parallel()
	setup := func(t *testing.T, handler http.HandlerFunc) *Exchange {
		t.Helper()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "client"})
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must update")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api/v5/"), "REST endpoint must update")
		return ex
	}
	t.Run("submit", func(t *testing.T) {
		t.Parallel()
		ex := setup(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v5/trade/order", r.URL.Path, "REST submit path should match")
			_, err := w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"rest-submit","sCode":"0","sMsg":""}]}`))
			assert.NoError(t, err, "mock response should write")
		})
		response, err := ex.SubmitOrder(t.Context(), &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1})
		require.NoError(t, err, "SubmitOrder must use REST")
		assert.Equal(t, "rest-submit", response.OrderID, "REST order ID should match")
	})
	t.Run("modify", func(t *testing.T) {
		t.Parallel()
		ex := setup(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v5/trade/amend-order", r.URL.Path, "REST modify path should match")
			_, err := w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"rest-modify","sCode":"0","sMsg":""}]}`))
			assert.NoError(t, err, "mock response should write")
		})
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "1", Pair: mainPair, AssetType: asset.Spot, Amount: 1, Price: 1})
		require.NoError(t, err, "ModifyOrder must use REST")
	})
	t.Run("cancel", func(t *testing.T) {
		t.Parallel()
		ex := setup(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v5/trade/cancel-order", r.URL.Path, "REST cancel path should match")
			_, err := w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"rest-cancel","sCode":"0","sMsg":""}]}`))
			assert.NoError(t, err, "mock response should write")
		})
		require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "1", Pair: mainPair, AssetType: asset.Spot}), "CancelOrder must use REST")
	})
	t.Run("batch cancel", func(t *testing.T) {
		t.Parallel()
		ex := setup(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v5/trade/cancel-batch-orders", r.URL.Path, "REST batch-cancel path should match")
			_, err := w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"rest-batch","sCode":"0","sMsg":""}]}`))
			assert.NoError(t, err, "mock response should write")
		})
		response, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", Pair: mainPair, AssetType: asset.Spot}})
		require.NoError(t, err, "CancelBatchOrders must use REST")
		assert.Equal(t, order.Cancelled.String(), response.Status["rest-batch"], "REST batch status should match")
	})
	t.Run("cancel all", func(t *testing.T) {
		t.Parallel()
		ex := setup(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v5/trade/orders-pending":
				_, err := w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"BTC-USDT","instType":"SPOT","ordId":"1","side":"buy"}]}`))
				assert.NoError(t, err, "mock order-list response should write")
			case "/api/v5/trade/cancel-batch-orders":
				_, err := w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"rest-all","sCode":"0","sMsg":""}]}`))
				assert.NoError(t, err, "mock cancel-all response should write")
			default:
				http.NotFound(w, r)
			}
		})
		response, err := ex.CancelAllOrders(t.Context(), &order.Cancel{Pair: mainPair, AssetType: asset.Spot})
		require.NoError(t, err, "CancelAllOrders must use REST")
		assert.Equal(t, order.Cancelled.String(), response.Status["rest-all"], "REST cancel-all status should match")
	})
}

func TestExchange_deriveSubmitOrderArguments(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	tests := []struct {
		name           string
		submit         order.Submit
		expectedAmount types.Number
		expectedTarget string
	}{
		{name: "limit", submit: order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 2}, expectedAmount: 1},
		{name: "quote amount", submit: order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 10}, expectedAmount: 10, expectedTarget: "quote_ccy"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			arguments, err := ex.deriveSubmitOrderArguments(&tc.submit)
			require.NoError(t, err, "deriveSubmitOrderArguments must not error")
			assert.Equal(t, tc.expectedAmount, arguments.Amount, "amount should match")
			assert.Equal(t, tc.expectedTarget, arguments.TargetCurrency, "target currency should match")
		})
	}
	errorTests := []struct {
		name     string
		submit   order.Submit
		expected error
	}{
		{name: "unsupported asset", submit: order.Submit{AssetType: asset.Empty}, expected: asset.ErrNotSupported},
		{name: "missing amount", submit: order.Submit{AssetType: asset.Spot}, expected: limits.ErrAmountBelowMin},
		{name: "spread", submit: order.Submit{AssetType: asset.Spread, Amount: 1}, expected: asset.ErrNotSupported},
		{name: "leverage", submit: order.Submit{AssetType: asset.Futures, Amount: 1, Leverage: 2}, expected: order.ErrSubmitLeverageNotSupported},
		{name: "empty pair", submit: order.Submit{AssetType: asset.Spot, Amount: 1}, expected: currency.ErrCurrencyPairEmpty},
		{name: "invalid side", submit: order.Submit{Pair: mainPair, AssetType: asset.Spot, Type: order.Limit, Amount: 1}, expected: order.ErrSideIsInvalid},
		{name: "algorithmic type", submit: order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Trigger, Amount: 1}, expected: order.ErrTypeIsInvalid},
	}
	for _, tc := range errorTests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ex.deriveSubmitOrderArguments(&tc.submit)
			require.ErrorIs(t, err, tc.expected, "deriveSubmitOrderArguments must return the expected error")
		})
	}
}

func TestIsSpotMarketOrder(t *testing.T) {
	t.Parallel()
	assert.True(t, isSpotMarketOrder(&order.Submit{AssetType: asset.Spot, Type: order.Market}), "spot market order should match")
	assert.False(t, isSpotMarketOrder(&order.Submit{AssetType: asset.Futures, Type: order.Market}), "futures market order should not match")
}

func TestIsSpotMarketBuyWithQuoteAmount(t *testing.T) {
	t.Parallel()
	assert.True(t, isSpotMarketBuyWithQuoteAmount(&order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Buy, QuoteAmount: 1}), "spot market buy with quote amount should match")
	assert.False(t, isSpotMarketBuyWithQuoteAmount(&order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Sell, QuoteAmount: 1}), "spot market sell should not match")
}

func TestDeriveOrderSide(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		side       order.Side
		reduceOnly bool
		expected   string
	}{
		{name: "buy", side: order.Buy, expected: "buy"},
		{name: "sell", side: order.Sell, expected: "sell"},
		{name: "reduce long", side: order.Long, reduceOnly: true, expected: "sell"},
		{name: "reduce short", side: order.Short, reduceOnly: true, expected: "buy"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, err := deriveOrderSide(tc.side, tc.reduceOnly)
			require.NoError(t, err, "deriveOrderSide must not error")
			assert.Equal(t, tc.expected, actual, "side should match")
		})
	}
	_, err := deriveOrderSide(order.UnknownSide, false)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "unknown side must return the expected error")
}

func TestDerivePositionSide(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		submit   order.Submit
		expected string
	}{
		{name: "spot", submit: order.Submit{AssetType: asset.Spot, Side: order.Long}},
		{name: "net", submit: order.Submit{AssetType: asset.Futures, Side: order.Buy}},
		{name: "long", submit: order.Submit{AssetType: asset.Futures, Side: order.Long}, expected: positionSideLong},
		{name: "short", submit: order.Submit{AssetType: asset.Futures, Side: order.Short}, expected: positionSideShort},
		{name: "bid", submit: order.Submit{AssetType: asset.Futures, Side: order.Bid}, expected: positionSideLong},
		{name: "reduce bid", submit: order.Submit{AssetType: asset.Futures, Side: order.Bid, ReduceOnly: true}, expected: positionSideShort},
		{name: "ask", submit: order.Submit{AssetType: asset.Futures, Side: order.Ask}, expected: positionSideShort},
		{name: "reduce ask", submit: order.Submit{AssetType: asset.Futures, Side: order.Ask, ReduceOnly: true}, expected: positionSideLong},
		{name: "unknown", submit: order.Submit{AssetType: asset.Futures, Side: order.UnknownSide}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, derivePositionSide(&tc.submit), "position side should match")
		})
	}
}

func TestDeriveOrderPositionArguments(t *testing.T) {
	t.Parallel()
	side, positionSide, reduceOnly, err := deriveOrderPositionArguments(&order.Submit{AssetType: asset.Futures, Side: order.Buy, ReduceOnly: true})
	require.NoError(t, err, "deriveOrderPositionArguments must not error")
	assert.Equal(t, "buy", side, "side should match")
	assert.Empty(t, positionSide, "position side should be empty in net mode")
	assert.True(t, reduceOnly, "reduce-only should be retained in net mode")
	side, positionSide, reduceOnly, err = deriveOrderPositionArguments(&order.Submit{AssetType: asset.Futures, Side: order.UnknownSide})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "unknown side must return the expected error")
	assert.Empty(t, side, "side should be empty after an error")
	assert.Empty(t, positionSide, "position side should be empty after an error")
	assert.False(t, reduceOnly, "reduce-only should be false after an error")
}

func TestExchange_deriveAmendOrderArguments(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	arguments, err := ex.deriveAmendOrderArguments(&order.Modify{OrderID: "1", Pair: mainPair, AssetType: asset.Spot, Amount: 0.5, Price: 1})
	require.NoError(t, err, "deriveAmendOrderArguments must not error")
	assert.Equal(t, types.Number(0.5), arguments.NewQuantity, "quantity should match")
	tests := []struct {
		name     string
		modify   *order.Modify
		expected error
	}{
		{name: "nil", expected: order.ErrModifyOrderIsNil},
		{name: "spread", modify: &order.Modify{OrderID: "1", AssetType: asset.Spread, Pair: spreadPair}, expected: asset.ErrNotSupported},
		{name: "fractional contract", modify: &order.Modify{OrderID: "1", AssetType: asset.Options, Pair: mainPair, Amount: 1.5}, expected: errContractAmountCanNotBeDecimal},
		{name: "empty pair", modify: &order.Modify{AssetType: asset.Spot, Amount: 1}, expected: order.ErrPairIsEmpty},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ex.deriveAmendOrderArguments(tc.modify)
			require.ErrorIs(t, err, tc.expected, "deriveAmendOrderArguments must return the expected error")
		})
	}
}

func TestExchange_deriveCancelOrderArguments(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	arguments, err := ex.deriveCancelOrderArguments(&order.Cancel{OrderID: "1", Pair: mainPair, AssetType: asset.Spot})
	require.NoError(t, err, "deriveCancelOrderArguments must not error")
	assert.Equal(t, mainPair.String(), arguments.InstrumentID, "instrument ID should match")
	tests := []struct {
		name         string
		cancellation *order.Cancel
		expected     error
	}{
		{name: "nil", expected: order.ErrCancelOrderIsNil},
		{name: "spread", cancellation: &order.Cancel{AssetType: asset.Spread}, expected: asset.ErrNotSupported},
		{name: "empty pair", cancellation: &order.Cancel{AssetType: asset.Spot}, expected: currency.ErrCurrencyPairEmpty},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ex.deriveCancelOrderArguments(tc.cancellation)
			require.ErrorIs(t, err, tc.expected, "deriveCancelOrderArguments must return the expected error")
		})
	}
}

func TestExchange_cachedInstrumentIDCode(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.instrumentsInfoMap[instTypeSpot] = []Instrument{{InstrumentID: mainPair, InstrumentIDCode: 42}}
	code, err := ex.cachedInstrumentIDCode(asset.Spot, mainPair.String())
	require.NoError(t, err, "cachedInstrumentIDCode must not error")
	assert.Equal(t, int64(42), code, "instrument ID code should match")
	_, err = ex.cachedInstrumentIDCode(asset.Spot, "")
	require.ErrorIs(t, err, errMissingInstrumentID, "empty instrument ID must return the expected error")
	_, err = ex.cachedInstrumentIDCode(asset.Empty, mainPair.String())
	require.ErrorIs(t, err, asset.ErrNotSupported, "unsupported asset must return the expected error")
	_, err = ex.cachedInstrumentIDCode(asset.Options, mainPair.String())
	require.ErrorIs(t, err, errMissingInstrumentIDCode, "missing instrument ID code must return the expected error")
}

func TestLookupInstrumentIDCode(t *testing.T) {
	t.Parallel()
	instruments := []Instrument{{InstrumentID: currency.NewBTCUSDT(), InstrumentIDCode: 42}}
	assert.Equal(t, int64(42), lookupInstrumentIDCode(instruments, currency.NewBTCUSDT().String()), "instrument ID code should match")
	assert.Zero(t, lookupInstrumentIDCode(instruments, "missing"), "missing instrument ID code should be zero")
}
