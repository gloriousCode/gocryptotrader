package report

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

func TestGenerateReport(t *testing.T) {
	e := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	ev := event.Event{
		Exchange:     e,
		Time:         time.Now(),
		Interval:     gctkline.OneHour,
		CurrencyPair: p,
		AssetType:    a,
		Why:          "coz",
	}
	d := Data{
		Config:       &config.Config{},
		OutputPath:   filepath.Join("..", "results"),
		TemplatePath: filepath.Join("tpl.gohtml"),
		OriginalCandles: []*gctkline.Item{
			{
				Candles: []gctkline.Candle{
					{},
				},
			},
		},
		EnhancedCandles: []DetailedKline{
			{
				Exchange:  e,
				Asset:     a,
				Pair:      p,
				Interval:  gctkline.OneHour,
				Watermark: "Binance - SPOT - BTC-USDT",
				Candles: []DetailedCandle{
					{
						Time:           time.Now().Add(-time.Hour * 5).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: gctorder.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(47, 194, 27, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 4).Unix(),
						Open:           1332,
						High:           1332,
						Low:            1330,
						Close:          1331,
						Volume:         2,
						MadeOrder:      true,
						OrderDirection: gctorder.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(252, 3, 3, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 3).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: gctorder.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(47, 194, 27, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 2).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: gctorder.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(252, 3, 3, 0.8)",
					},
					{
						Time:         time.Now().Unix(),
						Open:         1337,
						High:         1339,
						Low:          1336,
						Close:        1338,
						Volume:       3,
						VolumeColour: "rgba(47, 194, 27, 0.8)",
					},
				},
			},
			{
				Exchange:  "Bittrex",
				Asset:     a,
				Pair:      currency.NewPair(currency.BTC, currency.USD),
				Interval:  gctkline.OneDay,
				Watermark: "BITTREX - SPOT - BTC-USD - 1d",
				Candles: []DetailedCandle{
					{
						Time:           time.Now().Add(-time.Hour * 5).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: gctorder.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(47, 194, 27, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 4).Unix(),
						Open:           1332,
						High:           1332,
						Low:            1330,
						Close:          1331,
						Volume:         2,
						MadeOrder:      true,
						OrderDirection: gctorder.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(252, 3, 3, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 3).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: gctorder.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(47, 194, 27, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 2).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: gctorder.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(252, 3, 3, 0.8)",
					},
					{
						Time:         time.Now().Unix(),
						Open:         1337,
						High:         1339,
						Low:          1336,
						Close:        1338,
						Volume:       3,
						VolumeColour: "rgba(47, 194, 27, 0.8)",
					},
				},
			},
		},
		Statistics: &statistics.Statistic{
			StrategyName: "testStrat",
			ExchangeAssetPairStatistics: map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic{
				e: {
					a: {
						p: &currencystatistics.CurrencyStatistic{
							Events: []currencystatistics.EventStore{
								{
									Holdings:     holdings.Holding{},
									Transactions: compliance.Snapshot{},
									DataEvent: &kline.Kline{
										Event:  ev,
										Open:   1337,
										Close:  1337,
										Low:    1337,
										High:   1337,
										Volume: 1337,
									},
									SignalEvent: &signal.Signal{
										Event: ev,
										Price: 1337,
									},
									OrderEvent: &order.Order{
										Event: ev,
										Price: 1337,
									},
									FillEvent: &fill.Fill{
										Event:               ev,
										Amount:              1337,
										ClosePrice:          1337,
										VolumeAdjustedPrice: 1337,
										PurchasePrice:       1337,
										ExchangeFee:         1337,
										Slippage:            1337,
									},
								},
							},
							MaxDrawdown:              currencystatistics.Swing{},
							LowestClosePrice:         100,
							HighestClosePrice:        200,
							MarketMovement:           100,
							StrategyMovement:         100,
							RiskFreeRate:             1,
							CompoundAnnualGrowthRate: 1,
							BuyOrders:                1,
							SellOrders:               1,
							FinalHoldings: holdings.Holding{
								Pair:                         currency.Pair{},
								Asset:                        "",
								Exchange:                     "",
								Timestamp:                    time.Time{},
								InitialFunds:                 0,
								PositionsSize:                0,
								PositionsValue:               0,
								SoldAmount:                   0,
								SoldValue:                    0,
								BoughtAmount:                 0,
								BoughtValue:                  0,
								RemainingFunds:               0,
								TotalValueDifference:         0,
								ChangeInTotalValuePercent:    0,
								ExcessReturnPercent:          0,
								BoughtValueDifference:        0,
								SoldValueDifference:          0,
								PositionsValueDifference:     0,
								TotalValue:                   0,
								TotalFees:                    0,
								TotalValueLostToVolumeSizing: 0,
								TotalValueLostToSlippage:     0,
								RiskFreeRate:                 0,
							},
							FinalOrders: compliance.Snapshot{},
						},
					},
				},
			},
			RiskFreeRate:    0.03,
			TotalBuyOrders:  1337,
			TotalSellOrders: 1330,
			TotalOrders:     200,
			BiggestDrawdown: &statistics.FinalResultsHolder{
				Exchange: e,
				Asset:    a,
				Pair:     p,
				MaxDrawdown: currencystatistics.Swing{
					Highest: currencystatistics.Iteration{
						Time:  time.Now(),
						Price: 1337,
					},
					Lowest: currencystatistics.Iteration{
						Time:  time.Now(),
						Price: 137,
					},
					DrawdownPercent: 100,
				},
				MarketMovement:   1377,
				StrategyMovement: 1377,
			},
			BestStrategyResults: &statistics.FinalResultsHolder{
				Exchange: e,
				Asset:    a,
				Pair:     p,
				MaxDrawdown: currencystatistics.Swing{
					Highest: currencystatistics.Iteration{
						Time:  time.Now(),
						Price: 1337,
					},
					Lowest: currencystatistics.Iteration{
						Time:  time.Now(),
						Price: 137,
					},
					DrawdownPercent: 100,
				},
				MarketMovement:   1337,
				StrategyMovement: 1337,
			},
			BestMarketMovement: &statistics.FinalResultsHolder{
				Exchange: e,
				Asset:    a,
				Pair:     p,
				MaxDrawdown: currencystatistics.Swing{
					Highest: currencystatistics.Iteration{
						Time:  time.Now(),
						Price: 1337,
					},
					Lowest: currencystatistics.Iteration{
						Time:  time.Now(),
						Price: 137,
					},
					DrawdownPercent: 100,
				},
				MarketMovement:   1337,
				StrategyMovement: 1337,
			},
		},
	}
	err := d.GenerateReport()
	if err != nil {
		t.Error(err)
	}
}

