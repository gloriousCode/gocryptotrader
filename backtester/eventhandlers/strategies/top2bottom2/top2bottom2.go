package top2bottom2

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
	Name                     = "top2bottom2"
	description              = `This is an example strategy to highlight more complex strategy design. All signals are processed and then ranked. Only the top 2 and bottom 2 proceed further`
	defaultMaxMissingPeriods = 14
)

var (
	errStrategyOnlySupportsSimultaneousProcessing = errors.New("strategy only supports simultaneous processing")
	errStrategyCurrencyRequirements               = errors.New("top2bottom2 strategy requires at least 4 currencies")
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	strategybase.Strategy
	Settings CustomSettings
}

// CustomSettings holds the settings for the strategy
type CustomSettings struct {
	MaxMissingPeriods int64           `json:"max-missing-periods"`
	MFIPeriod         decimal.Decimal `json:"mfi-period"`
	MFILow            decimal.Decimal `json:"mfi-low"`
	MFIHigh           decimal.Decimal `json:"mfi-high"`
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
// however,this complex strategy cannot function on an individual basis
func (s *Strategy) OnSignal(_ data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	return nil, errStrategyOnlySupportsSimultaneousProcessing
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

type mfiFundEvent struct {
	event signal.Event
	mfi   decimal.Decimal
	funds funding.IFundReader
}

// ByPrice used for sorting orders by order date
type byMFI []mfiFundEvent

func (b byMFI) Len() int           { return len(b) }
func (b byMFI) Less(i, j int) bool { return b[i].mfi.LessThan(b[j].mfi) }
func (b byMFI) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// sortOrdersByPrice the caller function to sort orders
func sortByMFI(o *[]mfiFundEvent, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(byMFI(*o)))
	} else {
		sort.Sort(byMFI(*o))
	}
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, f funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	if len(d) < 4 {
		return nil, errStrategyCurrencyRequirements
	}
	mfiFundEvents := make([]mfiFundEvent, 0, len(d))
	var resp []signal.Event
	for i := range d {
		if d == nil {
			return nil, common.ErrNilEvent
		}
		es, err := s.GetBaseData(d[i])
		if err != nil {
			return nil, err
		}
		latest, err := d[i].Latest()
		if err != nil {
			return nil, err
		}
		es.SetPrice(latest.GetClosePrice())
		offset := latest.GetOffset()

		if offset <= s.Settings.MFIPeriod.IntPart() {
			es.AppendReason("Not enough data for signal generation")
			es.SetDirection(order.DoNothing)
			resp = append(resp, &es)
			continue
		}

		history, err := d[i].History()
		if err != nil {
			return nil, err
		}
		var (
			closeData  = make([]decimal.Decimal, len(history))
			volumeData = make([]decimal.Decimal, len(history))
			highData   = make([]decimal.Decimal, len(history))
			lowData    = make([]decimal.Decimal, len(history))
		)
		for i := range history {
			closeData[i] = history[i].GetClosePrice()
			volumeData[i] = history[i].GetVolume()
			highData[i] = history[i].GetHighPrice()
			lowData[i] = history[i].GetLowPrice()
		}
		var massagedCloseData, massagedVolumeData, massagedHighData, massagedLowData []float64
		massagedCloseData, err = s.massageMissingData(closeData, es.GetTime())
		if err != nil {
			return nil, err
		}
		massagedVolumeData, err = s.massageMissingData(volumeData, es.GetTime())
		if err != nil {
			return nil, err
		}
		massagedHighData, err = s.massageMissingData(highData, es.GetTime())
		if err != nil {
			return nil, err
		}
		massagedLowData, err = s.massageMissingData(lowData, es.GetTime())
		if err != nil {
			return nil, err
		}
		mfi := indicators.MFI(massagedHighData, massagedLowData, massagedCloseData, massagedVolumeData, int(s.Settings.MFIPeriod.IntPart()))
		latestMFI := decimal.NewFromFloat(mfi[len(mfi)-1])
		hasDataAtTime, err := d[i].HasDataAtTime(latest.GetTime())
		if err != nil {
			return nil, err
		}
		if !hasDataAtTime {
			es.SetDirection(order.MissingData)
			es.AppendReasonf("missing data at %v, cannot perform any actions. MFI %v", latest.GetTime(), latestMFI)
			resp = append(resp, &es)
			continue
		}

		es.SetDirection(order.DoNothing)
		es.AppendReasonf("MFI at %v", latestMFI)

		funds, err := f.GetFundingForEvent(&es)
		if err != nil {
			return nil, err
		}
		mfiFundEvents = append(mfiFundEvents, mfiFundEvent{
			event: &es,
			mfi:   latestMFI,
			funds: funds.FundReader(),
		})
	}

	return s.selectTopAndBottomPerformers(mfiFundEvents, resp)
}

