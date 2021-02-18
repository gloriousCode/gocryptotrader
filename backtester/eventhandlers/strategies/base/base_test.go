package base

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestGetBase(t *testing.T) {
	s := Strategy{}
	_, err := s.GetBase(nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("expected: %v, reveived %v", common.ErrNilArguments, err)
	}

	_, err = s.GetBase(&datakline.DataFromKline{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, reveived %v", common.ErrNilEvent, err)
	}
	tt := time.Now()
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := data.Base{}
	d.SetStream([]common.DataEventHandler{&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	}})

	d.Next()
	_, err = s.GetBase(&datakline.DataFromKline{
		Item:  gctkline.Item{},
		Base:  d,
		Range: gctkline.IntervalRangeHolder{},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSetSimultaneousProcessing(t *testing.T) {
	s := Strategy{}
	is := s.UseSimultaneousProcessing()
	if is {
		t.Error("expected false")
	}
	s.SetSimultaneousProcessing(true)
	is = s.UseSimultaneousProcessing()
	if !is {
		t.Error("expected true")
	}
}
