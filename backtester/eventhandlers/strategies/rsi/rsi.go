package rsi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name        = "rsi"
	description = `The relative strength index is a technical indicator used in the analysis of financial markets. It is intended to chart the current and historical strength or weakness of a stock or market based on the closing prices of a recent trading period`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	settings CustomSettings
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
	es.SetPrice(d.Latest().GetClosePrice())

	if offset := d.Offset(); offset <= int(s.settings.RSIPeriod) {
		es.AppendReason("Not enough data for signal generation")
		es.SetDirection(order.DoNothing)
		return &es, nil
	}

	dataRange := d.StreamClose()
	var massagedData []float64
	massagedData, err = s.massageMissingData(dataRange, es.GetTime())
	if err != nil {
		return nil, err
	}
	rsi := indicators.RSI(massagedData, int(s.settings.RSIPeriod))
	latestRSIValue := int64(rsi[len(rsi)-1])
	if !d.HasDataAtTime(d.Latest().GetTime()) {
		es.SetDirection(order.MissingData)
		es.AppendReasonf("missing data at %v, cannot perform any actions. RSI %v", d.Latest().GetTime(), latestRSIValue)
		return &es, nil
	}

	switch {
	case latestRSIValue >= s.settings.RSIHigh:
		es.SetDirection(order.Sell)
	case latestRSIValue <= s.settings.RSILow:
		es.SetDirection(order.Buy)
	default:
		es.SetDirection(order.DoNothing)
	}
	es.AppendReasonf("RSI at %v", latestRSIValue)

	return &es, nil
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
		sigEvent, err := s.OnSignal(d[i], nil, nil)
		if err != nil {
			errs = append(errs, fmt.Errorf("%v %v %v %w", d[i].Latest().GetExchange(), d[i].Latest().GetAssetType(), d[i].Latest().Pair(), err))
		} else {
			resp = append(resp, sigEvent)
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return resp, nil
}

type CustomSettings struct {
	rsiSettings
	AdditionalCandles []rsiSettings `json:"additional-candles"`
}

type rsiSettings struct {
	CandleInterval kline.Interval `json:"candle-interval"`
	RSILow         int64          `json:"rsi-low"`
	RSIHigh        int64          `json:"rsi-high"`
	RSIPeriod      int64          `json:"rsi-period"`
	Data           interface{}    `json:"-"`
}

// SetCustomSettings allows a user to modify the RSI limits in their config
func (s *Strategy) SetCustomSettings(customSettings json.RawMessage) error {
	s.SetDefaults()
	if len(customSettings) == 0 {
		return nil
	}
	var customData CustomSettings

	decoder := json.NewDecoder(bytes.NewReader(customSettings))
	// can't trust custom settings with extra fields
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&customData)
	if err != nil {
		// json decoder does not have an exported recognisable error
		// wrap it in something we can verify
		return fmt.Errorf("%w %s", base.ErrInvalidCustomSettings, err)
	}
	err = customData.validate()
	if err != nil {
		return err
	}
	if customData.RSILow > 0 {
		s.settings.RSILow = customData.RSILow
	}
	if customData.RSIHigh > 0 {
		s.settings.RSIHigh = customData.RSIHigh
	}
	if customData.RSIPeriod > 0 {
		s.settings.RSIPeriod = customData.RSIPeriod
	}

	for i := range customData.AdditionalCandles {
		err = customData.AdditionalCandles[i].validate()
		if err != nil {
			return err
		}
		s.settings.AdditionalCandles = append(s.settings.AdditionalCandles, customData.AdditionalCandles[i])
	}
	return nil
}

func (r *rsiSettings) validate() error {
	if r.RSIHigh == 0 && r.RSILow == 0 && r.RSIPeriod == 0 {
		return base.ErrInvalidCustomSettings
	}
	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	s.settings.RSIHigh = 70
	s.settings.RSILow = 30
	s.settings.RSIPeriod = 14
}

// massageMissingData will replace missing data with the previous candle's data
// this will ensure that RSI can be calculated correctly
// the decision to handle missing data occurs at the strategy level, not all strategies
// may wish to modify data
func (s *Strategy) massageMissingData(data []decimal.Decimal, t time.Time) ([]float64, error) {
	resp := make([]float64, len(data))
	var missingDataStreak int64
	for i := range data {
		if data[i].IsZero() && i > int(s.settings.RSIPeriod) {
			data[i] = data[i-1]
			missingDataStreak++
		} else {
			missingDataStreak = 0
		}
		if missingDataStreak >= s.settings.RSIPeriod {
			return nil, fmt.Errorf("missing data exceeds RSI period length of %v at %s and will distort results. %w",
				s.settings.RSIPeriod,
				t.Format(gctcommon.SimpleTimeFormat),
				base.ErrTooMuchBadData)
		}
		resp[i] = data[i].InexactFloat64()
	}
	return resp, nil
}
