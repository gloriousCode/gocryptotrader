package technicalanalysis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/strategybase"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// Name is the strategy name
	Name        = "technicalanalysis"
	description = `This strategy allows the use of multiple technical analysis indicators to make strategic decisions`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	strategybase.Strategy
	Settings CustomSettings
}

// New creates a new instance of a strategy
func (s *Strategy) New() strategybase.Handler {
	return &Strategy{
		Strategy: strategybase.Strategy{
			Name:        Name,
			Description: description,
		},
	}
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
		es.AppendReasonf("missing data at %v, cannot perform any actions", latest.GetTime())
		return &es, nil
	}

	for i := range s.Settings.groupedIndicators {
	groupAnalysis:
		for j := range s.Settings.groupedIndicators[i] {
			if offset := latest.GetOffset(); offset <= s.Settings.groupedIndicators[i][j].GetPeriod() {
				es.AppendReason("Not enough data for signal generation")
				es.SetDirection(order.DoNothing)
				if s.Settings.groupedIndicators[i][j].MustPass() {
					es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
					break groupAnalysis
				}
				continue
			}

			switch s.Settings.groupedIndicators[i][j].GetName() {
			case RSIName:
				rsi := indicators.RSI(massagedData, int(s.Settings.groupedIndicators[i][j].GetPeriod()))
				latestRSIValue := rsi[len(rsi)-1]
				switch {
				case latestRSIValue >= s.Settings.groupedIndicators[i][j].GetHigh():
					err = setDirection(&es, order.Sell)
					if err != nil && s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				case latestRSIValue <= s.Settings.groupedIndicators[i][j].GetLow():
					err = setDirection(&es, order.Buy)
					if err != nil && s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				default:
					_ = setDirection(&es, order.DoNothing)
					if s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				}
				es.AppendReasonf("RSI at %v", latestRSIValue)
			case MACDName:
				macd, signal, _ := indicators.MACD(massagedData, int(s.Settings.groupedIndicators[i][j].GetFastPeriod()), int(s.Settings.groupedIndicators[i][j].GetSlowPeriod()), int(s.Settings.groupedIndicators[i][j].GetPeriod()))
				latestMacd := macd[len(macd)-1]
				latestSignal := macd[len(signal)-1]
				previousMacd := macd[len(macd)-2]
				previousSignal := macd[len(signal)-2]
				switch {
				case latestMacd > latestSignal && previousMacd <= previousSignal:
					err = setDirection(&es, order.Sell)
					if err != nil && s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				case latestMacd < latestSignal && previousMacd >= previousSignal:
					err = setDirection(&es, order.Buy)
					if err != nil && s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				case latestMacd > 0 && previousMacd < 0:
					err = setDirection(&es, order.Buy)
					if err != nil && s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				case latestMacd < 0 && previousMacd > 0:
					err = setDirection(&es, order.Sell)
					if err != nil && s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				default:
					_ = setDirection(&es, order.DoNothing)
					if s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				}
			case BBandsName:
				upper, _, lower := indicators.BBANDS(massagedData, int(s.Settings.groupedIndicators[i][j].GetPeriod()), s.Settings.groupedIndicators[i][j].GetUp(), s.Settings.groupedIndicators[i][j].GetDown(), indicators.Sma)
				closePrice := latest.GetClosePrice().InexactFloat64()
				latestUpper := upper[len(upper)-1]
				latestDowner := lower[len(lower)-1]
				switch {
				case closePrice >= latestUpper:
					err = setDirection(&es, order.Sell)
					if err != nil && s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				case closePrice <= latestDowner:
					err = setDirection(&es, order.Buy)
					if err != nil && s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
				default:
					_ = setDirection(&es, order.DoNothing)
					if s.Settings.groupedIndicators[i][j].MustPass() {
						es.AppendReasonf("indicator %v of group %v failed check", s.Settings.groupedIndicators[i][j].GetName(), s.Settings.groupedIndicators[i][j].GetGroup())
						break groupAnalysis
					}
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
		groupMap := indicatorMap[settings.Indicators[i].GetGroup()]
		switch settings.Indicators[i].Name {
		case RSIName:
			rsi := RSI{settings.Indicators[i]}
			err = rsi.Validate()
			groupMap = append(groupMap, &rsi)
		}
		if err != nil {
			return err
		}
		indicatorMap[settings.Indicators[i].GetGroup()] = groupMap
	}
	for _, v := range indicatorMap {
		settings.groupedIndicators = append(settings.groupedIndicators, v)
	}
	s.Settings = settings
	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	if s.Settings.MaxMissingPeriods <= 0 {
		log.Warnf(common.Strategy, "invalid maximum missing price periods, defaulting to %v", defaultMaxMissingPeriods)
		s.Settings.MaxMissingPeriods = defaultMaxMissingPeriods
	}
}

// massageMissingData will replace missing data with the previous candle's data
// this will ensure that RSI can be calculated correctly
// the decision to handle missing data occurs at the strategy level, not all strategies
// may wish to modify data
func (s *Strategy) massageMissingData(data []decimal.Decimal, t time.Time) ([]float64, error) {
	resp := make([]float64, len(data))
	var missingDataStreak int64
	for i := range data {
		if data[i].IsZero() {
			data[i] = data[i-1]
			missingDataStreak++
		} else {
			missingDataStreak = 0
		}
		if missingDataStreak >= s.Settings.MaxMissingPeriods {
			return nil, fmt.Errorf("missing data exceeds maximum allowable length of missing data of %v at %s and will distort results. %w",
				s.Settings.MaxMissingPeriods,
				t.Format(gctcommon.SimpleTimeFormat),
				strategybase.ErrTooMuchBadData)
		}
		resp[i] = data[i].InexactFloat64()
	}
	return resp, nil
}
