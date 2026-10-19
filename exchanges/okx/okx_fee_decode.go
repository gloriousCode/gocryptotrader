package okx

import "github.com/thrasher-corp/gocryptotrader/encoding/json"

// UnmarshalJSON preserves presence of legacy fee fields, including explicit zero rates.
func (r *TradeFeeRate) UnmarshalJSON(data []byte) error {
	type rate TradeFeeRate
	var decoded rate
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields struct {
		Maker     *json.RawMessage `json:"maker"`
		Taker     *json.RawMessage `json:"taker"`
		MakerUSDT *json.RawMessage `json:"makerU"`
		TakerUSDT *json.RawMessage `json:"takerU"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	decoded.MakerPresent = fields.Maker != nil && string(*fields.Maker) != `""`
	decoded.TakerPresent = fields.Taker != nil && string(*fields.Taker) != `""`
	decoded.MakerUSDTPresent = fields.MakerUSDT != nil && string(*fields.MakerUSDT) != `""`
	decoded.TakerUSDTPresent = fields.TakerUSDT != nil && string(*fields.TakerUSDT) != `""`
	*r = TradeFeeRate(decoded)
	return nil
}
