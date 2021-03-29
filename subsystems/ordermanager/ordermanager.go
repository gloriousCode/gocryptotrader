package ordermanager

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

// vars for the fund manager package
var (
	orderManagerDelay      = time.Second * 10
	ErrOrdersAlreadyExists = errors.New("order already exists")
	ErrOrderNotFound       = errors.New("order does not exist")
)

// Started returns the status of the OrderManager
func (m *Manager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

// Start will boot up the OrderManager
func (m *Manager) Start(bot *engine.Engine) error {
	if bot == nil {
		return errors.New("cannot start with nil bot")
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("order manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}
	log.Debugln(log.OrderBook, "Order manager starting...")

	m.shutdown = make(chan struct{})
	m.orderStore.Orders = make(map[string][]*order.Detail)
	m.orderStore.bot = bot

	go m.run()
	return nil
}

// Stop will attempt to shutdown the OrderManager
func (m *Manager) Stop() error {
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", subsystems.ErrSubSystemNotStarted)
	}

	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	log.Debugln(log.OrderBook, "Order manager shutting down...")
	close(m.shutdown)
	return nil
}

func (m *Manager) gracefulShutdown() {
	if m.cfg.CancelOrdersOnShutdown {
		log.Debugln(log.OrderMgr, "Order manager: Cancelling any open orders...")
		m.CancelAllOrders(m.orderStore.bot.Config.GetEnabledExchanges())
	}
}

func (m *Manager) run() {
	log.Debugln(log.OrderBook, "Order manager started.")
	tick := time.NewTicker(orderManagerDelay)
	m.orderStore.bot.ServicesWG.Add(1)
	defer func() {
		log.Debugln(log.OrderMgr, "Order manager shutdown.")
		tick.Stop()
		m.orderStore.bot.ServicesWG.Done()
	}()

	for {
		select {
		case <-m.shutdown:
			m.gracefulShutdown()
			return
		case <-tick.C:
			go m.processOrders()
		}
	}
}

// CancelAllOrders iterates and cancels all orders for each exchange provided
func (m *Manager) CancelAllOrders(exchangeNames []string) {
	orders := m.orderStore.get()
	if orders == nil {
		return
	}

	for k, v := range orders {
		log.Debugf(log.OrderMgr, "Order manager: Cancelling order(s) for exchange %s.", k)
		if !common.StringDataCompareInsensitive(exchangeNames, k) {
			continue
		}

		for y := range v {
			err := m.Cancel(&order.Cancel{
				Exchange:      k,
				ID:            v[y].ID,
				AccountID:     v[y].AccountID,
				ClientID:      v[y].ClientID,
				WalletAddress: v[y].WalletAddress,
				Type:          v[y].Type,
				Side:          v[y].Side,
				Pair:          v[y].Pair,
				AssetType:     v[y].AssetType,
			})
			if err != nil {
				log.Error(log.OrderMgr, err)
				continue
			}
		}
	}
}

// Cancel will find the order in the OrderManager, send a cancel request
// to the exchange and if successful, update the status of the order
func (m *Manager) Cancel(cancel *order.Cancel) error {
	var err error
	defer func() {
		if err != nil {
			m.orderStore.bot.CommsManager.PushEvent(base.Event{
				Type:    "order",
				Message: err.Error(),
			})
		}
	}()

	if cancel == nil {
		err = errors.New("order cancel param is nil")
		return err
	}
	if cancel.Exchange == "" {
		err = errors.New("order exchange name is empty")
		return err
	}
	if cancel.ID == "" {
		err = errors.New("order id is empty")
		return err
	}

	exch := m.orderStore.bot.GetExchangeByName(cancel.Exchange)
	if exch == nil {
		err = exchangemanager.ErrExchangeNotFound
		return err
	}

	if cancel.AssetType.String() != "" && !exch.GetAssetTypes().Contains(cancel.AssetType) {
		err = errors.New("order asset type not supported by exchange")
		return err
	}

	log.Debugf(log.OrderMgr, "Order manager: Cancelling order ID %v [%+v]",
		cancel.ID, cancel)

	err = exch.CancelOrder(cancel)
	if err != nil {
		err = fmt.Errorf("%v - Failed to cancel order: %v", cancel.Exchange, err)
		return err
	}
	var od *order.Detail
	od, err = m.orderStore.getByExchangeAndID(cancel.Exchange, cancel.ID)
	if err != nil {
		err = fmt.Errorf("%v - Failed to retrieve order %v to update cancelled status: %v", cancel.Exchange, cancel.ID, err)
		return err
	}

	od.Status = order.Cancelled
	msg := fmt.Sprintf("Order manager: Exchange %s order ID=%v cancelled.",
		od.Exchange, od.ID)
	log.Debugln(log.OrderMgr, msg)
	m.orderStore.bot.CommsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})

	return nil
}

