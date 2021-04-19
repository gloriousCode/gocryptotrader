package currencypairsyncer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/apiserver"
)

// const holds the sync item types
const (
	SyncItemTicker = iota
	SyncItemOrderbook
	SyncItemTrade
	Name = "exchange_syncer"
)

var (
	createdCounter        = 0
	removedCounter        = 0
	DefaultSyncerWorkers  = 15
	DefaultSyncerTimeout  = time.Second * 15
	errNoSyncItemsEnabled = errors.New("no sync items enabled")
	errNilExchangeManager = errors.New("nil exchange manager received")
	errNilConfig          = errors.New("nil config received")
	errUnknownSyncItem    = errors.New("unknown sync item")
)

// Setup starts a new CurrencyPairSyncer
func Setup(c *Config, exchangeManager iExchangeManager, websocketDataReceiver iWebsocketDataReceiver, remoteConfig *config.RemoteControlConfig) (*Manager, error) {
	if !c.SyncOrderbook && !c.SyncTicker && !c.SyncTrades {
		return nil, errNoSyncItemsEnabled
	}
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if remoteConfig == nil {
		return nil, errNilConfig
	}

	if c.NumWorkers <= 0 {
		c.NumWorkers = DefaultSyncerWorkers
	}

	if c.SyncTimeout <= time.Duration(0) {
		c.SyncTimeout = DefaultSyncerTimeout
	}

	s := &Manager{
		config: Config{
			SyncTicker:       c.SyncTicker,
			SyncOrderbook:    c.SyncOrderbook,
			SyncTrades:       c.SyncTrades,
			SyncContinuously: c.SyncContinuously,
			SyncTimeout:      c.SyncTimeout,
			NumWorkers:       c.NumWorkers,
		},
		remoteConfig:          remoteConfig,
		exchangeManager:       exchangeManager,
		websocketDataReceiver: websocketDataReceiver,
	}

	s.tickerBatchLastRequested = make(map[string]time.Time)

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout: %v\n",
		s.config.SyncContinuously, s.config.SyncTicker, s.config.SyncOrderbook,
		s.config.SyncTrades, s.config.NumWorkers, s.config.Verbose, s.config.SyncTimeout)
	return s, nil
}