func TestEnhanceCandles(t *testing.T) {
	tt := time.Now()
	var d Data
	err := d.enhanceCandles()
	if err != nil && err.Error() != "no candles to enhance" {
		t.Error(err)
	}
	d.AddKlineItem(&gctkline.Item{})
	err = d.enhanceCandles()
	if err != nil && err.Error() != "unable to proceed with unset Statistics property" {
		t.Error(err)
	}
	d.Statistics = &statistics.Statistic{}
	err = d.enhanceCandles()
	if err != nil {
		t.Error(err)
	}

	d.Statistics.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange] = make(map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot] = make(map[currency.Pair]*currencystatistics.CurrencyStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot][currency.NewPair(currency.BTC, currency.USDT)] = &currencystatistics.CurrencyStatistic{}

	d.AddKlineItem(&gctkline.Item{
		Exchange: testExchange,
		Pair:     currency.NewPair(currency.BTC, currency.USDT),
		Asset:    asset.Spot,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   tt,
				Open:   1336,
				High:   1338,
				Low:    1336,
				Close:  1337,
				Volume: 1337,
			},
		},
	})
	err = d.enhanceCandles()
	if err != nil {
		t.Error(err)
	}

	d.AddKlineItem(&gctkline.Item{
		Exchange: testExchange,
		Pair:     currency.NewPair(currency.BTC, currency.USDT),
		Asset:    asset.Spot,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   tt,
				Open:   1336,
				High:   1338,
				Low:    1336,
				Close:  1336,
				Volume: 1337,
			},
			{
				Time:   tt,
				Open:   1336,
				High:   1338,
				Low:    1336,
				Close:  1335,
				Volume: 1337,
			},
		},
	})

	err = d.enhanceCandles()
	if err != nil {
		t.Error(err)
	}

	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot][currency.NewPair(currency.BTC, currency.USDT)].FinalOrders = compliance.Snapshot{
		Orders: []compliance.SnapshotOrder{
			{
				ClosePrice:          1335,
				VolumeAdjustedPrice: 1337,
				SlippageRate:        1,
				CostBasis:           1337,
				Detail:              nil,
			},
		},
		Timestamp: tt,
	}
	err = d.enhanceCandles()
	if err != nil {
		t.Error(err)
	}

	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot][currency.NewPair(currency.BTC, currency.USDT)].FinalOrders = compliance.Snapshot{
		Orders: []compliance.SnapshotOrder{
			{
				ClosePrice:          1335,
				VolumeAdjustedPrice: 1337,
				SlippageRate:        1,
				CostBasis:           1337,
				Detail: &gctorder.Detail{
					Date: tt,
					Side: gctorder.Buy,
				},
			},
		},
		Timestamp: tt,
	}
	err = d.enhanceCandles()
	if err != nil {
		t.Error(err)
	}

	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot][currency.NewPair(currency.BTC, currency.USDT)].FinalOrders = compliance.Snapshot{
		Orders: []compliance.SnapshotOrder{
			{
				ClosePrice:          1335,
				VolumeAdjustedPrice: 1337,
				SlippageRate:        1,
				CostBasis:           1337,
				Detail: &gctorder.Detail{
					Date: tt,
					Side: gctorder.Sell,
				},
			},
		},
		Timestamp: tt,
	}
	err = d.enhanceCandles()
	if err != nil {
		t.Error(err)
	}

	if len(d.EnhancedCandles) == 0 {
		t.Error("expected enhanced candles")
	}
}
