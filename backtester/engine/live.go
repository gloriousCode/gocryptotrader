package engine

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/ticker"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/ticker/live"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// RunLive is a proof of concept function that does not yet support multi currency usage
// It runs by constantly checking for new live datas and running through the list of events
// once new data is processed. It will run until application close event has been received
func (bt *BackTest) RunLive() error {
	log.Info(common.Backtester, "running backtester against live data")
	timeoutTimer := time.NewTimer(time.Minute * 5)
	// a frequent timer so that when a new candle is released by an exchange
	// that it can be processed quickly
	processEventTicker := time.NewTicker(time.Second)
	for {
		select {
		case <-bt.shutdown:
			return nil
		case <-timeoutTimer.C:
			return errLiveDataTimeout
		case <-processEventTicker.C:
			bt.Run()
		}
	}
}

// loadLiveDataLoop is an incomplete function to continuously retrieve exchange data on a loop
// from live. Its purpose is to be able to perform strategy analysis against current data
func (bt *BackTest) loadLiveDataLoop(resp *ticker.Data, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, checkInterval time.Duration) {
	var err error
	loadNewDataTimer := time.NewTimer(0)
	for {
		select {
		case <-bt.shutdown:
			return
		case <-loadNewDataTimer.C:
			log.Infof(common.Backtester, "fetching data for %v %v %v %v", exch.GetName(), a, fPair, checkInterval)
			loadNewDataTimer.Reset(checkInterval)
			err = bt.loadLiveData(resp, exch, fPair, a)
			if err != nil {
				log.Error(common.Backtester, err)
				return
			}
		}
	}
}

func (bt *BackTest) loadLiveData(resp *ticker.Data, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item) error {
	if resp == nil {
		return errNilData
	}
	if exch == nil {
		return errNilExchange
	}
	t, err := live.LoadData(context.TODO(),
		exch,
		fPair,
		a)
	if err != nil {
		return err
	}
	resp.AppendTicker(resp.UnderlyingPair, t)
	return nil
}
