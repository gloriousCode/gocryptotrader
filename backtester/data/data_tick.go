package data

import (
	"errors"

	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type DataFromTick struct {
	ticks []*ticker.Price

	Data
}

func (d *DataFromTick) Load() error {
	if len(d.ticks) == 0 {
		return errors.New("no tick data provided")
	}

	data := make([]portfolio.DataEventHandler, len(d.ticks))
	for i := range d.ticks {
		data[i] = &Tick{
			Event: event.Event{
				Time:         d.ticks[i].LastUpdated,
				CurrencyPair: d.ticks[i].Pair,
			},
			Ask: d.ticks[i].Ask,
			Bid: d.ticks[i].Bid,
		}
	}
	d.stream = data
	d.SortStream()
	return nil
}
