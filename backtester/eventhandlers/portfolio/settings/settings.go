package settings

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
)

// GetLatestHoldings returns the latest holdings after being sorted by time
func (e *Settings) GetLatestHoldings() holdings.Holding {
	if e.HoldingsSnapshots == nil {
		// no holdings yet
		return holdings.Holding{}
	}

	return e.HoldingsSnapshots[len(e.HoldingsSnapshots)-1]
}

// GetHoldingsForTime returns the holdings for a time period, or an empty holding if not found
func (e *Settings) GetHoldingsForTime(t time.Time) holdings.Holding {
	if e.HoldingsSnapshots == nil {
		// no holdings yet
		return holdings.Holding{}
	}
	// check if this is the latest, if so, it will save lots of time
	if t.Equal(e.HoldingsSnapshots[len(e.HoldingsSnapshots)-1].Timestamp) {
		return e.HoldingsSnapshots[len(e.HoldingsSnapshots)-1]
	}
	// and event the one before it. This will save thousands of iterations if true
	if t.Equal(e.HoldingsSnapshots[len(e.HoldingsSnapshots)-2].Timestamp) {
		return e.HoldingsSnapshots[len(e.HoldingsSnapshots)-2]
	}

	for i := range e.HoldingsSnapshots {
		if e.HoldingsSnapshots[i].Timestamp.Equal(t) {
			return e.HoldingsSnapshots[i]
		}
	}
	return holdings.Holding{}
}

// Value returns the total value of the latest holdings
func (e *Settings) Value() float64 {
	latest := e.GetLatestHoldings()
	return latest.TotalValue
}
