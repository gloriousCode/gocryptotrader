package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
)

// LoadData retrieves data from a GoCryptoTrader exchange wrapper which calls the exchange's API
func LoadData(ctx context.Context, startDate, endDate time.Time, exch exchange.IBotExchange, fPair currency.Pair, paymentCurrency currency.Code, a asset.Item) (*fundingrate.Rates, error) {
	rates, err := exch.GetFundingRates(ctx, &fundingrate.RatesRequest{
		Asset:           a,
		Pair:            fPair,
		PaymentCurrency: paymentCurrency,
		StartDate:       startDate,
		EndDate:         endDate,
	})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve candle data for %v %v %v, %v", exch.GetName(), a, fPair, err)
	}
	rates.Exchange = strings.ToLower(rates.Exchange)
	return rates, nil
}
