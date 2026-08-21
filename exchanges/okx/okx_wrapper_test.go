package okx

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := new(Exchange).MessageID()
	require.Len(t, id, 32, "Must return the correct length of message id")
	u, err := uuid.FromString(id)
	require.NoError(t, err, "MessageID must return a valid UUID")
	require.Equal(t, uuid.V7, u.Version(), "MessageID must return a V7 uuid")
	require.Len(t, u.String(), 36, "UUID v7 string representation must be 36 characters long")
}

// 7696807	       153.1 ns/op	      48 B/op	       2 allocs/op
func BenchmarkMessageID(b *testing.B) {
	e := new(Exchange)
	for b.Loop() {
		_ = e.MessageID()
	}
}

func TestOptionInstrumentSelectors(t *testing.T) {
	t.Parallel()

	underlying, family := optionInstrumentSelectors("BTC-USD-240329-70000-C")
	require.Equal(t, "BTC-USD", underlying, "underlying selector must parse option instrument ID")
	require.Equal(t, "BTC-USD", family, "family selector must parse option instrument ID")

	underlying, family = optionInstrumentSelectors("ETH_USD_240329_3500_P")
	require.Equal(t, "ETH_USD", underlying, "underlying selector must parse underscore instrument ID")
	require.Equal(t, "ETH_USD", family, "family selector must parse underscore instrument ID")

	underlying, family = optionInstrumentSelectors("INVALID")
	require.Equal(t, "INVALID", underlying, "fallback underlying must return raw instrument ID")
	require.Equal(t, "INVALID", family, "fallback family must return raw instrument ID")
}
