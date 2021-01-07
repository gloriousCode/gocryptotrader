package size

import (
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestSizingAccuracy(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  0,
		MaximumSize:  1,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02

	amountWithoutFee, err := sizer.calculateBuySize(price, availableFunds, feeRate, globalMinMax)
	if err != nil {
		t.Error(err)
	}
	totalWithFee := (price * amountWithoutFee) + (globalMinMax.MaximumTotal * feeRate)
	if totalWithFee != globalMinMax.MaximumTotal {
		t.Log("incorrect amount calculation")
	}
}

func TestSizingOverMaxSize(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  0,
		MaximumSize:  0.5,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02

	amount, err := sizer.calculateBuySize(price, availableFunds, feeRate, globalMinMax)
	if err != nil {
		t.Error(err)
	}
	if amount > globalMinMax.MaximumSize {
		t.Error("greater than max")
	}
}

func TestSizingUnderMinSize(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  2,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02

	_, err := sizer.calculateBuySize(price, availableFunds, feeRate, globalMinMax)
	if err != nil && !strings.Contains(err.Error(), "less than minimum '1'") {
		t.Error(err)
	}
}

func TestSizingErrors(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  2,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 0.0
	feeRate := 0.02

	_, err := sizer.calculateBuySize(price, availableFunds, feeRate, globalMinMax)
	if err != nil && err.Error() != "no fund available" {
		t.Error(err)
	}
}

func TestCalculateSellSize(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  2,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 0.0
	feeRate := 0.02

	_, err := sizer.calculateSellSize(price, availableFunds, feeRate, globalMinMax)
	if err != nil && err.Error() != "no fund available" {
		t.Error(err)
	}
	availableFunds = 1337
	_, err = sizer.calculateSellSize(price, availableFunds, feeRate, globalMinMax)
	if err != nil && !strings.Contains(err.Error(), "less than minimum '1'") {
		t.Error(err)
	}
	price = 12
	availableFunds = 1339
	_, err = sizer.calculateSellSize(price, availableFunds, feeRate, globalMinMax)
	if err != nil {
		t.Error(err)
	}
}

func TestSizeOrder(t *testing.T) {
	s := Size{}
	_, err := s.SizeOrder(nil, 0, nil)
	if err != nil && err.Error() != "nil arguments received, cannot size order" {
		t.Error(err)
	}
	o := &order.Order{}
	cs := &exchange.CurrencySettings{}
	_, err = s.SizeOrder(o, 0, cs)
	if err != nil && err.Error() != "received availableFunds <= 0, cannot size order" {
		t.Error(err)
	}

	_, err = s.SizeOrder(o, 1337, cs)
	if err != nil && !strings.Contains(err.Error(), "portfolio manager cannot allocate funds for an order at") {
		t.Error(err)
	}

	o.Direction = gctorder.Buy
	o.Price = 1
	s.BuySide.MaximumSize = 1
	s.BuySide.MinimumSize = 1
	_, err = s.SizeOrder(o, 1337, cs)
	if err != nil {
		t.Error(err)
	}

	o.Direction = gctorder.Sell
	_, err = s.SizeOrder(o, 1337, cs)
	if err != nil {
		t.Error(err)
	}

	s.SellSide.MaximumSize = 1
	s.SellSide.MinimumSize = 1
	_, err = s.SizeOrder(o, 1337, cs)
	if err != nil {
		t.Error(err)
	}
}
