package technicalanalysis

import (
	"errors"
	"fmt"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

const (
	rsiName  = "RSI"
	macdName = "SMA"
	mfiName  = "MFI"
	obvName  = "OBV"
)

var (
	errUnsetIndicatorValue   = errors.New("unset indicator value")
	errInvalidIndicatorValue = errors.New("invalid indicator value")
)

// Indicator contains all relevant usable information to perform TA
type Indicator interface {
	GetName() string
	GetPeriod() int64
	GetFastPeriod() int64
	GetSlowPeriod() int64
	GetLow() float64
	GetHigh() float64
	GetGroup() string
	MustPass() bool
	Validate() error
}

// CustomSettings holds all defined indicators
// seperated by group (if defined)
type CustomSettings struct {
	Indicators        []Indicator `json:"indicators"`
	GroupedIndicators [][]Indicator
}

// TABase contains implementations to satisfy the interface
// when something is unsupported
type TABase struct {
	Name         string  `json:"name"`
	Period       int64   `json:"period"`
	FastPeriod   int64   `json:"fast-period"`
	SlowPeriod   int64   `json:"slow-period"`
	Low          float64 `json:"low"`
	High         float64 `json:"high"`
	Group        string  `json:"group"`
	PassRequired bool    `json:"pass-required"`
}

// GetName returns the indicator Name
func (t *TABase) GetName() string {
	return t.Name
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
	*TABase
}

// Validate ensures the indicator's settings are all correct and usable
func (r *RSI) Validate() error {
	if r.High <= 0 {
		return fmt.Errorf("%w RSI High: %v", errUnsetIndicatorValue, r.High)
	}
	if r.Low <= 0 {
		return fmt.Errorf("%w RSI Low: %v", errUnsetIndicatorValue, r.Low)
	}
	if r.Period <= 0 {
		return fmt.Errorf("%w RSI Period: %v", errUnsetIndicatorValue, r.Period)
	}
	if r.Low > r.High {
		return fmt.Errorf("%w RSI Low %v > High %v: %v", errInvalidIndicatorValue, r.Low, r.High)
	}
	if r.GetSlowPeriod() > 0 || r.GetFastPeriod() > 0 {
		return gctcommon.ErrFunctionNotSupported
	}
	return nil
}

// SMA stands for Smoothed Moving Average
type SMA struct {
	*TABase
}

// Validate ensures the indicator's settings are all correct and usable
func (r *SMA) Validate() error {
	if r.Period <= 0 {
		return fmt.Errorf("%w RSI High: %v", errUnsetIndicatorValue, r.Period)
	}
	if r.FastPeriod <= 0 {
		return fmt.Errorf("%w RSI Low: %v", errUnsetIndicatorValue, r.FastPeriod)
	}
	if r.SlowPeriod <= 0 {
		return fmt.Errorf("%w RSI Low: %v", errUnsetIndicatorValue, r.SlowPeriod)
	}
	if r.SlowPeriod > r.Period {
		return fmt.Errorf("%w SMA slow period %v > period %v: %v", errInvalidIndicatorValue, r.SlowPeriod, r.Period)
	}
	if r.Period > r.FastPeriod {
		return fmt.Errorf("%w SMA period %v > fast period %v: %v", errInvalidIndicatorValue, r.Period, r.FastPeriod)
	}
	if r.Low > 0 || r.High > 0 {
		return gctcommon.ErrFunctionNotSupported
	}
	return nil
}
