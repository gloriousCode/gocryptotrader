package fees

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	ErrNoFeesLoaded = errors.New("no fees loaded")
	ErrNoFeeFound   = errors.New("no fee found")
	ErrInvalidFee   = errors.New("invalid fee configuration")

	errExchangeNameEmpty = errors.New("exchange name is empty")
	errAssetInvalid      = errors.New("asset is invalid")
	errPairNotSet        = errors.New("currency pair not set")
)

type Key struct {
	key.ExchangeAsset
	Credentials accounts.Credentials
}

type store struct {
	eaFees map[Key]*Fee
	mtx    sync.RWMutex
}

var manager = store{
	eaFees: make(map[Key]*Fee),
}

type Fee struct {
	Key       Key
	MakerFee  float64
	TakerFee  float64
	Tier      uint8
	UpdatedAt time.Time
}

// Load loads all limits into private limit holder
func Load(fees []Fee) error {
	return manager.load(fees)
}

// GetFee returns the fee matching the key
func GetFee(k Key) (*Fee, error) {
	return manager.getFee(k)
}

// EstimateFee is a convenience method to estimate fees without retrieving the fee structure first
func EstimateFee(k Key, price, amount float64, isMaker bool) (float64, error) {
	return manager.estimateFee(k, price, amount, isMaker)
}

func (e *store) load(fees []Fee) error {
	if len(fees) == 0 {
		return common.ErrEmptyParams
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if e.eaFees == nil {
		e.eaFees = make(map[Key]*Fee)
	}

	for x := range fees {
		if fees[x].Key.Exchange == "" {
			return fmt.Errorf("cannot load fees for %q: %w", fees[x].Key, errExchangeNameEmpty)
		}
		if !fees[x].Key.Asset.IsValid() {
			return fmt.Errorf("cannot load fees for %q: %w", fees[x].Key, errAssetInvalid)
		}
		if (fees[x].TakerFee == 0 && fees[x].MakerFee == 0) ||
			fees[x].TakerFee < fees[x].MakerFee {
			return fmt.Errorf("%w %q: maker: %f taker: %f",
				ErrInvalidFee,
				fees[x].Key,
				fees[x].MakerFee,
				fees[x].TakerFee)
		}
		fees[x].UpdatedAt = time.Now()
		e.eaFees[fees[x].Key] = &fees[x]
	}
	return nil
}

func (e *store) getFee(k Key) (*Fee, error) {
	e.mtx.RLock()
	defer e.mtx.RUnlock()
	if e.eaFees == nil {
		return nil, ErrNoFeesLoaded
	}
	f, ok := e.eaFees[k]
	if !ok {
		return nil, fmt.Errorf("%w for %s", ErrNoFeeFound, k)
	}
	return f, nil
}

func (e *store) estimateFee(k Key, price, amount float64, isMaker bool) (float64, error) {
	e.mtx.RLock()
	defer e.mtx.RUnlock()
	if e.eaFees == nil {
		return -1, ErrNoFeesLoaded
	}
	m1, ok := e.eaFees[k]
	if !ok {
		return -1, fmt.Errorf("%w for %s", ErrNoFeeFound, k)
	}
	feeRate := m1.TakerFee
	if isMaker {
		feeRate = m1.MakerFee
	}
	return price * amount * feeRate, nil
}

func NewKey(name string, a asset.Item, creds *accounts.Credentials) Key {
	return Key{
		ExchangeAsset: key.ExchangeAsset{Exchange: name, Asset: a},
		Credentials:   *creds,
	}
}

func (f *Key) String() string {
	return fmt.Sprintf("%s-%s-%s", f.Exchange, f.Asset, f.Credentials)
}
