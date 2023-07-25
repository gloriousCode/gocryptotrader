package stats

import (
	"errors"
	"sort"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// item holds various fields for storing currency pair stats
type item struct {
	Exchange  string
	Pair      currency.Pair
	AssetType asset.Item
	Price     float64
	Volume    float64
}

var itemLocker sync.Mutex

// items var array
var items []item

// ByPrice allows sorting by price
type ByPrice []item

func (b ByPrice) Len() int {
	return len(b)
}

func (b ByPrice) Less(i, j int) bool {
	return b[i].Price < b[j].Price
}

func (b ByPrice) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// ByVolume allows sorting by volume
type ByVolume []item

func (b ByVolume) Len() int {
	return len(b)
}

func (b ByVolume) Less(i, j int) bool {
	return b[i].Volume < b[j].Volume
}

func (b ByVolume) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// Add adds or updates the item stats
func Add(exchange string, p currency.Pair, a asset.Item, price, volume float64) error {
	if exchange == "" ||
		a == asset.Empty ||
		price == 0 ||
		volume == 0 ||
		p.Base.IsEmpty() ||
		p.Quote.IsEmpty() {
		return errors.New("cannot add or update, invalid params")
	}

	if p.Base.Equal(currency.XBT) {
		newPair, err := currency.NewPairFromStrings(currency.BTC.String(),
			p.Quote.String())
		if err != nil {
			return err
		}
		Append(exchange, newPair, a, price, volume)
	}

	if p.Quote.Equal(currency.USDT) {
		newPair, err := currency.NewPairFromStrings(p.Base.String(), currency.USD.String())
		if err != nil {
			return err
		}
		Append(exchange, newPair, a, price, volume)
	}

	Append(exchange, p, a, price, volume)
	return nil
}

// Append adds or updates the item stats for a specific
// currency pair and asset type
func Append(exchange string, p currency.Pair, a asset.Item, price, volume float64) {
	if AlreadyExists(exchange, p, a, price, volume) {
		return
	}

	i := item{
		Exchange:  exchange,
		Pair:      p,
		AssetType: a,
		Price:     price,
		Volume:    volume,
	}
	itemLocker.Lock()
	defer itemLocker.Unlock()
	items = append(items, i)
}

// AlreadyExists checks to see if item info already exists
// for a specific currency pair and asset type
func AlreadyExists(exchange string, p currency.Pair, assetType asset.Item, price, volume float64) bool {
	itemLocker.Lock()
	defer itemLocker.Unlock()
	for i := range items {
		if items[i].Exchange == exchange &&
			items[i].Pair.EqualIncludeReciprocal(p) &&
			items[i].AssetType == assetType {
			items[i].Price, items[i].Volume = price, volume
			return true
		}
	}
	return false
}

// SortExchangesByVolume sorts item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByVolume(p currency.Pair, assetType asset.Item, reverse bool) []item {
	var result []item
	itemLocker.Lock()
	defer itemLocker.Unlock()
	for x := range items {
		if items[x].Pair.EqualIncludeReciprocal(p) &&
			items[x].AssetType == assetType {
			result = append(result, items[x])
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(ByVolume(result)))
	} else {
		sort.Sort(ByVolume(result))
	}
	return result
}

// SortExchangesByPrice sorts item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByPrice(p currency.Pair, assetType asset.Item, reverse bool) []item {
	var result []item
	itemLocker.Lock()
	defer itemLocker.Unlock()
	for x := range items {
		if items[x].Pair.EqualIncludeReciprocal(p) &&
			items[x].AssetType == assetType {
			result = append(result, items[x])
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(ByPrice(result)))
	} else {
		sort.Sort(ByPrice(result))
	}
	return result
}
