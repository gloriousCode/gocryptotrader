package gateio

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// UnmarshalJSON accepts both Gate wire representations without floating-point conversion.
func (id *AccountID) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		*id = ""
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*id = AccountID(value)
		return nil
	}
	if _, err := strconv.ParseUint(string(data), 10, 64); err != nil {
		return fmt.Errorf("gate account identifier: %w", err)
	}
	*id = AccountID(data)
	return nil
}
