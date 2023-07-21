package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

var (
	btcUSDOnly bool = true
)

func main() {
	exchangeManager := engine.NewExchangeManager()
	wg := &sync.WaitGroup{}
	setupExchangeManager(wg, exchangeManager)

	formatting := currency.PairFormat{
		Uppercase: true,
		Delimiter: "-",
	}

	exchanges, err := exchangeManager.GetExchanges()
	if err != nil {
		fmt.Println(err)
		return
	}
	currencyComparer5000 := loadFuturesContracts(exchanges, wg, formatting)
	//outputLikeContracts(currencyComparer5000)
	cashAndCarryHolder := HolderHolder{currencyComparer5000}
	// now check for any of the good ones with SPOT crossovers
	disableIrrelevantSpotPairs(exchanges, wg, formatting, &cashAndCarryHolder)
	for i := range exchanges {
		wg.Add(1)
		go func(it int) {
			defer wg.Done()
			b := exchanges[it].GetBase()
			enabledAssets := b.GetAssetTypes(true)
			if b.Features.Supports.RESTCapabilities.TickerBatching {
				for j := range enabledAssets {
					if len(b.CurrencyPairs.Pairs[enabledAssets[j]].Enabled) == 0 {
						continue
					}
					err = exchanges[it].UpdateTickers(context.Background(), enabledAssets[j])
					if err != nil {
						if errors.Is(err, common.ErrFunctionNotSupported) {
							updateTickersByPair(it, enabledAssets, err, exchanges)
						} else {
							fmt.Println(err)
							continue
						}
					}
				}
			} else {
				err = updateTickersByPair(it, enabledAssets, err, exchanges)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	for k, v := range cashAndCarryHolder.ComparableCurrencyPairs {
		for k2, v2 := range v.ExchangeAssetTicker {
			var cp currency.Pair
			a := asset.Spot
			if v2.Contract == nil {
				cp = v2.SuperCompare
			} else {
				cp = v2.Contract.Name
				a = v2.Contract.Asset
			}
			tick, err := ticker.GetTicker(v2.Exchange.GetName(), cp, a)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if tick.Last == 0 && tick.Close == 0 && tick.Bid == 0 && tick.Ask == 0 {
				fmt.Println(v2.Exchange.GetName(), cp, a, "NO TICKER!")
				continue
			}
			v2.LastPrice = tick.Last
			v2.Close = tick.Close
			v2.Bid = tick.Bid
			v2.Ask = tick.Ask
			cashAndCarryHolder.ComparableCurrencyPairs[k].ExchangeAssetTicker[k2] = v2
		}
	}

	type result struct {
		contract   PairDetails
		comparison float64
	}

	type spotPairs struct {
		exchange    string
		pair        currency.Pair
		spotLast    float64
		comparisons []result
	}

	var butts []spotPairs
	for _, v := range cashAndCarryHolder.ComparableCurrencyPairs {
		for _, v2 := range v.ExchangeAssetTicker {
			if v2.Contract == nil {
				butts = append(butts, spotPairs{
					exchange: v2.Exchange.GetName(),
					pair:     v2.SuperCompare,
					spotLast: v2.LastPrice,
				})
			}
		}
	}
	for _, v := range cashAndCarryHolder.ComparableCurrencyPairs {
		for _, v2 := range v.ExchangeAssetTicker {
			if v2.Contract != nil {
				for i := range butts {
					butts[i].comparisons = append(butts[i].comparisons, result{
						contract: v2,
					})
				}
			}
		}
	}

	for i := range butts {
		for j := range butts[i].comparisons {
			if butts[i].comparisons[j].contract.LastPrice > 0 {
				butts[i].comparisons[j].comparison = math.CalculatePercentageDifference(butts[i].comparisons[j].contract.LastPrice, butts[i].spotLast)
			}
		}
	}

	// Traverse ALL exchanges that support the underlying currency
	// build final map of all pairings, spot and futures

	// retrieve ticker of all assets of interest

	// compare difference to spot prices
	fmt.Println("hi")
}

func updateTickersByPair(it int, enabledAssets asset.Items, err error, exchanges []exchange.IBotExchange) error {
	for j := range enabledAssets {
		var enabledPairs currency.Pairs
		enabledPairs, err = exchanges[it].GetEnabledPairs(enabledAssets[j])
		if err != nil {
			return err
		}
		for z := range enabledPairs {
			_, err = exchanges[it].UpdateTicker(context.Background(), enabledPairs[z], enabledAssets[j])
			if err != nil {
				//_, err = exchanges[it].UpdateTicker(context.Background(), enabledPairs[z], enabledAssets[j])
				fmt.Println(err)
				continue
			}
		}
	}
	return nil
}

func disableIrrelevantSpotPairs(exchs []exchange.IBotExchange, wg *sync.WaitGroup, formatting currency.PairFormat, contractComparer *HolderHolder) {
	for i := range exchs {
		wg.Add(1)
		go func(it int) {
			defer wg.Done()
			b := exchs[it].GetBase()
			enabledPairs := b.CurrencyPairs.Pairs[asset.Spot].Enabled
			enabledPairs = enabledPairs.Format(formatting)
			for j := range enabledPairs {
				m.Lock()
				lookup, ok := contractComparer.ComparableCurrencyPairs[enabledPairs[j].String()]
				if ok {
					if lookup.ExchangeAssetTicker == nil {
						lookup.ExchangeAssetTicker = make(map[string]PairDetails)
					}
					lookup.ExchangeAssetTicker[exchs[it].GetName()+asset.Spot.String()] = PairDetails{
						Exchange:     exchs[it],
						SuperCompare: enabledPairs[j],
					}
					contractComparer.ComparableCurrencyPairs[enabledPairs[j].String()] = lookup

					fmt.Println("found ", enabledPairs[j].String())
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
}

func outputLikeContracts(contractComparer map[string]ComboHolder) {
	//for k, v := range contractComparer {
	//	canWrite := false
	//	sb := strings.Builder{}
	//	sb.WriteString(k + ": \n")
	//	for z := range v {
	//		if !v.ExchangeAssetTicker[z].Contract.IsActive {
	//			continue
	//		}
	//		if v.ExchangeAssetTicker[z].Contract.Type == futures.Perpetual {
	//			// for now
	//			continue
	//		}
	//		canWrite = true
	//		sb.WriteString(v.ExchangeAssetTicker[z].Exchange + " " + v.ExchangeAssetTicker[z].Contract.Name.String() + "\n")
	//	}
	//	sb.WriteString("END \n\n")
	//	if canWrite {
	//		fmt.Println(sb.String())
	//	}
	//}
}

func loadFuturesContracts(exchs []exchange.IBotExchange, wg *sync.WaitGroup, formatting currency.PairFormat) map[string]ComboHolder {
	contractHolder := make(map[string]ComboHolder)
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
					// Add in base equivalents, mainly in the form of XBT == BTC
					pairStr := formatting.Format(contracts[x].Underlying)
					var superCompare currency.Pair
					if contracts[x].Underlying.Base == currency.XBT {
						superCompare = currency.NewPair(currency.BTC, contracts[x].Underlying.Quote)
						pairStr = formatting.Format(superCompare)
					}
					if contracts[x].Underlying.Quote == currency.XBT {
						superCompare = currency.NewPair(contracts[x].Underlying.Base, currency.BTC)
						pairStr = formatting.Format(superCompare)
					}
					// add option to group close enough currencies like USD, USDT, BUSD and USDC
					if contracts[x].Underlying.Quote == currency.USD ||
						contracts[x].Underlying.Quote == currency.USDT ||
						contracts[x].Underlying.Quote == currency.BUSD ||
						contracts[x].Underlying.Quote == currency.USDC {
						superCompare = currency.NewPair(contracts[x].Underlying.Base, currency.USD)
						pairStr = formatting.Format(superCompare)
					}
					if btcUSDOnly && pairStr != "BTC-USD" {
						continue
					}
					m.Lock()
					lookup := contractHolder[pairStr]
					if lookup.ExchangeAssetTicker == nil {
						lookup.ExchangeAssetTicker = make(map[string]PairDetails)
					}
					lookup.ExchangeAssetTicker[exchs[it].GetName()+enabledAssets[j].String()] = PairDetails{
						Exchange:     exchs[it],
						Contract:     &contracts[x],
						SuperCompare: superCompare,
					}
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
					contractHolder[pairStr] = lookup
					m.Unlock()
				}
			}
		}(i)
	}
	wg.Wait()
	return contractHolder
}

func setupExchangeManager(wg *sync.WaitGroup, exchangeManager *engine.ExchangeManager) {
	exchanges := exchange.Exchanges
	bannedExchanges := []string{"okcoin international", "itbit", "bitflyer", "alphapoint"}
	for i := range exchanges {
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
			conf, err := exch.GetDefaultConfig(context.Background())
			if err != nil {
				fmt.Println(err)
				return
			}
			assets := conf.CurrencyPairs.GetAssetTypes(false)
			var hasSpot, hasFutures bool
			var assetsToDisableAllPairs []asset.Item
			for j := range assets {
				if assets[j].IsFutures() {
					hasFutures = true
					fmt.Println(exch.GetName())
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
				b := exch.GetBase()

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
