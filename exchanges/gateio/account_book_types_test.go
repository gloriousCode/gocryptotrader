package gateio

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// TestAccountBookItem verifies stable futures account-book attribution fields are retained.
func TestAccountBookItem(t *testing.T) {
	t.Parallel()

	var item AccountBookItem
	require.NoError(t, json.Unmarshal([]byte(`{
		"time": 1682294400.123456,
		"change": "0.000010152188",
		"balance": "4.59316525194",
		"text": "ETH_USDT:6086261",
		"type": "fund",
		"contract": "ETH_USDT",
		"trade_id": "17",
		"id": "29"
	}`), &item))
	require.Equal(t, "ETH_USDT", item.Contract)
	require.Equal(t, "17", item.TradeID)
	require.Equal(t, "29", item.ID)
}
