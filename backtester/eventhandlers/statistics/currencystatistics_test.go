package statistics

import (
	"testing"
	"time"

	"github.com/quagmt/udecimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestCalculateResults(t *testing.T) {
	t.Parallel()
	a := asset.Spot
	cs := CurrencyPairStatistic{
		Asset: a,
	}
	tt1 := time.Now()
	tt2 := time.Now().Add(gctkline.OneDay.Duration())
	exch := testExchange
	p := currency.NewBTCUSDT()
	even := &event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
		Offset:       1,
	}
	ev := DataAtOffset{
		Offset:     1,
		Time:       tt1,
		ClosePrice: udecimal.MustFromFloat64(2000),
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: udecimal.MustFromFloat64(0.1333),
			Timestamp:                 tt1,
			QuoteInitialFunds:         udecimal.MustFromFloat64(1337),
		},
		ComplianceSnapshot: &compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          udecimal.MustFromFloat64(1338),
					VolumeAdjustedPrice: udecimal.MustFromFloat64(1338),
					SlippageRate:        udecimal.MustFromFloat64(1338),
					CostBasis:           udecimal.MustFromFloat64(1338),
					Order:               &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          udecimal.MustFromFloat64(1337),
					VolumeAdjustedPrice: udecimal.MustFromFloat64(1337),
					SlippageRate:        udecimal.MustFromFloat64(1337),
					CostBasis:           udecimal.MustFromFloat64(1337),
					Order:               &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even,
			Open:   udecimal.MustFromFloat64(2000),
			Close:  udecimal.MustFromFloat64(2000),
			Low:    udecimal.MustFromFloat64(2000),
			High:   udecimal.MustFromFloat64(2000),
			Volume: udecimal.MustFromFloat64(2000),
		},
		SignalEvent: &signal.Signal{
			Base:       even,
			ClosePrice: udecimal.MustFromFloat64(2000),
		},
	}
	even2 := even
	even2.Time = tt2
	even2.Offset = 2
	ev2 := DataAtOffset{
		Offset:     2,
		Time:       tt2,
		ClosePrice: udecimal.MustFromFloat64(1337),
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: udecimal.MustFromFloat64(0.1337),
			Timestamp:                 tt2,
			QuoteInitialFunds:         udecimal.MustFromFloat64(1337),
		},
		ComplianceSnapshot: &compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          udecimal.MustFromFloat64(1338),
					VolumeAdjustedPrice: udecimal.MustFromFloat64(1338),
					SlippageRate:        udecimal.MustFromFloat64(1338),
					CostBasis:           udecimal.MustFromFloat64(1338),
					Order:               &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          udecimal.MustFromFloat64(1337),
					VolumeAdjustedPrice: udecimal.MustFromFloat64(1337),
					SlippageRate:        udecimal.MustFromFloat64(1337),
					CostBasis:           udecimal.MustFromFloat64(1337),
					Order:               &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even2,
			Open:   udecimal.MustFromFloat64(1337),
			Close:  udecimal.MustFromFloat64(1337),
			Low:    udecimal.MustFromFloat64(1337),
			High:   udecimal.MustFromFloat64(1337),
			Volume: udecimal.MustFromFloat64(1337),
		},
		SignalEvent: &signal.Signal{
			Base:       even2,
			ClosePrice: udecimal.MustFromFloat64(1337),
			Direction:  order.MissingData,
		},
	}

	cs.Events = append(cs.Events, ev, ev2)
	err := cs.CalculateResults(udecimal.MustFromFloat64(0.03))
	assert.NoError(t, err)

	if !cs.MarketMovement.Equal(udecimal.MustFromFloat64(-33.15)) {
		t.Errorf("expected -33.15 received '%v'", cs.MarketMovement)
	}
	ev3 := ev2
	ev3.DataEvent = &kline.Kline{
		Base:   even2,
		Open:   udecimal.MustFromFloat64(1339),
		Close:  udecimal.MustFromFloat64(1339),
		Low:    udecimal.MustFromFloat64(1339),
		High:   udecimal.MustFromFloat64(1339),
		Volume: udecimal.MustFromFloat64(1339),
	}
	cs.Events = append(cs.Events, ev, ev3)
	cs.Events[0].DataEvent = &kline.Kline{
		Base: even2,
	}
	err = cs.CalculateResults(udecimal.MustFromFloat64(0.03))
	assert.NoError(t, err)

	cs.Events[1].DataEvent = &kline.Kline{
		Base: even2,
	}
	err = cs.CalculateResults(udecimal.MustFromFloat64(0.03))
	assert.NoError(t, err)
}

