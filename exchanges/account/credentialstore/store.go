package credentialstore

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrNoCredentialsForKey is an error returned when no credentials are found for a given key
	ErrNoCredentialsForKey = errors.New("no credentials found for key")
	// DefaultFilePath is a function that returns the default path for the credentials file
	DefaultFilePath = func() string {
		return filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "creds.creds")
	}
)

// Store struct holds different types of credentials
type Store struct {
	ExchangeAssetPairCredentials map[*key.ExchangeAssetPair]*account.Credentials
	m                            sync.Mutex // Mutex for thread-safe operations
}

func NewStore() *Store {
	return &Store{
		ExchangeAssetPairCredentials: make(map[*key.ExchangeAssetPair]*account.Credentials),
	}
}

// GetAll returns a map of all stored credentials
func (c *Store) GetAll() map[*key.ExchangeAssetPair]account.Credentials {
	c.m.Lock()
	defer c.m.Unlock()
	creds := make(map[*key.ExchangeAssetPair]account.Credentials)
	for k, v := range c.ExchangeAssetPairCredentials {
		creds[k] = *v
	}
	return creds
}

// Upsert is a convenience method to store credentials based on provided parameters
func (c *Store) Upsert(exch string, item asset.Item, base, quote *currency.Item, creds *account.Credentials) error {
	if creds == nil {
		return fmt.Errorf("%s requires credentials", common.ErrNilPointer)
	}
	if exch == "" {
		return engine.ErrExchangeNameIsEmpty
	}
	c.m.Lock()
	defer c.m.Unlock()
	if base != nil || quote != nil {
		c.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
			Exchange: strings.ToLower(exch),
			Base:     base,
			Quote:    quote,
			Asset:    item,
		}] = creds
		return nil
	}
	c.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
		Exchange: strings.ToLower(exch),
		Asset:    item,
	}] = creds
	return nil
}

func (c *Store) CredentialKeyUpsert(k ExchangeAssetPairCredentials) error {
	if k.Credentials.IsEmpty() {
		return fmt.Errorf("%s requires credentials", common.ErrNilPointer)
	}
	if k.Key.Exchange == "" {
		return engine.ErrExchangeNameIsEmpty
	}
	c.m.Lock()
	defer c.m.Unlock()
	c.ExchangeAssetPairCredentials[k.Key] = k.Credentials
	return nil
}

func (c *Store) ExchangeAssetPairKeyUpsert(k *key.ExchangeAssetPair, creds *account.Credentials) error {
	if creds == nil {
		return fmt.Errorf("%s requires credentials", common.ErrNilPointer)
	}
	if k.Exchange == "" {
		return engine.ErrExchangeNameIsEmpty
	}
	c.m.Lock()
	defer c.m.Unlock()
	c.ExchangeAssetPairCredentials[k] = creds
	return nil
}

// GetExactMatch retrieves credentials for a given ExchangeAssetPair key
func (c *Store) GetCredentialsForKey(k *key.ExchangeAssetPair) (account.Credentials, error) {
	c.m.Lock()
	defer c.m.Unlock()
	k.Exchange = strings.ToLower(k.Exchange)
	creds, ok := c.ExchangeAssetPairCredentials[k]
	if ok {
		return *creds, nil
	}
	return account.Credentials{}, fmt.Errorf("%w %v", ErrNoCredentialsForKey, k)
}

func (c *Store) GetAnyMatch(k *key.ExchangeAssetPair) (account.Credentials, error) {
	c.m.Lock()
	defer c.m.Unlock()
	k.Exchange = strings.ToLower(k.Exchange)
	// assume all fields are provided
	cred, ok := c.ExchangeAssetPairCredentials[k]
	if ok {
		return *cred, nil
	}
	// check if there is a credential for singular currencies
	if k.Base != nil {
		cred, ok = c.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
			Exchange: k.Exchange,
			Asset:    k.Asset,
			Base:     k.Base,
		}]
		if ok {
			return *cred, nil
		}
	}
	if k.Quote != nil {
		cred, ok = c.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
			Exchange: k.Exchange,
			Asset:    k.Asset,
			Quote:    k.Quote,
		}]
		if ok {
			return *cred, nil
		}
	}
	// check if there is a credential with only the exchange and asset
	if k.Asset.IsValid() {
		cred, ok = c.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
			Exchange: k.Exchange,
			Asset:    k.Asset,
		}]
		if ok {
			return *cred, nil
		}
	}
	// check if there is a credential with only the exchange
	cred, ok = c.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
		Exchange: k.Exchange,
	}]
	if ok {
		return *cred, nil
	}
	return account.Credentials{}, fmt.Errorf("%w %v", ErrNoCredentialsForKey, k)
}
