package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountIDUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, want string
		invalid           bool
	}{
		{"number", "123456", "123456", false},
		{"quoted", `"123456"`, "123456", false},
		{"masked string", `"110xxxxx"`, "110xxxxx", false},
		{"exact large identifier", "9007199254740993", "9007199254740993", false},
		{"whitespace", " 123 ", "123", false},
		{"null", "null", "", false},
		{"empty string", `""`, "", false},
		{"broken string", `"broken`, "", true},
		{"negative", "-1", "", true},
		{"fraction", "1.2", "", true},
		{"boolean", "true", "", true},
		{"object", "{}", "", true},
		{"overflow", "18446744073709551616", "", true},
		{"empty", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			id := AccountID("retained")
			err := id.UnmarshalJSON([]byte(tc.input))
			if tc.invalid {
				require.Error(t, err, "invalid wire value must fail")
				assert.Equal(t, AccountID("retained"), id, "failed decoding should preserve the prior identifier")
				return
			}
			require.NoError(t, err, "supported wire value must decode")
			assert.Equal(t, tc.want, string(id), "identifier should retain its source digits")
		})
	}
}
