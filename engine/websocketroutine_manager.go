package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// setupWebsocketRoutineManager creates a new websocket routine manager
func setupWebsocketRoutineManager(exchangeManager iExchangeManager, orderManager iOrderManager, syncer iCurrencyPairSyncer, cfg *currency.Config, verbose bool) (*WebsocketRoutineManager, error) {
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if syncer == nil {
		return nil, errNilCurrencyPairSyncer
	}
	if cfg == nil {
		return nil, errNilCurrencyConfig
	}
	if cfg.CurrencyPairFormat == nil {
		return nil, errNilCurrencyPairFormat
	}
	man := &WebsocketRoutineManager{
		verbose:         verbose,
		exchangeManager: exchangeManager,
		orderManager:    orderManager,
		syncer:          syncer,
		currencyConfig:  cfg,
	}
	man.ensureHandlersInit()
	return man, man.registerWebsocketDataHandler(man.websocketDataHandler, false)
}

// Start runs the subsystem
func (m *WebsocketRoutineManager) Start() error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}

	if m.currencyConfig == nil {
		return errNilCurrencyConfig
	}

	if m.currencyConfig.CurrencyPairFormat == nil {
		return errNilCurrencyPairFormat
	}

	if !atomic.CompareAndSwapInt32(&m.state, stoppedState, startingState) {
		return ErrSubSystemAlreadyStarted
	}

	m.shutdown = make(chan struct{})

	go func() {
		m.websocketRoutine()
		// It's okay for this to fail, just means shutdown has started
		atomic.CompareAndSwapInt32(&m.state, startingState, readyState)
	}()
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (m *WebsocketRoutineManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.state) == readyState
}

// Stop attempts to shutdown the subsystem
func (m *WebsocketRoutineManager) Stop() error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}

	m.mu.Lock()
	if atomic.LoadInt32(&m.state) == stoppedState {
		m.mu.Unlock()
		return fmt.Errorf("websocket routine manager %w", ErrSubSystemNotStarted)
	}
	atomic.StoreInt32(&m.state, stoppedState)
	m.mu.Unlock()

	close(m.shutdown)
	m.wg.Wait()

	return nil
}

// websocketRoutine Initial routine management system for websocket
func (m *WebsocketRoutineManager) websocketRoutine() {
	if m.verbose {
		log.Debugln(log.WebsocketMgr, "Connecting exchange websocket services...")
	}
	exchanges, err := m.exchangeManager.GetExchanges()
	if err != nil {
		log.Errorf(log.WebsocketMgr, "websocket routine manager cannot get exchanges: %v", err)
	}
	var wg sync.WaitGroup
	for _, exch := range exchanges {
		if !exch.SupportsWebsocket() {
			if m.verbose {
				log.Debugf(log.WebsocketMgr, "Exchange %s websocket support: No",
					exch.GetName())
			}
			continue
		}

		if m.verbose {
			log.Debugf(log.WebsocketMgr, "Exchange %s websocket support: Yes Enabled: %v",
				exch.GetName(),
				common.IsEnabled(exch.IsWebsocketEnabled()))
		}

		ws, err := exch.GetWebsocket()
		if err != nil {
			log.Errorf(log.WebsocketMgr, "Exchange %s GetWebsocket error: %s",
				exch.GetName(),
				err)
			continue
		}

		if !ws.IsEnabled() {
			continue
		}

		wg.Go(func() {
			if err := m.websocketDataReceiver(ws); err != nil {
				log.Errorf(log.WebsocketMgr, "%v", err)
			}

			if err := ws.Connect(context.TODO()); err != nil {
				log.Errorf(log.WebsocketMgr, "%v", err)
			}
		})
	}
	wg.Wait()
}

// websocketDataReceiver handles websocket data coming from a websocket feed
// associated with an exchange
func (m *WebsocketRoutineManager) websocketDataReceiver(ws *websocket.Manager) error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}

	if ws == nil {
		return errNilWebsocket
	}

	if atomic.LoadInt32(&m.state) == stoppedState {
		return errRoutineManagerNotStarted
	}

	m.wg.Go(func() {
		for {
			select {
			case <-m.shutdown:
				return
			case payload := <-ws.DataHandler.C:
				if payload.Data == nil {
					log.Errorf(log.WebsocketMgr, "exchange %s nil data sent to websocket", ws.GetName())
				}
				m.ensureHandlersInit()
				handlers := m.dataHandlers.Load().([]WebsocketDataHandler)
				for x := range handlers {
					if err := handlers[x](ws.GetName(), payload.Data); err != nil {
						log.Errorln(log.WebsocketMgr, err)
					}
				}
			}
		}
	})
	return nil
}