func (m *Manager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Start starts an exchange currency pair syncer
func (m *Manager) Start() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return subsystems.ErrSubSystemAlreadyStarted
	}
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer started.")
	exchanges := m.exchangeManager.GetExchanges()
	for x := range exchanges {
		exchangeName := exchanges[x].GetName()
		supportsWebsocket := exchanges[x].SupportsWebsocket()
		assetTypes := exchanges[x].GetAssetTypes()
		supportsREST := exchanges[x].SupportsREST()

		if !supportsREST && !supportsWebsocket {
			log.Warnf(log.SyncMgr,
				"Loaded exchange %s does not support REST or Websocket.\n",
				exchangeName)
			continue
		}

		var usingWebsocket bool
		var usingREST bool
		if supportsWebsocket && exchanges[x].IsWebsocketEnabled() {
			ws, err := exchanges[x].GetWebsocket()
			if err != nil {
				log.Errorf(log.SyncMgr,
					"%s failed to get websocket. Err: %s\n",
					exchangeName,
					err)
				usingREST = true
			}

			if !ws.IsConnected() && !ws.IsConnecting() {
				if m.websocketDataReceiver.IsRunning() {
					go m.websocketDataReceiver.WebsocketDataReceiver(ws)
				}

				err = ws.Connect()
				if err == nil {
					err = ws.FlushChannels()
				}
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s websocket failed to connect. Err: %s\n",
						exchangeName,
						err)
					usingREST = true
				} else {
					usingWebsocket = true
				}
			} else {
				usingWebsocket = true
			}
		} else if supportsREST {
			usingREST = true
		}

		for y := range assetTypes {
			if exchanges[x].GetBase().CurrencyPairs.IsAssetEnabled(assetTypes[y]) != nil {
				log.Warnf(log.SyncMgr,
					"%s asset type %s is disabled, fetching enabled pairs is paused",
					exchangeName,
					assetTypes[y])
				continue
			}

			enabledPairs, err := exchanges[x].GetEnabledPairs(assetTypes[y])
			if err != nil {
				log.Errorf(log.SyncMgr,
					"%s failed to get enabled pairs. Err: %s\n",
					exchangeName,
					err)
				continue
			}
			for i := range enabledPairs {
				if m.exists(exchangeName, enabledPairs[i], assetTypes[y]) {
					continue
				}

				c := currencyPairSyncAgent{
					AssetType: assetTypes[y],
					Exchange:  exchangeName,
					Pair:      enabledPairs[i],
				}

				if m.config.SyncTicker {
					c.Ticker = syncBase{
						IsUsingREST:      usingREST,
						IsUsingWebsocket: usingWebsocket,
					}
				}

				if m.config.SyncOrderbook {
					c.Orderbook = syncBase{
						IsUsingREST:      usingREST,
						IsUsingWebsocket: usingWebsocket,
					}
				}

				if m.config.SyncTrades {
					c.Trade = syncBase{
						IsUsingREST:      usingREST,
						IsUsingWebsocket: usingWebsocket,
					}
				}

				m.add(&c)
			}
		}
	}

	if atomic.CompareAndSwapInt32(&m.initSyncStarted, 0, 1) {
		log.Debugf(log.SyncMgr,
			"Exchange CurrencyPairSyncer initial sync started. %d items to process.\n",
			createdCounter)
		m.initSyncStartTime = time.Now()
	}

	go func() {
		m.initSyncWG.Wait()
		if atomic.CompareAndSwapInt32(&m.initSyncCompleted, 0, 1) {
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync is complete.\n")
			completedTime := time.Now()
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync took %v [%v sync items].\n",
				completedTime.Sub(m.initSyncStartTime), createdCounter)

			if !m.config.SyncContinuously {
				log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
				err := m.Stop()
				if err != nil {
					log.Error(log.SyncMgr, err)
				}
				return
			}
		}
	}()

	if atomic.LoadInt32(&m.initSyncCompleted) == 1 && !m.config.SyncContinuously {
		return nil
	}

	for i := 0; i < m.config.NumWorkers; i++ {
		go m.worker()
	}
	return nil
}

// Stop shuts down the exchange currency pair syncer
func (m *Manager) Stop() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystems.ErrSubSystemNotStarted)
	}
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	return nil
}

func (m *Manager) get(exchangeName string, p currency.Pair, a asset.Item) (*currencyPairSyncAgent, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for x := range m.currencyPairs {
		if m.currencyPairs[x].Exchange == exchangeName &&
			m.currencyPairs[x].Pair.Equal(p) &&
			m.currencyPairs[x].AssetType == a {
			return &m.currencyPairs[x], nil
		}
	}

	return nil, errors.New("exchange currency pair syncer not found")
}

func (m *Manager) exists(exchangeName string, p currency.Pair, a asset.Item) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	for x := range m.currencyPairs {
		if m.currencyPairs[x].Exchange == exchangeName &&
			m.currencyPairs[x].Pair.Equal(p) &&
			m.currencyPairs[x].AssetType == a {
			return true
		}
	}
	return false
}

func (m *Manager) add(c *currencyPairSyncAgent) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.config.SyncTicker {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added ticker sync item %v: using websocket: %v using REST: %v\n",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Ticker.IsUsingWebsocket,
				c.Ticker.IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if m.config.SyncOrderbook {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added orderbook sync item %v: using websocket: %v using REST: %v\n",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Orderbook.IsUsingWebsocket,
				c.Orderbook.IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if m.config.SyncTrades {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added trade sync item %v: using websocket: %v using REST: %v\n",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Trade.IsUsingWebsocket,
				c.Trade.IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			createdCounter++
		}
	}

	c.Created = time.Now()
	m.currencyPairs = append(m.currencyPairs, *c)
}

func (m *Manager) remove(c *currencyPairSyncAgent) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for x := range m.currencyPairs {
		if m.currencyPairs[x].Exchange == c.Exchange &&
			m.currencyPairs[x].Pair.Equal(c.Pair) &&
			m.currencyPairs[x].AssetType == c.AssetType {
			m.currencyPairs = append(m.currencyPairs[:x], m.currencyPairs[x+1:]...)
			return
		}
	}
}

