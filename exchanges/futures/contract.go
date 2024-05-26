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
	RequestKey         key.PairAsset
	Data               []ContractKline
	SpotData           *kline.Item
	Analytics          []ContractKlineAnalytics
	AnalyticsPerformed bool
}

type ContractKlineAnalytics struct {
	Start                          time.Time
	End                            time.Time
	StartSpotPrice                 float64
	StartContractPrice             float64
	StartPercentageDifference      float64
	EndSpotPrice                   float64
	EndContractPrice               float64
	EndPercentageDifference        float64
	EndedInContangoOnSameExchange  bool
	AchievedContangoOnSameExchange bool
	AchievedContangoTime           time.Time
}

type ContractKline struct {
	Contract *Contract
	Aliases  []string
	Kline    *kline.Item
}

func (c *HistoricalContractKline) Analyse() {
	if c.SpotData == nil || len(c.SpotData.Candles) == 0 || len(c.Data) == 0 {
		return
	}

	for i := range c.Data {
		analytics := ContractKlineAnalytics{}
		var spotStartCandle, spotEndCandle kline.Candle
		for j := range c.SpotData.Candles {
			if c.SpotData.Candles[j].Time.Equal(c.Data[i].Contract.StartDate) {
				spotStartCandle = c.SpotData.Candles[j]
			}
			if c.SpotData.Candles[j].Time.Equal(c.Data[i].Contract.EndDate) {
				spotEndCandle = c.SpotData.Candles[j]
			}
			if !spotStartCandle.Time.IsZero() && !spotEndCandle.Time.IsZero() {
				break
			}
		}

		for j := range c.Data[i].Kline.Candles {
			if c.Data[i].Kline.Candles[j].Close <= spotEndCandle.Close {
				analytics.AchievedContangoOnSameExchange = true
				analytics.AchievedContangoTime = c.Data[i].Kline.Candles[j].Time
				break
			}
		}

		analytics.Start = c.Data[i].Contract.StartDate
		analytics.End = c.Data[i].Contract.EndDate
		analytics.StartSpotPrice = spotStartCandle.Open
		analytics.StartContractPrice = c.Data[i].Kline.Candles[0].Open
		analytics.StartPercentageDifference = ((spotStartCandle.Open - c.Data[i].Kline.Candles[0].Open) / spotStartCandle.Open) * 100
		analytics.EndSpotPrice = spotEndCandle.Close
		analytics.EndContractPrice = c.Data[i].Kline.Candles[len(c.Data[i].Kline.Candles)-1].Close
		analytics.EndPercentageDifference = ((spotEndCandle.Close - c.Data[i].Kline.Candles[len(c.Data[i].Kline.Candles)-1].Close) / spotEndCandle.Close) * 100
		analytics.EndedInContangoOnSameExchange = spotEndCandle.Close >= c.Data[i].Kline.Candles[len(c.Data[i].Kline.Candles)-1].Close
		c.Analytics = append(c.Analytics, analytics)
	}
	if len(c.Analytics) > 0 {
		c.AnalyticsPerformed = true
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
		c == Monthly
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
