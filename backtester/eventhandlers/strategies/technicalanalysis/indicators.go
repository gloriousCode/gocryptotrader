package technicalanalysis

import (
	"errors"
	"fmt"

	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

const (
	RSIName                  = "RSI"
	MFIName                  = "MFI"
	OBVName                  = "OBV"
	BBandsName               = "BBANDS"
	MACDName                 = "MACD"
	defaultMaxMissingPeriods = 14
)

var (
	errUnsetIndicatorValue          = errors.New("unset indicator value")
	errInvalidIndicatorValue        = errors.New("invalid indicator value")
	errUnknownIndicatorAttributeSet = errors.New("unknown indicator attribute set, please check your config and read the readme")
)

// Indicator contains all relevant usable information to perform TA
type Indicator interface {
	GetName() string
	GetPeriod() int64
	GetFastPeriod() int64
	GetSlowPeriod() int64
	GetLow() float64
	GetHigh() float64
	GetUp() float64
	GetDown() float64
	GetGroup() string
	MustPass() bool
	Validate() error
}

// CustomSettings holds all defined indicators
// seperated by group (if defined)
type CustomSettings struct {
	MaxMissingPeriods int64         `json:"max-missing-periods"`
	Indicators        []TABase      `json:"indicators"`
	groupedIndicators [][]Indicator `json:"-"`
}

// TABase contains implementations to satisfy the interface
// when something is unsupported
type TABase struct {
	Name             string  `json:"name"`
	Period           int64   `json:"period,omitempty"`
	FastPeriod       int64   `json:"fast-period,omitempty"`
	SlowPeriod       int64   `json:"slow-period,omitempty"`
	Low              float64 `json:"low,omitempty"`
	High             float64 `json:"high,omitempty"`
	Down             float64 `json:"down,omitempty"`
	Up               float64 `json:"up,omitempty"`
	Group            string  `json:"group,omitempty"`
	PassRequired     bool    `json:"pass-required,omitempty"`
	UseDefaultValues bool    `json:"use-defaults"`
}

func (t *TABase) GetName() string {
	return ""
}

// GetPeriod returns the indicator period
func (t *TABase) GetPeriod() int64 {
	return t.Period
}

// GetLow returns the low indicator setting
func (t *TABase) GetLow() float64 {
	return t.Low
}

// GetHigh returns the high indicator setting
func (t *TABase) GetHigh() float64 {
	return t.High
}

// GetFastPeriod returns the fast period indicator setting
func (t *TABase) GetFastPeriod() int64 {
	return t.FastPeriod
}

// GetSlowPeriod returns the slow period indicator setting
func (t *TABase) GetSlowPeriod() int64 {
	return t.SlowPeriod
}

// GetUp returns the up value for BBands
func (t *TABase) GetUp() float64 {
	return t.Up
}

// GetDown returns the down value for BBands
func (t *TABase) GetDown() float64 {
	return t.Down
}

// GetGroup returns the group the indicator belongs to
func (t *TABase) GetGroup() string {
	return t.Group
}

// MustPass returns whether the indicator must pass in order to
// make a decision. Used when multiple indicators or groups are defined
func (t *TABase) MustPass() bool {
	return t.PassRequired
}

// Validate ensures the indicator's settings are all correct and usable
func (t *TABase) Validate() error {
	return gctcommon.ErrNotYetImplemented
}

// RSI stands for Relative Strength Indicator
type RSI struct {
	TABase `json:"rsi"`
}

// BBands are Bollinger bands, not boy bands
type BBands struct {
	TABase `json:"bbands"`
}

// OBV stands for on-balance-volume
type OBV struct {
	TABase `json:"obv"`
}

// MACD stands for Smoothed Moving Average
type MACD struct {
	TABase `json:"macd"`
}

// EMA stands for Exponential Moving Average
type EMA struct {
	TABase `json:"ema"`
}

// GetName returns the indicator's name
func (i *RSI) GetName() string {
	return RSIName
}

// Validate ensures the indicator's settings are all correct and usable
func (i *RSI) Validate() error {
	if i.High <= 0 {
		return fmt.Errorf("%w %s High: %v", errUnsetIndicatorValue, i.GetName(), i.High)
	}
	if i.Low <= 0 {
		return fmt.Errorf("%w %s Low: %v", errUnsetIndicatorValue, i.GetName(), i.Low)
	}
	if i.Period <= 0 {
		return fmt.Errorf("%w %s Period: %v", errUnsetIndicatorValue, i.GetName(), i.Period)
	}
	if i.Low > i.High {
		return fmt.Errorf("%w %s Low %v > High %v: %v", errInvalidIndicatorValue, i.GetName(), i.Low, i.High)
	}
	if i.SlowPeriod > 0 || i.FastPeriod > 0 || i.Up > 0 || i.Down > 0 {
		return errUnknownIndicatorAttributeSet
	}
	return nil
}

// GetName returns the indicator's name
func (i *MACD) GetName() string {
	return MACDName
}

// Validate ensures the indicator's settings are all correct and usable
func (i *MACD) Validate() error {
	if i.Period <= 0 {
		return fmt.Errorf("%w %s Period: %v", errUnsetIndicatorValue, i.GetName(), i.Period)
	}
	if i.FastPeriod <= 0 {
		return fmt.Errorf("%w %s FastPeriod: %v", errUnsetIndicatorValue, i.GetName(), i.FastPeriod)
	}
	if i.SlowPeriod <= 0 {
		return fmt.Errorf("%w %s SlowPeriod: %v", errUnsetIndicatorValue, i.GetName(), i.SlowPeriod)
	}
	if i.SlowPeriod > i.Period {
		return fmt.Errorf("%w %s slow period %v > period %v", errInvalidIndicatorValue, i.GetName(), i.SlowPeriod, i.Period)
	}
	if i.Period > i.FastPeriod {
		return fmt.Errorf("%w %s period %v > fast period %v", errInvalidIndicatorValue, i.GetName(), i.Period, i.FastPeriod)
	}
	if i.Up > 0 || i.Down > 0 || i.High > 0 || i.Low > 0 {
		return errUnknownIndicatorAttributeSet
	}
	return nil
}

// GetName returns the indicator's name
func (i *BBands) GetName() string {
	return BBandsName
}

// Validate ensures the indicator's settings are all correct and usable
func (i *BBands) Validate() error {
	if i.Period <= 0 {
		return fmt.Errorf("%w %s Period: %v", errUnsetIndicatorValue, i.GetName(), i.Period)
	}
	if i.Up <= 0 {
		return fmt.Errorf("%w %s High: %v", errUnsetIndicatorValue, i.GetName(), i.Period)
	}
	if i.Down <= 0 {
		return fmt.Errorf("%w %s Low: %v", errUnsetIndicatorValue, i.GetName(), i.Period)
	}
	if i.Up <= i.Down {
		return fmt.Errorf("%w %s High %v <= Low %v", errInvalidIndicatorValue, i.GetName(), i.Up, i.Down)
	}
	if i.SlowPeriod > 0 || i.FastPeriod > 0 || i.High > 0 || i.Low > 0 {
		return errUnknownIndicatorAttributeSet
	}

	return nil
}

// GetName returns the indicator's name
func (i *OBV) GetName() string {
	return OBVName
}

// Validate ensures the indicator's settings are all correct and usable
func (i *OBV) Validate() error {
	return nil
}