func (m *Manager) isProcessing(exchangeName string, p currency.Pair, a asset.Item, syncType int) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	for x := range m.currencyPairs {
		if m.currencyPairs[x].Exchange == exchangeName &&
			m.currencyPairs[x].Pair.Equal(p) &&
			m.currencyPairs[x].AssetType == a {
			switch syncType {
			case SyncItemTicker:
				return m.currencyPairs[x].Ticker.IsProcessing
			case SyncItemOrderbook:
				return m.currencyPairs[x].Orderbook.IsProcessing
			case SyncItemTrade:
				return m.currencyPairs[x].Trade.IsProcessing
			}
		}
	}

	return false
}

func (m *Manager) setProcessing(exchangeName string, p currency.Pair, a asset.Item, syncType int, processing bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for x := range m.currencyPairs {
		if m.currencyPairs[x].Exchange == exchangeName &&
			m.currencyPairs[x].Pair.Equal(p) &&
			m.currencyPairs[x].AssetType == a {
			switch syncType {
			case SyncItemTicker:
				m.currencyPairs[x].Ticker.IsProcessing = processing
			case SyncItemOrderbook:
				m.currencyPairs[x].Orderbook.IsProcessing = processing
			case SyncItemTrade:
				m.currencyPairs[x].Trade.IsProcessing = processing
			}
		}
	}
}

// Update notifies the Manager to change the last updated time for a exchange asset pair
func (m *Manager) Update(exchangeName string, p currency.Pair, a asset.Item, syncType int, err error) error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystems.ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystems.ErrSubSystemNotStarted)
	}

	if atomic.LoadInt32(&m.initSyncStarted) != 1 {
		return nil
	}

	switch syncType {
	case SyncItemOrderbook:
		if !m.config.SyncOrderbook {
			return nil
		}
	case SyncItemTicker:
		if !m.config.SyncTicker {
			return nil
		}
	case SyncItemTrade:
		if !m.config.SyncTrades {
			return nil
		}
	default:
		return fmt.Errorf("%v %w", syncType, errUnknownSyncItem)
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	for x := range m.currencyPairs {
		if m.currencyPairs[x].Exchange == exchangeName &&
			m.currencyPairs[x].Pair.Equal(p) &&
			m.currencyPairs[x].AssetType == a {
			switch syncType {
			case SyncItemTicker:
				origHadData := m.currencyPairs[x].Ticker.HaveData
				m.currencyPairs[x].Ticker.LastUpdated = time.Now()
				if err != nil {
					m.currencyPairs[x].Ticker.NumErrors++
				}
				m.currencyPairs[x].Ticker.HaveData = true
				m.currencyPairs[x].Ticker.IsProcessing = false
				if atomic.LoadInt32(&m.initSyncCompleted) != 1 && !origHadData {
					removedCounter++
					log.Debugf(log.SyncMgr, "%s ticker sync complete %v [%d/%d].\n",
						exchangeName,
						m.FormatCurrency(p).String(),
						removedCounter,
						createdCounter)
					m.initSyncWG.Done()
				}

			case SyncItemOrderbook:
				origHadData := m.currencyPairs[x].Orderbook.HaveData
				m.currencyPairs[x].Orderbook.LastUpdated = time.Now()
				if err != nil {
					m.currencyPairs[x].Orderbook.NumErrors++
				}
				m.currencyPairs[x].Orderbook.HaveData = true
				m.currencyPairs[x].Orderbook.IsProcessing = false
				if atomic.LoadInt32(&m.initSyncCompleted) != 1 && !origHadData {
					removedCounter++
					log.Debugf(log.SyncMgr, "%s orderbook sync complete %v [%d/%d].\n",
						exchangeName,
						m.FormatCurrency(p).String(),
						removedCounter,
						createdCounter)
					m.initSyncWG.Done()
				}

			case SyncItemTrade:
				origHadData := m.currencyPairs[x].Trade.HaveData
				m.currencyPairs[x].Trade.LastUpdated = time.Now()
				if err != nil {
					m.currencyPairs[x].Trade.NumErrors++
				}
				m.currencyPairs[x].Trade.HaveData = true
				m.currencyPairs[x].Trade.IsProcessing = false
				if atomic.LoadInt32(&m.initSyncCompleted) != 1 && !origHadData {
					removedCounter++
					log.Debugf(log.SyncMgr, "%s trade sync complete %v [%d/%d].\n",
						exchangeName,
						m.FormatCurrency(p).String(),
						removedCounter,
						createdCounter)
					m.initSyncWG.Done()
				}
			}
		}
	}
	return nil
}

