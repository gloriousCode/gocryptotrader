package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func (c *contractComparer) SortComparersByExpiry() {
	sort.Slice(c.Comparers, func(i, j int) bool {
		return c.Comparers[i].FuturesContract.EndDate.Before(c.Comparers[j].FuturesContract.EndDate)
	})
}

func (c *contractComparer) SortComparersByAnnualRateOfReturn() {
	sort.Slice(c.Comparers, func(i, j int) bool {
		return c.Comparers[i].AnnualisedRateOfReturn > c.Comparers[j].AnnualisedRateOfReturn
	})
}

type allOverComparer []contractComparer

func (a allOverComparer) SortByComparerAnnualRateOfReturn() {
	sort.Slice(a, func(i, j int) bool {
		return a[i].Comparers[0].AnnualisedRateOfReturn > a[j].Comparers[0].AnnualisedRateOfReturn
	})
}

var (
	binanceOnly = false
	btcUSDOnly  = true
	ignorePerps = true
	formatting  = currency.PairFormat{
		Uppercase: true,
		Delimiter: "-",
	}
	m sync.Mutex
)

func main() {
	defaultLogSettings := log.GenDefaultSettings()

	err := log.SetGlobalLogConfig(defaultLogSettings)
	if err != nil {
		panic(err)

	}
	err = log.SetupGlobalLogger("lol", false)
	if err != nil {
		panic(err)
	}
	exchangeManager := engine.NewExchangeManager()
	wg := &sync.WaitGroup{}
	setupExchanges(wg, exchangeManager)
	sm, err := engine.SetupSyncManager(&config.SyncManagerConfig{
		SynchronizeTicker:       true,
		SynchronizeContinuously: true,
		FiatDisplayCurrency:     currency.USD,
		PairFormatDisplay:       &formatting,
		Verbose:                 true,
	}, exchangeManager, &config.RemoteControlConfig{}, true)
	if err != nil {
		panic(err)
	}
	om, err := engine.SetupOrderManager(exchangeManager, &Butts{}, wg, false, false, 0)
	if err != nil {
		panic(err)
	}
	wrm, err := engine.SetupWebsocketRoutineManager(exchangeManager, om, sm, &currency.Config{
		CurrencyPairFormat:  &formatting,
		FiatDisplayCurrency: currency.USD,
	}, false)

	exchanges, err := exchangeManager.GetExchanges()
	if err != nil {
		fmt.Println(err)
		return
	}
	comparableCurrencyToContracts, err := loadAndCategoriseFuturesContracts(exchanges, wg, formatting)
	if err != nil {
		fmt.Println(err)
		return
	}
	//outputLikeContracts(comparableCurrencyToContracts)
	// now check for any of the good ones with SPOT crossovers
	comparableCurrencyToContracts = disableIrrelevantSpotPairs(exchanges, wg, formatting, comparableCurrencyToContracts)

	err = sm.Start()
	if err != nil {
		log.Errorln(log.SyncMgr, err)
	}
	err = wrm.Start()
	if err != nil {
		panic(err)
	}

	err = sm.WaitForInitialSync()
	if err != nil {
		log.Errorln(log.SyncMgr, err)
	}

	var spotComparers allOverComparer
	var futuresComparers allOverComparer
	uniqueHelperForDumbIdiotsLikeMe := make(map[string]bool)

	for k, v := range comparableCurrencyToContracts {
		var futuresOnly []PairDetails
		var spotComparer contractComparer
		for k2, v2 := range v.ExchangeAssetTicker {
			var cp currency.Pair
			a := asset.Spot
			if v2.FuturesContract == nil {
				cp = v2.SpotPair
			} else {
				cp = v2.FuturesContract.Name
				a = v2.FuturesContract.Asset

			}
			tick, err := ticker.GetTicker(v2.Exchange.GetName(), cp, a)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if tick.Last == 0 && tick.Close == 0 && tick.Bid == 0 && tick.Ask == 0 {
				b := v2.Exchange.GetBase()
				err = b.CurrencyPairs.DisablePair(a, cp)
				if err != nil {
					log.Errorln(log.ExchangeSys, err)
				}
				continue
			}
			v2.LastPrice = tick.Last
			v2.Volume = tick.Volume
			v2.QuoteVolume = tick.QuoteVolume
			v2.Close = tick.Close
			v2.Bid = tick.Bid
			v2.Ask = tick.Ask
			v2.Asset = a
			if a == asset.Spot {
				spotComparer.Main = v2
			} else {
				if v2.LastPrice != 0 {
					futuresOnly = append(futuresOnly, *v2)
				}
			}
			comparableCurrencyToContracts[k].ExchangeAssetTicker[k2] = v2
		}
		for i := range futuresOnly {
			spotComparer.Comparers = append(spotComparer.Comparers, futuresOnly[i])
		}
		spotComparers = append(spotComparers, spotComparer)

		for r := range futuresOnly {
			if _, ok := uniqueHelperForDumbIdiotsLikeMe[futuresOnly[r].GetUniqueMapKey()]; ok {
				continue
			} else {
				uniqueHelperForDumbIdiotsLikeMe[futuresOnly[r].GetUniqueMapKey()] = true
			}
			addition := contractComparer{
				Main: &futuresOnly[r],
			}
			var futuresCompare []PairDetails
			if r != 0 {
				futuresCompare = append(futuresCompare, futuresOnly[0:r]...)
			}
			if len(futuresOnly) > r+1 {
				futuresCompare = append(futuresCompare, futuresOnly[r+1:]...)
			}
			for i := range futuresCompare {
				addition.Comparers = append(addition.Comparers, futuresCompare[i])
			}
			futuresComparers = append(futuresComparers, addition)
		}
	}

	for i := range futuresComparers {
		futuresComparers[i].SortComparersByExpiry()
	}

	for i := range futuresComparers {
		for j := range futuresComparers[i].Comparers {
			if futuresComparers[i].Comparers[j].LastPrice == 0 || futuresComparers[i].Main.LastPrice == 0 {
				continue
			}
			futuresComparers[i].Comparers[j].ComparisonToContract = gctmath.CalculatePercentageDifference(futuresComparers[i].Comparers[j].LastPrice, futuresComparers[i].Main.LastPrice)
			futuresComparers[i].Comparers[j].AnnualisedRateOfReturn = calculateAnnualisedRateOfReturn(
				futuresComparers[i].Main.LastPrice,
				futuresComparers[i].Comparers[j].LastPrice,
				futuresComparers[i].Comparers[j].FuturesContract.EndDate.Sub(time.Now()))
		}
	}

	for i := range spotComparers {
		for j := range spotComparers[i].Comparers {
			if spotComparers[i].Comparers[j].LastPrice == 0 || spotComparers[i].Main.LastPrice == 0 {
				continue
			}
			spotComparers[i].Comparers[j].ComparisonToContract = gctmath.CalculatePercentageDifference(spotComparers[i].Comparers[j].LastPrice, spotComparers[i].Main.LastPrice)
			spotComparers[i].Comparers[j].AnnualisedRateOfReturn = calculateAnnualisedRateOfReturn(
				spotComparers[i].Main.LastPrice,
				spotComparers[i].Comparers[j].LastPrice,
				spotComparers[i].Comparers[j].FuturesContract.EndDate.Sub(time.Now()))
		}
	}

	for i := range futuresComparers {
		futuresComparers[i].SortComparersByExpiry()
	}
	for i := range spotComparers {
		spotComparers[i].SortComparersByExpiry()
	}

	futuresComparers.SortByComparerAnnualRateOfReturn()
	spotComparers.SortByComparerAnnualRateOfReturn()

	var spotVersusContracts []spotPairs
	for _, v := range comparableCurrencyToContracts {
		for _, v2 := range v.ExchangeAssetTicker {
			if v2.FuturesContract == nil {
				spotVersusContracts = append(spotVersusContracts, spotPairs{
					exchange:     v2.Exchange.GetName(),
					pair:         v2.SpotPair,
					superCompare: v2.ComparePair,
					spotLast:     v2.LastPrice,
					volume:       v2.Volume,
				})
			}
		}
	}
	for _, v := range comparableCurrencyToContracts {
		for _, v2 := range v.ExchangeAssetTicker {
			if v2.FuturesContract != nil {
				for i := range spotVersusContracts {
					contractCompare := getComparablePair(v2.FuturesContract.Underlying)
					if !contractCompare.Equal(spotVersusContracts[i].superCompare) {
						continue
					}
					spotVersusContracts[i].comparisons = append(spotVersusContracts[i].comparisons, result{
						contract:     v2,
						baseExchange: spotVersusContracts[i].exchange,
						baseAsset:    asset.Spot,
						baseCurr:     spotVersusContracts[i].pair,
					})
				}
			}
		}
	}

	for i := range spotVersusContracts {
		for j := range spotVersusContracts[i].comparisons {
			if spotVersusContracts[i].comparisons[j].contract.LastPrice > 0 {
				spotVersusContracts[i].comparisons[j].comparison = gctmath.CalculatePercentageDifference(spotVersusContracts[i].comparisons[j].contract.LastPrice, spotVersusContracts[i].spotLast)
				spotVersusContracts[i].comparisons[j].annualisedRateOfReturn = calculateAnnualisedRateOfReturn(
					spotVersusContracts[i].spotLast,
					spotVersusContracts[i].comparisons[j].contract.LastPrice,
					spotVersusContracts[i].comparisons[j].contract.FuturesContract.EndDate.Sub(time.Now()))
			}
		}
	}

	// Traverse ALL exchanges that support the underlying currency
	// build final map of all pairings, spot and futures

	// retrieve ticker of all assets of interest

	// compare difference to spot prices
	var index int64

	var biggest result
	for i := range spotVersusContracts {
		renderTable(&spotVersusContracts[i])
		for j := range spotVersusContracts[i].comparisons {
			index++
			if biggest.annualisedRateOfReturn < spotVersusContracts[i].comparisons[j].annualisedRateOfReturn {
				biggest = result{
					baseExchange:           spotVersusContracts[i].exchange,
					baseCurr:               spotVersusContracts[i].pair,
					baseAsset:              asset.Spot,
					contract:               spotVersusContracts[i].comparisons[j].contract,
					comparison:             spotVersusContracts[i].comparisons[j].comparison,
					annualisedRateOfReturn: spotVersusContracts[i].comparisons[j].annualisedRateOfReturn,
				}
			}
		}
	}
	fmt.Println(biggest)
}