func (s *Strategy) selectTopAndBottomPerformers(mfiFundEvents []mfiFundEvent, resp []signal.Event) ([]signal.Event, error) {
	if len(mfiFundEvents) == 0 {
		return resp, nil
	}
	sortByMFI(&mfiFundEvents, true)
	buyingOrSelling := false
	for i := range mfiFundEvents {
		if i < 2 && mfiFundEvents[i].mfi.GreaterThanOrEqual(s.Settings.MFIHigh) {
			mfiFundEvents[i].event.SetDirection(order.Sell)
			buyingOrSelling = true
		} else if i >= 2 {
			break
		}
	}
	sortByMFI(&mfiFundEvents, false)
	for i := range mfiFundEvents {
		if i < 2 && mfiFundEvents[i].mfi.LessThanOrEqual(s.Settings.MFILow) {
			mfiFundEvents[i].event.SetDirection(order.Buy)
			buyingOrSelling = true
		} else if i >= 2 {
			break
		}
	}
	for i := range mfiFundEvents {
		if buyingOrSelling && mfiFundEvents[i].event.GetDirection() == order.DoNothing {
			mfiFundEvents[i].event.AppendReason("MFI was not in the top or bottom two ranks")
		}
		resp = append(resp, mfiFundEvents[i].event)
	}
	return resp, nil
}

// SetCustomSettings allows a user to modify the MFI limits in their config
func (s *Strategy) SetCustomSettings(message json.RawMessage) error {
	if len(message) == 0 {
		return strategybase.ErrEmptyCustomSettings
	}
	var customSettings CustomSettings
	err := json.Unmarshal(message, &customSettings)
	if err != nil {
		return err
	}
	if customSettings.MaxMissingPeriods < 0 {
		return fmt.Errorf("%w max missing periods less than zero", strategybase.ErrInvalidCustomSettings)
	}
	if customSettings.MFIHigh.LessThan(decimal.Zero) {
		return fmt.Errorf("%w mfi high less than zero", strategybase.ErrInvalidCustomSettings)
	}
	if customSettings.MFILow.LessThan(decimal.Zero) {
		return fmt.Errorf("%w mfi low less than zero", strategybase.ErrInvalidCustomSettings)
	}
	if customSettings.MFIPeriod.LessThan(decimal.Zero) {
		return fmt.Errorf("%w mfi period less than zero", strategybase.ErrInvalidCustomSettings)
	}
	if customSettings.MFIHigh.LessThan(customSettings.MFILow) {
		return fmt.Errorf("%w MFI high %v less than MFI low %v make sure you know what you're doing",
			strategybase.ErrInvalidCustomSettings, customSettings.MFIHigh, customSettings.MFILow)
	}

	s.Settings = customSettings
	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	s.Settings.MFIHigh = decimal.NewFromInt(70)
	s.Settings.MFILow = decimal.NewFromInt(30)
	s.Settings.MFIPeriod = decimal.NewFromInt(14)
	if s.Settings.MaxMissingPeriods <= 0 {
		// only override <= 0 in case strategy user desires no tolerance for missing data
		// potentiall rename this, because a tolerance of 1 means no missing data
		log.Warnf(common.Strategy, "invalid maximum missing price periods, defaulting to %v", defaultMaxMissingPeriods)
		s.Settings.MaxMissingPeriods = defaultMaxMissingPeriods
	}
}

// massageMissingData will replace missing data with the previous candle's data
// this will ensure that mfi can be calculated correctly
// the decision to handle missing data occurs at the strategy level, not all strategies
// may wish to modify data
func (s *Strategy) massageMissingData(data []decimal.Decimal, t time.Time) ([]float64, error) {
	resp := make([]float64, len(data))
	var missingDataStreak int64
	for i := range data {
		if data[i].IsZero() && i > int(s.Settings.MFIPeriod.IntPart()) {
			data[i] = data[i-1]
			missingDataStreak++
		} else {
			missingDataStreak = 0
		}
		if missingDataStreak >= s.Settings.MaxMissingPeriods {
			return nil, fmt.Errorf("missing data exceeds mfi period length of %v at %s and will distort results. %w",
				s.Settings.MaxMissingPeriods,
				t.Format(gctcommon.SimpleTimeFormat),
				strategybase.ErrTooMuchBadData)
		}
		resp[i] = data[i].InexactFloat64()
	}
	return resp, nil
}
