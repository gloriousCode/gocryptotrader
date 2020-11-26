package exchange

import (
	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func (e *Exchange) ExecuteOrder(o OrderEvent, data interfaces.DataHandler) (*fill.Fill, error) {
	cs := e.GetCurrencySettings(o.GetExchange(), o.GetAssetType(), o.Pair())
	fillEvent := &fill.Fill{
		Event: event.Event{
			Exchange:     o.GetExchange(),
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
			AssetType:    o.GetAssetType(),
			Interval:     o.GetInterval(),
		},
		Direction:           o.GetDirection(),
		Amount:              o.GetAmount(),
		ClosePrice:          data.Latest().Price(),
		VolumeAdjustedPrice: 0,
		ExchangeFee:         cs.ExchangeFee, // defaulting to just using taker fee right now without orderbook
		Why:                 o.GetWhy(),
	}
	if o.GetAmount() <= 0 {
		fillEvent.Direction = common.DoNothing
		return fillEvent, nil
	}
	fillEvent.Direction = o.GetDirection()
	var slippageRate, estimatedPrice, amount float64
	if false /*e.UseRealOrders*/ {
		// get current orderbook
		// calculate an estimated slippage rate
		slippageRate = slippage.CalculateSlippage(nil)
		estimatedPrice = fillEvent.VolumeAdjustedPrice * slippageRate
	} else {
		// provide n history and estimate volatility
		slippageRate = slippage.EstimateSlippagePercentage(cs.MinimumSlippageRate, cs.MaximumSlippageRate, o.GetDirection())
		estimatedPrice = fillEvent.VolumeAdjustedPrice * slippageRate
		high := data.StreamHigh()
		low := data.StreamLow()
		volume := data.StreamVol()

		estimatedPrice, amount = e.ensureOrderFitsWithinHLV(estimatedPrice, o.GetAmount(), high[len(high)-1], low[len(low)-1], volume[len(volume)-1])
	}

	fillEvent.Slippage = (slippageRate * 100) - 100
	fillEvent.VolumeAdjustedPrice = estimatedPrice
	fillEvent.ExchangeFee = e.calculateExchangeFee(estimatedPrice, amount, cs.ExchangeFee)

	u, _ := uuid.NewV4()
	var orderID string
	o2 := &gctorder.Submit{
		Price:       estimatedPrice,
		Amount:      amount,
		Fee:         fillEvent.ExchangeFee,
		Exchange:    fillEvent.Exchange,
		ID:          u.String(),
		Side:        fillEvent.Direction,
		AssetType:   fillEvent.AssetType,
		Date:        o.GetTime(),
		LastUpdated: o.GetTime(),
		Pair:        o.Pair(),
		Type:        gctorder.Market,
	}

	if false /*e.UseRealOrders*/ {
		resp, err := engine.Bot.OrderManager.Submit(o2)
		if err != nil {
			return nil, err
		}
		orderID = resp.OrderID
	} else {
		o2Response := gctorder.SubmitResponse{
			IsOrderPlaced: true,
			OrderID:       u.String(),
			Rate:          fillEvent.Amount,
			Fee:           fillEvent.ExchangeFee,
			Cost:          estimatedPrice,
			FullyMatched:  true,
		}
		log.Debugf(log.BackTester, "submitting fake order for %v interval", o.GetTime())
		resp, err := engine.Bot.OrderManager.SubmitFakeOrder(o2, o2Response)
		if err != nil {
			return nil, err
		}
		orderID = resp.OrderID
	}
	ords, _ := engine.Bot.OrderManager.GetOrdersSnapshot("")
	for i := range ords {
		if ords[i].ID == orderID {
			ords[i].Date = o.GetTime()
			ords[i].LastUpdated = o.GetTime()
			ords[i].CloseTime = o.GetTime()
			fillEvent.Order = &ords[i]
			fillEvent.PurchasePrice = ords[i].Price
		}
	}

	return fillEvent, nil
}

func (e *Exchange) SetCurrency(exch string, a asset.Item, cp currency.Pair, c CurrencySettings) {
	for i := range e.CurrencySettings {
		if e.CurrencySettings[i].CurrencyPair == cp {
			if e.CurrencySettings[i].AssetType == a {
				if exch == e.CurrencySettings[i].ExchangeName {
					e.CurrencySettings[i] = c
				}
			}
		}
	}
}

func (e *Exchange) GetCurrencySettings(exch string, a asset.Item, cp currency.Pair) CurrencySettings {
	for i := range e.CurrencySettings {
		if e.CurrencySettings[i].CurrencyPair == cp {
			if e.CurrencySettings[i].AssetType == a {
				if exch == e.CurrencySettings[i].ExchangeName {
					return e.CurrencySettings[i]
				}
			}
		}
	}
	return CurrencySettings{}
}

func (e *Exchange) ensureOrderFitsWithinHLV(slippagePrice, amount, high, low, volume float64) (float64, float64) {
	if slippagePrice < low {
		slippagePrice = low
	}
	if slippagePrice > high {
		slippagePrice = high
	}

	if amount*slippagePrice > volume {
		// hey, this order is too big here
		for amount*slippagePrice > volume {
			amount *= 0.99999
		}
	}

	return slippagePrice, amount
}

func (e *Exchange) calculateExchangeFee(price, amount, fee float64) float64 {
	return fee * price * amount
}
