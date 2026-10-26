package okx

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
)

// FuturesContractFromInstrument converts native REST or websocket metadata without network requests.
func (e *Exchange) FuturesContractFromInstrument(instrument *Instrument, item asset.Item) (futures.Contract, error) {
	if instrument == nil {
		return futures.Contract{}, common.ErrNilPointer
	}
	if !item.IsFutures() {
		return futures.Contract{}, futures.ErrNotFuturesAsset
	}
	if item != asset.Futures && item != asset.PerpetualSwap {
		return futures.Contract{}, fmt.Errorf("%w %s", asset.ErrNotSupported, item)
	}
	if instrument.InstrumentType != "" && !strings.EqualFold(instrument.InstrumentType, GetInstrumentTypeFromAssetItem(item)) {
		return futures.Contract{}, fmt.Errorf("%w: instrument type %s", asset.ErrNotSupported, instrument.InstrumentType)
	}
	var err error

	var (
		underlying             currency.Pair
		settleCurr             currency.Code
		contractSettlementType futures.ContractSettlementType
	)

	if instrument.State == "live" {
		underlying, err = currency.NewPairFromString(instrument.Underlying)
		if err != nil {
			return futures.Contract{}, err
		}

		settleCurr = instrument.SettlementCurrency

		contractSettlementType, err = futures.StringToContractSettlementType(instrument.ContractType)
		if err != nil {
			return futures.Contract{}, err
		}
		if contractSettlementType == futures.UnsetSettlementType {
			contractSettlementType = futures.Linear
			if instrument.SettlementCurrency.Equal(underlying.Base) {
				contractSettlementType = futures.Inverse
			}
		}
	}

	var ct futures.ContractType
	if item == asset.PerpetualSwap {
		ct = futures.Perpetual
	} else {
		switch instrument.Alias {
		case "this_week", "next_week":
			ct = futures.Weekly
		case "quarter", "next_quarter":
			ct = futures.Quarterly
		}
	}

	contract := futures.Contract{
		Exchange:       e.Name,
		Name:           instrument.InstrumentID,
		Underlying:     underlying,
		Asset:          item,
		StartDate:      instrument.ListTime.Time(),
		EndDate:        instrument.ExpTime.Time(),
		IsActive:       instrument.State == "live",
		Status:         instrument.State,
		Type:           ct,
		SettlementType: contractSettlementType,
		MarginCurrency: settleCurr,
		Multiplier:     instrument.ContractValue.Float64(),
		MaxLeverage:    instrument.MaxLeverage.Float64(),
	}

	if !settleCurr.IsEmpty() {
		contract.SettlementCurrency = settleCurr
	}
	return contract, nil
}
