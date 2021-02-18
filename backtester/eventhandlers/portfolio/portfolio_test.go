package portfolio

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/settings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

func TestReset(t *testing.T) {
	t.Parallel()
	p := Portfolio{
		exchangeAssetPairSettings: make(map[string]map[asset.Item]map[currency.Pair]*settings.Settings),
	}
	p.Reset()
	if p.exchangeAssetPairSettings != nil {
		t.Error("expected nil")
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := Setup(nil, nil, -1)
	if err != nil && err.Error() != "received nil sizeHandler" {
		t.Error(err)
	}

	_, err = Setup(&size.Size{}, nil, -1)
	if err != nil && err.Error() != "received negative riskFreeRate" {
		t.Error(err)
	}

	_, err = Setup(&size.Size{}, nil, 1)
	if err != nil && err.Error() != "received nil risk handler" {
		t.Error(err)
	}
	var p *Portfolio
	p, err = Setup(&size.Size{}, &risk.Risk{}, 1)
	if err != nil {
		t.Error(err)
	}
	if p.riskFreeRate != 1 {
		t.Error("expected 1")
	}
}

func TestSetupCurrencySettingsMap(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	_, err := p.SetupCurrencySettingsMap("", "", currency.Pair{})
	if err != nil && err.Error() != "received empty exchange name" {
		t.Error(err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", "", currency.Pair{})
	if err != nil && err.Error() != "received empty asset" {
		t.Error(err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.Pair{})
	if err != nil && err.Error() != "received unset currency pair" {
		t.Error(err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
}

func TestSetHoldings(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}

	err := p.setHoldings("", "", currency.Pair{}, &holdings.Holding{}, false)
	if err != nil && !strings.Contains(err.Error(), "holding with unset timestamp received") {
		t.Error(err)
	}
	tt := time.Now()

	err = p.setHoldings("", "", currency.Pair{}, &holdings.Holding{Timestamp: tt}, false)
	if err != nil && !strings.Contains(err.Error(), "received empty exchange name") {
		t.Error(err)
	}

	err = p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt}, false)
	if err != nil {
		t.Error(err)
	}

	err = p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt}, false)
	if err != nil && !strings.Contains(err.Error(), "holdings for binance spot BTCUSD at") {
		t.Error(err)
	}

	err = p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt}, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLatestHoldingsForAllCurrencies(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	h := p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 0 {
		t.Error("expected 0")
	}
	tt := time.Now()
	err := p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt}, true)
	if err != nil {
		t.Error(err)
	}
	err = p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), &holdings.Holding{Timestamp: tt}, true)
	if err != nil {
		t.Error(err)
	}
	err = p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), &holdings.Holding{Timestamp: tt.Add(time.Minute)}, true)
	if err != nil {
		t.Error(err)
	}
	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 2 {
		t.Error("expected 2")
	}
}

func TestGetInitialFunds(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	f := p.GetInitialFunds("", "", currency.Pair{})
	if f != 0 {
		t.Error("expected zero")
	}

	err := p.SetInitialFunds("", "", currency.Pair{}, 1)
	if err != nil && err.Error() != "received empty exchange name" {
		t.Error(err)
	}

	err = p.SetInitialFunds(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), 1)
	if err != nil {
		t.Error(err)
	}

	f = p.GetInitialFunds(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE))
	if f != 1 {
		t.Error("expected 1")
	}
}