func TestPrintResults(t *testing.T) {
	cs := CurrencyPairStatistic{}
	tt1 := time.Now()
	tt2 := time.Now().Add(gctkline.OneDay.Duration())
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	even := &event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	ev := DataAtOffset{
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: udecimal.MustFromFloat64(0.1333),
			Timestamp:                 tt1,
			QuoteInitialFunds:         udecimal.MustFromFloat64(1337),
		},
		ComplianceSnapshot: &compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          udecimal.MustFromFloat64(1338),
					VolumeAdjustedPrice: udecimal.MustFromFloat64(1338),
					SlippageRate:        udecimal.MustFromFloat64(1338),
					CostBasis:           udecimal.MustFromFloat64(1338),
					Order:               &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          udecimal.MustFromFloat64(1337),
					VolumeAdjustedPrice: udecimal.MustFromFloat64(1337),
					SlippageRate:        udecimal.MustFromFloat64(1337),
					CostBasis:           udecimal.MustFromFloat64(1337),
					Order:               &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even,
			Open:   udecimal.MustFromFloat64(2000),
			Close:  udecimal.MustFromFloat64(2000),
			Low:    udecimal.MustFromFloat64(2000),
			High:   udecimal.MustFromFloat64(2000),
			Volume: udecimal.MustFromFloat64(2000),
		},
		SignalEvent: &signal.Signal{
			Base:       even,
			ClosePrice: udecimal.MustFromFloat64(2000),
		},
	}
	even2 := even
	even2.Time = tt2
	ev2 := DataAtOffset{
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: udecimal.MustFromFloat64(0.1337),
			Timestamp:                 tt2,
			QuoteInitialFunds:         udecimal.MustFromFloat64(1337),
		},
		ComplianceSnapshot: &compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          udecimal.MustFromFloat64(1338),
					VolumeAdjustedPrice: udecimal.MustFromFloat64(1338),
					SlippageRate:        udecimal.MustFromFloat64(1338),
					CostBasis:           udecimal.MustFromFloat64(1338),
					Order:               &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          udecimal.MustFromFloat64(1337),
					VolumeAdjustedPrice: udecimal.MustFromFloat64(1337),
					SlippageRate:        udecimal.MustFromFloat64(1337),
					CostBasis:           udecimal.MustFromFloat64(1337),
					Order:               &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even2,
			Open:   udecimal.MustFromFloat64(1337),
			Close:  udecimal.MustFromFloat64(1337),
			Low:    udecimal.MustFromFloat64(1337),
			High:   udecimal.MustFromFloat64(1337),
			Volume: udecimal.MustFromFloat64(1337),
		},
		SignalEvent: &signal.Signal{
			Base:       even2,
			ClosePrice: udecimal.MustFromFloat64(1337),
		},
	}

	cs.Events = append(cs.Events, ev, ev2)
	err := cs.PrintResults(exch, a, p, true)
	if err != nil {
		t.Error(err)
	}
}

