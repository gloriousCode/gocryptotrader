package credentials

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	storage CredStore
	// ErrNoCredentialsForKey errors when no key found wow
	ErrNoCredentialsForKey = errors.New("no credentials found for key")
)

func init() {
	storage = CredStore{
		ExchangePairAssetCredentials: make(map[key.ExchangePairAsset]*Credentials),
		ExchangeAssetCredentials:     make(map[key.ExchangeAsset]*Credentials),
		ExchangeCredentials:          make(map[string]*Credentials),
	}
}

// CredStore stores credentials
type CredStore struct {
	mu                           sync.Mutex
	ExchangePairAssetCredentials map[key.ExchangePairAsset]*Credentials
	ExchangeAssetCredentials     map[key.ExchangeAsset]*Credentials
	ExchangeCredentials          map[string]*Credentials
}

func GetCredsByKey(k key.ExchangePairAsset) (*Credentials, error) {
	return storage.GetCredentialsForKey(k)
}

// EasyStore stores credentials in an ez manner
func (c *CredStore) EasyStore(exch string, item asset.Item, base, quote *currency.Item, creds *Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if base != nil && quote != nil {
		bCode := currency.NewCode(base.Symbol)
		qCode := currency.NewCode(quote.Symbol)
		k := key.ExchangePairAsset{
			Exchange: exch,
			Base:     bCode.Item,
			Quote:    qCode.Item,
			Asset:    item,
		}

		c.ExchangePairAssetCredentials[k] = creds
		return
	}
	if item != asset.Empty {
		k := key.ExchangeAsset{
			Exchange: exch,
			Asset:    item,
		}
		c.ExchangeAssetCredentials[key.ExchangeAsset{
			Exchange: k.Exchange,
			Asset:    k.Asset,
		}] = creds
		return
	}
	c.ExchangeCredentials[exch] = creds
}

// StoreExchangePairAssetCredentials stores exchange pair asset credentials
func (c *CredStore) StoreExchangePairAssetCredentials(k key.ExchangePairAsset, creds *Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ExchangePairAssetCredentials[k] = creds
}

// StoreExchangeAssetCredentials stores exchange asset credentials
func (c *CredStore) StoreExchangeAssetCredentials(k key.ExchangeAsset, creds *Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ExchangeAssetCredentials[k] = creds
}

// StoreExchangeCredentials stores exchange credentials
// this is likely the main manner to store credentials
func (c *CredStore) StoreExchangeCredentials(k string, creds *Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ExchangeCredentials[k] = creds
}

// GetCredentialsForKey returns credentials for a given key
func (c *CredStore) GetCredentialsForKey(k key.ExchangePairAsset) (*Credentials, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ExchangePairAssetCredentials != nil {
		creds, ok := c.ExchangePairAssetCredentials[k]
		if ok {
			return creds, nil
		}
	}
	if c.ExchangeAssetCredentials != nil {
		creds, ok := c.ExchangeAssetCredentials[key.ExchangeAsset{
			Exchange: k.Exchange,
			Asset:    k.Asset,
		}]
		if ok {
			return creds, nil
		}
	}
	creds, ok := c.ExchangeCredentials[k.Exchange]
	if ok {
		return creds, nil
	}
	return nil, ErrNoCredentialsForKey
}

func (c *CredStore) PrepareForSaving() (marshaledKeys []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var resp AtRestCredStore
	for _, cred := range c.ExchangePairAssetCredentials {
		resp.ExchangePairAssetCredentials = append(resp.ExchangePairAssetCredentials, cred)
	}
	for _, cred := range c.ExchangeAssetCredentials {
		resp.ExchangeAssetCredentials = append(resp.ExchangeAssetCredentials, cred)
	}
	for _, cred := range c.ExchangeCredentials {
		resp.ExchangeCredentials = append(resp.ExchangeCredentials, cred)
	}
	return json.Marshal(&resp)
}
