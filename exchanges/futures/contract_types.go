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
	SettlementCurrency        currency.Code
	MarginCurrency            currency.Code
	Multiplier                float64
	ContractValueInSettlement float64
	MaxLeverage               float64
	LatestRate                fundingrate.Rate
	FundingRateFloor          decimal.Decimal
	FundingRateCeiling        decimal.Decimal
	ContractValue             ContractValue
}

type HistoricalContractKline struct {
	RequestKey              key.PairAsset   `json:"-"`
	Data                    []ContractKline `json:"-"`
	Analytics               []ContractKlineAnalytics
	AnalyticsPerformed      bool
	AnyContangos            bool
	AnyPositiveContangoes   bool
	ContangoPercent         float64
	PositiveContangoPercent float64
	PositiveOutcomePercent  float64
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
	Gain         float64
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

type ContractValue int64

type GetKlineContractRequest struct {
	ContractPair currency.Pair
	// used for okx
	UnderlyingPair                 currency.Pair
	Asset                          asset.Item
	StartDate                      time.Time
	EndDate                        time.Time
	Interval                       kline.Interval
	Contract                       ContractType
	IndividualContractDenomination ContractValue
}

var ErrUnderlyingPairRequired = errors.New("underlying pair required")

const (
	UnsetDenomination ContractValue = iota
	BaseContract
	QuoteContract
)

func (c *ContractValue) String() string {
	switch *c {
	case BaseContract:
		return "base"
	case QuoteContract:
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
