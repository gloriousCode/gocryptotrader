package withdrawmanager

import (
	"errors"
	"fmt"
	"time"

	dbwithdraw "github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

// Setup creates a new withdraw manager
func Setup(em iExchangeManager, pm iPortfolioManager, isDryRun bool) (*Manager, error) {
	if em == nil {
		return nil, errors.New("nil manager")
	}
	return &Manager{
		exchangeManager:  em,
		portfolioManager: pm,
		isDryRun:         isDryRun,
	}, nil
}

// SubmitWithdrawal performs validation and submits a new withdraw request to
// exchange
func (m *Manager) SubmitWithdrawal(req *withdraw.Request) (*withdraw.Response, error) {
	if m == nil {
		return nil, subsystems.ErrNilSubsystem
	}
	if req == nil {
		return nil, withdraw.ErrRequestCannotBeNil
	}

	exch := m.exchangeManager.GetExchangeByName(req.Exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	resp := &withdraw.Response{
		Exchange: withdraw.ExchangeResponse{
			Name: req.Exchange,
		},
		RequestDetails: *req,
	}

	var err error
	if m.isDryRun {
		log.Warnln(log.Global, "Dry run enabled, no withdrawal request will be submitted or have an event created")
		resp.ID = withdraw.DryRunID
		resp.Exchange.Status = "dryrun"
		resp.Exchange.ID = withdraw.DryRunID.String()
	} else {
		var ret *withdraw.ExchangeResponse
		if req.Type == withdraw.Crypto {
			if !m.portfolioManager.IsWhiteListed(req.Crypto.Address) {
				return nil, withdraw.ErrStrAddressNotWhiteListed
			}
			if !m.portfolioManager.IsExchangeSupported(req.Exchange, req.Crypto.Address) {
				return nil, withdraw.ErrStrExchangeNotSupportedByAddress
			}
		}
		err = req.Validate()
		if err != nil {
			return nil, err
		}
		if req.Type == withdraw.Fiat {
			ret, err = exch.WithdrawFiatFunds(req)
			if err != nil {
				resp.Exchange.Status = err.Error()
			} else {
				resp.Exchange.Status = ret.Status
				resp.Exchange.ID = ret.ID
			}
		} else if req.Type == withdraw.Crypto {
			ret, err = exch.WithdrawCryptocurrencyFunds(req)
			if err != nil {
				resp.Exchange.Status = err.Error()
			} else {
				resp.Exchange.Status = ret.Status
				resp.Exchange.ID = ret.ID
			}
		}
	}
	if err == nil {
		withdraw.Cache.Add(resp.ID, resp)
	}
	dbwithdraw.Event(resp)
	return resp, err
}

// WithdrawalEventByID returns a withdrawal request by ID
func (m *Manager) WithdrawalEventByID(id string) (*withdraw.Response, error) {
	if m == nil {
		return nil, subsystems.ErrNilSubsystem
	}
	v := withdraw.Cache.Get(id)
	if v != nil {
		return v.(*withdraw.Response), nil
	}

	l, err := dbwithdraw.GetEventByUUID(id)
	if err != nil {
		return nil, fmt.Errorf("%w %v", ErrWithdrawRequestNotFound, id)
	}
	withdraw.Cache.Add(id, l)
	return l, nil
}

// WithdrawalEventByExchange returns a withdrawal request by ID
func (m *Manager) WithdrawalEventByExchange(exchange string, limit int) ([]*withdraw.Response, error) {
	if m == nil {
		return nil, subsystems.ErrNilSubsystem
	}
	exch := m.exchangeManager.GetExchangeByName(exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	return dbwithdraw.GetEventsByExchange(exchange, limit)
}

// WithdrawEventByDate returns a withdrawal request by ID
func (m *Manager) WithdrawEventByDate(exchange string, start, end time.Time, limit int) ([]*withdraw.Response, error) {
	if m == nil {
		return nil, subsystems.ErrNilSubsystem
	}
	exch := m.exchangeManager.GetExchangeByName(exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	return dbwithdraw.GetEventsByDate(exchange, start, end, limit)
}

// WithdrawalEventByExchangeID returns a withdrawal request by Exchange ID
func (m *Manager) WithdrawalEventByExchangeID(exchange, id string) (*withdraw.Response, error) {
	if m == nil {
		return nil, subsystems.ErrNilSubsystem
	}
	exch := m.exchangeManager.GetExchangeByName(exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	return dbwithdraw.GetEventByExchangeID(exchange, id)
}
