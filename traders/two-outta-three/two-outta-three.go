package two_outta_three

import (
	"errors"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
	"sync/atomic"
	"time"
)

// Setup takes in important details for ONE INSTANCE of an auto trader
func (t *TOT) Setup(exchangeName string, timer time.Duration, cp currency.Pair, assetType asset.Item, interval kline.Interval) error {
	var trader TOT
	e := engine.GetExchangeByName(exchangeName)
	if e == nil {
		return errors.New("cmon dude")
	}
	if cp.IsEmpty() {
		return errors.New("bad currency")
	}
	if  e.GetBase().CurrencyPairs.Pairs[assetType].Enabled.Contains(cp, true) {
		return errors.New("currency not available in exchange")
	}
	trader.Lock()
	trader.exchange = e
	trader.shutdown = make(chan struct{})
	trader.TickerTime = timer
	trader.currencyPair = cp
	trader.assetType = assetType
	trader.Interval = interval
	atomic.CompareAndSwapInt32(&trader.IsSetup, 0, 1)
	trader.Unlock()
	return nil
}

// Start runs the application
func (t *TOT)  Start() error {
	if atomic.LoadInt32(&t.IsSetup) == 0 {
		return errors.New("need to setup before starting")
	}
	if atomic.LoadInt32(&t.Started) == 1 {
		return errors.New("already started")
	}
	t.Monitor()
	return nil
}

func (t *TOT) Stop() {
	close(t.shutdown)
}

func (t *TOT) DoesMarketMatchBuyConditions() (bool, error) {
	if !t.HasCapital() {
		return false, nil
	}
	var rsiPass, dpoPass, mfiPass bool
	ohlcv, err := t.exchange.GetHistoricCandles(t.currencyPair, t.assetType, time.Now().Add(-time.Hour * 24), time.Now(), t.Interval)
	if err !=  nil {
		return false, err
	}
	var closeValus []float64
	for i := range ohlcv.Candles {
		closeValus = append(closeValus, ohlcv.Candles[i].Close)
	}
	rsi := indicators.RSI(closeValus, 14)
	if rsi[len(rsi) -1] > 0 {
		rsiPass = true
	}
	if (rsiPass && dpoPass) || (rsiPass && mfiPass) || (dpoPass && mfiPass) {
		return true, nil
	}

	return false, nil
}


func (t *TOT) DoesMarketMatchSellConditions() (bool, error) {
	if !t.InMarket {
		return false, nil
	}
	var rsiPass, dpoPass, mfiPass bool
	ohlcv, err := t.exchange.GetHistoricCandles(t.currencyPair, t.assetType, time.Now().Add(-time.Hour * 24), time.Now(), t.Interval)
	if err !=  nil {
		return false, err
	}
	var closeValus []float64
	for i := range ohlcv.Candles {
		closeValus = append(closeValus, ohlcv.Candles[i].Close)
	}
	rsi := indicators.RSI(closeValus, 14)
	if rsi[len(rsi) -1] > 0 {
		rsiPass = true
	}
	if (rsiPass && dpoPass) || (rsiPass && mfiPass) || (dpoPass && mfiPass) {
		return true, nil
	}

	return false, nil
}

func (t *TOT) Monitor() {
	t.RLock()
	timer := time.NewTicker(t.TickerTime)
	t.RUnlock()
	defer func() {
		atomic.CompareAndSwapInt32(&t.Started, 1, 0)
	}()

	go func() {
		if !atomic.CompareAndSwapInt32(&t.Started, 0, 1) {
			return
		}
		for {
			select {
			case <- timer.C:
				if !t.InMarket {
					can, err := t.DoesMarketMatchBuyConditions()
					if err != nil {
						log.Error(log.GCTScriptMgr, "monitor DoesMarketMatchBuyConditions, " +err.Error())
					}
					if can {
						o, err := t.exchange.SubmitOrder(&order.Submit{
							Amount:            t.AvailableFunds,
							Type:              order.Market,
							Side:              order.Buy,
							AssetType:         t.assetType,
							Pair:              t.currencyPair,
						})
						if err != nil {
							log.Error(log.GCTScriptMgr, "monitor BUY SubmitOrder, " +err.Error())
						}
						if o.FullyMatched {
							t.InMarket = true
						}
					}
					t.HasCapital()
				} else {
					can, err := t.DoesMarketMatchSellConditions()
					if err != nil {
						log.Error(log.GCTScriptMgr, "monitor DoesMarketMatchSellConditions, " +err.Error())
					}
					if can {
						o, err := t.exchange.SubmitOrder(&order.Submit{
							Amount:            t.LockedFunds,
							Type:              order.Market,
							Side:              order.Sell,
							AssetType:         t.assetType,
							Pair:              t.currencyPair,
						})
						if err != nil {
							log.Error(log.GCTScriptMgr, "monitor SELL SubmitOrder, " +err.Error())
						}
						if o.FullyMatched {
							t.InMarket = false
							t.HasCapital()
						}
					}
				}

			case <- t.shutdown:
				return
			}
		}
	}()
}

func (t *TOT) GetCapital() error {
	var availFunds float64
	ai, err := t.exchange.FetchAccountInfo()
	if err != nil {
		return err
	}
	for i := range ai.Accounts {
		for j := range ai.Accounts[i].Currencies {
			if ai.Accounts[i].Currencies[j].CurrencyName.Match(t.currencyPair.Base) {
				availFunds = ai.Accounts[i].Currencies[j].TotalValue - ai.Accounts[i].Currencies[j].Hold
				t.AvailableFunds = availFunds
				t.LockedFunds = ai.Accounts[i].Currencies[j].Hold
				break
			}
		}
	}
	return nil
}

func (t *TOT) HasCapital() bool {
	t.GetCapital()
	return t.AvailableFunds > 0
}