// GetOrderInfo calls the exchange's wrapper GetOrderInfo function
// and stores the result in the order manager
func (m *Manager) GetOrderInfo(exchangeName, orderID string, cp currency.Pair, a asset.Item) (order.Detail, error) {
	if orderID == "" {
		return order.Detail{}, errOrderCannotBeEmpty
	}

	exch := m.orderStore.bot.GetExchangeByName(exchangeName)
	if exch == nil {
		return order.Detail{}, exchangemanager.ErrExchangeNotFound
	}
	result, err := exch.GetOrderInfo(orderID, cp, a)
	if err != nil {
		return order.Detail{}, err
	}

	err = m.orderStore.add(&result)
	if err != nil && err != ErrOrdersAlreadyExists {
		return order.Detail{}, err
	}

	return result, nil
}

func (m *Manager) validate(newOrder *order.Submit) error {
	if newOrder == nil {
		return errors.New("order cannot be nil")
	}

	if newOrder.Exchange == "" {
		return errors.New("order exchange name must be specified")
	}

	if err := newOrder.Validate(); err != nil {
		return err
	}

	if m.cfg.EnforceLimitConfig {
		if !m.cfg.AllowMarketOrders && newOrder.Type == order.Market {
			return errors.New("order market type is not allowed")
		}

		if m.cfg.LimitAmount > 0 && newOrder.Amount > m.cfg.LimitAmount {
			return errors.New("order limit exceeds allowed limit")
		}

		if len(m.cfg.AllowedExchanges) > 0 &&
			!common.StringDataCompareInsensitive(m.cfg.AllowedExchanges, newOrder.Exchange) {
			return errors.New("order exchange not found in allowed list")
		}

		if len(m.cfg.AllowedPairs) > 0 && !m.cfg.AllowedPairs.Contains(newOrder.Pair, true) {
			return errors.New("order pair not found in allowed list")
		}
	}
	return nil
}

// Submit will take in an order struct, send it to the exchange and
// populate it in the OrderManager if successful
func (m *Manager) Submit(newOrder *order.Submit) (*orderSubmitResponse, error) {
	err := m.validate(newOrder)
	if err != nil {
		return nil, err
	}
	exch := m.orderStore.bot.GetExchangeByName(newOrder.Exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}
	var result order.SubmitResponse
	result, err = exch.SubmitOrder(newOrder)
	if err != nil {
		return nil, err
	}

	return m.processSubmittedOrder(newOrder, result)
}

// SubmitFakeOrder runs through the same process as order submission
// but does not touch live endpoints
func (m *Manager) SubmitFakeOrder(newOrder *order.Submit, resultingOrder order.SubmitResponse) (*orderSubmitResponse, error) {
	err := m.validate(newOrder)
	if err != nil {
		return nil, err
	}
	exch := m.orderStore.bot.GetExchangeByName(newOrder.Exchange)
	if exch == nil {
		return nil, exchangemanager.ErrExchangeNotFound
	}

	return m.processSubmittedOrder(newOrder, resultingOrder)
}

// GetOrdersSnapshot returns a snapshot of all orders in the orderstore. It optionally filters any orders that do not match the status
// but a status of "" or ANY will include all
// the time adds contexts for the when the snapshot is relevant for
func (m *Manager) GetOrdersSnapshot(s order.Status) ([]order.Detail, time.Time) {
	var os []order.Detail
	var latestUpdate time.Time
	for _, v := range m.orderStore.Orders {
		for i := range v {
			if s != v[i].Status &&
				s != order.AnyStatus &&
				s != "" {
				continue
			}
			if v[i].LastUpdated.After(latestUpdate) {
				latestUpdate = v[i].LastUpdated
			}

			cpy := *v[i]
			os = append(os, cpy)
		}
	}

	return os, latestUpdate
}