func (m *Manager) worker() {
	cleanup := func() {
		log.Debugln(log.SyncMgr,
			"Exchange CurrencyPairSyncer worker shutting down.")
	}
	defer cleanup()

	for atomic.LoadInt32(&m.started) != 0 {
		exchanges := m.exchangeManager.GetExchanges()
		for x := range exchanges {
			exchangeName := exchanges[x].GetName()
			assetTypes := exchanges[x].GetAssetTypes()
			supportsREST := exchanges[x].SupportsREST()
			supportsRESTTickerBatching := exchanges[x].SupportsRESTTickerBatchUpdates()
			var usingREST bool
			var usingWebsocket bool
			var switchedToRest bool
			if exchanges[x].SupportsWebsocket() && exchanges[x].IsWebsocketEnabled() {
				ws, err := exchanges[x].GetWebsocket()
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s unable to get websocket pointer. Err: %s\n",
						exchangeName,
						err)
					usingREST = true
				}

				if ws.IsConnected() {
					usingWebsocket = true
				} else {
					usingREST = true
				}
			} else if supportsREST {
				usingREST = true
			}

			for y := range assetTypes {
				if exchanges[x].GetBase().CurrencyPairs.IsAssetEnabled(assetTypes[y]) != nil {
					continue
				}
				enabledPairs, err := exchanges[x].GetEnabledPairs(assetTypes[y])
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s failed to get enabled pairs. Err: %s\n",
						exchangeName,
						err)
					continue
				}
				for i := range enabledPairs {
					if atomic.LoadInt32(&m.started) == 0 {
						return
					}

					if !m.exists(exchangeName, enabledPairs[i], assetTypes[y]) {
						c := currencyPairSyncAgent{
							AssetType: assetTypes[y],
							Exchange:  exchangeName,
							Pair:      enabledPairs[i],
						}

						if m.config.SyncTicker {
							c.Ticker = syncBase{
								IsUsingREST:      usingREST,
								IsUsingWebsocket: usingWebsocket,
							}
						}

						if m.config.SyncOrderbook {
							c.Orderbook = syncBase{
								IsUsingREST:      usingREST,
								IsUsingWebsocket: usingWebsocket,
							}
						}

						if m.config.SyncTrades {
							c.Trade = syncBase{
								IsUsingREST:      usingREST,
								IsUsingWebsocket: usingWebsocket,
							}
						}

						m.add(&c)
					}

					c, err := m.get(exchangeName, enabledPairs[i], assetTypes[y])
					if err != nil {
						log.Errorf(log.SyncMgr, "failed to get item. Err: %s\n", err)
						continue
					}
					if switchedToRest && usingWebsocket {
						log.Warnf(log.SyncMgr,
							"%s %s: Websocket re-enabled, switching from rest to websocket\n",
							c.Exchange, m.FormatCurrency(enabledPairs[i]).String())
						switchedToRest = false
					}

					if m.config.SyncOrderbook {
						if !m.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemOrderbook) {
							if c.Orderbook.LastUpdated.IsZero() || time.Since(c.Orderbook.LastUpdated) > m.config.SyncTimeout {
								if c.Orderbook.IsUsingWebsocket {
									if time.Since(c.Created) < m.config.SyncTimeout {
										continue
									}
									if supportsREST {
										m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
										c.Orderbook.IsUsingWebsocket = false
										c.Orderbook.IsUsingREST = true
										log.Warnf(log.SyncMgr,
											"%s %s %s: No orderbook update after %s, switching from websocket to rest\n",
											c.Exchange,
											m.FormatCurrency(c.Pair).String(),
											strings.ToUpper(c.AssetType.String()),
											m.config.SyncTimeout,
										)
										switchedToRest = true
										m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, false)
									}
								}

								m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
								result, err := exchanges[x].UpdateOrderbook(c.Pair, c.AssetType)
								m.PrintOrderbookSummary(result, "REST", err)
								if err == nil {
									if m.remoteConfig.WebsocketRPC.Enabled {
										relayWebsocketEvent(result, "orderbook_update", c.AssetType.String(), exchangeName)
									}
								}
								updateErr := m.Update(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, err)
								if updateErr != nil {
									log.Error(log.SyncMgr, updateErr)
								}
							} else {
								time.Sleep(time.Millisecond * 50)
							}
						}

						if m.config.SyncTicker {
							if !m.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTicker) {
								if c.Ticker.LastUpdated.IsZero() || time.Since(c.Ticker.LastUpdated) > m.config.SyncTimeout {
									if c.Ticker.IsUsingWebsocket {
										if time.Since(c.Created) < m.config.SyncTimeout {
											continue
										}

										if supportsREST {
											m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
											c.Ticker.IsUsingWebsocket = false
											c.Ticker.IsUsingREST = true
											log.Warnf(log.SyncMgr,
												"%s %s %s: No ticker update after %s, switching from websocket to rest\n",
												c.Exchange,
												m.FormatCurrency(enabledPairs[i]).String(),
												strings.ToUpper(c.AssetType.String()),
												m.config.SyncTimeout,
											)
											switchedToRest = true
											m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, false)
										}
									}

									if c.Ticker.IsUsingREST {
										m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
										var result *ticker.Price
										var err error

										if supportsRESTTickerBatching {
											m.mux.Lock()
											batchLastDone, ok := m.tickerBatchLastRequested[exchangeName]
											if !ok {
												m.tickerBatchLastRequested[exchangeName] = time.Time{}
											}
											m.mux.Unlock()

											if batchLastDone.IsZero() || time.Since(batchLastDone) > m.config.SyncTimeout {
												m.mux.Lock()
												if m.config.Verbose {
													log.Debugf(log.SyncMgr, "%s Init'ing REST ticker batching\n", exchangeName)
												}
												result, err = exchanges[x].UpdateTicker(c.Pair, c.AssetType)
												m.tickerBatchLastRequested[exchangeName] = time.Now()
												m.mux.Unlock()
											} else {
												if m.config.Verbose {
													log.Debugf(log.SyncMgr, "%s Using recent batching cache\n", exchangeName)
												}
												result, err = exchanges[x].FetchTicker(c.Pair, c.AssetType)
											}
										} else {
											result, err = exchanges[x].UpdateTicker(c.Pair, c.AssetType)
										}
										m.PrintTickerSummary(result, "REST", err)
										if err == nil {
											if m.remoteConfig.WebsocketRPC.Enabled {
												relayWebsocketEvent(result, "ticker_update", c.AssetType.String(), exchangeName)
											}
										}
										updateErr := m.Update(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, err)
										if updateErr != nil {
											log.Error(log.SyncMgr, updateErr)
										}
									}
								} else {
									time.Sleep(time.Millisecond * 50)
								}
							}
						}

						if m.config.SyncTrades {
							if !m.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTrade) {
								if c.Trade.LastUpdated.IsZero() || time.Since(c.Trade.LastUpdated) > m.config.SyncTimeout {
									m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, true)
									err := m.Update(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, nil)
									if err != nil {
										log.Error(log.SyncMgr, err)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func printCurrencyFormat(price float64, displayCurrency currency.Code) string {
	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get display symbol: %s\n", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency currency.Code, origPrice float64, displayCurrency currency.Code) string {
	conv, err := currency.ConvertCurrency(origPrice,
		origCurrency,
		displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to convert currency: %s\n", err)
	}

	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get display symbol: %s\n", err)
	}

	origSymbol, err := currency.GetSymbolByCurrencyName(origCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get original currency symbol for %s: %s\n",
			origCurrency,
			err)
	}

	return fmt.Sprintf("%s%.2f %s (%s%.2f %s)",
		displaySymbol,
		conv,
		displayCurrency,
		origSymbol,
		origPrice,
		origCurrency,
	)
}

