package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var dataOptions = []string{
	"API",
	"CSV",
	"Database",
	"Live",
}

func main() {
	fmt.Print(common.ASCIILogo)
	fmt.Println("Welcome to the config generator!")
	reader := bufio.NewReader(os.Stdin)
	cfg := config.Config{
		StrategySettings: config.StrategySettings{
			Name:                         "",
			SimultaneousSignalProcessing: false,
			CustomSettings:               nil,
		},
		CurrencySettings: []config.CurrencySettings{},
		DataSettings: config.DataSettings{
			Interval:     0,
			DataType:     "",
			APIData:      nil,
			DatabaseData: nil,
			LiveData:     nil,
			CSVData:      nil,
		},
		PortfolioSettings: config.PortfolioSettings{
			Leverage: config.Leverage{},
			BuySide:  config.MinMax{},
			SellSide: config.MinMax{},
		},
		StatisticSettings:        config.StatisticSettings{},
		GoCryptoTraderConfigPath: "",
	}
	fmt.Println("-----Strategy Settings-----")
	strats, err := parseStrategySettings(&cfg, reader)

	fmt.Println("-----Exchange Settings-----")
	err = parseExchangeSettings(reader, &cfg, strats)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("-----Portfolio Settings-----")
	err = parsePortfolioSettings(reader, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("-----Data Settings-----")
	err = parseDataSettings(&cfg, reader)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("-----Statistics Settings-----")
	err = parseStatisticsSettings(&cfg, reader)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("-----GoCryptoTrader config Settings-----")
	fmt.Printf("Enter the path to the GoCryptoTrader config you wish to use. Leave blank to use \"%v\"\n", gctconfig.DefaultFilePath())
	cfg.GoCryptoTraderConfigPath = quickParse(reader)

	var resp []byte
	resp, err = json.MarshalIndent(cfg, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Write strat to file? If no, the output will be on screen y/n")
	yn := quickParse(reader)
	if yn == "y" || yn == "yes" {
		var wd string
		wd, err = os.Getwd()
		wd = filepath.Join(wd, cfg.StrategySettings.Name+"-"+cfg.Nickname, ".strat")
		fmt.Printf("Enter output file. If blank, will output to \"%v\"\n", wd)
		path := quickParse(reader)
		if path == "" {
			path = wd
		}
		err = os.WriteFile(path, resp, 0770)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Print(string(resp))
	}
	log.Println("Config creation complete!")
}

func parseStatisticsSettings(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println("Enter the risk free rate. eg 0.03")
	var err error
	cfg.StatisticSettings.RiskFreeRate, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return err
	}
	return nil
}

func parseDataSettings(cfg *config.Config, reader *bufio.Reader) error {
	var err error
	fmt.Println("Will you be using \"candle\" or \"trade\" data?")
	cfg.DataSettings.DataType = quickParse(reader)
	if cfg.DataSettings.DataType == common.TradeStr {
		fmt.Println("Trade data will be converted into candles")
	}
	fmt.Println("What candle time interval will you use?")
	cfg.DataSettings.Interval, err = parseKlineInterval(reader)
	if err != nil {
		return err
	}

	fmt.Println("Where will this data be sourced?")
	var choice string
	choice, err = parseDataChoice(reader, len(cfg.CurrencySettings) > 1)
	if err != nil {
		return err
	}
	switch choice {
	case "API":
		err = parseAPI(reader, cfg)
	case "Database":
		err = parseDatabase(reader, cfg)
	case "CSV":
		parseCSV(reader, cfg)
	case "Live":
		err = parseLive(reader, cfg)
	}
	if err != nil {
		return err
	}
	return nil
}

func parsePortfolioSettings(reader *bufio.Reader, cfg *config.Config) error {
	var err error
	fmt.Println("Will there be global portfolio buy-side limits? y/n")
	yn := quickParse(reader)
	if yn == "y" || yn == "yes" {
		cfg.PortfolioSettings.BuySide, err = minMaxParse("buy", reader)
		if err != nil {
			return err
		}
	}
	fmt.Println("Will there be global portfolio sell-side limits? y/n")
	yn = quickParse(reader)
	if yn == "y" || yn == "yes" {
		cfg.PortfolioSettings.SellSide, err = minMaxParse("sell", reader)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseExchangeSettings(reader *bufio.Reader, cfg *config.Config, strats []strategies.Handler) error {
	var err error
	addCurrency := "y"
	for strings.Contains(addCurrency, "y") {
		var currencySetting *config.CurrencySettings
		currencySetting, err = addCurrencySetting(reader)
		if err != nil {
			return err
		}

		cfg.CurrencySettings = append(cfg.CurrencySettings, *currencySetting)
		fmt.Println("Add another exchange currency setting? y/n")
		addCurrency = quickParse(reader)
	}

	if len(cfg.CurrencySettings) > 1 {
		for i := range strats {
			if strats[i].Name() == cfg.StrategySettings.Name &&
				strats[i].SupportsSimultaneousProcessing() {
				fmt.Println("Will this strategy use simultaneous processing? y/n")
				yn := quickParse(reader)
				if yn == "y" || yn == "yes" {
					cfg.StrategySettings.SimultaneousSignalProcessing = true
				}
			}
			break
		}
	}
	return nil
}

func parseStrategySettings(cfg *config.Config, reader *bufio.Reader) ([]strategies.Handler, error) {
	fmt.Println("Firstly, please select which strategy you wish to use")
	strats := strategies.GetStrategies()
	var strategiesToUse []string
	for i := range strats {
		fmt.Printf("%v. %s\n", i+1, strats[i].Name())
		strategiesToUse = append(strategiesToUse, strats[i].Name())
	}
	var err error
	cfg.StrategySettings.Name, err = parseStratName(quickParse(reader), strategiesToUse)

	fmt.Println("What is the goal of your strategy?")
	cfg.Goal = quickParse(reader)

	fmt.Println("Enter a nickname, it can help distinguish between different configs using the same strategy")
	cfg.Nickname = quickParse(reader)
	fmt.Println("Does this strategy have custom settings? y/n")
	customSettings := quickParse(reader)
	if strings.Contains(customSettings, "y") {
		cfg.StrategySettings.CustomSettings = customSettingsLoop(reader)
	}
	return strats, err
}

func parseAPI(reader *bufio.Reader, cfg *config.Config) error {
	cfg.DataSettings.APIData = &config.APIData{}
	var startDate, endDate, inclusive string
	var err error
	defaultStart := time.Now().Add(-time.Hour * 24 * 365)
	defaultEnd := time.Now()
	fmt.Printf("What is the start date? Leave blank for \"%v\"\n", defaultStart.Format(gctcommon.SimpleTimeFormat))
	startDate = quickParse(reader)
	if startDate != "" {
		cfg.DataSettings.APIData.StartDate, err = time.Parse(startDate, gctcommon.SimpleTimeFormat)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cfg.DataSettings.APIData.StartDate = defaultStart
	}

	fmt.Printf("What is the end date? Leave blank for \"%v\"\n", defaultStart.Format(gctcommon.SimpleTimeFormat))
	endDate = quickParse(reader)
	if endDate != "" {
		cfg.DataSettings.APIData.EndDate, err = time.Parse(endDate, gctcommon.SimpleTimeFormat)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cfg.DataSettings.APIData.EndDate = defaultEnd
	}
	fmt.Println("Is the end date inclusive? y/n")
	inclusive = quickParse(reader)
	cfg.DataSettings.APIData.InclusiveEndDate = inclusive == "y" || inclusive == "yes"

	return nil
}

func parseCSV(reader *bufio.Reader, cfg *config.Config) {
	cfg.DataSettings.CSVData = &config.CSVData{}
	fmt.Println("What is path of the CSV file to read?")
	cfg.DataSettings.CSVData.FullPath = quickParse(reader)
}

func parseDatabase(reader *bufio.Reader, cfg *config.Config) error {
	cfg.DataSettings.DatabaseData = &config.DatabaseData{}
	var input string
	var err error
	fmt.Printf("What is the start date? Leave blank for \"%v\"\n", time.Now().Add(-time.Hour*24*365).Format(gctcommon.SimpleTimeFormat))
	input = quickParse(reader)
	cfg.DataSettings.DatabaseData.StartDate, err = time.Parse(input, gctcommon.SimpleTimeFormat)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("What is the end date? Leave blank for \"%v\"\n", time.Now().Format(gctcommon.SimpleTimeFormat))
	input = quickParse(reader)
	cfg.DataSettings.DatabaseData.EndDate, err = time.Parse(input, gctcommon.SimpleTimeFormat)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Is the end date inclusive? y/n")
	input = quickParse(reader)
	cfg.DataSettings.DatabaseData.InclusiveEndDate = input == "y" || input == "yes"

	fmt.Println("Do you wish to override GoCryptoTrader's database config? y/n")
	input = quickParse(reader)
	if input == "y" || input == "yes" {
		cfg.DataSettings.DatabaseData.ConfigOverride = &database.Config{
			Enabled: true,
		}
		fmt.Println("Do you want database verbose output? y/n")
		input = quickParse(reader)
		cfg.DataSettings.DatabaseData.ConfigOverride.Verbose = input == "y" || input == "yes"

		fmt.Println("What database driver to use? eg sqlite")
		cfg.DataSettings.DatabaseData.ConfigOverride.Driver = quickParse(reader)

		fmt.Println("What is the database host? eg localhost")
		cfg.DataSettings.DatabaseData.ConfigOverride.Host = quickParse(reader)

		fmt.Println("What is the database username?")
		cfg.DataSettings.DatabaseData.ConfigOverride.Username = quickParse(reader)

		fmt.Println("What is the database password?")
		cfg.DataSettings.DatabaseData.ConfigOverride.Password = quickParse(reader)

		fmt.Println("What is the database? eg database.db")
		cfg.DataSettings.DatabaseData.ConfigOverride.Database = quickParse(reader)

		fmt.Println("What is the database SSLMode? eg disable")
		cfg.DataSettings.DatabaseData.ConfigOverride.SSLMode = quickParse(reader)

		fmt.Println("What is the database Port? eg 1337")
		input = quickParse(reader)
		var port float64
		port, err = strconv.ParseFloat(input, 64)
		if err != nil {
			log.Fatal(err)
		}
		cfg.DataSettings.DatabaseData.ConfigOverride.Port = uint16(port)
	}

	return nil
}

func parseLive(reader *bufio.Reader, cfg *config.Config) error {
	cfg.DataSettings.LiveData = &config.LiveData{}
	fmt.Println("Do you wish to use live trading? It's highly recommended that you do not. y/n")
	input := quickParse(reader)
	cfg.DataSettings.LiveData.RealOrders = input == "y" || input == "yes"
	if cfg.DataSettings.LiveData.RealOrders {
		fmt.Printf("Do you want to override GoCryptoTrader's API credentials for %s? y/n\n", cfg.CurrencySettings[0].ExchangeName)
		input = quickParse(reader)
		if input == "y" || input == "yes" {
			fmt.Println("What is the API key?")
			cfg.DataSettings.DatabaseData.ConfigOverride.Database = quickParse(reader)
			fmt.Println("What is the API secret?")
			cfg.DataSettings.DatabaseData.ConfigOverride.Database = quickParse(reader)
			fmt.Println("What is the Client ID?")
			cfg.DataSettings.DatabaseData.ConfigOverride.Database = quickParse(reader)
			fmt.Println("What is the 2FA seed?")
			cfg.DataSettings.DatabaseData.ConfigOverride.Database = quickParse(reader)
		}
	}
	return nil
}

func parseDataChoice(reader *bufio.Reader, multiCurrency bool) (string, error) {
	if multiCurrency {
		// live trading does not support multiple currencies
		dataOptions = dataOptions[:3]
	}
	for i := range dataOptions {
		fmt.Printf("%v. %s\n", i+1, dataOptions[i])
	}
	response := quickParse(reader)
	num, err := strconv.ParseFloat(response, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(dataOptions) {
			return "", errors.New("unknown option")
		}
		return dataOptions[intNum-1], nil
	}
	for i := range dataOptions {
		if strings.EqualFold(response, dataOptions[i]) {
			return dataOptions[i], nil
		}
	}
	return "", errors.New("unrecognised data option")
}

func parseKlineInterval(reader *bufio.Reader) (time.Duration, error) {
	allCandles := gctkline.SupportedIntervals
	for i := range allCandles {
		fmt.Printf("%v. %s\n", i+1, allCandles[i].Word())
	}
	response := quickParse(reader)
	num, err := strconv.ParseFloat(response, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(allCandles) {
			return 0, errors.New("unknown option")
		}
		return allCandles[intNum-1].Duration(), nil
	}
	for i := range allCandles {
		if strings.EqualFold(response, allCandles[i].Word()) {
			return allCandles[i].Duration(), nil
		}
	}
	return 0, errors.New("unrecognised interval")
}

func parseStratName(name string, strategiesToUse []string) (string, error) {
	num, err := strconv.ParseFloat(name, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(strategiesToUse) {
			return "", errors.New("unknown option")
		}
		return strategiesToUse[intNum-1], nil
	}
	for i := range strategiesToUse {
		if strings.EqualFold(name, strategiesToUse[i]) {
			return strategiesToUse[i], nil
		}
	}
	return "", errors.New("unrecognised strategy")
}

func customSettingsLoop(reader *bufio.Reader) map[string]interface{} {
	resp := make(map[string]interface{})
	customSettingField := "loopTime!"
	for customSettingField != "" {
		fmt.Println("Enter a custom setting name. Enter nothing to stop")
		customSettingField = quickParse(reader)
		if customSettingField != "" {
			fmt.Println("Enter a custom setting value")
			resp[customSettingField] = quickParse(reader)
		}
	}
	return resp
}

func addCurrencySetting(reader *bufio.Reader) (*config.CurrencySettings, error) {
	setting := config.CurrencySettings{
		BuySide:  config.MinMax{},
		SellSide: config.MinMax{},
	}
	fmt.Println("Enter the exchange name. eg Binance")
	setting.ExchangeName = quickParse(reader)

	fmt.Println("Please select an asset")
	supported := asset.Supported()
	for i := range supported {
		fmt.Printf("%v. %s\n", i+1, supported[i])
	}
	response := quickParse(reader)
	num, err := strconv.ParseFloat(response, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(dataOptions) {
			return nil, errors.New("unknown option")
		}
		return dataOptions[intNum-1], nil
	}
	for i := range dataOptions {
		if strings.EqualFold(response, dataOptions[i]) {
			return dataOptions[i], nil
		}
	}
	setting.Asset = quickParse(reader)
	if setting.Asset == "help" {
		supported := asset.Supported()
		for i := range supported {
			fmt.Println(supported[i].String())
		}
		fmt.Println("Enter the asset. eg spot")
		setting.Asset = quickParse(reader)
	}

	fmt.Println("Enter the currency base. eg BTC")
	setting.Base = quickParse(reader)

	fmt.Println("Enter the currency quote. eg USDT")
	setting.Quote = quickParse(reader)

	fmt.Println("Enter the initial funds. eg 10000")
	var err error
	setting.InitialFunds, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return nil, err
	}

	fmt.Println("Enter the maker-fee. eg 0.001")
	setting.MakerFee, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return nil, err
	}
	fmt.Println("Enter the taker-fee. eg 0.01")
	setting.TakerFee, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return nil, err
	}

	fmt.Println("Will there be buy-side limits? y/n")
	yn := quickParse(reader)
	if yn == "y" || yn == "yes" {
		setting.BuySide, err = minMaxParse("buy", reader)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("Will there be sell-side limits? y/n")
	yn = quickParse(reader)
	if yn == "y" || yn == "yes" {
		setting.SellSide, err = minMaxParse("sell", reader)
		if err != nil {
			return nil, err
		}
	}

	return &setting, nil
}

func minMaxParse(buySell string, reader *bufio.Reader) (config.MinMax, error) {
	resp := config.MinMax{}
	var err error
	fmt.Printf("What is the maximum %s size? eg 1\n", buySell)
	resp.MaximumSize, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return resp, err
	}
	fmt.Printf("What is the minimum %s size? eg 0.1\n", buySell)
	resp.MinimumSize, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return resp, err
	}
	fmt.Printf("What is the maximum spend %s buy? eg 12000\n", buySell)
	resp.MaximumTotal, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func quickParse(reader *bufio.Reader) string {
	customSettingField, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(customSettingField, "\n", "", 1)
}