func TestCalculateHighestCommittedFunds(t *testing.T) {
	t.Parallel()
	c := CurrencyPairStatistic{
		Asset: asset.Spot,
	}
	err := c.calculateHighestCommittedFunds()
	assert.NoError(t, err)

	if !c.HighestCommittedFunds.Time.IsZero() {
		t.Error("expected no time with not committed funds")
	}
	tt1 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	tt2 := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
	tt3 := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	c.Events = append(c.Events,
		DataAtOffset{DataEvent: &kline.Kline{Close: udecimal.MustFromFloat64(1337)}, Time: tt1, Holdings: holdings.Holding{Timestamp: tt1, CommittedFunds: udecimal.MustFromFloat64(10), BaseSize: udecimal.MustFromFloat64(10)}},
		DataAtOffset{DataEvent: &kline.Kline{Close: udecimal.MustFromFloat64(1338)}, Time: tt2, Holdings: holdings.Holding{Timestamp: tt2, CommittedFunds: udecimal.MustFromFloat64(1337), BaseSize: udecimal.MustFromFloat64(1337)}},
		DataAtOffset{DataEvent: &kline.Kline{Close: udecimal.MustFromFloat64(1339)}, Time: tt3, Holdings: holdings.Holding{Timestamp: tt3, CommittedFunds: udecimal.MustFromFloat64(11), BaseSize: udecimal.MustFromFloat64(11)}},
	)
	err = c.calculateHighestCommittedFunds()
	assert.NoError(t, err)

	if c.HighestCommittedFunds.Time != tt2 {
		t.Errorf("expected %v, received %v", tt2, c.HighestCommittedFunds.Time)
	}

	c.Asset = asset.Futures
	c.HighestCommittedFunds = ValueAtTime{}
	err = c.calculateHighestCommittedFunds()
	assert.NoError(t, err)

	c.Asset = asset.Binary
	err = c.calculateHighestCommittedFunds()
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestAnalysePNLGrowth(t *testing.T) {
	t.Parallel()
	c := CurrencyPairStatistic{}
	c.analysePNLGrowth()
	if !c.HighestUnrealisedPNL.Value.IsZero() ||
		!c.LowestUnrealisedPNL.Value.IsZero() ||
		!c.LowestRealisedPNL.Value.IsZero() ||
		!c.HighestRealisedPNL.Value.IsZero() {
		t.Error("expected unset")
	}

	e := testExchange
	a := asset.Futures
	p := currency.NewBTCUSDT()
	c.Asset = asset.Futures
	c.Events = append(c.Events,
		DataAtOffset{PNL: &portfolio.PNLSummary{
			Exchange: e,
			Asset:    a,
			Pair:     p,
			Result: futures.PNLResult{
				Time:          time.Now(),
				UnrealisedPNL: udecimal.MustFromFloat64(1),
				RealisedPNL:   udecimal.MustFromFloat64(2),
			},
		}},
	)

	c.analysePNLGrowth()
	if !c.HighestRealisedPNL.Value.Equal(udecimal.MustFromFloat64(2)) {
		t.Errorf("received %v expected 2", c.HighestRealisedPNL.Value)
	}
	if !c.LowestUnrealisedPNL.Value.Equal(udecimal.MustFromFloat64(1)) {
		t.Errorf("received %v expected 1", c.LowestUnrealisedPNL.Value)
	}

	c.Events = append(c.Events,
		DataAtOffset{PNL: &portfolio.PNLSummary{
			Exchange: e,
			Asset:    a,
			Pair:     p,
			Result: futures.PNLResult{
				Time:          time.Now(),
				UnrealisedPNL: udecimal.MustFromFloat64(0.5),
				RealisedPNL:   udecimal.MustFromFloat64(1),
			},
		}},
	)

	c.analysePNLGrowth()
	if !c.HighestRealisedPNL.Value.Equal(udecimal.MustFromFloat64(2)) {
		t.Errorf("received %v expected 2", c.HighestRealisedPNL.Value)
	}
	if !c.LowestUnrealisedPNL.Value.Equal(udecimal.MustFromFloat64(0.5)) {
		t.Errorf("received %v expected 0.5", c.LowestUnrealisedPNL.Value)
	}
}
