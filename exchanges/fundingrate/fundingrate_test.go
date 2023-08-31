package fundingrate

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestMain(m *testing.M) {
	err := dispatch.Start(1, dispatch.DefaultJobsLimit)
	if err != nil {
		log.Fatal(err)
	}

	cpyMux = service.mux

	os.Exit(m.Run())
}

var cpyMux *dispatch.Mux

func TestSubscribeFundingRate(t *testing.T) {
	_, err := SubscribeFundingRate("", currency.EMPTYPAIR, asset.Empty)
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USD)

	// force error
	service.mux = nil
	err = ProcessFundingRate(&Price{
		Pair:         p,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot})
	if err == nil {
		t.Error("error cannot be nil")
	}

	sillyP := p
	sillyP.Base = currency.GALA_NEO
	err = ProcessFundingRate(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot})
	if err == nil {
		t.Error("error cannot be nil")
	}

	sillyP.Quote = currency.AAA
	err = ProcessFundingRate(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = ProcessFundingRate(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    asset.DownsideProfitContract,
	})
	if err == nil {
		t.Error("error cannot be nil")
	}
	// reinstate mux
	service.mux = cpyMux

	err = ProcessFundingRate(&Price{
		Pair:         p,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot})
	if err != nil {
		t.Fatal(err)
	}

	_, err = SubscribeFundingRate("subscribetest", p, asset.Spot)
	if err != nil {
		t.Error("cannot subscribe to fundingRate", err)
	}
}

func TestSubscribeToExchangeFundingRates(t *testing.T) {
	_, err := SubscribeToExchangeFundingRates("")
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USD)

	err = ProcessFundingRate(&Price{
		Pair:         p,
		ExchangeName: "subscribeExchangeTest",
		AssetType:    asset.Spot})
	if err != nil {
		t.Error(err)
	}

	_, err = SubscribeToExchangeFundingRates("subscribeExchangeTest")
	if err != nil {
		t.Error("error cannot be nil", err)
	}
}

func TestGetFundingRate(t *testing.T) {
	newPair, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	priceStruct := Price{
		Pair:         newPair,
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
		ExchangeName: "bitfinex",
		AssetType:    asset.Spot,
	}

	err = ProcessFundingRate(&priceStruct)
	if err != nil {
		t.Fatal("ProcessFundingRate error", err)
	}

	fundingRatePrice, err := GetFundingRate("bitfinex", newPair, asset.Spot)
	if err != nil {
		t.Errorf("LatestRateWithDispatchIDs GetFundingRate init error: %s", err)
	}
	if !fundingRatePrice.Pair.Equal(newPair) {
		t.Error("fundingRate fundingRatePrice.CurrencyPair value is incorrect")
	}

	_, err = GetFundingRate("blah", newPair, asset.Spot)
	if err == nil {
		t.Fatal("TestGetFundingRate returned nil error on invalid exchange")
	}

	newPair.Base = currency.ETH
	_, err = GetFundingRate("bitfinex", newPair, asset.Spot)
	if err == nil {
		t.Fatal("TestGetFundingRate returned fundingRate for invalid first currency")
	}

	btcltcPair, err := currency.NewPairFromStrings("BTC", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetFundingRate("bitfinex", btcltcPair, asset.Spot)
	if err == nil {
		t.Fatal("TestGetFundingRate returned fundingRate for invalid second currency")
	}

	priceStruct.PriceATH = 9001
	priceStruct.Pair.Base = currency.ETH
	priceStruct.AssetType = asset.DownsideProfitContract
	err = ProcessFundingRate(&priceStruct)
	if err != nil {
		t.Fatal("ProcessFundingRate error", err)
	}

	fundingRatePrice, err = GetFundingRate("bitfinex", newPair, asset.DownsideProfitContract)
	if err != nil {
		t.Errorf("LatestRateWithDispatchIDs GetFundingRate init error: %s", err)
	}

	if fundingRatePrice.PriceATH != 9001 {
		t.Error("fundingRate fundingRatePrice.PriceATH value is incorrect")
	}

	_, err = GetFundingRate("bitfinex", newPair, asset.UpsideProfitContract)
	if err == nil {
		t.Error("LatestRateWithDispatchIDs GetFundingRate error cannot be nil")
	}

	priceStruct.AssetType = asset.UpsideProfitContract
	err = ProcessFundingRate(&priceStruct)
	if err != nil {
		t.Fatal("ProcessFundingRate error", err)
	}

	// process update again
	err = ProcessFundingRate(&priceStruct)
	if err != nil {
		t.Fatal("ProcessFundingRate error", err)
	}
}

func TestFindLast(t *testing.T) {
	cp := currency.NewPair(currency.BTC, currency.XRP)
	_, err := FindLast(cp, asset.Spot)
	if !errors.Is(err, errFundingRateNotFound) {
		t.Errorf("received: %v but expected: %v", err, errFundingRateNotFound)
	}

	err = service.update(&Price{Last: 0, ExchangeName: "testerinos", Pair: cp, AssetType: asset.Spot})
	if err != nil {
		t.Fatal(err)
	}

	_, err = FindLast(cp, asset.Spot)
	if !errors.Is(err, errInvalidFundingRate) {
		t.Errorf("received: %v but expected: %v", err, errInvalidFundingRate)
	}

	err = service.update(&Price{Last: 1337, ExchangeName: "testerinos", Pair: cp, AssetType: asset.Spot})
	if err != nil {
		t.Fatal(err)
	}

	last, err := FindLast(cp, asset.Spot)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v but expected: %v", err, nil)
	}

	if last != 1337 {
		t.Fatal("unexpected value")
	}
}

