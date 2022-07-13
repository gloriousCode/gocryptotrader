package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	yes = "yes"
	y   = "y"
)

var (
	apiKey     = ""
	apiSecret  = ""
	subAccount = ""
)

func main() {
	f := ftx.FTX{}
	f.SetDefaults()
	response := PositionCostingsEstimate{}

	cfg, err := f.GetDefaultConfig()
	closeOnErr(err)

	err = f.SetupDefaults(cfg)
	closeOnErr(err)

	f.SetCredentials(apiKey, apiSecret, "", subAccount, "", "")
	b := f.GetBase()
	b.API.AuthenticatedSupport = true
	b.API.AuthenticatedWebsocketSupport = true
	f.CurrencyPairs.Pairs[asset.Futures].Enabled = f.CurrencyPairs.Pairs[asset.Futures].Available
	f.CurrencyPairs.Pairs[asset.Spot].Enabled = f.CurrencyPairs.Pairs[asset.Spot].Available
	reader := bufio.NewReader(os.Stdin)

	collateral, err := f.GetCollateral(context.Background(), false)
	closeOnErr(err)
	response.AvailableCollateral = collateral.CollateralAvailable
	fmt.Printf("You have %v collateral available\n", collateral.CollateralAvailable)
	fmt.Println("how much do you want to spend?")
	sizeStr := quickParse(reader)
	spendAmount, err := strconv.ParseFloat(sizeStr, 64)
	closeOnErr(err)

	if spendAmount > collateral.CollateralAvailable.InexactFloat64() {
		fmt.Println("spendAmount greater than collateral, do you want to use margin?")
		yn := quickParse(reader)
		if yn == y || yn == yes {
			response.UsingMargin = true
		} else {
			fmt.Println("well then you can't do this!")
			os.Exit(0)
		}
	}
	spotPairs, err := f.CurrencyPairs.GetPairs(asset.Spot, true)
	closeOnErr(err)
	futuresPairs, err := f.CurrencyPairs.GetPairs(asset.Futures, true)
	closeOnErr(err)

	response.LongTrade.SetPairValues(reader, order.Long, futuresPairs, spotPairs, &f)
	response.ShotTrade.SetPairValues(reader, order.Short, futuresPairs, spotPairs, &f)

	acc, err := f.GetAccountInfo(context.Background())
	closeOnErr(err)
	response.FeeRate = decimal.NewFromFloat(acc.TakerFee)

	response.TheThing(&f)

	fmt.Printf("When will these positions close? format: %s\n", common.SimpleTimeFormat)

	positionCloseStr := quickParse(reader)
	positionCloseTime, err := time.Parse(common.SimpleTimeFormat, positionCloseStr)
	closeOnErr(err)
	response.StartDate = time.Now()
	response.EndDate = positionCloseTime

	positionLife := time.Until(positionCloseTime)
	positionHours := decimal.NewFromFloat(positionLife.Hours())

	isLongPerp, err := f.IsPerpetualFutureCurrency(response.LongTrade.Asset, response.LongTrade.Pair)
	closeOnErr(err)
	if isLongPerp {
		response.LongTrade.SetPairFundingRates(&f, positionLife, positionHours, response.FeeRate)
	}

	isShortPerp, err := f.IsPerpetualFutureCurrency(response.ShotTrade.Asset, response.ShotTrade.Pair)
	closeOnErr(err)
	if isShortPerp {
		response.ShotTrade.SetPairFundingRates(&f, positionLife, positionHours, response.FeeRate)
	}

	response.TotalBorrowRates = response.ShotTrade.BorrowRateCosts.Add(response.LongTrade.BorrowRateCosts)

	log.Printf("available collateral: %v\n", response.AvailableCollateral)
	log.Printf("desired spend: %v\n", response.DesiredSpend)
	log.Printf("position start: %v\n", response.StartDate)
	log.Printf("position end: %v\n", response.EndDate)
	log.Printf("fee rate: %v\n", response.FeeRate)
	log.Println("-----Long position details-----")
	log.Printf("asset: %v\n", response.LongTrade.Asset)
	log.Printf("pair: %v\n", response.LongTrade.Pair)
	log.Printf("is perpetual future: %v\n", response.LongTrade.IsPerp)
	log.Printf("position size: %v\n", response.LongTrade.Size)
	log.Printf("perp cost using predicted rate: %v\n", response.LongTrade.RateCosts.PredictedRate)
	log.Printf("perp cost using latest rate: %v\n", response.LongTrade.RateCosts.LatestRate)
	log.Printf("perp cost using year average rate: %v\n", response.LongTrade.RateCosts.YearAverageRate)
	log.Printf("perp cost using position length average rate: %v\n", response.LongTrade.RateCosts.PositionTimeAverageRate)

	log.Println("-----Short position details-----")
	log.Printf("asset: %v\n", response.ShotTrade.Asset)
	log.Printf("pair: %v\n", response.ShotTrade.Pair)
	log.Printf("is perpetual future: %v\n", response.ShotTrade.IsPerp)
	log.Printf("position size: %v\n", response.ShotTrade.Size)
	log.Printf("perp cost using predicted rate: %v\n", response.ShotTrade.RateCosts.PredictedRate)
	log.Printf("perp cost using latest rate: %v\n", response.ShotTrade.RateCosts.LatestRate)
	log.Printf("perp cost using year average rate: %v\n", response.ShotTrade.RateCosts.YearAverageRate)
	log.Printf("perp cost using position length average rate: %v\n", response.ShotTrade.RateCosts.PositionTimeAverageRate)

}

