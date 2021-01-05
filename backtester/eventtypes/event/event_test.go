package event

import (
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestEvent_AppendWhy(t *testing.T) {
	e := &Event{}
	e.AppendWhy("test")
	y := e.GetWhy()
	if !strings.Contains(y, "test") {
		t.Error("expected test")
	}
}

func TestEvent_GetAssetType(t *testing.T) {
	e := &Event{
		AssetType: asset.Spot,
	}
	y := e.GetAssetType()
	if y != asset.Spot {
		t.Error("expected spot")
	}
}

func TestEvent_GetExchange(t *testing.T) {
	e := &Event{
		Exchange: "test",
	}
	y := e.GetExchange()
	if y != "test" {
		t.Error("expected test")
	}
}

func TestEvent_GetInterval(t *testing.T) {
	e := &Event{
		Interval: gctkline.OneMin,
	}
	y := e.GetInterval()
	if y != gctkline.OneMin {
		t.Error("expected one minute")
	}
}

func TestEvent_GetTime(t *testing.T) {
	tt := time.Now()
	e := &Event{
		Time: tt,
	}
	y := e.GetTime()
	if !y.Equal(tt) {
		t.Errorf("expected %v", tt)
	}
}

func TestEvent_IsEvent(t *testing.T) {
	e := &Event{}
	y := e.IsEvent()
	if !y {
		t.Error("it is an event")
	}
}

func TestEvent_Pair(t *testing.T) {
	e := &Event{
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
	}
	y := e.Pair()
	if y.IsEmpty() {
		t.Error("expected currency")
	}
}
