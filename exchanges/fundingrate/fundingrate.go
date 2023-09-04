package fundingrate

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func init() {
	service = newService()
}

func newService() *Service {
	return &Service{
		FundingRates: make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*LatestRateWithDispatchIDs),
		Exchange:     make(map[string]uuid.UUID),
		mux:          dispatch.GetNewMux(nil),
	}
}

// SubscribeFundingRate subscribes to a fundingRate and returns a communication channel to
// stream new fundingRate updates
func SubscribeFundingRate(exchange string, p currency.Pair, a asset.Item) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	return service.subscribeFundingRate(exchange, p, a)
}

func (s *Service) subscribeFundingRate(exchange string, p currency.Pair, a asset.Item) (dispatch.Pipe, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tick, ok := s.FundingRates[exchange][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%w for %s %s %s",
			errFundingRateNotFound,
			exchange,
			p,
			a)
	}
	return s.mux.Subscribe(tick.Main)
}

// SubscribeToExchangeFundingRates subscribes to all fundingRates on an exchange
func SubscribeToExchangeFundingRates(exchange string) (dispatch.Pipe, error) {
	return service.subscribeToExchangeFundingRates(exchange)
}

func (s *Service) subscribeToExchangeFundingRates(exchange string) (dispatch.Pipe, error) {
	if exchange == "" {
		return dispatch.Pipe{}, ErrExchangeNameUnset
	}
	exchange = strings.ToLower(exchange)
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.Exchange[exchange]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%s exchange fundingRates not found",
			exchange)
	}

	return s.mux.Subscribe(id)
}

// GetFundingRate checks and returns a requested fundingRate if it exists
func GetFundingRate(exchange string, p currency.Pair, a asset.Item) (*LatestRateResponse, error) {
	exchange = strings.ToLower(exchange)
	return service.getFundingRate(exchange, p, a)
}

func (s *Service) getFundingRate(exchange string, p currency.Pair, a asset.Item) (*LatestRateResponse, error) {
	if exchange == "" {
		return nil, ErrExchangeNameUnset
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m1, ok := s.FundingRates[exchange]
	if !ok {
		return nil, fmt.Errorf("%w for %s exchange", errFundingRateNotFound, exchange)
	}

	m2, ok := m1[p.Base.Item]
	if !ok {
		return nil, fmt.Errorf("%w for base currency %s for %s %s %s",
			errFundingRateNotFound, p.Base, exchange, a, p)
	}

	m3, ok := m2[p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("%w for quote currency %s for %s %s %s",
			errFundingRateNotFound, p.Quote, exchange, a, p)
	}

	t, ok := m3[a]
	if !ok {
		return nil, fmt.Errorf("%w forasset type %s",
			errFundingRateNotFound, a)
	}

	cpy := t.LatestRateResponse // Don't let external functions have access to underlying
	return &cpy, nil
}

// ProcessFundingRate processes incoming fundingRates, creating or updating the FundingRates
// list
func ProcessFundingRate(p *LatestRateResponse) error {
	return service.processFundingRate(p)
}

// processFundingRate updates fundingRate price
func (s *Service) processFundingRate(p *LatestRateResponse) error {
	if p == nil {
		return fmt.Errorf("%w LatestRateResponse", common.ErrNilPointer)
	}

	if p.Exchange == "" {
		return ErrExchangeNameUnset
	}

	if p.Pair.IsEmpty() {
		return fmt.Errorf("%s %w", p.Exchange, errPairNotSet)
	}

	if p.LatestRate.Time.IsZero() {
		return fmt.Errorf("%s %s %w",
			p.Exchange,
			p.Pair,
			errFundingRateTimeUnset)
	}

	if p.Asset == asset.Empty {
		return fmt.Errorf("%s %s %w",
			p.Exchange,
			p.Pair,
			errAssetTypeNotSet)
	}
	if !p.Asset.IsFutures() {
		return fmt.Errorf("%s %s %w not futures",
			p.Exchange,
			p.Pair,
			asset.ErrNotSupported)
	}
	name := strings.ToLower(p.Exchange)
	s.mu.Lock()

	m1, ok := s.FundingRates[name]
	if !ok {
		m1 = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*LatestRateWithDispatchIDs)
		s.FundingRates[name] = m1
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
	if p.TimeChecked.IsZero() {
		// used to help a trader know when the funding rate was last checked
		p.TimeChecked = time.Now()
	}
	t.LatestRateResponse = *p
	//nolint: gocritic // combining lists into a new one isn't a crime
	ids := append(t.Assoc, t.Main)
	s.mu.Unlock()
	s.alerter.Alert()
	return s.mux.Publish(p, ids...)
}

// GetAllFundingRates returns all stored funding rates
func GetAllFundingRates() []LatestRateResponse {
	return service.getAllRates()
}

// ReturnAllRatesOnUpdate helper func which returns all rates on an update
func ReturnAllRatesOnUpdate(timeout time.Duration) []LatestRateResponse {
	if !service.waitForUpdate(timeout) {
		return nil
	}
	return service.getAllRates()
}

// waitForUpdate allows a caller to wait for any fundingRate processFundingRate
func (s *Service) waitForUpdate(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	ch := make(chan struct{})
	go func(timer *time.Timer, ch chan struct{}) {
		select {
		case <-timer.C:
			close(ch)
		}
	}(timer, ch)
	return <-s.alerter.Wait(ch)
}

// getAllRates returns all fundingRates
func (s *Service) getAllRates() []LatestRateResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rates []LatestRateResponse
	for _, exchangeMap := range s.FundingRates {
		for _, baseMap := range exchangeMap {
			for _, quoteMap := range baseMap {
				for _, rate := range quoteMap {
					rates = append(rates, rate.LatestRateResponse)
				}
			}
		}
	}
	return rates
}

// GetFundingRatesForExchange returns all fundingRates for a given exchange
func GetFundingRatesForExchange(exch string) ([]LatestRateResponse, error) {
	return service.getAllRatesForExchange(exch)
}

// getAllRates returns all fundingRates
func (s *Service) getAllRatesForExchange(exch string) ([]LatestRateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rates []LatestRateResponse
	exch = strings.ToLower(exch)
	exchangeRates, ok := s.FundingRates[exch]
	if !ok {
		return nil, fmt.Errorf("%w for %s exchange", errFundingRateNotFound, exch)
	}
	for _, baseMap := range exchangeRates {
		for _, quoteMap := range baseMap {
			for _, rate := range quoteMap {
				rates = append(rates, rate.LatestRateResponse)
			}
		}
	}
	return rates, nil
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

// getAssociations links a singular book with its dispatch associations
func (s *Service) getAssociations(exch string) ([]uuid.UUID, error) {
	if exch == "" {
		return nil, ErrExchangeNameUnset
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