func (p *PairDetails) GetUniqueMapKey() string {
	if p.Key != "" {
		return p.Key
	}
	if p.Asset.IsFutures() {
		p.Key = strings.ToLower(p.Exchange.GetName() + "-" + p.Asset.String() + "-" + p.FuturesContract.Name.String())
	} else {
		p.Key = strings.ToLower(p.Exchange.GetName() + "-" + p.Asset.String() + "-" + p.SpotPair.String())
	}
	return p.Key
}

func calculateAnnualisedRateOfReturn(spotPrice, futuresPrice float64, timeUntilExpiry time.Duration) float64 {
	if spotPrice == 0 || futuresPrice == 0 {
		return 0
	}
	// Calculate the rate of return
	rateOfReturn := (futuresPrice - spotPrice) / spotPrice

	// Convert the remaining time to years
	years := timeUntilExpiry.Seconds() / kline.OneYear.Duration().Seconds()

	// Calculate the annualized rate of return
	annualizedReturn := math.Pow(1+rateOfReturn, 1/float64(years)) - 1

	return annualizedReturn * 100
}

func renderTable(pairs *spotPairs) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{pairs.exchange, asset.Spot, pairs.pair})
	t.AppendHeader(table.Row{"#", "Exchange", "Asset", "Pair", "Start", "End", "Diff", "ARoR"})
	for i := range pairs.comparisons {
		t.AppendRow(table.Row{i + 1,
			pairs.comparisons[i].contract.Exchange.GetName(),
			pairs.comparisons[i].contract.FuturesContract.Asset,
			pairs.comparisons[i].contract.FuturesContract.Name,
			pairs.comparisons[i].contract.FuturesContract.StartDate,
			pairs.comparisons[i].contract.FuturesContract.EndDate,
			pairs.comparisons[i].comparison,
			pairs.comparisons[i].annualisedRateOfReturn})
	}
	t.AppendSeparator()
	t.Render()
}

