package data

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// NewHandlerHolder returns a new HandlerHolder
func NewHandlerHolder() *HandlerHolder {
	return &HandlerHolder{
		data: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler),
	}
}

// SetDataForCurrency assigns a Data Handler to the Data map by exchange, asset and currency
func (h *HandlerHolder) SetDataForCurrency(e string, a asset.Item, p currency.Pair, k Handler) error {
	if h == nil {
		return fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	h.m.Lock()
	defer h.m.Unlock()
	if h.data == nil {
		h.data = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
	}
	e = strings.ToLower(e)
	m1, ok := h.data[e]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
		h.data[e] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]Handler)
		m1[a] = m2
	}

	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]Handler)
		m2[p.Base.Item] = m3
	}

	m3[p.Quote.Item] = k
	return nil
}

// GetAllData returns all set Data in the Data map
func (h *HandlerHolder) GetAllData() ([]Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	h.m.Lock()
	defer h.m.Unlock()
	var resp []Handler
	for _, exchMap := range h.data {
		for _, assetMap := range exchMap {
			for _, baseMap := range assetMap {
				for _, handler := range baseMap {
					resp = append(resp, handler)
				}
			}
		}
	}
	return resp, nil
}

// GetDataForCurrency returns the Handler for a specific exchange, asset, currency
func (h *HandlerHolder) GetDataForCurrency(ev common.Event) (Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	h.m.Lock()
	defer h.m.Unlock()
	exch := ev.GetExchange()
	a := ev.GetAssetType()
	p := ev.Pair()
	handler, ok := h.data[exch][a][p.Base.Item][p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w", exch, a, p, ErrHandlerNotFound)
	}
	return handler, nil
}

// Reset returns the struct to defaults
func (h *HandlerHolder) Reset() error {
	if h == nil {
		return gctcommon.ErrNilPointer
	}
	h.m.Lock()
	defer h.m.Unlock()
	h.data = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
	return nil
}
