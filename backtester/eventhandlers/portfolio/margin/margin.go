package margin

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

type MarginCalculator struct {
	exch     exchange.IBotExchange
	currency currency.Code
}

func SetupMarginCalculator(exch exchange.IBotExchange, code currency.Code) (*MarginCalculator, error) {
	if exch == nil {
		return nil, common.ErrNilPointer
	}
	return &MarginCalculator{
		exch:     exch,
		currency: code,
	}, nil
}

func (m *MarginCalculator) CalculateMaxLeverage(accountLeverage decimal.Decimal) (decimal.Decimal, error) {
	requirements, err := m.exch.GetMarginRequirementsForCurrency(m.currency)
	if err != nil {
		return decimal.Zero, err
	}
	return requirements.MaxLeverage, nil
}

func (m *MarginCalculator) CalculateInitialMargin(collateral, orderAmount decimal.Decimal) error {
	requirements, err := m.exch.GetMarginRequirementsForCurrency(m.currency)
	if err != nil {
		return err
	}
	something := collateral.Mul(requirements.InitialMargin)
	return nil
}

func (m *MarginCalculator) CheckMargin(collateral, orderAmount, unrealisedPNL decimal.Decimal) error {
	requirements, err := m.exch.GetMarginRequirementsForCurrency(m.currency)
	if err != nil {
		return err
	}

	return nil
}
