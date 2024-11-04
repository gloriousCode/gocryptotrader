package key

import (
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account/credentials"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ExchangePairAsset is a unique map key signature for exchange, currency pair and asset
type ExchangePairAsset struct {
	Exchange string
	Base     *currency.Item
	Quote    *currency.Item
	Asset    asset.Item
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

type ExchangeCredentials struct {
	Exchange    string
	Credentials credentials.Credentials
}

type ExchangeAssetCredentials struct {
	Exchange    string
	Asset       asset.Item
	Credentials *credentials.Credentials
}

type ExchangePairAssetCredentials struct {
	Exchange    string
	Asset       asset.Item     `json:"asset,omitempty"`
	Base        *currency.Item `json:"base,omitempty"`
	Quote       *currency.Item `json:"quote,omitempty"`
	Credentials *credentials.Credentials
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

func (k *ExchangePairAssetCredentials) KeyNoCreds() ExchangePairAsset {
	return ExchangePairAsset{
		Exchange: k.Exchange,
		Base:     k.Base,
		Quote:    k.Quote,
		Asset:    k.Asset,
	}
}

func (k *ExchangePairAssetCredentials) Pair() currency.Pair {
	if k == nil || (k.Base == nil && k.Quote == nil) {
		return currency.EMPTYPAIR
	}
	return currency.NewPair(k.Base.Currency(), k.Quote.Currency())
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

type Pair struct {
	Base  *currency.Item
	Quote *currency.Item
}

// SubAccountCurrencyAsset is a unique map key signature for subaccount, currency code and asset
type SubAccountCurrencyAsset struct {
	SubAccount string
	Currency   *currency.Item
	Asset      asset.Item
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
