package data_filler

import (
	"flag"
	"log"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func main() {
	var startDate, endDate, exchange, assetType, pair string
	var interval int64

	flag.String(
		startDate,
		"startdate",
		time.Now().Add(-time.Hour*24*7).String(),
	)
	flag.String(
		endDate,
		"enddate",
		time.Now().String(),
	)
	flag.String(
		exchange,
		"exchange",
		"binance",
	)
	flag.String(
		assetType,
		"asset",
		"SPOT",
	)
	flag.String(
		pair,
		"pair",
		"BTC-USDT",
	)
	flag.Int64(
		interval,
		86400,
		"interval represented in seconds",
	)
	flag.Parse()

	bot, err := engine.NewFromSettings(nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	err = bot.LoadExchange(exchange, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	exch := bot.GetExchangeByName(exchange)

	var currencyPair currency.Pair
	currencyPair, err = currency.NewPairFromString(pair)

	var a asset.Item
	a, err = asset.New(assetType)
	if err != nil {
		log.Fatal(err)
	}

	var start, end time.Time
	start, err = time.Parse(common.SimpleTimeFormat, startDate)
	if err != nil {

	}
	end, err = time.Parse(common.SimpleTimeFormat, endDate)

	candleInterval := time.Duration(interval) * time.Second
	var butts kline.Item
	err = dates.VerifyResultsHaveData(ret.Candles)
	if err != nil {
		log.Fatal(err)
	}
	butts, err = exch.GetHistoricCandlesExtended(currencyPair, a, start, end, kline.Interval(candleInterval))

}
