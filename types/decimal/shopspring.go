//go:build !sonic_on

package decimal

import "github.com/shopspring/decimal"

// Decimal is a wrapper around shopspring/decimal.Decimal
type Decimal struct {
	d decimal.Decimal
}

// NewDecimal creates a new Decimal from a float64
func NewDecimal(value float64) Decimal {
	return Decimal{d: decimal.NewFromFloat(value)}
}

// Add adds two Decimals
func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{d: d.d.Add(other.d)}
}

// Sub subtracts two Decimals
func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{d: d.d.Sub(other.d)}
}

// Mul multiplies two Decimals
func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{d: d.d.Mul(other.d)}
}

// Div divides two Decimals
func (d Decimal) Div(other Decimal) Decimal {
	return Decimal{d: d.d.Div(other.d)}
}

// String returns the string representation
func (d Decimal) String() string {
	return d.d.String()
}

// Cmp compares two Decimals
func (d Decimal) Cmp(other Decimal) int {
	return d.d.Cmp(other.d)
}