// websocketDataHandler is the default central point for exchange websocket
// implementations to send processed data which will then pass that to an
// appropriate handler.
func (m *WebsocketRoutineManager) websocketDataHandler(exchName string, data any) error {
	switch d := data.(type) {
	case string:
		if !m.verbose {
			return nil
		}
		log.Infoln(log.WebsocketMgr, d)
	case error:
		return fmt.Errorf("exchange %s websocket error - %v", exchName, data)
	case websocket.FundingData:
		if !m.verbose {
			return nil
		}
		log.Infof(log.WebsocketMgr, "%s websocket %s %s funding updated %+v",
			exchName,
			m.FormatCurrency(d.CurrencyPair),
			d.AssetType,
			d)
	case *ticker.Price:
		if !m.syncer.IsRunning() {
			return nil
		}
		err := m.syncer.WebsocketUpdate(exchName,
			d.Pair,
			d.AssetType,
			SyncItemTicker,
			nil)
		if err != nil {
			return err
		}
		m.syncer.PrintTickerSummary(d, "websocket", err)
	case []ticker.Price:
		if !m.syncer.IsRunning() {
			return nil
		}
		for x := range d {
			err := m.syncer.WebsocketUpdate(exchName,
				d[x].Pair,
				d[x].AssetType,
				SyncItemTicker,
				nil)
			if err != nil {
				return err
			}
			m.syncer.PrintTickerSummary(&d[x], "websocket", err)
		}
	case order.Detail, ticker.Price, orderbook.Depth:
		return errUseAPointer
	case websocket.KlineData:
		if !m.verbose {
			return nil
		}
		log.Infof(log.WebsocketMgr, "%s websocket %s %s kline updated %+v",
			exchName,
			m.FormatCurrency(d.Pair),
			d.AssetType,
			d)
	case []websocket.KlineData:
		if !m.verbose {
			return nil
		}
		for x := range d {
			log.Infof(log.WebsocketMgr, "%s websocket %s %s kline updated %+v",
				exchName,
				m.FormatCurrency(d[x].Pair),
				d[x].AssetType,
				d)
		}
	case *orderbook.Depth:
		if !m.syncer.IsRunning() {
			return nil
		}
		if err := m.syncer.WebsocketUpdate(exchName, d.Pair(), d.Asset(), SyncItemOrderbook, nil); err != nil {
			return err
		}
	case *order.Detail:
		if !m.orderManager.IsRunning() {
			return nil
		}
		if m.orderManager.Exists(d) {
			od, err := m.orderManager.GetByExchangeAndID(d.Exchange, d.OrderID)
			if err != nil {
				return err
			}
			err = od.UpdateOrderFromDetail(d)
			if err != nil {
				return err
			}
			err = m.orderManager.UpdateExistingOrder(od)
			if err != nil {
				return err
			}
		} else {
			err := m.orderManager.Add(d)
			if err != nil {
				return err
			}
		}
	case []order.Detail:
		if !m.orderManager.IsRunning() {
			return nil
		}
		for x := range d {
			if m.orderManager.Exists(&d[x]) {
				od, err := m.orderManager.GetByExchangeAndID(d[x].Exchange, d[x].OrderID)
				if err != nil {
					return err
				}
				err = od.UpdateOrderFromDetail(&d[x])
				if err != nil {
					return err
				}
				err = m.orderManager.UpdateExistingOrder(od)
				if err != nil {
					return err
				}
			} else {
				err := m.orderManager.Add(&d[x])
				if err != nil {
					return err
				}
			}
		}
	case websocket.UnhandledMessageWarning:
		if !m.verbose {
			return nil
		}
		log.Warnf(log.WebsocketMgr, "%s unhandled message - %s", exchName, d.Message)
	case []accounts.Change, accounts.Change:
		if !m.verbose {
			return nil
		}
		log.Debugf(log.WebsocketMgr, "%s %+v", exchName, d)
	case []trade.Data, trade.Data:
		if !m.verbose {
			return nil
		}
		log.Infof(log.Trade, "%+v", d)
	case []fill.Data:
		if !m.verbose {
			return nil
		}
		log.Infof(log.Fill, "%+v", d)
	default:
		if !m.verbose {
			return nil
		}
		log.Warnf(log.WebsocketMgr, "%s websocket Unknown type: %+v", exchName, d)
	}
	return nil
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (m *WebsocketRoutineManager) FormatCurrency(p currency.Pair) currency.Pair {
	if m == nil || atomic.LoadInt32(&m.state) == stoppedState {
		return p
	}
	return p.Format(*m.currencyConfig.CurrencyPairFormat)
}

// registerWebsocketDataHandler registers an externally (GCT Library) defined
// dedicated filter specific data types for internal & external strategy use.
// InterceptorOnly as true will purge all other registered handlers
// (including default) bypassing all other handling.
func (m *WebsocketRoutineManager) registerWebsocketDataHandler(fn WebsocketDataHandler, interceptorOnly bool) error {
	if m == nil {
		return fmt.Errorf("%T %w", m, ErrNilSubsystem)
	}

	if fn == nil {
		return errNilWebsocketDataHandlerFunction
	}

	if interceptorOnly {
		return m.setWebsocketDataHandler(fn)
	}

	m.ensureHandlersInit()
	m.mu.Lock()
	handlers := m.dataHandlers.Load().([]WebsocketDataHandler)
	// Push front so that any registered data handler has first preference
	// over the gct default handler.
	next := make([]WebsocketDataHandler, 0, len(handlers)+1)
	next = append(next, fn)
	next = append(next, handlers...)
	m.dataHandlers.Store(next)
	m.mu.Unlock()
	return nil
}

// setWebsocketDataHandler sets a single websocket data handler, removing all
// pre-existing handlers.
func (m *WebsocketRoutineManager) setWebsocketDataHandler(fn WebsocketDataHandler) error {
	if m == nil {
		return fmt.Errorf("%T %w", m, ErrNilSubsystem)
	}
	if fn == nil {
		return errNilWebsocketDataHandlerFunction
	}
	m.ensureHandlersInit()
	m.mu.Lock()
	m.dataHandlers.Store([]WebsocketDataHandler{fn})
	m.mu.Unlock()
	return nil
}

func (m *WebsocketRoutineManager) ensureHandlersInit() {
	if m == nil || atomic.LoadUint32(&m.handlersInit) == 1 {
		return
	}
	m.mu.Lock()
	if m.handlersInit == 0 {
		m.dataHandlers.Store([]WebsocketDataHandler{})
		atomic.StoreUint32(&m.handlersInit, 1)
	}
	m.mu.Unlock()
}