func TestViewHoldingAtTimePeriod(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	tt := time.Now()
	_, err := p.ViewHoldingAtTimePeriod("", "", currency.Pair{}, tt)
	if err != nil && !strings.Contains(err.Error(), "no holdings found for") {
		t.Error(err)
	}

	err = p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt}, true)
	if err != nil {
		t.Error(err)
	}
	err = p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt.Add(time.Hour)}, true)
	if err != nil {
		t.Error(err)
	}
	_, err = p.ViewHoldingAtTimePeriod(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), tt)
	if err != nil && !strings.Contains(err.Error(), "no holdings found for") {
		t.Error(err)
	}

	var h holdings.Holding
	h, err = p.ViewHoldingAtTimePeriod(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), tt)
	if err != nil {
		t.Error(err)
	}
	if !h.Timestamp.Equal(tt) {
		t.Errorf("expected %v received %v", tt, h.Timestamp)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	err := p.Update(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, reveived %v", common.ErrNilEvent, err)
	}

	err = p.Update(&kline.Kline{})
	if err != nil && !strings.Contains(err.Error(), "no holdings for") {
		t.Error(err)
	}

	err = p.Update(&kline.Kline{
		Base: event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "no holdings for") {
		t.Error(err)
	}

	tt := time.Now()
	err = p.setHoldings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt, PositionsSize: 1337}, true)
	if err != nil {
		t.Error(err)
	}

	err = p.Update(&kline.Kline{
		Base: event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
			Time:         tt,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	f := p.GetFee("", "", currency.Pair{})
	if f != 0 {
		t.Error("expected 0")
	}

	_, err := p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}

	p.SetFee("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD), 1337)
	f = p.GetFee("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if f != 1337 {
		t.Error("expected 1337")
	}
}

func TestGetComplianceManager(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.GetComplianceManager("", "", currency.Pair{})
	if err != nil && !strings.Contains(err.Error(), "no exchange settings found for") {
		t.Error(err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
	var cm *compliance.Manager
	cm, err = p.GetComplianceManager("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
	if cm == nil {
		t.Error("expected not nil")
	}
}

func TestAddComplianceSnapshot(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	err := p.addComplianceSnapshot(&fill.Fill{})
	if err != nil && !strings.Contains(err.Error(), "could not retrieve compliance manager") {
		t.Error(err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}

	err = p.addComplianceSnapshot(&fill.Fill{
		Base: event.Base{
			Exchange:     "hi",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  "hi",
			Pair:      currency.NewPair(currency.BTC, currency.USD),
			AssetType: asset.Spot,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestOnFill(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.OnFill(nil)
	if err != nil && err.Error() != "nil fill event received, cannot process OnFill" {
		t.Error(err)
	}

	f := &fill.Fill{
		Base: event.Base{
			Exchange:     "hi",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  "hi",
			Pair:      currency.NewPair(currency.BTC, currency.USD),
			AssetType: asset.Spot,
		},
	}
	_, err = p.OnFill(f)
	if err != nil && !strings.Contains(err.Error(), "no currency settings found for") {
		t.Error(err)
	}
	var s *settings.Settings
	s, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
	_, err = p.OnFill(f)
	if err != nil && err.Error() != "initial funds <= 0" {
		t.Error(err)
	}

	s.InitialFunds = 1337
	_, err = p.OnFill(f)
	if err != nil {
		t.Error(err)
	}

	f.Direction = gctorder.Buy
	_, err = p.OnFill(f)
	if err != nil {
		t.Error(err)
	}
}

func TestOnSignal(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.OnSignal(nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}

	s := &signal.Signal{}
	_, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil && err.Error() != "no size manager set" {
		t.Error(err)
	}
	p.sizeManager = &size.Size{}

	_, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil && err.Error() != "no risk manager set" {
		t.Error(err)
	}

	p.riskManager = &risk.Risk{}

	_, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil && err.Error() != "invalid Direction" {
		t.Error(err)
	}

	s.Direction = gctorder.Buy
	_, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil && !strings.Contains(err.Error(), "no portfolio settings for") {
		t.Error(err)
	}
	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
	s = &signal.Signal{
		Base: event.Base{
			Exchange:     "hi",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
		Direction: gctorder.Buy,
	}
	var resp *order.Order
	resp, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil {
		t.Error(err)
	}
	if resp.Reason == "" {
		t.Error("expected issue")
	}

	s.Direction = gctorder.Sell
	_, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil {
		t.Error(err)
	}
	if resp.Reason == "" {
		t.Error("expected issue")
	}

	s.Direction = common.MissingData
	_, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil {
		t.Error(err)
	}

	s.Direction = gctorder.Buy
	err = p.setHoldings("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: time.Now(), RemainingFunds: 1337}, true)
	if err != nil {
		t.Error(err)
	}
	resp, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil {
		t.Error(err)
	}
	if resp.Direction != common.CouldNotBuy {
		t.Errorf("expected common.CouldNotBuy, received %v", resp.Direction)
	}

	s.Price = 10
	s.Direction = gctorder.Buy
	resp, err = p.OnSignal(s, &exchange.Settings{})
	if err != nil {
		t.Error(err)
	}
	if resp.Amount == 0 {
		t.Error("expected an amount to be sized")
	}
}