func (m *Manager) processSubmittedOrder(newOrder *order.Submit, result order.SubmitResponse) (*orderSubmitResponse, error) {
	if !result.IsOrderPlaced {
		return nil, errors.New("order unable to be placed")
	}

	id, err := uuid.NewV4()
	if err != nil {
		log.Warnf(log.OrderMgr,
			"Order manager: Unable to generate UUID. Err: %s",
			err)
	}
	msg := fmt.Sprintf("Order manager: Exchange %s submitted order ID=%v [Ours: %v] pair=%v price=%v amount=%v side=%v type=%v for time %v.",
		newOrder.Exchange,
		result.OrderID,
		id.String(),
		newOrder.Pair,
		newOrder.Price,
		newOrder.Amount,
		newOrder.Side,
		newOrder.Type,
		newOrder.Date)

	log.Debugln(log.OrderMgr, msg)
	m.orderStore.bot.CommsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})
	status := order.New
	if result.FullyMatched {
		status = order.Filled
	}
	err = m.orderStore.add(&order.Detail{
		ImmediateOrCancel: newOrder.ImmediateOrCancel,
		HiddenOrder:       newOrder.HiddenOrder,
		FillOrKill:        newOrder.FillOrKill,
		PostOnly:          newOrder.PostOnly,
		Price:             newOrder.Price,
		Amount:            newOrder.Amount,
		LimitPriceUpper:   newOrder.LimitPriceUpper,
		LimitPriceLower:   newOrder.LimitPriceLower,
		TriggerPrice:      newOrder.TriggerPrice,
		TargetAmount:      newOrder.TargetAmount,
		ExecutedAmount:    newOrder.ExecutedAmount,
		RemainingAmount:   newOrder.RemainingAmount,
		Fee:               newOrder.Fee,
		Exchange:          newOrder.Exchange,
		InternalOrderID:   id.String(),
		ID:                result.OrderID,
		AccountID:         newOrder.AccountID,
		ClientID:          newOrder.ClientID,
		WalletAddress:     newOrder.WalletAddress,
		Type:              newOrder.Type,
		Side:              newOrder.Side,
		Status:            status,
		AssetType:         newOrder.AssetType,
		Date:              time.Now(),
		LastUpdated:       time.Now(),
		Pair:              newOrder.Pair,
		Leverage:          newOrder.Leverage,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to add %v order %v to orderStore: %s", newOrder.Exchange, result.OrderID, err)
	}

	return &orderSubmitResponse{
		SubmitResponse: order.SubmitResponse{
			IsOrderPlaced: result.IsOrderPlaced,
			OrderID:       result.OrderID,
		},
		InternalOrderID: id.String(),
	}, nil
}

func (m *Manager) processOrders() {
	authExchanges := m.orderStore.bot.GetAuthAPISupportedExchanges()
	for x := range authExchanges {
		log.Debugf(log.OrderMgr,
			"Order manager: Processing orders for exchange %v.",
			authExchanges[x])

		exch := m.orderStore.bot.GetExchangeByName(authExchanges[x])
		supportedAssets := exch.GetAssetTypes()
		for y := range supportedAssets {
			pairs, err := exch.GetEnabledPairs(supportedAssets[y])
			if err != nil {
				log.Errorf(log.OrderMgr,
					"Order manager: Unable to get enabled pairs for %s and asset type %s: %s",
					authExchanges[x],
					supportedAssets[y],
					err)
				continue
			}

			if len(pairs) == 0 {
				if m.orderStore.bot.Settings.Verbose {
					log.Debugf(log.OrderMgr,
						"Order manager: No pairs enabled for %s and asset type %s, skipping...",
						authExchanges[x],
						supportedAssets[y])
				}
				continue
			}

			req := order.GetOrdersRequest{
				Side:      order.AnySide,
				Type:      order.AnyType,
				Pairs:     pairs,
				AssetType: supportedAssets[y],
			}
			result, err := exch.GetActiveOrders(&req)
			if err != nil {
				log.Warnf(log.OrderMgr,
					"Order manager: Unable to get active orders for %s and asset type %s: %s",
					authExchanges[x],
					supportedAssets[y],
					err)
				continue
			}

			for z := range result {
				ord := &result[z]
				result := m.orderStore.add(ord)
				if result != ErrOrdersAlreadyExists {
					msg := fmt.Sprintf("Order manager: Exchange %s added order ID=%v pair=%v price=%v amount=%v side=%v type=%v.",
						ord.Exchange, ord.ID, ord.Pair, ord.Price, ord.Amount, ord.Side, ord.Type)
					log.Debugf(log.OrderMgr, "%v", msg)
					m.orderStore.bot.CommsManager.PushEvent(base.Event{
						Type:    "order",
						Message: msg,
					})
					continue
				}
			}
		}
	}
}