func disableIrrelevantSpotPairs(exchs []exchange.IBotExchange, wg *sync.WaitGroup, formatting currency.PairFormat, contractComparer map[string]ComboHolder) map[string]ComboHolder {
	for i := range exchs {
		wg.Add(1)
		go func(it int) {
			defer wg.Done()
			b := exchs[it].GetBase()
			enabledPairs := b.CurrencyPairs.Pairs[asset.Spot].Enabled
			enabledPairs = enabledPairs.Format(formatting)
			for j := range enabledPairs {
				m.Lock()
				superCompare := getComparablePair(enabledPairs[j])
				lookup, ok := contractComparer[superCompare.String()]
				if ok {
					lookup.ExchangeAssetTicker = append(lookup.ExchangeAssetTicker, &PairDetails{
						Exchange:    exchs[it],
						ComparePair: superCompare,
						SpotPair:    enabledPairs[j],
					})
					contractComparer[enabledPairs[j].String()] = lookup
				}
				m.Unlock()
				if !ok {
					err := b.CurrencyPairs.DisablePair(asset.Spot, enabledPairs[j])
					if err != nil {
						fmt.Println(err)
						return
					}
					continue
				}
			}
		}(i)
	}
	wg.Wait()
	return contractComparer
}

func getComparablePair(cp currency.Pair) currency.Pair {
	superCompare := cp
	if cp.Base == currency.XBT {
		superCompare = currency.NewPair(currency.BTC, cp.Quote)
	}
	if cp.Quote == currency.XBT {
		superCompare = currency.NewPair(cp.Base, currency.BTC)
	}
	// add option to group close enough currencies like USD, USDT, BUSD and USDC
	if cp.Quote == currency.USDT ||
		cp.Quote == currency.BUSD ||
		cp.Quote == currency.USDC {
		superCompare = currency.NewPair(cp.Base, currency.USD)
	}

	return superCompare.Format(formatting)
}

