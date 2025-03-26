package types

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

var errInvalidNumberValue = errors.New("invalid value for Number type")

// Number represents a floating point number, and implements json.Unmarshaller and json.Marshaller
// Upon unmarshalling, it stores the string value and the float64 value
type Number struct {
	f float64
	s string
}

// UnmarshalJSON implements json.Unmarshaler
func (f *Number) UnmarshalJSON(data []byte) error {
	switch c := data[0]; c { // From json.decode literalInterface
	case 'n', 't', 'f': // null, true, false
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
		*f = Number{f: 0, s: "0"}
		return nil
	}
	s := string(data)
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidNumberValue, data) // We don't use err; We know it's not valid and errInvalidNumberValue is clearer
	}

	*f = Number{
		f: val,
		s: s,
	}
	return nil
}

func NewNumberFromFloat(f float64) Number {
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

// IsZero returns true if the number is zero
func (f Number) IsZero() bool {
	return f.f == 0
}

// IsValid is only relevant if someone has tried to instantiate a Number struct manually
// that really shouldn't happen and so you probably won't ever call this
func (f Number) IsValid() bool {
	return f.s != ""
}

// MarshalJSON implements json.Marshaler by formatting to a json string
// 1337.37 will marshal to "1337.37"
// 0 will marshal to an empty string: ""
func (f Number) MarshalJSON() ([]byte, error) {
	if f.f == 0 {
		return []byte(`""`), nil
	}
	return fmt.Appendf(nil, "%q", f.s), nil
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
	return decimal.NewFromFloat(f.f)
}

// DecimalFromString returns a decimal.Decimal.
// It is faster and more precise than Decimal(), but returns an error if Number is invalid
// The likelihood of Number being invalid is very low, but a well-regarded programmer could call it after UnmarshalJSON errors
func (f Number) DecimalFromString() (decimal.Decimal, error) {
	return decimal.NewFromString(f.s)
}

// String returns a string representation of the number
func (f Number) String() string {
	return f.s
}
