package credentialstore

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// credStoreService is a global variable of type Store to manage credentials
var credStoreService Store

// init initializes the global Credentials variable with an empty Store
func init() {
	credStoreService = *NewStore()
}

// GetAll returns a map of all stored credentials
func GetAll() map[*key.ExchangeAssetPair]account.Credentials {
	credStoreService.m.Lock()
	defer credStoreService.m.Unlock()
	creds := make(map[*key.ExchangeAssetPair]account.Credentials)
	for k, v := range credStoreService.ExchangeAssetPairCredentials {
		creds[k] = *v
	}
	return creds
}

// Upsert is a convenience method to store credentials based on provided parameters
func Upsert(exch string, item asset.Item, base, quote *currency.Item, creds *account.Credentials) error {
	if creds == nil {
		return fmt.Errorf("%s requires credentials", common.ErrNilPointer)
	}
	if exch == "" {
		return engine.ErrExchangeNameIsEmpty
	}
	credStoreService.m.Lock()
	defer credStoreService.m.Unlock()
	if base != nil || quote != nil {
		credStoreService.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
			Exchange: exch,
			Base:     base,
			Quote:    quote,
			Asset:    item,
		}] = creds
		return nil
	}
	credStoreService.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
		Exchange: exch,
		Asset:    item,
	}] = creds
	return nil
}

func CredentialKeyUpsert(k ExchangeAssetPairCredentials) error {
	if k.Credentials.IsEmpty() {
		return fmt.Errorf("%s requires credentials", common.ErrNilPointer)
	}
	if k.Key.Exchange == "" {
		return engine.ErrExchangeNameIsEmpty
	}
	credStoreService.m.Lock()
	defer credStoreService.m.Unlock()
	credStoreService.ExchangeAssetPairCredentials[k.Key] = k.Credentials
	return nil
}

func ExchangeAssetPairKeyUpsert(k *key.ExchangeAssetPair, creds *account.Credentials) error {
	if creds == nil {
		return fmt.Errorf("%s requires credentials", common.ErrNilPointer)
	}
	if k.Exchange == "" {
		return engine.ErrExchangeNameIsEmpty
	}
	credStoreService.m.Lock()
	defer credStoreService.m.Unlock()
	credStoreService.ExchangeAssetPairCredentials[k] = creds
	return nil
}

// GetExactMatch retrieves credentials for a given ExchangeAssetPair key
func GetExactMatch(k *key.ExchangeAssetPair) (*account.Credentials, error) {
	credStoreService.m.Lock()
	defer credStoreService.m.Unlock()
	k.Exchange = strings.ToLower(k.Exchange)
	creds, ok := credStoreService.ExchangeAssetPairCredentials[k]
	if ok {
		return creds, nil
	}
	return nil, fmt.Errorf("%w %v", ErrNoCredentialsForKey, k)
}

// UpsertCredentialsFromConfig stores credentials from a provided Config struct
func UpsertCredentialsFromConfig(data *Config) error {
	for _, v := range data.ExchangeAssetPairCredentials {
		if !v.Enabled {
			continue
		}
		if err := credStoreService.Upsert(v.Key.Exchange, v.Key.Asset, v.Key.Base, v.Key.Quote, v.Credentials); err != nil {
			return err
		}
	}
	return nil
}

func GetExchangeCredentials(exch string) (*account.Credentials, error) {
	credStoreService.m.Lock()
	defer credStoreService.m.Unlock()

	cred, ok := credStoreService.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{Exchange: exch}]
	if ok {
		return cred, nil
	}
	return nil, fmt.Errorf("%w for exchange %s", ErrNoCredentialsForKey, exch)
}

func GetExchangeAssetCredentials(exch string, item asset.Item) (*account.Credentials, error) {
	credStoreService.m.Lock()
	defer credStoreService.m.Unlock()

	cred, ok := credStoreService.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{Exchange: exch, Asset: item}]
	if ok {
		return cred, nil
	}
	return nil, fmt.Errorf("%w for exchange %s", ErrNoCredentialsForKey, exch)
}

func GetAnyMatch(k *key.ExchangeAssetPair) (*account.Credentials, error) {
	credStoreService.m.Lock()
	defer credStoreService.m.Unlock()
	// assume all fields are provided
	cred, ok := credStoreService.ExchangeAssetPairCredentials[k]
	if ok {
		return cred, nil
	}
	// check if there is a credential for singular currencies
	if k.Base != nil {
		cred, ok = credStoreService.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
			Exchange: k.Exchange,
			Asset:    k.Asset,
			Base:     k.Base,
		}]
		if ok {
			return cred, nil
		}
	}
	if k.Quote != nil {
		cred, ok = credStoreService.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
			Exchange: k.Exchange,
			Asset:    k.Asset,
			Quote:    k.Quote,
		}]
		if ok {
			return cred, nil
		}
	}
	cred, ok = credStoreService.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
		Exchange: k.Exchange,
		Asset:    k.Asset,
	}]
	if ok {
		return cred, nil
	}
	// check if there is a credential with only the exchange
	cred, ok = credStoreService.ExchangeAssetPairCredentials[&key.ExchangeAssetPair{
		Exchange: k.Exchange,
	}]
	if ok {
		return cred, nil
	}
	return nil, fmt.Errorf("%w %v", ErrNoCredentialsForKey, k)
}
