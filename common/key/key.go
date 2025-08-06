package key

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	ErrNotFound = errors.New("key not found")
)

// ExchangePairAsset is a unique map key signature for exchange, currency pair and asset
type ExchangePairAsset struct {
	Exchange string         `json:"exchange"`
	Base     *currency.Item `json:"base,omitempty"`
	Quote    *currency.Item `json:"quote,omitempty"`
	Asset    asset.Item     `json:"asset,omitempty"`
}

func NewExchangePairAssetKey(exch string, a asset.Item, cp currency.Pair) ExchangePairAsset {
	return ExchangePairAsset{
		Exchange: exch,
		Base:     cp.Base.Item,
		Quote:    cp.Quote.Item,
		Asset:    a,
	}
}

type ExchangePairAssetUnderlyingContractExpiry struct {
	Exchange         string
	Base             *currency.Item
	Quote            *currency.Item
	Asset            asset.Item
	Contract         string
	ContractDecimals float64
	Expiry           time.Time `json:"Expiry,omitempty"`
	UnderlyingBase   *currency.Item
	UnderlyingQuote  *currency.Item
}

type OrderKey struct {
	Exchange  string
	Base      *currency.Item
	Quote     *currency.Item
	Asset     asset.Item
	Time      time.Time
	OrderID   string
	OrderSide string
	OrderSize float64
}

func (k *ExchangePairAssetUnderlyingContractExpiry) ToEPA() ExchangePairAsset {
	return ExchangePairAsset{
		Exchange: k.Exchange,
		Base:     k.Base,
		Quote:    k.Quote,
		Asset:    k.Asset,
	}
}

// ExchangeAsset is a unique map key signature for exchange and asset
type ExchangeAsset struct {
	Exchange string
	Asset    asset.Item
}

// PairAsset is a unique map key signature for currency pair and asset
type PairAsset struct {
	Base  *currency.Item
	Quote *currency.Item
	Asset asset.Item
}

// SubAccountAsset is a unique map key signature for subaccount and asset
type SubAccountAsset struct {
	SubAccount string
	Asset      asset.Item
}

type Pair struct {
	Base  *currency.Item
	Quote *currency.Item
}

// SubAccountCurrencyAsset is a unique map key signature for subaccount, currency code and asset
type SubAccountCurrencyAsset struct {
	SubAccount string
	Asset      asset.Item
	Currency   *currency.Item
}

// Pair combines the base and quote into a pair
func (k *PairAsset) Pair() currency.Pair {
	if k == nil || (k.Base == nil && k.Quote == nil) {
		return currency.EMPTYPAIR
	}
	return currency.NewPair(k.Base.Currency(), k.Quote.Currency())
}

// Pair combines the base and quote into a pair
func (k *ExchangePairAsset) Pair() currency.Pair {
	if k == nil || (k.Base == nil && k.Quote == nil) {
		return currency.EMPTYPAIR
	}
	return currency.NewPair(k.Base.Currency(), k.Quote.Currency())
}

// Pair combines the base and quote into a pair
func (k *ExchangePairAsset) String() string {
	return fmt.Sprintf("%s %s %s-%s", k.Exchange, k.Asset, k.Base, k.Quote)
}

// MatchesExchangeAsset checks if the key matches the exchange and asset
func (k *ExchangePairAsset) MatchesExchangeAsset(exch string, item asset.Item) bool {
	if k == nil {
		return false
	}
	return strings.EqualFold(k.Exchange, exch) && k.Asset == item
}

// MatchesPairAsset checks if the key matches the pair and asset
func (k *ExchangePairAsset) MatchesPairAsset(pair currency.Pair, item asset.Item) bool {
	if k == nil {
		return false
	}
	return k.Base == pair.Base.Item &&
		k.Quote == pair.Quote.Item &&
		k.Asset == item
}

// MatchesExchange checks if the exchange matches
func (k *ExchangePairAsset) MatchesExchange(exch string) bool {
	if k == nil {
		return false
	}
	return strings.EqualFold(k.Exchange, exch)
}
