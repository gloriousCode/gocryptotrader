package futures

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
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
	SettlementCurrency             currency.Code
	MarginCurrency                 currency.Code
	Multiplier                     float64
	MaxLeverage                    float64
	LatestRate                     fundingrate.Rate
	FundingRateFloor               decimal.Decimal
	FundingRateCeiling             decimal.Decimal
	AdditionalSettlementCurrencies currency.Currencies
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
