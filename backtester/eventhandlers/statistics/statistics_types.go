package statistics

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Statistic
type Statistic struct {
	StrategyName                string                                                                           `json:"strategy-name"`
	ExchangeAssetPairStatistics map[string]map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic `json:"-"`
	RiskFreeRate                float64                                                                          `json:"risk-free-rate"`
	TotalBuyOrders              int64                                                                            `json:"total-buy-orders"`
	TotalSellOrders             int64                                                                            `json:"total-sell-orders"`
	TotalOrders                 int64                                                                            `json:"total-orders"`
	BiggestDrawdown             *FinalResultsHolder                                                              `json:"biggest-drawdown,omitempty"`
	BestStrategyResults         *FinalResultsHolder                                                              `json:"best-start-results,omitempty"`
	BestMarketMovement          *FinalResultsHolder                                                              `json:"best-market-movement,omitempty"`
	AllStats                    []currencystatstics.CurrencyStatistic                                            `json:"results"` // as ExchangeAssetPairStatistics cannot be rendered via json.Marshall, we append all result to this slice instead
}

type FinalResultsHolder struct {
	Exchange         string                  `json:"exchange"`
	Asset            asset.Item              `json:"asset"`
	Pair             currency.Pair           `json:"currency"`
	MaxDrawdown      currencystatstics.Swing `json:"max-drawdown"`
	MarketMovement   float64                 `json:"market-movement"`
	StrategyMovement float64                 `json:"strategy-movement"`
}

// Handler interface handles
type Handler interface {
	SetStrategyName(string)
	AddDataEventForTime(common.DataEventHandler) error
	AddSignalEventForTime(signal.Event) error
	AddOrderEventForTime(order.Event) error
	AddFillEventForTime(fill.Event) error
	AddHoldingsForTime(holdings.Holding) error
	AddComplianceSnapshotForTime(compliance.Snapshot, fill.Event) error
	CalculateTheResults() error
	Reset()
	Serialise() (string, error)
}

type Results struct {
	Pair              string               `json:"pair"`
	TotalEvents       int                  `json:"totalEvents"`
	TotalTransactions int                  `json:"totalTransactions"`
	Events            []ResultEvent        `json:"events"`
	Transactions      []ResultTransactions `json:"transactions"`
	SharpieRatio      float64              `json:"sharpieRatio"`
	StrategyName      string               `json:"strategyName"`
}

type ResultTransactions struct {
	Time      time.Time     `json:"time"`
	Direction gctorder.Side `json:"direction"`
	Price     float64       `json:"price"`
	Amount    float64       `json:"amount"`
	Why       string        `json:"why,omitempty"`
}

type ResultEvent struct {
	Time time.Time `json:"time"`
}
