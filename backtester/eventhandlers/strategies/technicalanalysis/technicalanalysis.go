package technicalanalysis

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	var massagedClosePrices []float64
	massagedClosePrices, err = s.massageMissingData(dataRange, es.GetTime())
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
			groupedIndicator := s.Settings.groupedIndicators[i][j]
			if offset := latest.GetOffset(); offset <= groupedIndicator.GetPeriod() {
				es.AppendReason("Not enough data for signal generation")
				es.SetDirection(order.DoNothing)
				if groupedIndicator.MustPass() {
					es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
					// break groupAnalysis
					return nil, err
				}
				continue
			}

			switch groupedIndicator.GetName() {
			case RSIName:
				err = s.processRSI(massagedClosePrices, groupedIndicator, &es)
				if err != nil {
					break groupAnalysis
				}
			case MACDName:
				err = s.processMACD(massagedClosePrices, groupedIndicator, &es)
				if err != nil {
					break groupAnalysis
				}
			case BBandsName:
				err = s.processBBands(massagedClosePrices, groupedIndicator, latest, &es)
				if err != nil {
					break groupAnalysis
				}
			case OBVName:
				var volumeRange []decimal.Decimal
				volumeRange, err = d.StreamVol()
				if err != nil {
					return nil, err
				}
				var massagedVolume []float64
				massagedVolume, err = s.massageMissingData(volumeRange, es.GetTime())
				if err != nil {
					return nil, err
				}

				err = s.processOBV(massagedClosePrices, massagedVolume, groupedIndicator, &es)
				if err != nil {
					break groupAnalysis
				}
			case MFIName:
				var highRange []decimal.Decimal
				highRange, err = d.StreamHigh()
				if err != nil {
					return nil, err
				}
				var lowRange []decimal.Decimal
				lowRange, err = d.StreamLow()
				if err != nil {
					return nil, err
				}
				var volRange []decimal.Decimal
				volRange, err = d.StreamVol()
				if err != nil {
					return nil, err
				}
				var massagedVolume []float64
				massagedVolume, err = s.massageMissingData(volRange, es.GetTime())
				if err != nil {
					return nil, err
				}
				var massagedHigh []float64
				massagedHigh, err = s.massageMissingData(highRange, es.GetTime())
				if err != nil {
					return nil, err
				}
				var massagedLow []float64
				massagedLow, err = s.massageMissingData(lowRange, es.GetTime())
				if err != nil {
					return nil, err
				}

				err = s.processMFI(massagedHigh, massagedLow, massagedClosePrices, massagedVolume, groupedIndicator, &es)
				if err != nil {
					break groupAnalysis
				}
			case ATRName:
				var highRange []decimal.Decimal
				highRange, err = d.StreamHigh()
				if err != nil {
					return nil, err
				}
				var lowRange []decimal.Decimal
				lowRange, err = d.StreamLow()
				if err != nil {
					return nil, err
				}
				var massagedHigh []float64
				massagedHigh, err = s.massageMissingData(highRange, es.GetTime())
				if err != nil {
					return nil, err
				}
				var massagedLow []float64
				massagedLow, err = s.massageMissingData(lowRange, es.GetTime())
				if err != nil {
					return nil, err
				}

				err = s.processATR(massagedHigh, massagedLow, massagedClosePrices, groupedIndicator, &es)
				if err != nil {
					break groupAnalysis
				}
			}
		}
	}

	return &es, nil
}

func (s *Strategy) processATR(high, low, closePrices []float64, groupedIndicator Indicator, es signal.Event) error {
	atr := indicators.ATR(high, low, closePrices, int(groupedIndicator.GetPeriod()))
	
}

func (s *Strategy) processMFI(high, low, closePrices, volume []float64, groupedIndicator Indicator, es signal.Event) error {
	mfi := indicators.MFI(high, low, closePrices, volume, int(groupedIndicator.GetPeriod()))
	latestMFI := mfi[len(mfi)-1]
	var err error
	switch {
	case latestMFI >= groupedIndicator.GetHigh():
		err = setDirection(es, order.Sell)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	case latestMFI <= groupedIndicator.GetLow():
		err = setDirection(es, order.Buy)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			return err
		}
	default:
		_ = setDirection(es, order.DoNothing)
		if groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			return err
		}
	}
	return nil
}

func (s *Strategy) processOBV(massagedClosePrices, massagedVolume []float64, groupedIndicator Indicator, es signal.Event) error {
	obv := indicators.OBV(massagedClosePrices, massagedVolume)
	if len(obv) == 0 {
		return nil
	}
	rsiObv := indicators.RSI(obv, int(groupedIndicator.GetPeriod()))
	latestRSIValue := rsiObv[len(rsiObv)-1]
	var err error
	switch {
	case latestRSIValue >= groupedIndicator.GetHigh():
		err = setDirection(es, order.Sell)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	case latestRSIValue <= groupedIndicator.GetLow():
		err = setDirection(es, order.Buy)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			return err
		}
	default:
		_ = setDirection(es, order.DoNothing)
		if groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			return err
		}
	}
	es.AppendReasonf("OBV RSI at %v", latestRSIValue)
	return nil
}