// PrintTickerSummary outputs the ticker results
func (m *Manager) PrintTickerSummary(result *ticker.Price, protocol string, err error) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	if err != nil {
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.Ticker, "Failed to get %s ticker. Error: %s\n",
				protocol,
				err)
			return
		}
		log.Errorf(log.Ticker, "Failed to get %s ticker. Error: %s\n",
			protocol,
			err)
		return
	}

	err = stats.Add(result.ExchangeName, result.Pair, result.AssetType, result.Last, result.Volume)
	if err != nil && protocol != "websocket" {
		// websocket does not always contain all fields
		log.Error(log.SyncMgr, err)
	}
	if result.Pair.Quote.IsFiatCurrency() &&
		result.Pair.Quote != m.fiatDisplayCurrency &&
		!m.fiatDisplayCurrency.IsEmpty() {
		origCurrency := result.Pair.Quote.Upper()
		log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
			result.ExchangeName,
			protocol,
			m.FormatCurrency(result.Pair),
			strings.ToUpper(result.AssetType.String()),
			printConvertCurrencyFormat(origCurrency, result.Last, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Ask, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Bid, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.High, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Low, m.fiatDisplayCurrency),
			result.Volume)
	} else {
		if result.Pair.Quote.IsFiatCurrency() &&
			result.Pair.Quote == m.fiatDisplayCurrency &&
			!m.fiatDisplayCurrency.IsEmpty() {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
				result.ExchangeName,
				protocol,
				m.FormatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				printCurrencyFormat(result.Last, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Ask, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Bid, m.fiatDisplayCurrency),
				printCurrencyFormat(result.High, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Low, m.fiatDisplayCurrency),
				result.Volume)
		} else {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f\n",
				result.ExchangeName,
				protocol,
				m.FormatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				result.Last,
				result.Ask,
				result.Bid,
				result.High,
				result.Low,
				result.Volume)
		}
	}
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (m *Manager) FormatCurrency(p currency.Pair) currency.Pair {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return p
	}
	return p.Format(m.delimiter, m.uppercase)
}

