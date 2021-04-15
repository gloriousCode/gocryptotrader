package websocketroutinemanager

import (
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/currencypairsyncer"
)

func Setup(exchangeManager iExchangeManager, orderManager iOrderManager, syncer iCurrencyPairSyncer, cfg *config.CurrencyConfig, verbose bool) (*Manager, error) {
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if orderManager == nil {
		return nil, errNilOrderManager
	}
	if syncer == nil {
		return nil, errNilCurrencyPairSyncer
	}
	if cfg == nil {
		return nil, errNilCurrencyConfig
	}
	if cfg.CurrencyPairFormat == nil && verbose {
		return nil, errNilCurrencyPairFormat
	}
	return &Manager{
		verbose:         verbose,
		exchangeManager: exchangeManager,
		orderManager:    orderManager,
		syncer:          syncer,
		currencyConfig:  cfg,
		shutdown:        make(chan struct{}),
	}, nil
}

func (m *Manager) Start() error {
	if m == nil {
		return subsystems.ErrNilSubsystem
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return subsystems.ErrSubSystemAlreadyStarted
	}
	m.shutdown = make(chan struct{})
	go m.websocketRoutine()
	return nil
}

func (m *Manager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

func (m *Manager) Stop() error {
	if m == nil {
		return subsystems.ErrNilSubsystem
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return subsystems.ErrSubSystemNotStarted
	}
	close(m.shutdown)
	m.wg.Wait()
	return nil
}

// websocketRoutine Initial routine management system for websocket
func (m *Manager) websocketRoutine() {
	if m.verbose {
		log.Debugln(log.WebsocketMgr, "Connecting exchange websocket services...")
	}
	exchanges := m.exchangeManager.GetExchanges()
	for i := range exchanges {
		go func(i int) {
			if exchanges[i].SupportsWebsocket() {
				if m.verbose {
					log.Debugf(log.WebsocketMgr,
						"Exchange %s websocket support: Yes Enabled: %v\n",
						exchanges[i].GetName(),
						common.IsEnabled(exchanges[i].IsWebsocketEnabled()),
					)
				}

				ws, err := exchanges[i].GetWebsocket()
				if err != nil {
					log.Errorf(
						log.WebsocketMgr,
						"Exchange %s GetWebsocket error: %s\n",
						exchanges[i].GetName(),
						err,
					)
					return
				}

				// Exchange sync manager might have already started ws
				// service or is in the process of connecting, so check
				if ws.IsConnected() || ws.IsConnecting() {
					return
				}

				// Data handler routine
				go m.WebsocketDataReceiver(ws)

				if ws.IsEnabled() {
					err = ws.Connect()
					if err != nil {
						log.Errorf(log.WebsocketMgr, "%v\n", err)
					}
					err = ws.FlushChannels()
					if err != nil {
						log.Errorf(log.WebsocketMgr, "Failed to subscribe: %v\n", err)
					}
				}
			} else if m.verbose {
				log.Debugf(log.WebsocketMgr,
					"Exchange %s websocket support: No\n",
					exchanges[i].GetName(),
				)
			}
		}(i)
	}
}

// WebsocketDataReceiver handles websocket data coming from a websocket feed
// associated with an exchange
func (m *Manager) WebsocketDataReceiver(ws *stream.Websocket) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	m.wg.Add(1)
	defer m.wg.Done()

	for {
		select {
		case <-m.shutdown:
			return
		case data := <-ws.ToRoutine:
			err := m.WebsocketDataHandler(ws.GetName(), data)
			if err != nil {
				log.Error(log.WebsocketMgr, err)
			}
		}
	}
}

// WebsocketDataHandler is a central point for exchange websocket implementations to send
// processed data. WebsocketDataHandler will then pass that to an appropriate handler
func (m *Manager) WebsocketDataHandler(exchName string, data interface{}) error {
	if data == nil {
		return fmt.Errorf("exchange %s nil data sent to websocket",
			exchName)
	}

	switch d := data.(type) {
	case string:
		log.Info(log.WebsocketMgr, d)
	case error:
		return fmt.Errorf("exchange %s websocket error - %s", exchName, data)
	case stream.FundingData:
		if m.verbose {
			log.Infof(log.WebsocketMgr, "%s websocket %s %s funding updated %+v",
				exchName,
				m.FormatCurrency(d.CurrencyPair),
				d.AssetType,
				d)
		}
	case *ticker.Price:
		if m.syncer.IsRunning() {
			err := m.syncer.Update(exchName,
				d.Pair,
				d.AssetType,
				currencypairsyncer.SyncItemTicker,
				nil)
			if err != nil {
				return err
			}
		}
		err := ticker.ProcessTicker(d)
		if err != nil {
			return err
		}
		m.syncer.PrintTickerSummary(d, "websocket", err)
	case stream.KlineData:
		if m.verbose {
			log.Infof(log.WebsocketMgr, "%s websocket %s %s kline updated %+v",
				exchName,
				m.FormatCurrency(d.Pair),
				d.AssetType,
				d)
		}
	case *orderbook.Base:
		if m.syncer.IsRunning() {
			err := m.syncer.Update(exchName,
				d.Pair,
				d.AssetType,
				currencypairsyncer.SyncItemOrderbook,
				nil)
			if err != nil {
				return err
			}
		}
		m.syncer.PrintOrderbookSummary(d, "websocket", nil)
	case *order.Detail:
		if !m.orderManager.Exists(d) {
			err := m.orderManager.Add(d)
			if err != nil {
				return err
			}
		} else {
			od, err := m.orderManager.GetByExchangeAndID(d.Exchange, d.ID)
			if err != nil {
				return err
			}
			od.UpdateOrderFromDetail(d)

			err = m.orderManager.UpdateExistingOrder(od)
			if err != nil {
				return err
			}
		}
	case *order.Cancel:
		return m.orderManager.Cancel(d)
	case *order.Modify:
		od, err := m.orderManager.GetByExchangeAndID(d.Exchange, d.ID)
		if err != nil {
			return err
		}
		od.UpdateOrderFromModify(d)
		err = m.orderManager.UpdateExistingOrder(od)
		if err != nil {
			return err
		}
	case order.ClassificationError:
		return fmt.Errorf("%w %s", d.Err, d.Error())
	case stream.UnhandledMessageWarning:
		log.Warn(log.WebsocketMgr, d.Message)
	default:
		if m.verbose {
			log.Warnf(log.WebsocketMgr,
				"%s websocket Unknown type: %+v",
				exchName,
				d)
		}
	}
	return nil
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (m *Manager) FormatCurrency(p currency.Pair) currency.Pair {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return p
	}
	return p.Format(m.currencyConfig.CurrencyPairFormat.Delimiter,
		m.currencyConfig.CurrencyPairFormat.Uppercase)
}
