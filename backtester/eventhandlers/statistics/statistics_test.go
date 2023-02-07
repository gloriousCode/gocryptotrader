package statistics

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

var (
	eleeg  = decimal.NewFromInt(1336)
	eleet  = decimal.NewFromInt(1337)
	eleeet = decimal.NewFromInt(13337)
	eleeb  = decimal.NewFromInt(1338)
)

func TestOBV(t *testing.T) {
	expectedOutput := []float64{0, -6710.77, -10905.36, -12409.890000000001, -15526.640000000001, -11502.070000000002, -9922.37, -7338.52, -11061.43, -13700.12, -12581.01, -16553.72, -8480.990000000002, -5224.250000000002, -2516.980000000002, 4211.299999999997, 16369.619999999997, 4456.389999999998, -1500.8500000000022, 7499.139999999998, 3710.8099999999977, 5766.899999999998, 1755.4599999999978, 18764.89, 10158.539999999999, 3445.279999999999, 12808.439999999999, 16486.449999999997, 7700.999999999996, 3280.7999999999965, 6675.489999999996, 3531.679999999996, -5079.330000000004, 1351.0999999999967, -1461.6900000000032, 4878.899999999997, 14498.669999999998, 24064.229999999996, 17582.309999999998, 25527.469999999998, 20517.85, 22186.85, 19171.059999999998, 14412.929999999997, 9545.699999999997, 17609.67, 25164.159999999996, 29177.649999999994, 32446.389999999992, 37276.729999999996, 29716.059999999998, 37843.659999999996, 44397.06999999999, 35049.619999999995, 41239.99999999999, 35477.689999999995, 39611.17, 33697.97, 40452.32, 30515.5, 37123.99, 41395.46, 39298.9, 42628.12, 36751.11, 30471.11, 18906.91, 28180.809999999998, 21633.829999999998, 19295.76, 23452.329999999998, 29639.19, 25009.219999999998, 21655.989999999998, 28104.679999999997, 32241.1, 29435.51, 20366.62, 8189.23, -85.72000000000116, 7203.8899999999985, -51309.5, 3110.1399999999994, -12260.730000000001, 4328.569999999998, -30576.89, -13059.05, 9500.18, 34283.91, 64366.41, 52569.68000000001, 41756.30000000001, 60810.100000000006, 80529.19, 63853.45, 71247.16, 60430.87, 48817.380000000005, 39518.44, 40067.060000000005}
	testClose := []float64{7509.7, 7316.17, 7251.52, 7195.79, 7188.3, 7246, 7296.24, 7385.54, 7220.24, 7168.36, 7178.68, 6950.56, 7338.91, 7344.48, 7356.7, 7762.74, 8159.01, 8044.44, 7806.78, 8200, 8016.22, 8180.76, 8105.01, 8813.04, 8809.17, 8710.15, 8892.63, 8908.53, 8696.6, 8625.17, 8717.89, 8655.93, 8378.44, 8422.13, 8329.5, 8590.48, 8894.54, 9400, 9289.18, 9500, 9327.85, 9377.17, 9329.39, 9288.09, 9159.37, 9618.42, 9754.63, 9803.42, 9902, 10173.97, 9850.01, 10268.98, 10348.78, 10228.67, 10364.04, 9899.78, 9912.89, 9697.15, 10185.17, 9595.72, 9612.76, 9696.13, 9668.13, 9965.21, 9652.58, 9305.4, 8779.36, 8816.5, 8703.84, 8527.74, 8528.95, 8917.34, 8755.45, 8753.28, 9066.65, 9153.79, 8893.93, 8033.7, 7936.25, 7885.92, 7934.57, 4841.67, 5622.74, 5169.37, 5343.64, 5033.42, 5324.99, 5406.92, 6181.18, 6210.14, 6187.78, 5813.15, 6493.51, 6768.64, 6692.22, 6760.72, 6376.03, 6253.08, 5870.9, 5947.01}
	testVolume := []float64{3796.23, 6710.77, 4194.59, 1504.53, 3116.75, 4024.57, 1579.7, 2583.85, 3722.91, 2638.69, 1119.11, 3972.71, 8072.73, 3256.74, 2707.27, 6728.28, 12158.32, 11913.23, 5957.24, 8999.99, 3788.33, 2056.09, 4011.44, 17009.43, 8606.35, 6713.26, 9363.16, 3678.01, 8785.45, 4420.2, 3394.69, 3143.81, 8611.01, 6430.43, 2812.79, 6340.59, 9619.77, 9565.56, 6481.92, 7945.16, 5009.62, 1669.0, 3015.79, 4758.13, 4867.23, 8063.97, 7554.49, 4013.49, 3268.74, 4830.34, 7560.67, 8127.6, 6553.41, 9347.45, 6190.38, 5762.31, 4133.48, 5913.2, 6754.35, 9936.82, 6608.49, 4271.47, 2096.56, 3329.22, 5877.01, 6280.0, 11564.2, 9273.9, 6546.98, 2338.07, 4156.57, 6186.86, 4629.97, 3353.23, 6448.69, 4136.42, 2805.59, 9068.89, 12177.39, 8274.95, 7289.61, 58513.39, 54419.64, 15370.87, 16589.3, 34905.46, 17517.84, 22559.23, 24783.73, 30082.5, 11796.73, 10813.38, 19053.8, 19719.09, 16675.74, 7393.71, 10816.29, 11613.49, 9298.94, 548.62}

	ret := indicators.OBV(testClose, testVolume)
	if len(ret) != len(expectedOutput) {
		t.Fatalf("unexpected length of return slice %v", len(ret))
	}
	for i := range ret {
		t.Log(ret[i])
	}

	for x := range ret {
		if ret[x] != expectedOutput[x] {
			t.Fatalf("unexpected value returned %v", ret[x])
		}
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	s := &Statistic{
		TotalOrders: 1,
	}
	err := s.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if s.TotalOrders != 0 {
		t.Error("expected 0")
	}

	s = nil
	err = s.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}
}

func TestAddDataEventForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if s.ExchangeAssetPairStatistics == nil {
		t.Error("expected not nil")
	}
	if len(s.ExchangeAssetPairStatistics[exch][a][p.Base.Item][p.Quote.Item].Events) != 1 {
		t.Error("expected 1 event")
	}
}

func TestAddSignalEventForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&signal.Signal{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
	b := &event.Base{}
	err = s.SetEventForOffset(&signal.Signal{
		Base: b,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b.Exchange = exch
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = p
	b.AssetType = a
	err = s.SetEventForOffset(&kline.Kline{
		Base:   b,
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base:       b,
		ClosePrice: eleet,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddExchangeEventForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&order.Order{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
	b := &event.Base{}

	b.Exchange = exch
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = p
	b.AssetType = a
	err = s.SetEventForOffset(&kline.Kline{
		Base:   b,
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&order.Order{
		Base:       b,
		ID:         "elite",
		Direction:  gctorder.Buy,
		Status:     gctorder.New,
		ClosePrice: eleet,
		Amount:     eleet,
		OrderType:  gctorder.Stop,
		Leverage:   eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddFillEventForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&fill.Fill{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
	b := &event.Base{}
	err = s.SetEventForOffset(&fill.Fill{
		Base: b,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	b.Exchange = exch
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = p
	b.AssetType = a

	err = s.SetEventForOffset(&kline.Kline{
		Base:   b,
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&fill.Fill{
		Base:                b,
		Direction:           gctorder.Buy,
		Amount:              eleet,
		ClosePrice:          eleet,
		VolumeAdjustedPrice: eleet,
		PurchasePrice:       eleet,
		ExchangeFee:         eleet,
		Slippage:            eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddHoldingsForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.AddHoldingsForTime(&holdings.Holding{})
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("received: %v, expected: %v", err, errExchangeAssetPairStatsUnset)
	}
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
	err = s.AddHoldingsForTime(&holdings.Holding{})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("received: %v, expected: %v", err, errCurrencyStatisticsUnset)
	}

	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.AddHoldingsForTime(&holdings.Holding{
		Pair:                         p,
		Asset:                        a,
		Exchange:                     exch,
		Timestamp:                    tt,
		QuoteInitialFunds:            eleet,
		BaseSize:                     eleet,
		BaseValue:                    eleet,
		SoldAmount:                   eleet,
		BoughtAmount:                 eleet,
		QuoteSize:                    eleet,
		TotalValueDifference:         eleet,
		ChangeInTotalValuePercent:    eleet,
		PositionsValueDifference:     eleet,
		TotalValue:                   eleet,
		TotalFees:                    eleet,
		TotalValueLostToVolumeSizing: eleet,
		TotalValueLostToSlippage:     eleet,
		TotalValueLost:               eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddComplianceSnapshotForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}

	err := s.AddComplianceSnapshotForTime(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.AddComplianceSnapshotForTime(nil, &fill.Fill{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	err = s.AddComplianceSnapshotForTime(&compliance.Snapshot{}, &fill.Fill{})
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("received: %v, expected: %v", err, errExchangeAssetPairStatsUnset)
	}
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
	b := &event.Base{}
	err = s.AddComplianceSnapshotForTime(&compliance.Snapshot{}, &fill.Fill{Base: b})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("received: %v, expected: %v", err, errCurrencyStatisticsUnset)
	}
	b.Exchange = exch
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = p
	b.AssetType = a
	err = s.SetEventForOffset(&kline.Kline{
		Base:   b,
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.AddComplianceSnapshotForTime(&compliance.Snapshot{
		Timestamp: tt,
	}, &fill.Fill{
		Base: b,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestSerialise(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	if _, err := s.Serialise(); err != nil {
		t.Error(err)
	}
}

func TestSetStrategyName(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	s.SetStrategyName("test")
	if s.StrategyName != "test" {
		t.Error("expected test")
	}
}

func TestPrintTotalResults(t *testing.T) {
	t.Parallel()
	s := Statistic{
		FundingStatistics: &FundingStatistics{},
	}
	s.BiggestDrawdown = s.GetTheBiggestDrawdownAcrossCurrencies([]FinalResultsHolder{
		{
			Exchange: "test",
			MaxDrawdown: Swing{
				DrawdownPercent: eleet,
			},
		},
	})
	s.BestStrategyResults = s.GetBestStrategyPerformer([]FinalResultsHolder{
		{
			Exchange:         "test",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      Swing{},
			MarketMovement:   eleet,
			StrategyMovement: eleet,
		},
	})
	s.BestMarketMovement = s.GetBestMarketPerformer([]FinalResultsHolder{
		{
			Exchange:       "test",
			MarketMovement: eleet,
		},
	})
	s.PrintTotalResults()
}

func TestGetBestStrategyPerformer(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	resp := s.GetBestStrategyPerformer(nil)
	if resp.Exchange != "" {
		t.Error("expected unset details")
	}

	resp = s.GetBestStrategyPerformer([]FinalResultsHolder{
		{
			Exchange:         "test",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      Swing{},
			MarketMovement:   eleet,
			StrategyMovement: eleet,
		},
		{
			Exchange:         "test2",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      Swing{},
			MarketMovement:   eleeb,
			StrategyMovement: eleeb,
		},
	})

	if resp.Exchange != "test2" {
		t.Error("expected test2")
	}
}

func TestGetTheBiggestDrawdownAcrossCurrencies(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	result := s.GetTheBiggestDrawdownAcrossCurrencies(nil)
	if result.Exchange != "" {
		t.Error("expected empty")
	}

	result = s.GetTheBiggestDrawdownAcrossCurrencies([]FinalResultsHolder{
		{
			Exchange: "test",
			MaxDrawdown: Swing{
				DrawdownPercent: eleet,
			},
		},
		{
			Exchange: "test2",
			MaxDrawdown: Swing{
				DrawdownPercent: eleeb,
			},
		},
	})
	if result.Exchange != "test2" {
		t.Error("expected test2")
	}
}

func TestGetBestMarketPerformer(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	result := s.GetBestMarketPerformer(nil)
	if result.Exchange != "" {
		t.Error("expected empty")
	}

	result = s.GetBestMarketPerformer([]FinalResultsHolder{
		{
			Exchange:       "test",
			MarketMovement: eleet,
		},
		{
			Exchange:       "test2",
			MarketMovement: eleeg,
		},
	})
	if result.Exchange != "test" {
		t.Error("expected test")
	}
}

func TestPrintAllEventsChronologically(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	s.PrintAllEventsChronologically()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&fill.Fill{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Direction:           gctorder.Buy,
		Amount:              eleet,
		ClosePrice:          eleet,
		VolumeAdjustedPrice: eleet,
		PurchasePrice:       eleet,
		ExchangeFee:         eleet,
		Slippage:            eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		ClosePrice: eleet,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	s.PrintAllEventsChronologically()
}

func TestCalculateTheResults(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	err := s.CalculateAllResults()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	tt := time.Now().Add(-gctkline.OneDay.Duration() * 7)
	tt2 := time.Now().Add(-gctkline.OneDay.Duration() * 6)
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	p2 := currency.NewPair(currency.XRP, currency.DOGE)
	err = s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       1,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       1,
		},
		OpenPrice:  eleet,
		HighPrice:  eleet,
		LowPrice:   eleet,
		ClosePrice: eleet,
		Volume:     eleet,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
			Offset:       2,
		},
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
			Offset:       2,
		},
		OpenPrice:  eleet,
		HighPrice:  eleet,
		LowPrice:   eleet,
		ClosePrice: eleet,
		Volume:     eleet,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       3,
		},
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       3,
		},
		OpenPrice:  eleeb,
		HighPrice:  eleeb,
		LowPrice:   eleeb,
		ClosePrice: eleeb,
		Volume:     eleeb,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
			Offset:       4,
		},
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	signal4 := &signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
			Offset:       4,
		},
		OpenPrice:  eleeb,
		HighPrice:  eleeb,
		LowPrice:   eleeb,
		ClosePrice: eleeb,
		Volume:     eleeb,
		Direction:  gctorder.Buy,
	}
	err = s.SetEventForOffset(signal4)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	s.ExchangeAssetPairStatistics[exch][a][p.Base.Item][p.Quote.Item].Events[1].Holdings.QuoteInitialFunds = eleet
	s.ExchangeAssetPairStatistics[exch][a][p.Base.Item][p.Quote.Item].Events[1].Holdings.TotalValue = eleeet
	s.ExchangeAssetPairStatistics[exch][a][p2.Base.Item][p2.Quote.Item].Events[1].Holdings.QuoteInitialFunds = eleet
	s.ExchangeAssetPairStatistics[exch][a][p2.Base.Item][p2.Quote.Item].Events[1].Holdings.TotalValue = eleeet

	funds, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pBase, err := funding.CreateItem(exch, a, p.Base, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pQuote, err := funding.CreateItem(exch, a, p.Quote, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	pair, err := funding.CreatePair(pBase, pQuote)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.AddPair(pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pBase2, err := funding.CreateItem(exch, a, p2.Base, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pQuote2, err := funding.CreateItem(exch, a, p2.Quote, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pair2, err := funding.CreatePair(pBase2, pQuote2)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.AddPair(pair2)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	s.FundManager = funds
	err = s.CalculateAllResults()
	if !errors.Is(err, errMissingSnapshots) {
		t.Errorf("received '%v' expected '%v'", err, errMissingSnapshots)
	}
	err = s.CalculateAllResults()
	if !errors.Is(err, errMissingSnapshots) {
		t.Errorf("received '%v' expected '%v'", err, errMissingSnapshots)
	}

	funds, err = funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.AddPair(pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.AddPair(pair2)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	s.FundManager = funds
	err = s.CalculateAllResults()
	if !errors.Is(err, errMissingSnapshots) {
		t.Errorf("received '%v' expected '%v'", err, errMissingSnapshots)
	}

	err = s.AddComplianceSnapshotForTime(&compliance.Snapshot{Timestamp: tt2}, signal4)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestCalculateBiggestEventDrawdown(t *testing.T) {
	tt1 := time.Now().Add(-gctkline.OneDay.Duration() * 7).Round(gctkline.OneDay.Duration())
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var events []data.Event
	for i := int64(0); i < 100; i++ {
		tt1 = tt1.Add(gctkline.OneDay.Duration())
		even := &event.Base{
			Exchange:     exch,
			Time:         tt1,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		}
		if i == 50 {
			// throw in a wrench, a spike in price
			events = append(events, &kline.Kline{
				Base:  even,
				Close: decimal.NewFromInt(1336),
				High:  decimal.NewFromInt(1336),
				Low:   decimal.NewFromInt(1336),
			})
		} else {
			events = append(events, &kline.Kline{
				Base:  even,
				Close: decimal.NewFromInt(1337).Sub(decimal.NewFromInt(i)),
				High:  decimal.NewFromInt(1337).Sub(decimal.NewFromInt(i)),
				Low:   decimal.NewFromInt(1337).Sub(decimal.NewFromInt(i)),
			})
		}
	}

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even := &event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Base:  even,
		Close: decimal.NewFromInt(1338),
		High:  decimal.NewFromInt(1338),
		Low:   decimal.NewFromInt(1338),
	})

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even = &event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Base:  even,
		Close: decimal.NewFromInt(1337),
		High:  decimal.NewFromInt(1337),
		Low:   decimal.NewFromInt(1337),
	})

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even = &event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Base:  even,
		Close: decimal.NewFromInt(1339),
		High:  decimal.NewFromInt(1339),
		Low:   decimal.NewFromInt(1339),
	})

	_, err := CalculateBiggestEventDrawdown(nil)
	if !errors.Is(err, errReceivedNoData) {
		t.Errorf("received %v expected %v", err, errReceivedNoData)
	}

	resp, err := CalculateBiggestEventDrawdown(events)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if resp.Highest.Value != decimal.NewFromInt(1337) && !resp.Lowest.Value.Equal(decimal.NewFromInt(1238)) {
		t.Error("unexpected max drawdown")
	}

	// bogus scenario
	bogusEvent := []data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				CurrencyPair: p,
				AssetType:    a,
			},
			Close: decimal.NewFromInt(1339),
			High:  decimal.NewFromInt(1339),
			Low:   decimal.NewFromInt(1339),
		},
	}
	_, err = CalculateBiggestEventDrawdown(bogusEvent)
	if !errors.Is(err, gctcommon.ErrDateUnset) {
		t.Errorf("received %v expected %v", err, gctcommon.ErrDateUnset)
	}
}

func TestCalculateBiggestValueAtTimeDrawdown(t *testing.T) {
	var interval gctkline.Interval
	_, err := CalculateBiggestValueAtTimeDrawdown(nil, interval)
	if !errors.Is(err, errReceivedNoData) {
		t.Errorf("received %v expected %v", err, errReceivedNoData)
	}

	_, err = CalculateBiggestValueAtTimeDrawdown(nil, interval)
	if !errors.Is(err, errReceivedNoData) {
		t.Errorf("received %v expected %v", err, errReceivedNoData)
	}
}

func TestAddPNLForTime(t *testing.T) {
	t.Parallel()
	s := &Statistic{}
	err := s.AddPNLForTime(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received %v expected %v", err, gctcommon.ErrNilPointer)
	}

	sum := &portfolio.PNLSummary{}
	err = s.AddPNLForTime(sum)
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("received %v expected %v", err, errExchangeAssetPairStatsUnset)
	}

	tt := time.Now().Add(-gctkline.OneDay.Duration() * 7)
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       1,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.AddPNLForTime(sum)
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("received %v expected %v", err, errCurrencyStatisticsUnset)
	}

	sum.Exchange = exch
	sum.Asset = a
	sum.Pair = p
	err = s.AddPNLForTime(sum)
	if !errors.Is(err, errNoDataAtOffset) {
		t.Errorf("received %v expected %v", err, errNoDataAtOffset)
	}

	sum.Offset = 1
	err = s.AddPNLForTime(sum)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
}
