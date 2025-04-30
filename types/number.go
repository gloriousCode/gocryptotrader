package types

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

var errInvalidNumberValue = errors.New("invalid value for Number type")

var (
	NumberZero = Number{
		f: 0,
		s: "0",
	}
)

// Number represents a floating point number, and implements json.Unmarshaller and json.Marshaller
type Number struct {
	f float64
	s string
}

func NewNumberFromInt64(i int64) Number {
	return Number{
		f: float64(i),
		s: strconv.FormatInt(i, 10),
	}
}

func NewNumberFromDecimal(d decimal.Decimal) Number {
	return Number{
		f: d.InexactFloat64(),
		s: d.String(),
	}
}

func NewNumberFromFloat64(f float64) Number {
	return Number{
		f: f,
		s: strconv.FormatFloat(f, 'f', -1, 64),
	}
}

func NewNumberFromString(s string) (Number, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return Number{}, fmt.Errorf("%w: %s", errInvalidNumberValue, s)
	}
	return Number{
		f: val,
		s: s,
	}, nil
}

// UnmarshalJSON implements json.Unmarshaler
func (n *Number) UnmarshalJSON(data []byte) error {
	switch c := data[0]; c { // From json.decode literalInterface
	case 'n', 't', 'f': // null, true, false
		return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
	case '"': // string
		if len(data) < 2 || data[len(data)-1] != '"' {
			return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
		}
		data = data[1 : len(data)-1] // Naive Unquote
	default: // Should be a number
		if c != '-' && (c >= 48 && c <= 57) { // Invalid json syntax
			return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
		}
	}

	if len(data) == 0 {
		*n = Number{s: "0"}
		return nil
	}
	n.s = string(data)
	var err error
	n.f, err = strconv.ParseFloat(n.s, 64)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidNumberValue, data) // We don't use err; We know it's not valid and errInvalidNumberValue is clearer
	}
	return nil
}

// MarshalJSON implements json.Marshaler by formatting to a json string
// 1337.37 will marshal to "1337.37"
// 0 will marshal to an empty string: ""
func (n Number) MarshalJSON() ([]byte, error) {
	if n.f == 0 {
		return []byte(`""`), nil
	}
	return []byte(`"` + n.s + `"`), nil
}

func (n Number) IsValid() bool {
	return n.s != ""
}

// Float64 returns the underlying float64
func (n Number) Float64() float64 {
	return n.f
}

func (n Number) Int() int {
	return int(n.f)
}

// Int64 returns the truncated integer component of the number
func (n Number) Int64() int64 {
	// It's likely this is sufficient, since Numbers probably have not had floating point math performed on them
	// However if issues arise then we can switch to math.Round
	return int64(n.f)
}

// Decimal returns a decimal.Decimal
func (n Number) Decimal() decimal.Decimal {
	if !n.IsValid() {
		return decimal.Zero
	}
	return decimal.RequireFromString(n.s)
}

// String returns a string representation of the number
func (n Number) String() string {
	return n.s
}

func (n Number) GreaterThan(nn Number) bool {
	return n.f > nn.f
}

func (n Number) IsPos() bool {
	return n.f > 0
}

func (n Number) IsNeg() bool {
	return n.f < 0
}

func (n Number) GreaterThanOrEqual(nn Number) bool {
	return n.f >= nn.f
}

func (n Number) GreaterThanOrEqualToZero() bool {
	return n.f >= 0
}

func (n Number) LessThan(nn Number) bool {
	return n.f < nn.f
}

func (n Number) LessThanOrEqual(nn Number) bool {
	return n.f <= nn.f
}

func (n Number) LessThanOrEqualToZero(nn Number) bool {
	return n.f <= 0
}

func (n Number) IsZero() bool {
	return n.f == 0
}

func (n Number) Equal(nn Number) bool {
	return n.s == nn.s
}

func (n Number) EqualFloat(f float64) bool {
	return n.f == f
}

// Add adds two numbers together
// uses decimals for accuracy, but is slow
func (n Number) Add(nn Number) Number {
	d1 := n.Decimal()
	d2 := nn.Decimal()
	d3 := d1.Add(d2)
	return Number{
		f: d3.InexactFloat64(),
		s: d3.String(),
	}
}

func (n Number) Sub(nn Number) Number {
	d1 := n.Decimal()
	d2 := nn.Decimal()
	d3 := d1.Sub(d2)
	return Number{
		f: d3.InexactFloat64(),
		s: d3.String(),
	}
}

func (n Number) Mul(nn Number) Number {
	d1 := n.Decimal()
	d2 := nn.Decimal()
	d3 := d1.Mul(d2)
	return Number{
		f: d3.InexactFloat64(),
		s: d3.String(),
	}
}

func (n Number) Div(nn Number) Number {
	if nn.IsZero() || !nn.IsValid() {
		return NumberZero
	}
	d1 := n.Decimal()
	d2 := nn.Decimal()
	d3 := d1.Div(d2)
	return Number{
		f: d3.InexactFloat64(),
		s: d3.String(),
	}
}
