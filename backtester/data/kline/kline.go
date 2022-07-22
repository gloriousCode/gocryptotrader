package kline

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/ticker"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctticker "github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// HasDataAtTime verifies checks the underlying range data
// To determine whether there is any candle data present at the time provided
func (d *PriceData) HasDataAtTime(t time.Time) bool {
	if d.RangeHolder == nil {
		return false
	}
	return d.RangeHolder.HasDataAtDate(t)
}

// Load sets the candle data to the stream for processing
func (d *PriceData) Load() error {
	d.addedTimes = make(map[int64]bool)
	if len(d.KLine.Candles) == 0 {
		return errNoCandleData
	}

	klineData := make([]common.DataEventHandler, len(d.KLine.Candles))
	for i := range d.KLine.Candles {
		newKline := &kline.Kline{
			Base: &event.Base{
				Offset:         int64(i + 1),
				Exchange:       d.KLine.Exchange,
				Time:           d.KLine.Candles[i].Time.UTC(),
				Interval:       d.KLine.Interval,
				CurrencyPair:   d.KLine.Pair,
				AssetType:      d.KLine.Asset,
				UnderlyingPair: d.KLine.UnderlyingPair,
			},
			Open:             decimal.NewFromFloat(d.KLine.Candles[i].Open),
			High:             decimal.NewFromFloat(d.KLine.Candles[i].High),
			Low:              decimal.NewFromFloat(d.KLine.Candles[i].Low),
			Close:            decimal.NewFromFloat(d.KLine.Candles[i].Close),
			Volume:           decimal.NewFromFloat(d.KLine.Candles[i].Volume),
			ValidationIssues: d.KLine.Candles[i].ValidationIssues,
		}
		klineData[i] = newKline
		d.addedTimes[d.KLine.Candles[i].Time.UTC().UnixNano()] = true
	}

	d.SetStream(klineData)
	d.SortStream()
	return nil
}

// AppendTicker is used to append ticker data for live data or something
func (d *PriceData) AppendTicker(exch string, a asset.Item, cp currency.Pair, t *gctticker.Price) {
	if d.addedTimes == nil {
		d.addedTimes = make(map[int64]bool)
	}

	if _, ok := d.addedTimes[t.LastUpdated.UnixNano()]; ok {
		return
	}
	d.addedTimes[t.LastUpdated.UnixNano()] = true
	offset := int64(len(d.List())) + 1
	d.AppendStream(&ticker.Ticker{
		Base: &event.Base{
			Offset:       offset,
			Exchange:     exch,
			Time:         t.LastUpdated,
			CurrencyPair: cp,
			AssetType:    a,
			Interval:     gctkline.Interval(time.Second * 5),
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

// AppendKLine adds a candle item to the data stream and sorts it to ensure it is all in order
func (d *PriceData) AppendKLine(ki *gctkline.Item) {
	if d.addedTimes == nil {
		d.addedTimes = make(map[int64]bool)
	}

	var gctCandles []gctkline.Candle
	for i := range ki.Candles {
		if _, ok := d.addedTimes[ki.Candles[i].Time.UnixNano()]; !ok {
			gctCandles = append(gctCandles, ki.Candles[i])
			d.addedTimes[ki.Candles[i].Time.UnixNano()] = true
		}
	}

	klineData := make([]common.DataEventHandler, len(gctCandles))
	candleTimes := make([]time.Time, len(gctCandles))
	for i := range gctCandles {
		klineData[i] = &kline.Kline{
			Base: &event.Base{
				Offset:       int64(i + 1),
				Exchange:     ki.Exchange,
				Time:         gctCandles[i].Time,
				Interval:     ki.Interval,
				CurrencyPair: ki.Pair,
				AssetType:    ki.Asset,
			},
			Open:             decimal.NewFromFloat(gctCandles[i].Open),
			High:             decimal.NewFromFloat(gctCandles[i].High),
			Low:              decimal.NewFromFloat(gctCandles[i].Low),
			Close:            decimal.NewFromFloat(gctCandles[i].Close),
			Volume:           decimal.NewFromFloat(gctCandles[i].Volume),
			ValidationIssues: gctCandles[i].ValidationIssues,
		}
		candleTimes[i] = gctCandles[i].Time
	}
	for i := range d.RangeHolder.Ranges {
		for j := range d.RangeHolder.Ranges[i].Intervals {
			d.RangeHolder.Ranges[i].Intervals[j].HasData = true
		}
	}
	log.Debugf(common.Data, "appending %v candle intervals: %v", len(gctCandles), candleTimes)
	d.AppendStream(klineData...)
	d.SortStream()
}

// StreamOpen returns all Open prices from the beginning until the current iteration
func (d *PriceData) StreamOpen() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.Open
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamHigh returns all High prices from the beginning until the current iteration
func (d *PriceData) StreamHigh() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.High
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamLow returns all Low prices from the beginning until the current iteration
func (d *PriceData) StreamLow() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.Low
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamClose returns all Close prices from the beginning until the current iteration
func (d *PriceData) StreamClose() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.Close
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamVol returns all Volume prices from the beginning until the current iteration
func (d *PriceData) StreamVol() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.Volume
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}
