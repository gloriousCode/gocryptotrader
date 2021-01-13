package statistics

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func (s *Statistic) Reset() {
	*s = Statistic{}
}

// AddDataEventForTime sets up the big map for to store important data at each time interval
func (s *Statistic) AddDataEventForTime(e common.DataEventHandler) error {
	if e == nil {
		return errors.New("nil data event received")
	}
	ex := e.GetExchange()
	a := e.GetAssetType()
	p := e.Pair()
	if s.ExchangeAssetPairStatistics == nil {
		s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic)
	}
	if s.ExchangeAssetPairStatistics[ex] == nil {
		s.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic)
	}
	if s.ExchangeAssetPairStatistics[ex][a] == nil {
		s.ExchangeAssetPairStatistics[ex][a] = make(map[currency.Pair]*currencystatstics.CurrencyStatistic)
	}
	lookup := s.ExchangeAssetPairStatistics[ex][a][p]
	if lookup == nil {
		lookup = &currencystatstics.CurrencyStatistic{}
	}
	lookup.Events = append(lookup.Events,
		currencystatstics.EventStore{
			DataEvent: e,
		},
	)
	s.ExchangeAssetPairStatistics[ex][a][p] = lookup
	return nil
}

// AddSignalEventForTime adds strategy signal event to the statistics at the time period
func (s *Statistic) AddSignalEventForTime(e signal.SignalEvent) error {
	if e == nil {
		return errors.New("nil signal event received")
	}
	if s.ExchangeAssetPairStatistics == nil {
		return errors.New("ExchangeAssetPairStatistics not setup")
	}
	if s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()] == nil {
		return fmt.Errorf("no data for %v %v %v to set signal event", e.GetExchange(), e.GetAssetType(), e.Pair())
	}
	lookup := s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup.Events[i].SignalEvent = e
			s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
	return nil
}

// AddOrderEventForTime adds order event to the statistics at the time period
func (s *Statistic) AddOrderEventForTime(e order.OrderEvent) error {
	if e == nil {
		return errors.New("nil order event received")
	}
	if s.ExchangeAssetPairStatistics == nil {
		return errors.New("ExchangeAssetPairStatistics not setup")
	}
	if s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()] == nil {
		return fmt.Errorf("no data for %v %v %v to set exchange event", e.GetExchange(), e.GetAssetType(), e.Pair())
	}
	lookup := s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup.Events[i].OrderEvent = e
			s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
	return nil
}

// AddFillEventForTime adds fill event to the statistics at the time period
func (s *Statistic) AddFillEventForTime(e fill.FillEvent) error {
	if e == nil {
		return errors.New("nil fill event received")
	}
	if s.ExchangeAssetPairStatistics == nil {
		return errors.New("ExchangeAssetPairStatistics not setup")
	}
	if s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()] == nil {
		return fmt.Errorf("no data for %v %v %v to set fill event", e.GetExchange(), e.GetAssetType(), e.Pair())
	}
	lookup := s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup.Events[i].FillEvent = e
			s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
	return nil
}

// AddHoldingsForTime adds all holdings to the statistics at the time period
func (s *Statistic) AddHoldingsForTime(h holdings.Holding) error {
	if s.ExchangeAssetPairStatistics == nil {
		return errors.New("ExchangeAssetPairStatistics not setup")
	}
	if s.ExchangeAssetPairStatistics[h.Exchange][h.Asset][h.Pair] == nil {
		return fmt.Errorf("no data for %v %v %v to set holding event", h.Exchange, h.Asset, h.Pair)
	}
	lookup := s.ExchangeAssetPairStatistics[h.Exchange][h.Asset][h.Pair]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(h.Timestamp) {
			lookup.Events[i].Holdings = h
			s.ExchangeAssetPairStatistics[h.Exchange][h.Asset][h.Pair] = lookup
		}
	}
	return nil
}

