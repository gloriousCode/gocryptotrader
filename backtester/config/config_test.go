package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	makerFee     = 0.002
	takerFee     = 0.001
	testExchange = "binance"
	dca          = "dollarcostaverage"
	// change this if you modify a config and want it to save to the example folder
	saveConfig = true
)

var (
	startDate time.Time
	endDate   time.Time
)

func TestMain(m *testing.M) {
	startDate = time.Now().Add(-time.Hour * 24 * 30)
	endDate = time.Now()
	startDate = startDate.Truncate(kline.OneDay.Duration())
	endDate = endDate.Truncate(kline.OneDay.Duration())

	os.Exit(m.Run())
}

func TestLoadConfig(t *testing.T) {
	_, err := LoadConfig([]byte(`{}`))
	if err != nil {
		t.Error(err)
	}
}

func TestReadConfigFromFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Problem creating temp dir at %s: %s\n", tempDir, err)
	}
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			t.Error(err)
		}
	}()
	var passFile *os.File
	passFile, err = ioutil.TempFile(tempDir, "*.start")
	if err != nil {
		t.Fatalf("Problem creating temp file at %v: %s\n", passFile, err)
	}
	_, err = passFile.WriteString("{}")
	if err != nil {
		t.Error(err)
	}
	err = passFile.Close()
	if err != nil {
		t.Error(err)
	}
	_, err = ReadConfigFromFile(passFile.Name())
	if err != nil {
		t.Error(err)
	}
}

func TestPrintSettings(t *testing.T) {
	cfg := Config{
		Nickname: "super fun run",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate: startDate,
				EndDate:   endDate,
			},
			CSVData: &CSVData{
				FullPath: "fake",
			},
			LiveData: &LiveData{
				APIKeyOverride:      "",
				APISecretOverride:   "",
				APIClientIDOverride: "",
				API2FAOverride:      "",
				RealOrders:          false,
			},
			DatabaseData: &DatabaseData{
				StartDate:      startDate,
				EndDate:        endDate,
				ConfigOverride: nil,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	cfg.PrintSetting()
}

func TestGenerateConfigForDCAAPICandles(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCAAPICandles",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-candles.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPITrades(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCAAPITrades",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.TradeStr,
			APIData: &APIData{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-trades.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPICandlesMultipleCurrencies(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCAAPICandlesMultipleCurrencies",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.ETH.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-candles-multiple-currencies.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

// these are tests for experimentation more than anything
func TestGenerateConfigForDCAAPICandlesMultiCurrencyAssessment(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCAAPICandlesMultiCurrencyAssessment",
		StrategySettings: StrategySettings{
			Name:            dca,
			IsMultiCurrency: true,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 1000000,
				BuySide: MinMax{
					MinimumSize:  0,
					MaximumSize:  0,
					MaximumTotal: 1000,
				},
				SellSide: MinMax{
					MinimumSize:  0,
					MaximumSize:  0,
					MaximumTotal: 1000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.ETH.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-candles-multi-currency-assessment.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCALiveCandles(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCALiveCandles",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			LiveData: &LiveData{
				APIKeyOverride:      "",
				APISecretOverride:   "",
				APIClientIDOverride: "",
				API2FAOverride:      "",
				RealOrders:          false,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-candles-live.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForRSIAPICustomSettings(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateRSICandleAPICustomSettingsStrat",
		StrategySettings: StrategySettings{
			Name: "rsi",
			CustomSettings: map[string]interface{}{
				"rsi-low":    30.0,
				"rsi-high":   70.0,
				"rsi-period": 14,
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 1000000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.ETH.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "rsi-api-candles.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCACSVCandles(t *testing.T) {
	fp := filepath.Join("..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv")
	cfg := Config{
		Nickname: "TestGenerateConfigForDCACSVCandles",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			CSVData: &CSVData{
				FullPath: fp,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-csv-candles.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCACSVTrades(t *testing.T) {
	fp := filepath.Join("..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv")
	cfg := Config{
		Nickname: "TestGenerateConfigForDCACSVTrades",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.TradeStr,
			CSVData: &CSVData{
				FullPath: fp,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-csv-trades.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCADatabaseCandles(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCADatabaseCandles",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			DatabaseData: &DatabaseData{
				StartDate: startDate,
				EndDate:   endDate,
				ConfigOverride: &database.Config{
					Enabled: true,
					Verbose: false,
					Driver:  "sqlite",
					ConnectionDetails: drivers.ConnectionDetails{
						Host:     "localhost",
						Database: "testsqlite.db",
					},
				},
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-database-candles.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestValidate(t *testing.T) {
	m := MinMax{
		MinimumSize:  -1,
		MaximumSize:  -1,
		MaximumTotal: -1,
	}
	m.Validate()
	if m.MinimumSize > m.MaximumSize {
		t.Errorf("expected %v > %v", m.MaximumSize, m.MinimumSize)
	}
	if m.MinimumSize < 0 {
		t.Errorf("expected %v > %v", m.MinimumSize, 0)
	}
	if m.MaximumSize < 0 {
		t.Errorf("expected %v > %v", m.MaximumSize, 0)
	}
	if m.MaximumTotal < 0 {
		t.Errorf("expected %v > %v", m.MaximumTotal, 0)
	}
}