var errMustPass = errors.New("must pass")

func (s *Strategy) processBBands(massagedClosePrices []float64, groupedIndicator Indicator, latest data.Event, es signal.Event) error {
	upper, _, lower := indicators.BBANDS(massagedClosePrices, int(groupedIndicator.GetPeriod()), groupedIndicator.GetUp(), groupedIndicator.GetDown(), indicators.Sma)
	closePrice := latest.GetClosePrice().InexactFloat64()
	latestUpper := upper[len(upper)-1]
	latestDowner := lower[len(lower)-1]
	var err error
	switch {
	case closePrice >= latestUpper:
		err = setDirection(es, order.Sell)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return errMustPass
		}
	case closePrice <= latestDowner:
		err = setDirection(es, order.Buy)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return errMustPass
		}
	default:
		_ = setDirection(es, order.DoNothing)
		if groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return errMustPass
		}
	}
	return nil
}

func (s *Strategy) processMACD(massagedClosePrices []float64, groupedIndicator Indicator, es signal.Event) error {
	macd, signal, _ := indicators.MACD(massagedClosePrices, int(groupedIndicator.GetFastPeriod()), int(groupedIndicator.GetSlowPeriod()), int(groupedIndicator.GetPeriod()))
	var err error
	if len(macd) == 0 {
		es.AppendReason("Not enough data for signal generation")
		es.SetDirection(order.DoNothing)
		if groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			return err
		}
		return nil
	}
	latestMacd := macd[len(macd)-1]
	latestSignal := macd[len(signal)-1]
	previousMacd := macd[len(macd)-2]
	previousSignal := macd[len(signal)-2]
	switch {
	case latestMacd > latestSignal && previousMacd <= previousSignal:
		err = setDirection(es, order.Sell)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	case latestMacd < latestSignal && previousMacd >= previousSignal:
		err = setDirection(es, order.Buy)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	case latestMacd > 0 && previousMacd < 0:
		err = setDirection(es, order.Buy)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	case latestMacd < 0 && previousMacd > 0:
		err = setDirection(es, order.Sell)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	default:
		_ = setDirection(es, order.DoNothing)
		if groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return errMustPass
		}
	}
	return nil
}

func (s *Strategy) processRSI(massagedClosePrices []float64, groupedIndicator Indicator, es signal.Event) error {
	rsi := indicators.RSI(massagedClosePrices, int(groupedIndicator.GetPeriod()))
	latestRSIValue := rsi[len(rsi)-1]
	var err error
	switch {
	case latestRSIValue >= groupedIndicator.GetHigh():
		err = setDirection(es, order.Sell)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	case latestRSIValue <= groupedIndicator.GetLow():
		err = setDirection(es, order.Buy)
		if err != nil && groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	default:
		_ = setDirection(es, order.DoNothing)
		if groupedIndicator.MustPass() {
			es.AppendReasonf("indicator %v of group %v failed check", groupedIndicator.GetName(), groupedIndicator.GetGroup())
			// break groupAnalysis
			return err
		}
	}
	es.AppendReasonf("RSI at %v", latestRSIValue)
	return nil
}

var errCannotSetDirection = errors.New("cannot set direction")

func setDirection(es signal.Event, direction order.Side) error {
	switch es.GetDirection() {
	case order.Buy:
		if direction == order.Sell {
			es.AppendReason("conflicting indicators results, cannot switch from buy to sell")
			es.SetDirection(order.DoNothing)
			return errCannotSetDirection
		}
	case order.Sell:
		if direction == order.Buy {
			es.AppendReason("conflicting indicators results, cannot switch from sell to buy")
			es.SetDirection(order.DoNothing)
			return errCannotSetDirection
		}
	}
	es.SetDirection(direction)
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

var errUnknownIndicator = errors.New("unknown indicator")

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
		var indicator Indicator
		switch strings.ToUpper(settings.Indicators[i].Name) {
		case RSIName:
			indicator = &RSI{settings.Indicators[i]}
		case BBandsName:
			indicator = &BBands{settings.Indicators[i]}
		case MACDName:
			indicator = &MACD{settings.Indicators[i]}
		case ATRName:
			indicator = &ATR{settings.Indicators[i]}
		case MFIName:
			indicator = &MFI{settings.Indicators[i]}
		case OBVName:
			indicator = &OBV{settings.Indicators[i]}
		default:
			return fmt.Errorf("%w %v in group '%v'",
				errUnknownIndicator, settings.Indicators[i].Name, settings.Indicators[i].Group)
		}
		if settings.Indicators[i].UseDefaultValues {
			indicator.SetDefaults()
		}
		err = indicator.Validate()
		if err != nil {
			return err
		}
		groupMap = append(groupMap, indicator)
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