// AddComplianceSnapshotForTime adds the compliance snapshot to the statistics at the time period
func (s *Statistic) AddComplianceSnapshotForTime(c compliance.Snapshot, e fill.FillEvent) error {
	if e == nil {
		return errors.New("nil fill event received")
	}
	if s.ExchangeAssetPairStatistics == nil {
		return errors.New("ExchangeAssetPairStatistics not setup")
	}
	if s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()] == nil {
		return fmt.Errorf("no data for %v %v %v to set compliance snapshot", e.GetExchange(), e.GetAssetType(), e.Pair())
	}
	lookup := s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(c.Timestamp) {
			lookup.Events[i].Transactions = c
			s.ExchangeAssetPairStatistics[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
	return nil
}

func (s *Statistic) CalculateTheResults() error {
	s.PrintAllEvents()
	currCount := 0
	var finalResults []FinalResultsHolder
	for exchangeName, exchangeMap := range s.ExchangeAssetPairStatistics {
		for assetItem, assetMap := range exchangeMap {
			for pair, stats := range assetMap {
				currCount++
				stats.CalculateResults()
				stats.PrintResults(exchangeName, assetItem, pair)
				last := stats.Events[len(stats.Events)-1]
				stats.FinalHoldings = last.Holdings
				stats.FinalOrders = last.Transactions
				s.AllStats = append(s.AllStats, *stats)

				finalResults = append(finalResults, FinalResultsHolder{
					Exchange:         exchangeName,
					Asset:            assetItem,
					Pair:             pair,
					MaxDrawdown:      stats.DrawDowns.MaxDrawDown,
					MarketMovement:   stats.MarketMovement,
					StrategyMovement: stats.StrategyMovement,
				})
				s.TotalBuyOrders += stats.BuyOrders
				s.TotalSellOrders += stats.SellOrders
			}
		}
	}
	s.TotalOrders = s.TotalBuyOrders + s.TotalSellOrders
	if currCount > 1 {
		s.BiggestDrawdown = s.GetTheBiggestDrawdownAcrossCurrencies(finalResults)
		s.BestMarketMovement = s.GetBestMarketPerformer(finalResults)
		s.BestStrategyResults = s.GetBestStrategyPerformer(finalResults)
		s.PrintTotalResults()
	}
	return nil
}

func (s *Statistic) PrintTotalResults() {
	log.Info(log.BackTester, "------------------Total Results------------------------------")
	log.Info(log.BackTester, "------------------Orders----------------------------------")
	log.Infof(log.BackTester, "Total buy orders: %v", s.TotalBuyOrders)
	log.Infof(log.BackTester, "Total sell orders: %v", s.TotalSellOrders)
	log.Infof(log.BackTester, "Total orders: %v\n\n", s.TotalOrders)

	if s.BiggestDrawdown != nil {
		log.Info(log.BackTester, "------------------Biggest Drawdown------------------------")
		log.Infof(log.BackTester, "Exchange: %v Asset: %v Currency: %v", s.BiggestDrawdown.Exchange, s.BiggestDrawdown.Asset, s.BiggestDrawdown.Pair)
		log.Infof(log.BackTester, "Highest Price: $%.2f", s.BiggestDrawdown.MaxDrawdown.Highest.Price)
		log.Infof(log.BackTester, "Highest Price Time: %v", s.BiggestDrawdown.MaxDrawdown.Highest.Time)
		log.Infof(log.BackTester, "Lowest Price: $%v", s.BiggestDrawdown.MaxDrawdown.Lowest.Price)
		log.Infof(log.BackTester, "Lowest Price Time: %v", s.BiggestDrawdown.MaxDrawdown.Lowest.Time)
		log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", s.BiggestDrawdown.MaxDrawdown.CalculatedDrawDown)
		log.Infof(log.BackTester, "Difference: $%.2f", s.BiggestDrawdown.MaxDrawdown.Highest.Price-s.BiggestDrawdown.MaxDrawdown.Lowest.Price)
		log.Infof(log.BackTester, "Drawdown length: %v\n\n", len(s.BiggestDrawdown.MaxDrawdown.Iterations))
	}
	if s.BestMarketMovement != nil && s.BestStrategyResults != nil {
		log.Info(log.BackTester, "------------------Orders----------------------------------")
		log.Infof(log.BackTester, "Best performing market movement: %v %v %v %v%%", s.BestMarketMovement.Exchange, s.BestMarketMovement.Asset, s.BestMarketMovement.Pair, s.BestMarketMovement.MarketMovement)
		log.Infof(log.BackTester, "Best performing strategy movement: %v %v %v %v%%", s.BestStrategyResults.Exchange, s.BestStrategyResults.Asset, s.BestStrategyResults.Pair, s.BestStrategyResults.StrategyMovement)
	}
}

func (s *Statistic) GetBestMarketPerformer(results []FinalResultsHolder) *FinalResultsHolder {
	result := &FinalResultsHolder{}
	for i := range results {
		if results[i].MarketMovement > result.MarketMovement || result.MarketMovement == 0 {
			result = &results[i]
		}
	}

	return result
}

func (s *Statistic) GetBestStrategyPerformer(results []FinalResultsHolder) *FinalResultsHolder {
	result := &FinalResultsHolder{}
	for i := range results {
		if results[i].StrategyMovement > result.StrategyMovement || result.StrategyMovement == 0 {
			result = &results[i]
		}
	}

	return result
}

func (s *Statistic) GetTheBiggestDrawdownAcrossCurrencies(results []FinalResultsHolder) *FinalResultsHolder {
	result := &FinalResultsHolder{
		MaxDrawdown: currencystatstics.Swing{},
	}
	for i := range results {
		if results[i].MaxDrawdown.CalculatedDrawDown > result.MaxDrawdown.CalculatedDrawDown || result.MaxDrawdown.CalculatedDrawDown == 0 {
			result = &results[i]
		}
	}

	return result
}

func (s *Statistic) PrintAllEvents() {
	log.Info(log.BackTester, "------------------Events-------------------------------------")
	var errs gctcommon.Errors
	for e, x := range s.ExchangeAssetPairStatistics {
		for a, y := range x {
			for p, c := range y {
				for i := range c.Events {
					if c.Events[i].FillEvent != nil {
						direction := c.Events[i].FillEvent.GetDirection()
						if direction == common.CouldNotBuy ||
							direction == common.CouldNotSell ||
							direction == common.DoNothing ||
							direction == common.MissingData ||
							direction == "" {
							log.Infof(log.BackTester, "%v | Price: $%v - Direction: %v - Why: %s",
								c.Events[i].FillEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
								c.Events[i].FillEvent.GetClosePrice(),
								c.Events[i].FillEvent.GetDirection(),
								c.Events[i].FillEvent.GetWhy())
						} else {
							log.Infof(log.BackTester, "%v | Price: $%v - Amount: %v - Fee: $%v - Direction %v - Why: %s",
								c.Events[i].FillEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
								c.Events[i].FillEvent.GetPurchasePrice(),
								c.Events[i].FillEvent.GetAmount(),
								c.Events[i].FillEvent.GetExchangeFee(),
								c.Events[i].FillEvent.GetDirection(),
								c.Events[i].FillEvent.GetWhy(),
							)
						}
					} else if c.Events[i].SignalEvent != nil {
						log.Infof(log.BackTester, "%v | Price: $%v - Why: %v",
							c.Events[i].SignalEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
							c.Events[i].SignalEvent.GetPrice(),
							c.Events[i].SignalEvent.GetWhy())
					} else {
						errs = append(errs, fmt.Errorf("%v %v %v unexpected data received %+v", e, a, p, c.Events[i]))
					}
				}
			}
		}
	}
	if len(errs) > 0 {
		log.Info(log.BackTester, "------------------Errors-------------------------------------")
		for i := range errs {
			log.Info(log.BackTester, errs[i].Error())
		}
	}
}

func (s *Statistic) SetStrategyName(name string) {
	s.StrategyName = name
}

func (s *Statistic) Serialise() (string, error) {
	resp, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return "", err
	}

	return string(resp), nil
}