func TestProcessFundingRate(t *testing.T) { // non-appending function to fundingRates
	exchName := "bitstamp"
	newPair, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	priceStruct := Price{
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	err = ProcessFundingRate(&priceStruct)
	if err == nil {
		t.Fatal("empty exchange should throw an err")
	}

	priceStruct.ExchangeName = exchName

	// test for empty pair
	err = ProcessFundingRate(&priceStruct)
	if err == nil {
		t.Fatal("empty pair should throw an err")
	}

	// test for empty asset type
	priceStruct.Pair = newPair
	err = ProcessFundingRate(&priceStruct)
	if err == nil {
		t.Fatal("ProcessFundingRate error cannot be nil")
	}
	priceStruct.AssetType = asset.Spot
	// now process a valid fundingRate
	err = ProcessFundingRate(&priceStruct)
	if err != nil {
		t.Fatal("ProcessFundingRate error", err)
	}
	result, err := GetFundingRate(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessFundingRate failed to create and return a new fundingRate")
	}
	if !result.Pair.Equal(newPair) {
		t.Fatal("TestProcessFundingRate pair mismatch")
	}

	err = ProcessFundingRate(&Price{
		ExchangeName: "Bitfinex",
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		AssetType:    asset.Margin,
		Bid:          1337,
		Ask:          1337,
	})
	if !errors.Is(err, errBidEqualsAsk) {
		t.Errorf("received: %v but expected: %v", err, errBidEqualsAsk)
	}

	err = ProcessFundingRate(&Price{
		ExchangeName: "Bitfinex",
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		AssetType:    asset.Margin,
		Bid:          1338,
		Ask:          1336,
	})
	if !errors.Is(err, errBidGreaterThanAsk) {
		t.Errorf("received: %v but expected: %v", err, errBidGreaterThanAsk)
	}

	err = ProcessFundingRate(&Price{
		ExchangeName: "Bitfinex",
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		AssetType:    asset.MarginFunding,
		Bid:          1338,
		Ask:          1336,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v but expected: %v", err, nil)
	}

	// now test for processing a pair with a different quote currency
	newPair, err = currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	priceStruct.Pair = newPair
	err = ProcessFundingRate(&priceStruct)
	if err != nil {
		t.Fatal("ProcessFundingRate error", err)
	}
	_, err = GetFundingRate(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessFundingRate failed to create and return a new fundingRate")
	}
	_, err = GetFundingRate(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessFundingRate failed to return an existing fundingRate")
	}

	// now test for processing a pair which has a different base currency
	newPair, err = currency.NewPairFromStrings("LTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	priceStruct.Pair = newPair
	err = ProcessFundingRate(&priceStruct)
	if err != nil {
		t.Fatal("ProcessFundingRate error", err)
	}
	_, err = GetFundingRate(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessFundingRate failed to create and return a new fundingRate")
	}
	_, err = GetFundingRate(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessFundingRate failed to return an existing fundingRate")
	}

	type quick struct {
		Name string
		P    currency.Pair
		TP   Price
	}

	var testArray []quick

	_ = rand.NewSource(time.Now().Unix())

	var wg sync.WaitGroup
	var sm sync.Mutex

	var catastrophicFailure bool
	for i := 0; i < 500; i++ {
		if catastrophicFailure {
			break
		}

		wg.Add(1)
		go func() {
			//nolint:gosec // no need to import crypo/rand for testing
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10)
			newPairs, err := currency.NewPairFromStrings("BTC"+strconv.FormatInt(rand.Int63(), 10), //nolint:gosec // no need to import crypo/rand for testing
				"USD"+strconv.FormatInt(rand.Int63(), 10)) //nolint:gosec // no need to import crypo/rand for testing
			if err != nil {
				log.Fatal(err)
			}

			tp := Price{
				Pair:         newPairs,
				Last:         rand.Float64(), //nolint:gosec // no need to import crypo/rand for testing
				ExchangeName: newName,
				AssetType:    asset.Spot,
			}

			sm.Lock()
			err = ProcessFundingRate(&tp)
			if err != nil {
				t.Error(err)
				catastrophicFailure = true
				return
			}

			testArray = append(testArray, quick{Name: newName, P: newPairs, TP: tp})
			sm.Unlock()
			wg.Done()
		}()
	}

	if catastrophicFailure {
		t.Fatal("ProcessFundingRate error")
	}

	wg.Wait()

	for _, test := range testArray {
		wg.Add(1)
		fatalErr := false
		go func(test quick) {
			result, err := GetFundingRate(test.Name, test.P, asset.Spot)
			if err != nil {
				fatalErr = true
				return
			}

			if result.Last != test.TP.Last {
				t.Error("TestProcessFundingRate failed bad values")
			}

			wg.Done()
		}(test)

		if fatalErr {
			t.Fatal("TestProcessFundingRate failed to retrieve new fundingRate")
		}
	}
	wg.Wait()
}

func TestGetAssociation(t *testing.T) {
	_, err := service.getAssociations("")
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Errorf("received: %v but expected: %v", err, errExchangeNameIsEmpty)
	}

	service.mux = nil

	_, err = service.getAssociations("getassociation")
	if err == nil {
		t.Error("error cannot be nil")
	}

	service.mux = cpyMux
}
