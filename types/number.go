package types

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

var errInvalidNumberValue = errors.New("invalid value for Number type")

// Number represents a floating point number, and implements json.Unmarshaller and json.Marshaller
type Number struct {
	f       float64
	s       string
	d       decimal.Decimal
	isEmpty bool
}

// UnmarshalJSON implements json.Unmarshaler
func (f *Number) UnmarshalJSON(data []byte) error {
	switch c := data[0]; c { // From json.decode literalInterface
	case 'n': // null
		*f = Number(0)
		return nil
	case 't', 'f': // true, false
		return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
	case '"': // string
		if len(data) < 2 || data[len(data)-1] != '"' {
			return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
		}
		data = data[1 : len(data)-1] // Naive Unquote
	default: // Should be a number
		if c != '-' && (c < '0' || c > '9') { // Invalid json syntax
			return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
		}
	}

	if len(data) == 0 {
		*f = Number{}
		return nil
	}
	s := string(data)
	if s == "" {
		*f = Number{
			isEmpty: true,
		}
		return nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidNumberValue, data) // We don't use err; We know it's not valid and errInvalidNumberValue is clearer
	}

	*f = Number{f: val, s: s, d: decimal.RequireFromString(s)}

	return nil
}

func (f Number) IsEmpty() bool {
	return f.isEmpty
}

func NumberFromString(s string) (Number, error) {
	if s == "" {
		return Number{isEmpty: true}, nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return Number{}, fmt.Errorf("%w: %s", errInvalidNumberValue, s)
	}

	return Number{f: val, s: s}, nil
}

func NumberFromFloat64(f float64) Number {
	return Number{f: f, s: strconv.FormatFloat(f, 'f', -1, 64)}
}

// MarshalJSON implements json.Marshaler by formatting to a json string
// 1337.37 will marshal to "1337.37"
// 0 will marshal to an empty string: ""
func (f Number) MarshalJSON() ([]byte, error) {
	if f.f == 0 {
		return []byte(`""`), nil
	}
	return []byte(`"` + f.s + `"`), nil
}

// Float64 returns the underlying float64
func (f Number) Float64() float64 {
	return f.f
}

// Int64 returns the truncated integer component of the number
func (f Number) Int64() int64 {
	// It's likely this is sufficient, since Numbers probably have not had floating point math performed on them
	// However if issues arise then we can switch to math.Round
	return int64(f.f)
}

// Decimal returns a decimal.Decimal
func (f Number) Decimal() decimal.Decimal {
	if f.s == "" {
		return decimal.Zero
	}
	return f.d
}

// String returns a string representation of the number
func (f Number) String() string {
	return f.s
}