func (p *PositionCostingsEstimate) TheThing(f *ftx.FTX) {
	longScale, err := f.ScaleCollateral(context.TODO(), &order.CollateralCalculator{
		CollateralCurrency: p.LongTrade.Pair.Base,
		Asset:              p.LongTrade.Asset,
		Side:               order.Long,
		USDPrice:           p.LongTrade.LastPrice,
		IsForNewPosition:   true,
		FreeCollateral:     p.AvailableCollateral,
	})
	closeOnErr(err)

	initialAmount := p.DesiredSpend.Mul(longScale.Weighting).Div(p.LongTrade.LastPrice)
	sizedAmount := initialAmount.Mul(p.LongTrade.LastPrice)
	scaledCollateralFromAmount := sizedAmount.Mul(p.LongTrade.Weight)
	excess := p.AvailableCollateral.Sub(sizedAmount).Add(scaledCollateralFromAmount)
	if excess.IsNegative() {
		os.Exit(-1)
	}
	p.LongTrade.Size = sizedAmount
	p.ShotTrade.Size = sizedAmount

}

func (p *PairData) SetPairValues(reader *bufio.Reader, side order.Side, futuresPairs currency.Pairs, spotPairs currency.Pairs, f *ftx.FTX) {
	fmt.Printf("What currency pair do you want to %v? format 'btcusd'", side)
	longStr := quickParse(reader)
	var err error
	p.Asset = asset.Futures
	p.Pair, err = futuresPairs.DeriveFrom(longStr)
	if err != nil {
		p.Pair, err = spotPairs.DeriveFrom(longStr)
		closeOnErr(err)
		p.Asset = asset.Spot
	}

	longTick, err := f.FetchTicker(context.Background(), p.Pair, p.Asset)
	closeOnErr(err)
	p.LastPrice = decimal.NewFromFloat(longTick.Last)
	scaling, err := f.ScaleCollateral(context.Background(), &order.CollateralCalculator{
		CollateralCurrency: p.Pair.Base,
		Asset:              p.Asset,
		Side:               order.Long,
		IsForNewPosition:   true,
	})
	closeOnErr(err)
	p.Weight = scaling.Weighting
}

