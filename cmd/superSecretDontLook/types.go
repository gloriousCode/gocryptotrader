package main

import (
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
)

type Butts struct{}

func (b *Butts) PushEvent(evt base.Event) {}

type Key struct {
	Exchange        string
	Asset           asset.Item
	PairBase        *currency.Item
	PairQuote       *currency.Item
	UnderlyingBase  *currency.Item
	UnderlyingQuote *currency.Item
}

type PairDetails struct {
	ComparePair currency.Pair
	Key         Key

	Exchange               exchange.IBotExchange
	Asset                  asset.Item
	FuturesContract        *futures.Contract
	SpotPair               currency.Pair
	LastPrice              float64
	Volume                 float64
	QuoteVolume            float64
	Ask                    float64
	Bid                    float64
	Close                  float64
	OBSize                 float64
	ComparisonToContract   float64
	AnnualisedRateOfReturn float64
	DailyRateOfReturn      float64
}

type ComboHolder struct {
	ExchangeAssetTicker PairDetailsPointerHolder
}

type PairDetailsPointerHolder []*PairDetails

type result struct {
	Key                    Key
	baseExchange           string
	baseCurr               currency.Pair
	baseAsset              asset.Item
	contract               *PairDetails
	comparison             float64
	annualisedRateOfReturn float64
}

type spotPairs struct {
	exchange     string
	pair         currency.Pair
	superCompare currency.Pair
	spotLast     float64
	volume       float64
	comparisons  []result
}

type contractComparer struct {
	Main      *PairDetails
	Comparers []PairDetails
}
