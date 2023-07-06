package rsi

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name         = "rsi"
	rsiPeriodKey = "rsi-period"
	rsiLowKey    = "rsi-low"
	rsiHighKey   = "rsi-high"
	description  = `The relative strength index is a technical indicator used in the analysis of financial markets. It is intended to chart the current and historical strength or weakness of a stock or market based on the closing prices of a recent trading period`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	rsiPeriod decimal.Decimal
	rsiLow    decimal.Decimal
	rsiHigh   decimal.Decimal
}

// Name returns the name of the strategy
func (s *Strategy) Name() string {
	return Name
}

// Description provides a nice overview of the strategy
// be it definition of terms or to highlight its purpose
func (s *Strategy) Description() string {
	return description
}

// Execute analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) Execute(d []data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	var resp []signal.Event
	var errs error
	for i := range d {
		latest, err := d[i].Latest()
		if err != nil {
			return nil, err
		}
		es, err := s.GetBaseData(d[i])
		if err != nil {
			return nil, err
		}

		es.SetPrice(latest.GetClosePrice())

		if offset := latest.GetOffset(); offset <= s.rsiPeriod.IntPart() {
			es.AppendReason("Not enough data for signal generation")
			es.SetDirection(order.DoNothing)
			continue
		}

		dataRange, err := d[i].StreamClose()
		if err != nil {
			errs = gctcommon.AppendError(errs, fmt.Errorf("%v %v %v %w",
				latest.GetExchange(),
				latest.GetAssetType(),
				latest.Pair(),
				err))
			continue
		}
		var massagedData []float64
		massagedData, err = s.massageMissingData(dataRange, es.GetTime())
		if err != nil {
			errs = gctcommon.AppendError(errs, fmt.Errorf("%v %v %v %w",
				latest.GetExchange(),
				latest.GetAssetType(),
				latest.Pair(),
				err))
			continue
		}
		rsi := indicators.RSI(massagedData, int(s.rsiPeriod.IntPart()))
		latestRSIValue := decimal.NewFromFloat(rsi[len(rsi)-1])
		hasDataAtTime, err := d[i].HasDataAtTime(latest.GetTime())
		if err != nil {
			errs = gctcommon.AppendError(errs, fmt.Errorf("%v %v %v %w",
				latest.GetExchange(),
				latest.GetAssetType(),
				latest.Pair(),
				err))
			continue
		}
		if !hasDataAtTime {
			es.SetDirection(order.MissingData)
			es.AppendReasonf("missing data at %v, cannot perform any actions. RSI %v", latest.GetTime(), latestRSIValue)
			continue
		}

		switch {
		case latestRSIValue.GreaterThanOrEqual(s.rsiHigh):
			es.SetDirection(order.Sell)
		case latestRSIValue.LessThanOrEqual(s.rsiLow):
			es.SetDirection(order.Buy)
		default:
			es.SetDirection(order.DoNothing)
		}
		es.AppendReasonf("RSI at %v", latestRSIValue)
		resp = append(resp, &es)
	}
	return resp, errs
}

// SetCustomSettings allows a user to modify the RSI limits in their config
func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	for k, v := range customSettings {
		switch k {
		case rsiHighKey:
			rsiHigh, ok := v.(float64)
			if !ok || rsiHigh <= 0 {
				return fmt.Errorf("%w provided rsi-high value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiHigh = decimal.NewFromFloat(rsiHigh)
		case rsiLowKey:
			rsiLow, ok := v.(float64)
			if !ok || rsiLow <= 0 {
				return fmt.Errorf("%w provided rsi-low value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiLow = decimal.NewFromFloat(rsiLow)
		case rsiPeriodKey:
			rsiPeriod, ok := v.(float64)
			if !ok || rsiPeriod <= 0 {
				return fmt.Errorf("%w provided rsi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiPeriod = decimal.NewFromFloat(rsiPeriod)
		default:
			return fmt.Errorf("%w unrecognised custom setting key %v with value %v. Cannot apply", base.ErrInvalidCustomSettings, k, v)
		}
	}

	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	s.rsiHigh = decimal.NewFromInt(70)
	s.rsiLow = decimal.NewFromInt(30)
	s.rsiPeriod = decimal.NewFromInt(14)
}

// massageMissingData will replace missing data with the previous candle's data
// this will ensure that RSI can be calculated correctly
// the decision to handle missing data occurs at the strategy level, not all strategies
// may wish to modify data
func (s *Strategy) massageMissingData(data []decimal.Decimal, t time.Time) ([]float64, error) {
	resp := make([]float64, len(data))
	var missingDataStreak int64
	for i := range data {
		if data[i].IsZero() && i > int(s.rsiPeriod.IntPart()) {
			data[i] = data[i-1]
			missingDataStreak++
		} else {
			missingDataStreak = 0
		}
		if missingDataStreak >= s.rsiPeriod.IntPart() {
			return nil, fmt.Errorf("missing data exceeds RSI period length of %v at %s and will distort results. %w",
				s.rsiPeriod,
				t.Format(gctcommon.SimpleTimeFormat),
				base.ErrTooMuchBadData)
		}
		resp[i] = data[i].InexactFloat64()
	}
	return resp, nil
}
