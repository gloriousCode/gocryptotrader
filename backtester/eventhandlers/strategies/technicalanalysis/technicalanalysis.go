package technicalanalysis

import (
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"time"
)

const (
	// Name is the strategy name
	Name        = "technical analysis"
	description = `This strategy allows the use of multiple technical analysis indicators to make decisions`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	Settings CustomSettings
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

// OnSignal handles a data event and returns what action the strategy believes should occur
// For rsi, this means returning a buy signal when rsi is at or below a certain level, and a
// sell signal when it is at or above a certain level
func (s *Strategy) OnSignal(d data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	if d == nil {
		return nil, common.ErrNilEvent
	}
	es, err := s.GetBaseData(d)
	if err != nil {
		return nil, err
	}

	latest, err := d.Latest()
	if err != nil {
		return nil, err
	}

	es.SetPrice(latest.GetClosePrice())
	dataRange, err := d.StreamClose()
	if err != nil {
		return nil, err
	}
	var massagedData []float64
	massagedData, err = s.massageMissingData(dataRange, es.GetTime())
	if err != nil {
		return nil, err
	}

	hasDataAtTime, err := d.HasDataAtTime(latest.GetTime())
	if err != nil {
		return nil, err
	}
	if !hasDataAtTime {
		es.SetDirection(order.MissingData)
		es.AppendReasonf("missing data at %v, cannot perform any actions. RSI %v", latest.GetTime(), latestRSIValue)
		return &es, nil
	}

	for i := range s.Settings.GroupedIndicators {
	groupAnalysis:
		for j := range s.Settings.GroupedIndicators[i] {
			if offset := latest.GetOffset(); offset <= s.Settings.GroupedIndicators[i][j].GetPeriod() {
				es.AppendReason("Not enough data for signal generation")
				es.SetDirection(order.DoNothing)
				if s.Settings.GroupedIndicators[i][j].MustPass() {
					es.AppendReasonf("group %v failed check", s.Settings.GroupedIndicators[i][j].GetGroup())
					break groupAnalysis
				}
				continue
			}

			switch s.Settings.GroupedIndicators[i][j].GetName() {
			case rsiName:
				rsi := indicators.RSI(massagedData, int(s.Settings.GroupedIndicators[i][j].GetPeriod()))
				latestRSIValue := rsi[len(rsi)-1]
				switch {
				case latestRSIValue >= s.Settings.GroupedIndicators[i][j].GetHigh():
					err = setDirection(&es, order.Sell)
					if err != nil && s.Settings.GroupedIndicators[i][j].MustPass() {
						es.AppendReasonf("group %v failed check", s.Settings.GroupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				case latestRSIValue <= s.Settings.GroupedIndicators[i][j].GetLow():
					err = setDirection(&es, order.Buy)
					if err != nil && s.Settings.GroupedIndicators[i][j].MustPass() {
						es.AppendReasonf("group %v failed check", s.Settings.GroupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				default:
					es.SetDirection(order.DoNothing)
					if s.Settings.GroupedIndicators[i][j].MustPass() {
						es.AppendReasonf("group %v failed check", s.Settings.GroupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				}
				es.AppendReasonf("RSI at %v", latestRSIValue)
			case macdName:
				macd, signal, _ := indicators.MACD(massagedData, int(s.Settings.GroupedIndicators[i][j].GetFastPeriod()), int(s.Settings.GroupedIndicators[i][j].GetSlowPeriod()), int(s.Settings.GroupedIndicators[i][j].GetPeriod()))
				latestMacd := macd[len(macd)-1]
				latestSignal := macd[len(signal)-1]
				previousMacd := macd[len(macd)-2]
				previousSignal := macd[len(signal)-2]

				if latestMacd > latestSignal && previousMacd <= previousSignal {
					err = setDirection(&es, order.Sell)
					if err != nil && s.Settings.GroupedIndicators[i][j].MustPass() {
						es.AppendReasonf("group %v failed check", s.Settings.GroupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
					continue
				}
				if latestMacd < latestSignal && previousMacd >= previousSignal {
					err = setDirection(&es, order.Buy)
					if err != nil && s.Settings.GroupedIndicators[i][j].MustPass() {
						es.AppendReasonf("group %v failed check", s.Settings.GroupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
					continue
				}
				if latestMacd > 0 && previousMacd < 0 {
					err = setDirection(&es, order.Buy)
					if err != nil && s.Settings.GroupedIndicators[i][j].MustPass() {
						es.AppendReasonf("group %v failed check", s.Settings.GroupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
					continue
				}
				if latestMacd < 0 && previousMacd > 0 {
					err = setDirection(&es, order.Sell)
					if err != nil && s.Settings.GroupedIndicators[i][j].MustPass() {
						es.AppendReasonf("group %v failed check", s.Settings.GroupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
					continue
				}
			}
		}
	}

	return &es, nil
}

func setDirection(es *signal.Signal, direction order.Side) error {
	switch es.Direction {
	case order.Buy:
		if direction == order.Sell {
			es.AppendReason("conflicting indicators results, cannot switch from buy to sell")
			es.SetDirection(order.DoNothing)
		}
	case order.Sell:
		if direction == order.Buy {
			es.AppendReason("conflicting indicators results, cannot switch from sell to buy")
			es.SetDirection(order.DoNothing)
		}
	}
	es.Direction = direction
	return nil
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	var resp []signal.Event
	var errs gctcommon.Errors
	for i := range d {
		latest, err := d[i].Latest()
		if err != nil {
			return nil, err
		}
		sigEvent, err := s.OnSignal(d[i], nil, nil)
		if err != nil {
			errs = append(errs, fmt.Errorf("%v %v %v %w", latest.GetExchange(), latest.GetAssetType(), latest.Pair(), err))
		} else {
			resp = append(resp, sigEvent)
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return resp, nil
}

// SetCustomSettings allows a user to modify the RSI limits in their config
func (s *Strategy) SetCustomSettings(customSettings json.RawMessage) error {
	var settings CustomSettings
	err := json.Unmarshal(customSettings, &settings)
	if err != nil {
		return err
	}
	indicatorMap := make(map[string][]Indicator)
	for i := range settings.Indicators {
		if err = settings.Indicators[i].Validate(); err != nil {
			return err
		}
		groupMap := indicatorMap[settings.Indicators[i].GetGroup()]
		groupMap = append(groupMap, settings.Indicators[i])
	}
	for _, v := range indicatorMap {
		settings.GroupedIndicators = append(settings.GroupedIndicators, v)
	}
	s.Settings = settings
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