func (p *PairData) SetPairFundingRates(f *ftx.FTX, positionLife time.Duration, positionHours, feeRate decimal.Decimal) {
	shortFundingRates, err := f.GetFundingRates(context.Background(), &order.FundingRatesRequest{
		Asset:                p.Asset,
		Pairs:                currency.Pairs{p.Pair},
		StartDate:            time.Now().Add(-positionLife),
		EndDate:              time.Now(),
		IncludePredictedRate: true,
	})
	closeOnErr(err)
	if len(shortFundingRates) != 1 {
		os.Exit(-1)
	}
	var averageRate float64
	for i := range shortFundingRates[0].FundingRates {
		averageRate += shortFundingRates[0].FundingRates[i].Rate.InexactFloat64()
	}
	averageRate /= float64(len(shortFundingRates[0].FundingRates))
	ar := decimal.NewFromFloat(averageRate)
	p.Rates.PositionTimeAverageRate = ar.Mul(positionHours)

	shortFundingRates, err = f.GetFundingRates(context.Background(), &order.FundingRatesRequest{
		Asset:                p.Asset,
		Pairs:                currency.Pairs{p.Pair},
		StartDate:            time.Now().Add(-time.Hour * 24 * 365),
		EndDate:              time.Now(),
		IncludePredictedRate: true,
	})
	closeOnErr(err)
	if len(shortFundingRates) != 1 {
		os.Exit(-1)
	}
	for i := range shortFundingRates[0].FundingRates {
		averageRate += shortFundingRates[0].FundingRates[i].Rate.InexactFloat64()
	}
	averageRate /= float64(len(shortFundingRates[0].FundingRates))
	ar = decimal.NewFromFloat(averageRate)
	p.Rates.YearAverageRate = ar.Mul(positionHours)
	p.Rates.PredictedRate = shortFundingRates[0].PredictedUpcomingRate.Rate
	p.Rates.LatestRate = shortFundingRates[0].LatestRate.Rate

	var (
		one         = decimal.NewFromInt(1)
		fiveHundred = decimal.NewFromInt(500)
	)

	p.RateCosts.YearAverageRate = p.Rates.YearAverageRate.Mul(one.Add(fiveHundred.Mul(feeRate))).Mul(positionHours)
	p.RateCosts.PositionTimeAverageRate = p.Rates.PositionTimeAverageRate.Mul(one.Add(fiveHundred.Mul(feeRate))).Mul(positionHours)
	p.RateCosts.PredictedRate = p.Rates.PredictedRate.Mul(one.Add(fiveHundred.Mul(feeRate))).Mul(positionHours)
	p.RateCosts.LatestRate = p.Rates.LatestRate.Mul(one.Add(fiveHundred.Mul(feeRate))).Mul(positionHours)
}

type PositionCostingsEstimate struct {
	LongTrade           PairData
	ShotTrade           PairData
	StartDate           time.Time
	EndDate             time.Time
	AvailableCollateral decimal.Decimal
	DesiredSpend        decimal.Decimal
	UsingMargin         bool
	MarginLeverage      decimal.Decimal
	TotalBorrowRates    decimal.Decimal
	FeeRate             decimal.Decimal
}

type PairData struct {
	Asset           asset.Item
	Pair            currency.Pair
	IsPerp          bool
	Size            decimal.Decimal
	Weight          decimal.Decimal
	Rates           FundingRates
	RateCosts       FundingRates
	BorrowRateCosts decimal.Decimal
	LastPrice       decimal.Decimal
	Fee             decimal.Decimal
	AllCostings     decimal.Decimal
}

type FundingRates struct {
	LatestRate              decimal.Decimal
	PredictedRate           decimal.Decimal
	PositionTimeAverageRate decimal.Decimal
	YearAverageRate         decimal.Decimal
}

func closeOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func quickParse(reader *bufio.Reader) string {
	customSettingField, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	customSettingField = strings.Replace(customSettingField, "\r", "", -1)
	return strings.Replace(customSettingField, "\n", "", -1)
}
