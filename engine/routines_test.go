package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestWebsocketDataHandlerProcess(t *testing.T) {
	t.Parallel()
	bot := OrdersSetup(t)
	ws := sharedtestvalues.NewTestWebsocket()
	go WebsocketDataReceiver(bot, ws)
	ws.DataHandler <- "string"
	time.Sleep(time.Second)
	close(shutdowner)
}

func TestHandleData(t *testing.T) {
	t.Parallel()
	bot := OrdersSetup(t)
	var exchName = "exch"
	var orderID = "testOrder.Detail"
	err := WebsocketDataHandler(bot, exchName, errors.New("error"))
	if err == nil {
		t.Error("Error not handled correctly")
	}
	err = WebsocketDataHandler(bot, exchName, nil)
	if err == nil {
		t.Error("Expected nil data error")
	}
	err = WebsocketDataHandler(bot, exchName, stream.TradeData{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(bot, exchName, stream.FundingData{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(bot, exchName, &ticker.Price{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(bot, exchName, stream.KlineData{})
	if err != nil {
		t.Error(err)
	}
	origOrder := &order.Detail{
		Exchange: fakePassExchange,
		ID:       orderID,
		Amount:   1337,
		Price:    1337,
	}
	err = WebsocketDataHandler(bot, exchName, origOrder)
	if err != nil {
		t.Error(err)
	}
	// Send it again since it exists now
	err = WebsocketDataHandler(bot, exchName, &order.Detail{
		Exchange: fakePassExchange,
		ID:       orderID,
		Amount:   1338,
	})
	if err != nil {
		t.Error(err)
	}
	if origOrder.Amount != 1338 {
		t.Error("Bad pipeline")
	}

	err = WebsocketDataHandler(bot, exchName, &order.Modify{
		Exchange: fakePassExchange,
		ID:       orderID,
		Status:   order.Active,
	})
	if err != nil {
		t.Error(err)
	}
	if origOrder.Status != order.Active {
		t.Error("Expected order to be modified to Active")
	}

	err = WebsocketDataHandler(bot, exchName, &order.Cancel{
		Exchange: fakePassExchange,
		ID:       orderID,
	})
	if err != nil {
		t.Error(err)
	}
	if origOrder.Status != order.Cancelled {
		t.Error("Expected order status to be cancelled")
	}
	// Send some gibberish
	err = WebsocketDataHandler(bot, exchName, order.Stop)
	if err != nil {
		t.Error(err)
	}

	err = WebsocketDataHandler(bot, exchName, stream.UnhandledMessageWarning{
		Message: "there's an issue here's a tissue"},
	)
	if err != nil {
		t.Error(err)
	}

	classificationError := order.ClassificationError{
		Exchange: "test",
		OrderID:  "one",
		Err:      errors.New("lol"),
	}
	err = WebsocketDataHandler(bot, exchName, classificationError)
	if err == nil {
		t.Fatal("Expected error")
	}
	if err.Error() != classificationError.Error() {
		t.Errorf("Problem formatting error. Expected %v Received %v", classificationError.Error(), err.Error())
	}

	err = WebsocketDataHandler(bot, exchName, &orderbook.Base{
		ExchangeName: fakePassExchange,
		Pair:         currency.NewPair(currency.BTC, currency.USD),
	})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(bot, exchName, "this is a test string")
	if err != nil {
		t.Error(err)
	}
}
