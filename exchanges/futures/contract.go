package futures

import (
	"errors"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Butts struct {
	PremiumCandle kline.Candle
	BaseCandle    kline.Candle
// var error definitions
var (
	ErrInvalidContractSettlementType = errors.New("invalid contract settlement type")
)

// Contract holds details on futures contracts
type Contract struct {
	Exchange       string
	Name           currency.Pair
	Underlying     currency.Pair
	Asset          asset.Item
	StartDate      time.Time
	EndDate        time.Time
	IsActive       bool
	Status         string
	Type           ContractType
	SettlementType ContractSettlementType
	// Optional values if the exchange offers them
	SettlementCurrencies currency.Currencies
	MarginCurrency       currency.Code
	Multiplier           float64
	MaxLeverage          float64
	LatestRate           fundingrate.Rate
	FundingRateFloor     decimal.Decimal
	FundingRateCeiling   decimal.Decimal
}

type Butteroo map[time.Time]*Butts

	// ContractSettlementType definitions
	const (
	UnsetSettlementType ContractSettlementType = iota
	Linear
	Inverse
	Quanto
	LinearOrInverse
	Hybrid
	)

	// String returns the string representation of a contract settlement type
	func (d ContractSettlementType) String() string {
	switch d {
	case UnsetSettlementType:
	return "unset"
	case Linear:
	return "linear"
	case Inverse:
	return "inverse"
	case Quanto:
	return "quanto"
	case LinearOrInverse:
	return "linearOrInverse"
	case Hybrid:
	return "hybrid"
	default:
	return "unknown"
	}
	}

func (c *HistoricalContractKline) Analyse() {
	if len(c.Data) == 0 {
		return
	}
	for i := range c.Data {
		c.Data[i].PremiumKline.ClearEmpty()
		c.Data[i].BaseKline.ClearEmpty()
	}
	for i := range c.Data {
		butts := make(Butteroo)
		for j := range c.Data[i].PremiumKline.Candles {
			if hello, ok := butts[c.Data[i].PremiumKline.Candles[j].Time]; ok {
				hello.PremiumCandle = c.Data[i].PremiumKline.Candles[j]
			} else {
				butts[c.Data[i].PremiumKline.Candles[j].Time] = &Butts{
					PremiumCandle: c.Data[i].PremiumKline.Candles[j],
				}
			}
		}
		for k := range c.Data[i].BaseKline.Candles {
			if hello, ok := butts[c.Data[i].BaseKline.Candles[k].Time]; ok {
				hello.BaseCandle = c.Data[i].BaseKline.Candles[k]
			} else {
				butts[c.Data[i].BaseKline.Candles[k].Time] = &Butts{
					BaseCandle: c.Data[i].BaseKline.Candles[k],
				}
			}
		}
		for k, v := range butts {
			if v.PremiumCandle.Close == 0 || v.BaseCandle.Close == 0 {
				delete(butts, k)
			}
		}
		if len(butts) == 0 {
			return
		}


		analytics := ContractKlineAnalytics{
			BaseCurrency:    c.Data[i].BaseKline.Pair,
			PremiumCurrency: c.Data[i].PremiumKline.Pair,
		}
		firstDone := false
		x := 0
		last := len(butts) - 1
		for k, v := range butts {
			if !firstDone {
				analytics.Start = k
				analytics.BaseOpenPrice = v.BaseCandle.Open
				analytics.PremiumOpenPrice = v.PremiumCandle.Open
				analytics.StartPercentageDifference = ((analytics.PremiumOpenPrice - analytics.BaseOpenPrice) / analytics.PremiumOpenPrice) * 100
				firstDone = true
			}
			if v.PremiumCandle.Close < v.BaseCandle.Close {
				analytics.AchievedContango = true
				c.AnyContangos = true
				ct := ContangoTime{
					Time:         k,
					BasePrice:    v.BaseCandle.Close,
					PremiumPrice: v.PremiumCandle.Close,
				}
				if analytics.PremiumOpenPrice > 0 {
					ct.Gain = ((analytics.PremiumOpenPrice - v.PremiumCandle.Close) / analytics.PremiumOpenPrice) * 100
				}
				analytics.ContagoTimes = append(analytics.ContagoTimes, ct)
			}
			x++
			if x == last {
				analytics.End = k
				analytics.BaseClosePrice = v.BaseCandle.Close
				analytics.PremiumClosePrice = v.PremiumCandle.Close
				analytics.EndPercentageDifference = ((analytics.PremiumClosePrice - analytics.BaseClosePrice) / analytics.PremiumClosePrice) * 100
				analytics.EndResult = analytics.EndPercentageDifference - analytics.StartPercentageDifference
			}
		}
		c.Analytics = append(c.Analytics, analytics)
	}


	if len(c.Analytics) > 0 {
		c.AnalyticsPerformed = true
		var contangos, positiveContangos, positiveEndResultPercent float64
		for i := range c.Analytics {
			switch {
			case c.Analytics[i].AchievedContango && c.Analytics[i].EndResult > 0:
				positiveContangos++
				positiveEndResultPercent++
				contangos++
			case c.Analytics[i].AchievedContango:
				contangos++
			case c.Analytics[i].EndResult > 0:
				positiveEndResultPercent++
			}
		}
		c.ContangoPercent = (contangos / float64(len(c.Analytics))) * 100
		c.PositiveContangoPercent = (positiveContangos / float64(len(c.Analytics))) * 100
		c.PositiveOutcomePercent = (positiveEndResultPercent / float64(len(c.Analytics))) * 100
	}
}

// StringToContractSettlementType for converting case insensitive contract settlement type
func StringToContractSettlementType(cstype string) (ContractSettlementType, error) {
	cstype = strings.ToLower(cstype)
	switch cstype {
	case UnsetSettlementType.String(), "":
		return UnsetSettlementType, nil
	case Linear.String():
		return Linear, nil
	case Inverse.String():
		return Inverse, nil
	case Quanto.String():
		return Quanto, nil
	case "linearorinverse":
		return LinearOrInverse, nil
	case Hybrid.String():
		return Hybrid, nil
	default:
		return UnsetSettlementType, ErrInvalidContractSettlementType
	}
}

// ContractType holds the various style of contracts offered by futures exchanges
type ContractType uint8

func (c ContractType) IsLongDated() bool {
	return c == LongDated ||
		c == Quarterly ||
		c == SemiAnnually ||
		c == HalfYearly ||
		c == NineMonthly ||
		c == Yearly ||
		c == Weekly ||
		c == Fortnightly ||
		c == ThreeWeekly ||
		c == Monthly ||
		c == BiMonthly ||
		c == BiQuarterly
}

// String returns the string representation of the contract type
func (c ContractType) String() string {
	switch c {
	case Daily:
		return "day"
	case Perpetual:
		return "perpetual"
	case LongDated:
		return "long_dated"
	case Weekly:
		return "weekly"
	case Fortnightly:
		return "fortnightly"
	case ThreeWeekly:
		return "three-weekly"
	case Monthly:
		return "monthly"
	case BiMonthly:
		return "bi-monthly"
	case Quarterly:
		return "quarterly"
	case BiQuarterly:
		return "bi-quarterly"
	case SemiAnnually:
		return "semi-annually"
	case HalfYearly:
		return "half-yearly"
	case NineMonthly:
		return "nine-monthly"
	case Yearly:
		return "yearly"
	case Unknown:
		return "unknown"
	default:
		return "unset/undefined contract type"
	}
}

func (c ContractType) Duration() time.Duration {
	switch c {
	case Daily:
		return time.Hour * 24
	case Weekly:
		return time.Hour * 24 * 7
	case Fortnightly:
		return time.Hour * 24 * 14
	case ThreeWeekly:
		return time.Hour * 24 * 21
	case Monthly:
		return time.Hour * 24 * 30
	case BiMonthly:
		return time.Hour * 24 * 60
	case Quarterly:
		return time.Hour * 24 * 90
	case BiQuarterly:
		return time.Hour * 24 * 180
	case SemiAnnually:
		return time.Hour * 24 * 180
	case HalfYearly:
		return time.Hour * 24 * 180
	case NineMonthly:
		return time.Hour * 24 * 270
	case Yearly:
		return time.Hour * 24 * 365
	default:
		return 0
	}
}
