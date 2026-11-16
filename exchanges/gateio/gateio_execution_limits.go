package gateio

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// NormaliseSpotExecutionLimits converts an observed spot catalogue without network or registry access.
func NormaliseSpotExecutionLimits(name string, pairsData []CurrencyPairDetail, observedAt time.Time) []limits.MinMaxLevel {
	a := asset.Spot
	l := make([]limits.MinMaxLevel, 0, len(pairsData))
	for i := range pairsData {
		if pairsData[i].TradeStatus == "untradable" {
			continue
		}
		// Minimum base amounts are not always provided this will default to
		// precision for base deployment. This can't be done for quote.
		minBaseAmount := pairsData[i].MinBaseAmount.Float64()
		if minBaseAmount == 0 {
			minBaseAmount = math.Pow10(-int(pairsData[i].AmountPrecision))
		}
		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(name, a, currency.NewPair(pairsData[i].Base, pairsData[i].Quote)),
			PriceStepIncrementSize:  math.Pow10(-int(pairsData[i].PricePrecision)),
			AmountStepIncrementSize: math.Pow10(-int(pairsData[i].AmountPrecision)),
			MinimumBaseAmount:       minBaseAmount,
			MinimumQuoteAmount:      pairsData[i].MinQuoteAmount.Float64(),
			MaximumBaseAmount:       pairsData[i].MaxBaseAmount.Float64(),
			MaximumQuoteAmount:      pairsData[i].MaxQuoteAmount.Float64(),
			Delisted:                pairsData[i].DelistingTime.Time(),
			Listed:                  earliestTime(observedAt, pairsData[i].SellStart.Time(), pairsData[i].BuyStart.Time()),
			MultiplierUp:            pairsData[i].MaximumQuoteRisePercentage.Float64(),
			MultiplierDown:          pairsData[i].MaximumQuoteDeclinePercentage.Float64(),
			MarketMaxQty:            pairsData[i].MarketOrderMaxStock.Float64(),
		})
	}
	return l
}

// NormaliseFuturesExecutionLimits preserves exact native contract units from a public catalogue.
func NormaliseFuturesExecutionLimits(name string, a asset.Item, contractInfo []FuturesContract) ([]limits.MinMaxLevel, error) {
	if a != asset.USDTMarginedFutures && a != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	l := make([]limits.MinMaxLevel, 0, len(contractInfo))
	for i := range contractInfo {
		pd, err := priceDivisor(a, contractInfo[i].Name)
		if err != nil {
			return nil, err
		}

		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(name, a, contractInfo[i].Name),
			MinimumBaseAmount:       contractInfo[i].OrderSizeMin.Float64(),
			MaximumBaseAmount:       contractInfo[i].OrderSizeMax.Float64(),
			PriceStepIncrementSize:  contractInfo[i].OrderPriceRound.Float64(),
			AmountStepIncrementSize: 1, // 1 Contract
			MultiplierDecimal:       contractInfo[i].QuantoMultiplier.Float64(),
			PriceDivisor:            pd,
			Delisting:               contractInfo[i].DelistingTime.Time(),
			Delisted:                contractInfo[i].DelistedTime.Time(),
			Listed:                  contractInfo[i].LaunchTime.Time(),
		})
	}
	return l, nil
}

// NormaliseDeliveryExecutionLimits converts one delivery catalogue without fetching another settlement.
func NormaliseDeliveryExecutionLimits(name string, contractInfo []DeliveryContract) ([]limits.MinMaxLevel, error) {
	a := asset.DeliveryFutures
	l := make([]limits.MinMaxLevel, 0, len(contractInfo))
	for x := range contractInfo {
		p := strings.ToUpper(contractInfo[x].Name)
		cp, err := currency.NewPairFromString(p)
		if err != nil {
			return nil, err
		}
		pd, err := priceDivisor(a, cp)
		if err != nil {
			return nil, err
		}
		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(name, a, cp),
			MinimumBaseAmount:       contractInfo[x].OrderSizeMin.Float64(),
			MaximumBaseAmount:       contractInfo[x].OrderSizeMax.Float64(),
			PriceStepIncrementSize:  contractInfo[x].OrderPriceRound.Float64(),
			AmountStepIncrementSize: 1,
			MultiplierDecimal:       contractInfo[x].QuantoMultiplier.Float64(),
			PriceDivisor:            pd,
			Expiry:                  contractInfo[x].ExpireTime.Time(),
		})
	}
	return l, nil
}
