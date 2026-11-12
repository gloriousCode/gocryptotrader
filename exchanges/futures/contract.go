package futures

import (
	"errors"
	"strings"
	"time"
)

// var error definitions
var (
	ErrInvalidContractSettlementType = errors.New("invalid contract settlement type")
	ErrContractNotSupported          = errors.New("unsupported contract")
)

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

// IsLongDated reports whether the contract type has a fixed expiry.
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
		return "unset"
	}
}

// Duration returns the nominal duration represented by the contract type.
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
