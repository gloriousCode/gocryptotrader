package futures

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/common/math"
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
	RequestKey         key.PairAsset
	Data               []ContractKline
	Analytics          []ContractKlineAnalytics
	AnalyticsPerformed bool
	AnyContangos       bool
	ContangoPercent    float64
}

type HistoricalContractKlineFrontEnd struct {
	Analytics          []ContractKlineAnalytics
	AnalyticsPerformed bool
	AnyContangos       bool
	ContangoPercent    float64
}

type ContractKlineAnalytics struct {
	Contract                       currency.Pair
	SpotCurrency                   currency.Pair
	Start                          time.Time
	End                            time.Time
	SpotOpenPrice                  float64
	ContractOpenPrice              float64
	StartPercentageDifference      float64
	SpotClosePrice                 float64
	ContractClosePrice             float64
	EndPercentageDifference        float64
	AchievedContangoOnSameExchange bool
	ContagoTimes                   []ContangoTime
}

type ContangoTime struct {
	Time          time.Time
	SpotPrice     float64
	ContractPrice float64
}

type ContractKline struct {
	Contract      *Contract
	Aliases       []string
	ContractKline *kline.Item
	SpotKline     *kline.Item
}

func (c *HistoricalContractKline) Analyse() {
	if len(c.Data) == 0 {
		return
	}
	for i := range c.Data {
		c.Data[i].ContractKline.ClearEmpty()
		c.Data[i].SpotKline.ClearEmpty()
		analytics := ContractKlineAnalytics{
			SpotCurrency: c.Data[i].SpotKline.Pair,
			Contract:     c.Data[i].ContractKline.Pair,
		}
		for j := range c.Data[i].ContractKline.Candles {
			if c.Data[i].ContractKline.Candles[j].Close == 0 {
				continue
			}
			if c.Data[i].SpotKline.Candles[j].Close == 0 {
				continue
			}
			if c.Data[i].ContractKline.Candles[j].Close < c.Data[i].SpotKline.Candles[j].Close {
				analytics.AchievedContangoOnSameExchange = true
				c.AnyContangos = true
				analytics.ContagoTimes = append(analytics.ContagoTimes, ContangoTime{
					Time:          c.Data[i].ContractKline.Candles[j].Time,
					SpotPrice:     c.Data[i].SpotKline.Candles[j].Close,
					ContractPrice: c.Data[i].ContractKline.Candles[j].Close,
				})
			}
		}

		analytics.Start = c.Data[i].Contract.StartDate
		analytics.End = c.Data[i].Contract.EndDate
		analytics.SpotOpenPrice = c.Data[i].SpotKline.Candles[0].Open
		analytics.ContractOpenPrice = c.Data[i].ContractKline.Candles[0].Open

		analytics.SpotClosePrice = c.Data[i].SpotKline.Candles[len(c.Data[i].SpotKline.Candles)-1].Close
		analytics.ContractClosePrice = c.Data[i].ContractKline.Candles[len(c.Data[i].ContractKline.Candles)-1].Close

		analytics.StartPercentageDifference = ((analytics.ContractOpenPrice - analytics.SpotOpenPrice) / analytics.ContractOpenPrice) * 100
		analytics.EndPercentageDifference = ((analytics.ContractClosePrice - analytics.SpotClosePrice) / analytics.ContractClosePrice) * 100

		c.Analytics = append(c.Analytics, analytics)
	}
	if len(c.Analytics) > 0 {
		c.AnalyticsPerformed = true
		var contangos float64
		for i := range c.Analytics {
			if c.Analytics[i].AchievedContangoOnSameExchange {
				contangos++
			}
		}
		c.ContangoPercent = math.CalculatePercentageDifference(float64(len(c.Analytics)), contangos)
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
