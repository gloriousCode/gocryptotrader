package fundingrate

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var ()

func init() {
	service = new(Service)
	service.FundingRates = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*LatestRateWithDispatchIDs)
	service.Exchange = make(map[string]uuid.UUID)
	service.mux = dispatch.GetNewMux(nil)
}

// SubscribeFundingRate subscribes to a fundingRate and returns a communication channel to
// stream new fundingRate updates
func SubscribeFundingRate(exchange string, p currency.Pair, a asset.Item) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.mu.Lock()
	defer service.mu.Unlock()

	tick, ok := service.FundingRates[exchange][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("fundingRate item not found for %s %s %s",
			exchange,
			p,
			a)
	}
	return service.mux.Subscribe(tick.Main)
}

// SubscribeToExchangeFundingRates subscribes to all fundingRates on an exchange
func SubscribeToExchangeFundingRates(exchange string) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.mu.Lock()
	defer service.mu.Unlock()
	id, ok := service.Exchange[exchange]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%s exchange fundingRates not found",
			exchange)
	}

	return service.mux.Subscribe(id)
}

// GetFundingRate checks and returns a requested fundingRate if it exists
func GetFundingRate(exchange string, p currency.Pair, a asset.Item) (*LatestRateResponse, error) {
	if exchange == "" {
		return nil, errExchangeNameIsEmpty
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	exchange = strings.ToLower(exchange)
	service.mu.Lock()
	defer service.mu.Unlock()
	m1, ok := service.FundingRates[exchange]
	if !ok {
		return nil, fmt.Errorf("no fundingRates for %s exchange", exchange)
	}

	m2, ok := m1[p.Base.Item]
	if !ok {
		return nil, fmt.Errorf("no fundingRates associated with base currency %s for %s %s %s",
			p.Base, exchange, a, p)
	}

	m3, ok := m2[p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("no fundingRates associated with quote currency %s for %s %s %s",
			p.Quote, exchange, a, p)
	}

	t, ok := m3[a]
	if !ok {
		return nil, fmt.Errorf("no fundingRates associated with asset type %s",
			a)
	}

	cpy := t.LatestRateResponse // Don't let external functions have access to underlying
	return &cpy, nil
}

// ProcessFundingRate processes incoming fundingRates, creating or updating the FundingRates
// list
func ProcessFundingRate(p *LatestRateResponse) error {
	if p == nil {
		return fmt.Errorf("%w LatestRateResponse", common.ErrNilPointer)
	}

	if p.Exchange == "" {
		return ErrExchangeNameUnset
	}

	if p.Pair.IsEmpty() {
		return fmt.Errorf("%s %s", p.Exchange, errPairNotSet)
	}

	if p.LatestRate.Time.IsZero() {
		return fmt.Errorf("%s %s %w",
			p.Exchange,
			p.Pair,
			errFundingRateTimeUnset)
	}

	if p.Asset == asset.Empty {
		return fmt.Errorf("%s %s %s",
			p.Exchange,
			p.Pair,
			errAssetTypeNotSet)
	}

	return service.update(p)
}

// update updates fundingRate price
func (s *Service) update(p *LatestRateResponse) error {
	name := strings.ToLower(p.Exchange)
	s.mu.Lock()

	m1, ok := service.FundingRates[name]
	if !ok {
		m1 = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*LatestRateWithDispatchIDs)
		service.FundingRates[name] = m1
	}

	m2, ok := m1[p.Pair.Base.Item]
	if !ok {
		m2 = make(map[*currency.Item]map[asset.Item]*LatestRateWithDispatchIDs)
		m1[p.Pair.Base.Item] = m2
	}

	m3, ok := m2[p.Pair.Quote.Item]
	if !ok {
		m3 = make(map[asset.Item]*LatestRateWithDispatchIDs)
		m2[p.Pair.Quote.Item] = m3
	}

	t, ok := m3[p.Asset]
	if !ok || t == nil {
		newFundingRate := &LatestRateWithDispatchIDs{}
		err := s.setItemID(newFundingRate, p, name)
		if err != nil {
			s.mu.Unlock()
			return err
		}
		m3[p.Asset] = newFundingRate
		s.mu.Unlock()
		return nil
	}

	t.LatestRateResponse = *p
	//nolint: gocritic
	ids := append(t.Assoc, t.Main)
	s.mu.Unlock()
	return s.mux.Publish(p, ids...)
}

// setItemID retrieves and sets dispatch mux publish IDs
func (s *Service) setItemID(t *LatestRateWithDispatchIDs, p *LatestRateResponse, exch string) error {
	ids, err := s.getAssociations(exch)
	if err != nil {
		return err
	}
	singleID, err := s.mux.GetID()
	if err != nil {
		return err
	}

	t.LatestRateResponse = *p
	t.Main = singleID
	t.Assoc = ids
	return nil
}

// getAssociations links a singular book with it's dispatch associations
func (s *Service) getAssociations(exch string) ([]uuid.UUID, error) {
	if exch == "" {
		return nil, errExchangeNameIsEmpty
	}
	var ids []uuid.UUID
	exchangeID, ok := s.Exchange[exch]
	if !ok {
		var err error
		exchangeID, err = s.mux.GetID()
		if err != nil {
			return nil, err
		}
		s.Exchange[exch] = exchangeID
	}
	ids = append(ids, exchangeID)
	return ids, nil
}