const (
	book = "%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s\n"
)

// PrintOrderbookSummary outputs orderbook results
func (m *Manager) PrintOrderbookSummary(result *orderbook.Base, protocol string, err error) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	if err != nil {
		if result == nil {
			log.Errorf(log.OrderBook, "Failed to get %s orderbook. Error: %s\n",
				protocol,
				err)
			return
		}
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s\n",
				protocol,
				result.ExchangeName,
				result.Pair,
				result.AssetType,
				err)
			return
		}
		log.Errorf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s\n",
			protocol,
			result.ExchangeName,
			result.Pair,
			result.AssetType,
			err)
		return
	}

	bidsAmount, bidsValue := result.TotalBidsAmount()
	asksAmount, asksValue := result.TotalAsksAmount()

	var bidValueResult, askValueResult string
	switch {
	case result.Pair.Quote.IsFiatCurrency() && result.Pair.Quote != m.fiatDisplayCurrency && !m.fiatDisplayCurrency.IsEmpty():
		origCurrency := result.Pair.Quote.Upper()
		bidValueResult = printConvertCurrencyFormat(origCurrency, bidsValue, m.fiatDisplayCurrency)
		askValueResult = printConvertCurrencyFormat(origCurrency, asksValue, m.fiatDisplayCurrency)
	case result.Pair.Quote.IsFiatCurrency() && result.Pair.Quote == m.fiatDisplayCurrency && !m.fiatDisplayCurrency.IsEmpty():
		bidValueResult = printCurrencyFormat(bidsValue, m.fiatDisplayCurrency)
		askValueResult = printCurrencyFormat(asksValue, m.fiatDisplayCurrency)
	default:
		bidValueResult = strconv.FormatFloat(bidsValue, 'f', -1, 64)
		askValueResult = strconv.FormatFloat(asksValue, 'f', -1, 64)
	}

	log.Infof(log.OrderBook, book,
		result.ExchangeName,
		protocol,
		m.FormatCurrency(result.Pair),
		strings.ToUpper(result.AssetType.String()),
		len(result.Bids),
		bidsAmount,
		result.Pair.Base,
		bidValueResult,
		len(result.Asks),
		asksAmount,
		result.Pair.Base,
		askValueResult,
	)
}

func relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	evt := apiserver.WebsocketEvent{
		Data:      result,
		Event:     event,
		AssetType: assetType,
		Exchange:  exchangeName,
	}
	err := apiserver.BroadcastWebsocketMessage(evt)
	if !errors.Is(err, apiserver.ErrWebsocketServiceNotRunning) {
		log.Errorf(log.WebsocketMgr, "Failed to broadcast websocket event %v. Error: %s\n",
			event, err)
	}
}
