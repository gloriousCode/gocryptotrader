package dollarcostaverage

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name        = "dollarcostaverage"
	description = `Dollar-cost averaging (DCA) is an investment strategy in which an investor divides up the total amount to be invested across periodic purchases of a target asset in an effort to reduce the impact of volatility on the overall purchase. The purchases occur regardless of the asset's price and at regular intervals. In effect, this strategy removes much of the detailed work of attempting to time the market in order to make purchases of equities at the best prices.`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
}

// Name returns the name
func (s *Strategy) Name() string {
	return Name
}

// Description provides a nice overview of the strategy
// be it definition of terms or to highlight its purpose
func (s *Strategy) Description() string {
	return description
}

// Execute analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
// For dollarcostaverage, the strategy is always "buy", so it uses the OnSignal function
func (s *Strategy) Execute(d []data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	var resp []signal.Event
	var errs error
	for i := range d {
		if d[i] == nil {
			return nil, common.ErrNilEvent
		}
		es, err := s.GetBaseData(d[i])
		if err != nil {
			return nil, err
		}

		latest, err := d[i].Latest()
		if err != nil {
			return nil, err
		}
		hasDataAtTime, err := d[i].HasDataAtTime(latest.GetTime())
		if err != nil {
			return nil, err
		}
		if !hasDataAtTime {
			es.SetDirection(order.MissingData)
			es.AppendReasonf("missing data at %v, cannot perform any actions", latest.GetTime())
			continue
		}

		es.SetPrice(latest.GetClosePrice())
		es.SetDirection(order.Buy)
		es.AppendReason("DCA purchases on every iteration")
		if err != nil {
			errs = gctcommon.AppendError(errs, err)
		} else {
			resp = append(resp, &es)
		}
	}
	return resp, errs
}

// SetCustomSettings not required for DCA
func (s *Strategy) SetCustomSettings(_ map[string]interface{}) error {
	return base.ErrCustomSettingsUnsupported
}

// SetDefaults not required for DCA
func (s *Strategy) SetDefaults() {}
