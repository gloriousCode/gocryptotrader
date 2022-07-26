package ticker

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/ticker"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctticker "github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// HasDataAtTime verifies checks the underlying range data
// To determine whether there is any candle data present at the time provided
func (d *Data) HasDataAtTime(t time.Time) bool {
	_, ok := d.addedTimes[t.Round(time.Minute).UnixNano()]
	return ok
}

// Load sets the candle data to the stream for processing
func (d *Data) Load() error {
	d.addedTimes = make(map[int64]bool)
	if len(d.Tickers) == 0 {
		return errNoTickerData
	}

	data := make([]common.DataEventHandler, len(d.Tickers))
	for i := range d.Tickers {
		t := &ticker.Ticker{
			Base: &event.Base{
				Offset:         int64(i + 1),
				Exchange:       d.Tickers[i].ExchangeName,
				Time:           d.Tickers[i].LastUpdated.Round(time.Minute).UTC(),
				Interval:       d.PollTime,
				CurrencyPair:   d.Tickers[i].Pair,
				AssetType:      d.Tickers[i].AssetType,
				UnderlyingPair: d.UnderlyingPair,
			},
			Last:        decimal.NewFromFloat(d.Tickers[i].Last),
			High:        decimal.NewFromFloat(d.Tickers[i].High),
			Low:         decimal.NewFromFloat(d.Tickers[i].Low),
			Bid:         decimal.NewFromFloat(d.Tickers[i].Bid),
			Ask:         decimal.NewFromFloat(d.Tickers[i].Ask),
			Volume:      decimal.NewFromFloat(d.Tickers[i].Volume),
			QuoteVolume: decimal.NewFromFloat(d.Tickers[i].QuoteVolume),
			Open:        decimal.NewFromFloat(d.Tickers[i].Open),
			Close:       decimal.NewFromFloat(d.Tickers[i].Close),
		}
		data[i] = t
		d.addedTimes[t.Time.Round(time.Minute).UnixNano()] = true
	}

	d.SetStream(data)
	d.SortStream()
	return nil
}

// AppendTicker is used to append ticker data for live data or something
func (d *Data) AppendTicker(underlyingPair currency.Pair, t *gctticker.Price) {
	if d.addedTimes == nil {
		d.addedTimes = make(map[int64]bool)
	}

	if _, ok := d.addedTimes[t.LastUpdated.Round(time.Minute).UnixNano()]; ok {
		return
	}
	d.addedTimes[t.LastUpdated.Round(time.Minute).UnixNano()] = true
	offset := int64(len(d.List())) + 1
	d.AppendStream(&ticker.Ticker{
		Base: &event.Base{
			Offset:         offset,
			Exchange:       t.ExchangeName,
			Time:           t.LastUpdated.Round(time.Minute),
			CurrencyPair:   t.Pair,
			AssetType:      t.AssetType,
			Interval:       gctkline.Interval(time.Second * 5),
			UnderlyingPair: underlyingPair,
		},
		Last:        decimal.NewFromFloat(t.Last),
		High:        decimal.NewFromFloat(t.High),
		Low:         decimal.NewFromFloat(t.Low),
		Bid:         decimal.NewFromFloat(t.Bid),
		Ask:         decimal.NewFromFloat(t.Ask),
		Volume:      decimal.NewFromFloat(t.Volume),
		QuoteVolume: decimal.NewFromFloat(t.QuoteVolume),
		Open:        decimal.NewFromFloat(t.Open),
		Close:       decimal.NewFromFloat(t.Close),
	})
	d.SortStream()
}

func (d *Data) GetDataType() uint8 {
	return 1
}

// StreamOpen returns all Open prices from the beginning until the current iteration
func (d *Data) StreamOpen() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*ticker.Ticker); ok {
			ret[x] = val.Open
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamHigh returns all High prices from the beginning until the current iteration
func (d *Data) StreamHigh() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*ticker.Ticker); ok {
			ret[x] = val.High
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamLow returns all Low prices from the beginning until the current iteration
func (d *Data) StreamLow() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*ticker.Ticker); ok {
			ret[x] = val.Low
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamClose returns all Close prices from the beginning until the current iteration
func (d *Data) StreamClose() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*ticker.Ticker); ok {
			ret[x] = val.Close
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamVol returns all Volume prices from the beginning until the current iteration
func (d *Data) StreamVol() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*ticker.Ticker); ok {
			ret[x] = val.Volume
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}
