package okx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestTradeFeeRateUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload string
		present       bool
		invalid       bool
	}{
		{name: "explicit zero strings", payload: `{"maker":"0","taker":"0","makerU":"0","takerU":"0"}`, present: true},
		{name: "explicit zero numbers", payload: `{"maker":0,"taker":0,"makerU":0,"takerU":0}`, present: true},
		{name: "present charges", payload: `{"maker":"-0.001","taker":"-0.002","makerU":"-0.001","takerU":"-0.002"}`, present: true},
		{name: "empty", payload: `{"maker":"","taker":"","makerU":"","takerU":""}`},
		{name: "null", payload: `{"maker":null,"taker":null,"makerU":null,"takerU":null}`},
		{name: "missing", payload: `{}`},
		{name: "invalid", payload: `{"maker":"invalid"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := TradeFeeRate{MakerPresent: true, TakerPresent: true, MakerUSDTPresent: true, TakerUSDTPresent: true}
			err := json.Unmarshal([]byte(tc.payload), &row)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.present, row.MakerPresent)
			assert.Equal(t, tc.present, row.TakerPresent)
			assert.Equal(t, tc.present, row.MakerUSDTPresent)
			assert.Equal(t, tc.present, row.TakerUSDTPresent)
		})
	}
	t.Run("modern fee group", func(t *testing.T) {
		t.Parallel()
		var row TradeFeeRate
		require.NoError(t, json.Unmarshal([]byte(`{"feeGroup":[{"groupId":"4","maker":"0","taker":"-0.0005"}]}`), &row))
		require.Len(t, row.FeeGroup, 1)
		assert.Equal(t, TradeFeeGroup{GroupID: "4", Maker: "0", Taker: "-0.0005"}, row.FeeGroup[0])
		var instrument Instrument
		require.NoError(t, json.Unmarshal([]byte(`{"instId":"BTC-USDT-SWAP","groupId":"4"}`), &instrument))
		assert.Equal(t, "4", instrument.GroupID)
	})
}