func loadAndCategoriseFuturesContracts(exchs []exchange.IBotExchange, wg *sync.WaitGroup, formatting currency.PairFormat) (map[string]ComboHolder, error) {
	response := make(map[string]ComboHolder)
	for i := range exchs {

		wg.Add(1)
		go func(it int) {
			defer wg.Done()
			b := exchs[it].GetBase()
			enabledAssets := b.CurrencyPairs.GetAssetTypes(true)
			for j := range enabledAssets {
				if enabledAssets[j] == asset.Spot {
					continue
				}

				contracts, err := exchs[it].GetFuturesContractDetails(context.Background(), enabledAssets[j])
				if err != nil && !errors.Is(err, common.ErrFunctionNotSupported) &&
					!(errors.Is(err, asset.ErrNotSupported)) &&
					!(errors.Is(err, futures.ErrNotFuturesAsset)) {
					fmt.Println(err)
					return
				}
				for x := range contracts {
					if ignorePerps && contracts[x].Type == futures.Perpetual {
						continue
					}
					// Add in base equivalents, mainly in the form of XBT == BTC
					superCompare := getComparablePair(contracts[x].Underlying)
					pairStr := formatting.Format(superCompare)
					if btcUSDOnly && pairStr != "BTC-USD" {
						continue
					}
					m.Lock()
					lookup := response[pairStr]
					lookup.ExchangeAssetTicker = append(lookup.ExchangeAssetTicker, &PairDetails{
						Exchange:        exchs[it],
						FuturesContract: &contracts[x],
						ComparePair:     superCompare,
					})
					err = b.CurrencyPairs.EnablePair(contracts[x].Asset, contracts[x].Name)
					if err != nil {
						if errors.Is(err, currency.ErrPairNotFound) {
							availPairs, err := b.CurrencyPairs.GetPairs(contracts[x].Asset, false)
							availPairs = availPairs.Add(contracts[x].Name)
							err = b.CurrencyPairs.StorePairs(contracts[x].Asset, availPairs, false)
							if err != nil {
								fmt.Println(err)
								m.Unlock()
								continue
							}
							err = b.CurrencyPairs.EnablePair(contracts[x].Asset, contracts[x].Name)
							if err != nil {
								fmt.Println(err)
								m.Unlock()
								continue
							}
						} else {
							fmt.Println(err)
							m.Unlock()
							continue
						}
					}
					response[pairStr] = lookup
					m.Unlock()
				}
			}
		}(i)
	}
	wg.Wait()
	return response, nil
}

