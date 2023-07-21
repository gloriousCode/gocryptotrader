package main

import (
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
)

type PairDetails struct {
	Exchange     exchange.IBotExchange
	Contract     *futures.Contract
	SuperCompare currency.Pair
	LastPrice    float64
	Ask          float64
	Bid          float64
	Close        float64
}

var m sync.Mutex

type SpotPairDetails struct {
	Exchange  string
	Pair      currency.Pair
	LastPrice float64
}

type ComboHolder struct {
	ExchangeAssetTicker map[string]PairDetails
}

type HolderHolder struct {
	ComparableCurrencyPairs map[string]ComboHolder
}

type Deals []CashCarryDeal

func (d Deals) BestDeal() *CashCarryDeal {
	var bestDeal *CashCarryDeal
	for i := range d {
		if bestDeal == nil {
			bestDeal = &d[i]
		}
		if d[i].PriceDifference.GreaterThan(bestDeal.PriceDifference) {
			bestDeal = &d[i]
		}
	}
	return bestDeal
}

type CashCarryDeal struct {
	PriceDifference decimal.Decimal
	BasePrice       decimal.Decimal
	EndPrice        decimal.Decimal
	IsFuturesBase   bool
	BaseSpot        SpotPairDetails
	BaseFutures     PairDetails
	EndFutures      PairDetails
}
