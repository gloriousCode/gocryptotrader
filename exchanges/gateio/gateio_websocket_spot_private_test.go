package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWsHandleSpotDataPrivateLifecycleEvents(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	ex.SetFillsFeedStatus(true)
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	require.NoError(t, ex.WsHandleSpotData(t.Context(), nil, []byte(wsSpotOrderPushDataJSON)))
	require.NoError(t, ex.WsHandleSpotData(t.Context(), nil, []byte(wsUserTradePushDataJSON)))
	ex.Websocket.DataHandler.Close()

	payloads := make([]any, 0, 4)
	for payload := range ex.Websocket.DataHandler.C {
		payloads = append(payloads, payload.Data)
	}
	require.Len(t, payloads, 4)

	ordersEvent, ok := payloads[0].(*WsSpotOrdersEvent)
	require.True(t, ok, "raw order event must be emitted before its canonical projection")
	require.Equal(t, spotOrdersChannel, ordersEvent.Channel)
	require.Equal(t, "update", ordersEvent.Event)
	require.Equal(t, time.Unix(1605175506, 0), ordersEvent.Time.Time())
	require.Len(t, ordersEvent.Result, 1)
	spotOrder := ordersEvent.Result[0]
	require.Equal(t, "30784435", spotOrder.ID)
	require.Equal(t, "t-abc", spotOrder.Text)
	require.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "_"), spotOrder.CurrencyPair)
	require.Equal(t, "sell", spotOrder.Side)
	require.Equal(t, "open", spotOrder.Status)
	require.Equal(t, 1.0, spotOrder.Amount.Float64())
	require.Equal(t, 0.25, spotOrder.Left.Float64())
	require.Equal(t, 0.75, spotOrder.FilledTotal.Float64())
	require.Equal(t, 10000.5, spotOrder.AverageDealPrice.Float64())
	require.Equal(t, 7500.375, spotOrder.FillPrice.Float64())
	require.Equal(t, 0.0015, spotOrder.Fee.Float64())
	require.Equal(t, currency.USDT.String(), spotOrder.FeeCurrency)
	require.Equal(t, "0.0005", spotOrder.PointFee)
	require.Equal(t, "update", spotOrder.Event)
	require.Equal(t, time.Unix(1605175506, 0), spotOrder.CreateTimeSeconds.Time())
	require.Equal(t, time.Unix(1605175506, 123000000), spotOrder.CreateTime.Time())
	require.Equal(t, time.Unix(1605175507, 0), spotOrder.UpdateTimeSeconds.Time())
	require.Equal(t, time.Unix(1605175507, 456000000), spotOrder.UpdateTime.Time())

	orders, ok := payloads[1].([]order.Detail)
	require.True(t, ok, "canonical order projection must follow its raw event")
	require.Len(t, orders, 1)
	require.Equal(t, spotOrder.ID, orders[0].OrderID)
	require.Equal(t, spotOrder.Amount.Float64(), orders[0].Amount)
	require.Equal(t, spotOrder.Amount.Float64()-spotOrder.Left.Float64(), orders[0].ExecutedAmount)
	require.Equal(t, spotOrder.CreateTime.Time(), orders[0].Date)
	require.Equal(t, spotOrder.UpdateTime.Time(), orders[0].LastUpdated)

	userTradesEvent, ok := payloads[2].(*WsSpotUserTradesEvent)
	require.True(t, ok, "raw user trade event must be emitted before its canonical projection")
	require.Equal(t, spotUserTradesChannel, userTradesEvent.Channel)
	require.Equal(t, "update", userTradesEvent.Event)
	require.Equal(t, time.Unix(1605176741, 0), userTradesEvent.Time.Time())
	require.Len(t, userTradesEvent.Result, 1)
	userTrade := userTradesEvent.Result[0]
	require.Equal(t, int64(5736713), userTrade.ID)
	require.Equal(t, "30784428", userTrade.OrderID)
	require.Equal(t, "t-abc", userTrade.Text)
	require.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "_"), userTrade.CurrencyPair)
	require.Equal(t, "sell", userTrade.Side)
	require.Equal(t, "taker", userTrade.Role)
	require.Equal(t, 1.0, userTrade.Amount.Float64())
	require.Equal(t, 10000.0, userTrade.Price.Float64())
	require.Equal(t, 0.002, userTrade.Fee.Float64())
	require.Equal(t, currency.USDT, userTrade.FeeCurrency)
	require.Equal(t, 0.0005, userTrade.PointFee.Float64())
	require.Equal(t, time.Unix(1605176741, 0), userTrade.CreateTimeSeconds.Time())
	require.Equal(t, time.Unix(1605176741, 123456000), userTrade.CreateTime.Time())

	fills, ok := payloads[3].([]fill.Data)
	require.True(t, ok, "canonical fill projection must follow its raw event")
	require.Len(t, fills, 1)
	require.Equal(t, "5736713", fills[0].TradeID)
	require.Equal(t, userTrade.OrderID, fills[0].OrderID)
	require.Equal(t, userTrade.Amount.Float64(), fills[0].Amount)
	require.Equal(t, userTrade.Price.Float64(), fills[0].Price)
	require.Equal(t, userTrade.CreateTime.Time(), fills[0].Timestamp)
}