func setupExchanges(wg *sync.WaitGroup, exchangeManager *engine.ExchangeManager) {
	exchanges := exchange.Exchanges
	bannedExchanges := []string{"okcoin international", "itbit", "bitflyer", "alphapoint", "yobit"}
	for i := range exchanges {
		if binanceOnly && strings.ToLower(exchanges[i]) != "binance" {
			continue
		}
		wg.Add(1)
		go func(it int) {
			defer wg.Done()
			if common.StringDataContains(bannedExchanges, strings.ToLower(exchanges[it])) {
				return
			}
			exch, err := exchangeManager.NewExchangeByName(exchanges[it])
			if err != nil {
				fmt.Println(err)
				return
			}
			b := exch.GetBase()
			conf, err := exch.GetDefaultConfig(context.Background())
			if err != nil {
				fmt.Println(err)
				return
			}
			if b.Features.Supports.Websocket {
				conf.Features.Enabled.Websocket = true
				conf.Websocket = convert.BoolPtr(true)
			}

			assets := conf.CurrencyPairs.GetAssetTypes(false)
			var hasSpot, hasFutures bool
			var assetsToDisableAllPairs []asset.Item
			for j := range assets {
				if assets[j].IsFutures() {
					hasFutures = true
					assetsToDisableAllPairs = append(assetsToDisableAllPairs, assets[j])
					conf.CurrencyPairs.Pairs[assets[j]].AssetEnabled = convert.BoolPtr(true)
					continue
				}
				if assets[j] == asset.Spot {
					conf.CurrencyPairs.Pairs[assets[j]].AssetEnabled = convert.BoolPtr(true)
					hasSpot = true
					continue
				}
				conf.CurrencyPairs.Pairs[assets[j]].AssetEnabled = convert.BoolPtr(false)
			}

			if hasSpot || hasFutures {
				exch.SetDefaults()
				conf.Enabled = true
				conf.WebsocketTrafficTimeout = time.Minute
				err = exch.Setup(conf)
				if err != nil {
					fmt.Println(err)
					return
				}
				err = exch.UpdateTradablePairs(context.Background(), true)
				if err != nil {
					fmt.Println(err)
					return
				}

				for j := range assetsToDisableAllPairs {
					b.CurrencyPairs.Pairs[assetsToDisableAllPairs[j]].Enabled = currency.Pairs{}
				}
				b.CurrencyPairs.Pairs[asset.Spot].Enabled = b.CurrencyPairs.Pairs[asset.Spot].Available
				exchangeManager.Add(exch)
			}
		}(i)
	}
	wg.Wait()
}
