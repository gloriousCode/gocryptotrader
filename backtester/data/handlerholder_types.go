package data

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// HandlerHolder stores an event handler per exchange asset pair
type HandlerHolder struct {
	m    sync.Mutex
	data map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler
}

// Holder interface dictates what a Data holder is expected to do
type Holder interface {
	SetDataForCurrency(string, asset.Item, currency.Pair, Handler) error
	GetAllData() ([]Handler, error)
	GetDataForCurrency(ev common.Event) (Handler, error)
	Reset() error
}
