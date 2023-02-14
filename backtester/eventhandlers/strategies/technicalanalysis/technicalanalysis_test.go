package technicalanalysis

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/strategybase"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	eventkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestName(t *testing.T) {
	t.Parallel()
	d := Strategy{}
	if n := d.GetName(); n != Name {
		t.Errorf("expected %v", Name)
	}
}

func TestSupportsSimultaneousProcessing(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	if !s.SupportsSimultaneousProcessing() {
		t.Error("expected true")
	}
}

func TestSetCustomSettings(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	err := s.SetCustomSettings(nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestOnSignal(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := &data.Base{}
	err = d.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
			Offset:       3,
			Exchange:     exch,
			Time:         dStart,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	da := &kline.DataFromKline{
		Item:        &gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	var resp signal.Event
	_, err = s.OnSignal(da, nil, nil)
	if !errors.Is(err, strategybase.ErrTooMuchBadData) {
		t.Fatalf("expected: %v, received %v", strategybase.ErrTooMuchBadData, err)
	}

	rsiIndicator := &RSI{}
	rsiIndicator.SetDefaults()
	s.SetDefaults()
	s.Settings.groupedIndicators = append(s.Settings.groupedIndicators, []Indicator{rsiIndicator})
	_, err = s.OnSignal(da, nil, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	da.Item = &gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   dStart,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	err = da.Load()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	ranger, err := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	da.RangeHolder = ranger
	err = da.RangeHolder.SetHasDataFromCandles(da.Item.Candles)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	resp, err = s.OnSignal(da, nil, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if resp.GetDirection() != order.DoNothing {
		t.Error("expected do nothing")
	}
}

func TestOnSignals(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := &data.Base{}
	err = d.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         dInsert,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	da := &kline.DataFromKline{
		Item:        &gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	_, err = s.OnSimultaneousSignals([]data.Handler{da}, nil, nil)
	if !strings.Contains(err.Error(), strategybase.ErrTooMuchBadData.Error()) {
		// common.Errs type doesn't keep type
		t.Errorf("received: %v, expected: %v", err, strategybase.ErrTooMuchBadData)
	}
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	s.SetDefaults()

}

func TestProcessOBV(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	b := binance.Binance{}
	b.SetDefaults()
	conf, _ := b.GetDefaultConfig()
	b.Setup(conf)
	b.CurrencyPairs.EnablePair(asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	candles, err := b.GetHistoricCandlesExtended(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot, gctkline.OneDay, time.Now().AddDate(-1, 0, 0), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	testClose := make([]float64, len(candles.Candles))
	testVolume := make([]float64, len(candles.Candles))
	for i := range candles.Candles {
		testClose[i] = candles.Candles[i].Close
		testVolume[i] = candles.Candles[i].Volume
	}
	obv := &OBV{}
	obv.SetDefaults()

	sig := &signal.Signal{
		Base: &event.Base{},
	}

	for i := range testClose {
		sig.Direction = order.UnknownSide
		if i <= 14 {
			continue
		}
		err := s.processOBV(testClose[:i], testVolume[:i], obv, sig)
		if err != nil {
			t.Error(err)
		}
		obvDir := sig.Direction
		sig.Direction = order.UnknownSide
		err = s.processRSI(testClose[:i], obv, sig)
		if err != nil {
			t.Error(err)
		}
		rsiDir := sig.Direction

		t.Logf("obv: %v, rsi: %v", obvDir, rsiDir)
	}

}

func TestProcessATR(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	b := binance.Binance{}
	b.SetDefaults()
	conf, _ := b.GetDefaultConfig()
	b.Setup(conf)
	b.CurrencyPairs.EnablePair(asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	candles, err := b.GetHistoricCandlesExtended(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot, gctkline.OneDay, time.Now().AddDate(-1, 0, 0), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	testClose := make([]float64, len(candles.Candles))
	testHigh := make([]float64, len(candles.Candles))
	testLow := make([]float64, len(candles.Candles))
	for i := range candles.Candles {
		testClose[i] = candles.Candles[i].Close
		testHigh[i] = candles.Candles[i].High
		testLow[i] = candles.Candles[i].Low
	}
	atr := &ATR{}
	atr.SetDefaults()

	sig := &signal.Signal{
		Base: &event.Base{},
	}
	err = s.processATR(testHigh, testLow, testClose, atr, sig)
	if err != nil {
		t.Error(err)
	}
}
