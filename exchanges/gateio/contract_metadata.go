package gateio

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
)

var errContractObservationTime = errors.New("contract observation time is required")

// NormaliseFuturesContracts converts one retained perpetual response without fetching it again.
// The observation time determines activity; conversion never advances source freshness.
func NormaliseFuturesContracts(exchangeName string, a asset.Item, contracts []FuturesContract, observedAt time.Time) ([]futures.Contract, error) {
	if a != asset.CoinMarginedFutures && a != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if observedAt.IsZero() {
		return nil, errContractObservationTime
	}
	settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(contracts))
	for i := range contracts {
		divisor, err := priceDivisor(a, contracts[i].Name)
		if err != nil {
			return nil, err
		}
		underlying := contracts[i].Name
		if underlying.Base.Equal(divisorCurrency) {
			underlying.Base = currency.BABYDOGE
		}
		contractSettlementType := futures.Linear
		switch {
		case contracts[i].Name.Base.Equal(currency.BTC) && settle.Equal(currency.BTC):
			contractSettlementType = futures.Inverse
		case !contracts[i].Name.Base.Equal(settle) && !settle.Equal(currency.USDT):
			contractSettlementType = futures.Quanto
		}
		c := futures.Contract{
			Exchange:           exchangeName,
			Name:               contracts[i].Name,
			Underlying:         underlying,
			Asset:              a,
			IsActive:           contracts[i].DelistedTime.Time().IsZero() || contracts[i].DelistedTime.Time().After(observedAt),
			Type:               futures.Perpetual,
			SettlementType:     contractSettlementType,
			SettlementCurrency: settle,
			Multiplier:         contracts[i].QuantoMultiplier.Float64(),
			PriceDivisor:       divisor,
			MaxLeverage:        contracts[i].LeverageMax.Float64(),
			LatestRate: fundingrate.Rate{
				Time: contracts[i].FundingNextApply.Time().Add(-time.Duration(contracts[i].FundingInterval) * time.Second),
				Rate: contracts[i].FundingRate.Decimal(),
			},
		}
		resp[i] = c
	}
	return resp, nil
}
