package futures

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
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
	SettlementCurrencies      currency.Currencies
	MarginCurrency            currency.Code
	ContractMultiplier        float64
	ContractValueInSettlement float64
	MaxLeverage               float64
	LatestRate                fundingrate.Rate
	FundingRateFloor          decimal.Decimal
	FundingRateCeiling        decimal.Decimal
	ContractValueDenomination ContractDenomination
}

type HistoricalContractKline struct {
	RequestKey             key.PairAsset
	Data                   []ContractKline
	Analytics              []ContractKlineAnalytics
	AnalyticsPerformed     bool
	AnyContangos           bool
	ContangoPercent        float64
	PositiveOutcomePercent float64
}

type HistoricalContractKlineFrontEnd struct {
	Analytics              []ContractKlineAnalytics
	AnalyticsPerformed     bool
	AnyContangos           bool
	ContangoPercent        float64
	PositiveOutcomePercent float64
}

type ContractKlineAnalytics struct {
	PremiumCurrency           currency.Pair
	BaseCurrency              currency.Pair
	Start                     time.Time
	End                       time.Time
	BaseOpenPrice             float64
	PremiumOpenPrice          float64
	StartPercentageDifference float64
	BaseClosePrice            float64
	PremiumClosePrice         float64
	EndPercentageDifference   float64
	EndResult                 float64
	AchievedContango          bool
	ContagoTimes              []ContangoTime
}

type ContangoTime struct {
	Time         time.Time
	BasePrice    float64
	PremiumPrice float64
}

type ContractKline struct {
	PremiumContract *Contract
	BaseContract    *Contract
	Aliases         []string
	PremiumKline    *kline.Item
	BaseKline       *kline.Item
}

func (c *HistoricalContractKline) Analyse() {
	if len(c.Data) == 0 {
		return
	}
	for i := range c.Data {
		c.Data[i].PremiumKline.ClearEmpty()
		c.Data[i].BaseKline.ClearEmpty()
		analytics := ContractKlineAnalytics{
			BaseCurrency:    c.Data[i].BaseKline.Pair,
			PremiumCurrency: c.Data[i].PremiumKline.Pair,
		}
		for j := range c.Data[i].PremiumKline.Candles {
			if c.Data[i].PremiumKline.Candles[j].Close == 0 {
				continue
			}
			if c.Data[i].BaseKline.Candles[j].Close == 0 {
				continue
			}
			if c.Data[i].PremiumKline.Candles[j].Close < c.Data[i].BaseKline.Candles[j].Close {
				analytics.AchievedContango = true
				c.AnyContangos = true
				analytics.ContagoTimes = append(analytics.ContagoTimes, ContangoTime{
					Time:         c.Data[i].PremiumKline.Candles[j].Time,
					BasePrice:    c.Data[i].BaseKline.Candles[j].Close,
					PremiumPrice: c.Data[i].PremiumKline.Candles[j].Close,
				})
			}
		}

		analytics.Start = c.Data[i].PremiumContract.StartDate
		analytics.End = c.Data[i].PremiumContract.EndDate
		analytics.BaseOpenPrice = c.Data[i].BaseKline.Candles[0].Open
		analytics.PremiumOpenPrice = c.Data[i].PremiumKline.Candles[0].Open

		analytics.BaseClosePrice = c.Data[i].BaseKline.Candles[len(c.Data[i].BaseKline.Candles)-1].Close
		analytics.PremiumClosePrice = c.Data[i].PremiumKline.Candles[len(c.Data[i].PremiumKline.Candles)-1].Close

		analytics.StartPercentageDifference = ((analytics.PremiumOpenPrice - analytics.BaseOpenPrice) / analytics.PremiumOpenPrice) * 100
		analytics.EndPercentageDifference = ((analytics.PremiumClosePrice - analytics.BaseClosePrice) / analytics.PremiumClosePrice) * 100

		analytics.EndResult = analytics.EndPercentageDifference - analytics.StartPercentageDifference

		c.Analytics = append(c.Analytics, analytics)
	}
	if len(c.Analytics) > 0 {
		c.AnalyticsPerformed = true
		var contangos, positiveEndResultPercent float64
		for i := range c.Analytics {
			if c.Analytics[i].AchievedContango {
				contangos++
				positiveEndResultPercent++
			} else if c.Analytics[i].EndResult > 0 {
				positiveEndResultPercent++
			}
		}
		c.ContangoPercent = (contangos / float64(len(c.Analytics))) * 100
		c.PositiveOutcomePercent = (positiveEndResultPercent / float64(len(c.Analytics))) * 100
	}
}

type ContractDenomination int64

type GetKlineContractRequest struct {
	ContractPair currency.Pair
	// used for okx
	UnderlyingPair currency.Pair
	Asset          asset.Item
	StartDate      time.Time
	EndDate        time.Time
	Interval       kline.Interval
	Contract       ContractType
}

var ErrUnderlyingPairRequired = errors.New("underlying pair required")

const (
	UnsetDenomination ContractDenomination = iota
	BaseDenomination
	QuoteDenomination
)

func (c *ContractDenomination) String() string {
	switch *c {
	case BaseDenomination:
		return "base"
	case QuoteDenomination:
		return "quote"
	default:
		return "unknown"
	}
}

// ContractSettlementType holds the various style of contracts offered by futures exchanges
type ContractSettlementType uint8

// ContractSettlementType definitions
const (
	UnsetSettlementType ContractSettlementType = iota
	Linear
	Inverse
	Quanto
	LinearOrInverse
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
	default:
		return "unknown"
	}
}

// ContractType holds the various style of contracts offered by futures exchanges
type ContractType uint8

// ContractType definitions
const (
	UnsetContractType ContractType = iota
	Perpetual
	LongDated
	Weekly
	Fortnightly
	ThreeWeekly
	Monthly
	BiMonthly
	Quarterly
	BiQuarterly
	SemiAnnually
	HalfYearly
	NineMonthly
	Yearly
	Unknown
	Daily
)

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
