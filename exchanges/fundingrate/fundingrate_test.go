package fundingrate

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestMain(m *testing.M) {
	err := dispatch.Start(1, dispatch.DefaultJobsLimit)
	if !errors.Is(err, nil) {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestSubscribeFundingRate(t *testing.T) {
	t.Parallel()
	s := newService()
	_, err := s.subscribeFundingRate("", currency.EMPTYPAIR, asset.Empty)
	if !errors.Is(err, errFundingRateNotFound) {
		t.Error(err)
	}

	p := currency.NewPair(currency.BTC, currency.USD)
	err = s.processFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       p,
		LatestRate: Rate{Time: time.Now()},
		Asset:      asset.Futures,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	_, err = s.subscribeFundingRate("test", p, asset.Futures)
	if !errors.Is(err, nil) {
		t.Error("cannot subscribe to fundingRate", err)
	}
}

func TestSubscribeToExchangeFundingRates(t *testing.T) {
	t.Parallel()
	s := newService()
	_, err := s.subscribeToExchangeFundingRates("")
	if !errors.Is(err, ErrExchangeNameUnset) {
		t.Error(err)
	}

	err = s.processFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       currency.NewPair(currency.BTC, currency.USD),
		LatestRate: Rate{Time: time.Now()},
		Asset:      asset.Futures,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	_, err = s.subscribeToExchangeFundingRates("test")
	if !errors.Is(err, nil) {
		t.Error(err, err)
	}
}

func TestGetFundingRate(t *testing.T) {
	t.Parallel()
	s := newService()
	_, err := s.getFundingRate("", currency.EMPTYPAIR, asset.Empty)
	if !errors.Is(err, ErrExchangeNameUnset) {
		t.Error(err)
	}
	_, err = s.getFundingRate("test", currency.EMPTYPAIR, asset.Empty)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	p := currency.NewPair(currency.BTC, currency.USD)
	_, err = s.getFundingRate("test", p, asset.Empty)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}

	priceStruct := LatestRateResponse{
		Pair: p,
		LatestRate: Rate{
			Time: time.Now(),
			Rate: decimal.NewFromInt(1337),
		},
		Exchange: "bitfinex",
		Asset:    asset.Futures,
	}

	err = s.processFundingRate(&priceStruct)
	if !errors.Is(err, nil) {
		t.Fatal("ProcessFundingRate error", err)
	}

	fundingRatePrice, err := s.getFundingRate("bitfinex", p, asset.Futures)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if !fundingRatePrice.Pair.Equal(p) {
		t.Error("fundingRate fundingRatePrice.CurrencyPair value is incorrect")
	}

	_, err = s.getFundingRate("blah", p, asset.Futures)
	if !errors.Is(err, errFundingRateNotFound) {
		t.Error(err)
	}
	p.Base = currency.ETH
	_, err = s.getFundingRate("bitfinex", p, asset.Futures)
	if !errors.Is(err, errFundingRateNotFound) {
		t.Error(err)
	}
}

func TestProcessFundingRate(t *testing.T) {
	t.Parallel()
	s := newService()
	p := currency.NewPair(currency.BTC, currency.USD)

	err := s.processFundingRate(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Error(err)
	}

	err = s.processFundingRate(&LatestRateResponse{})
	if !errors.Is(err, ErrExchangeNameUnset) {
		t.Error(err)
	}

	err = s.processFundingRate(&LatestRateResponse{
		Exchange: "test",
	})
	if !errors.Is(err, errPairNotSet) {
		t.Error(err)
	}

	err = s.processFundingRate(&LatestRateResponse{
		Exchange: "test",
		Pair:     p,
	})
	if !errors.Is(err, errFundingRateTimeUnset) {
		t.Error(err)
	}

	err = s.processFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       p,
		LatestRate: Rate{Time: time.Now()},
	})
	if !errors.Is(err, errAssetTypeNotSet) {
		t.Error(err)
	}

	err = s.processFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       p,
		LatestRate: Rate{Time: time.Now()},
		Asset:      asset.Spot,
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}

	err = s.processFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       p,
		LatestRate: Rate{Time: time.Now()},
		Asset:      asset.Futures,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
}

func TestGetAssociation(t *testing.T) {
	t.Parallel()
	s := newService()
	_, err := s.getAssociations("")
	if !errors.Is(err, ErrExchangeNameUnset) {
		t.Errorf("received: %v but expected: %v", err, ErrExchangeNameUnset)
	}

	s.mux = nil
	_, err = s.getAssociations("getassociation")
	if !errors.Is(err, dispatch.ErrMuxIsNil) {
		t.Error(err)
	}
}

// the following tests use the internal service variable
func TestSubscribeFundingRateInternal(t *testing.T) {
	t.Parallel()
	_, err := SubscribeFundingRate("", currency.EMPTYPAIR, asset.Empty)
	if !errors.Is(err, errFundingRateNotFound) {
		t.Error(err)
	}

	p := currency.NewPair(currency.BTC, currency.USD)
	err = service.processFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       p,
		LatestRate: Rate{Time: time.Now()},
		Asset:      asset.Futures,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	_, err = SubscribeFundingRate("test", p, asset.Futures)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
}

func TestSubscribeToExchangeFundingRatesInternal(t *testing.T) {
	t.Parallel()
	_, err := SubscribeToExchangeFundingRates("")
	if !errors.Is(err, ErrExchangeNameUnset) {
		t.Error(err)
	}

	p := currency.NewPair(currency.BTC, currency.USD)
	err = service.processFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       p,
		LatestRate: Rate{Time: time.Now()},
		Asset:      asset.Futures,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	_, err = SubscribeToExchangeFundingRates("test")
	if !errors.Is(err, nil) {
		t.Error(err)
	}
}

func TestGetFundingRateInternal(t *testing.T) {
	t.Parallel()
	p := currency.NewPair(currency.BTC, currency.USD)
	err := service.processFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       p,
		LatestRate: Rate{Time: time.Now()},
		Asset:      asset.Futures,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	_, err = GetFundingRate("test", p, asset.Futures)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
}

func TestReturnAllRatesOnUpdateInternal(t *testing.T) {
	t.Parallel()
	resp := ReturnAllRatesOnUpdate(time.Nanosecond)
	if resp != nil {
		t.Error("expected nil")
	}

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		p := currency.NewPair(currency.BTC, currency.USD)
		err := service.processFundingRate(&LatestRateResponse{
			Exchange:   "test",
			Pair:       p,
			LatestRate: Rate{Time: time.Now()},
			Asset:      asset.Futures,
		})
		if !errors.Is(err, nil) {
			t.Error(err)
		}
	}()
	resp = ReturnAllRatesOnUpdate(time.Second)
	<-ch
	if len(resp) != 1 {
		t.Error("expected 1")
	}
}

func TestProcessFundingRateInternal(t *testing.T) {
	t.Parallel()
	err := ProcessFundingRate(&LatestRateResponse{
		Exchange:   "test",
		Pair:       currency.NewPair(currency.BTC, currency.USD),
		LatestRate: Rate{Time: time.Now()},
		Asset:      asset.Futures,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
}

func TestNewService(t *testing.T) {
	t.Parallel()
	s := newService()
	if s == nil {
		t.Error("expected service")
	}
}
