package eventmanager

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/subsystems"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Setup loads and validates the communications manager config
func Setup(comManager iCommsManager, exchangeManager iExchangeManager, sleepDelay time.Duration, verbose bool) (*Manager, error) {
	if comManager == nil {
		return nil, errNilComManager
	}
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if sleepDelay <= 0 {
		sleepDelay = EventSleepDelay
	}
	return &Manager{
		comms:           comManager,
		exchangeManager: exchangeManager,
		verbose:         verbose,
		sleepDelay:      sleepDelay,
		shutdown:        make(chan struct{}),
	}, nil
}

// Start is the overarching routine that will iterate through the Events
// chain
func (m *Manager) Start() error {
	if m == nil {
		return fmt.Errorf("event manager %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("event manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}
	log.Debugf(log.EventMgr, "Event Manager started. SleepDelay: %v\n", EventSleepDelay.String())
	m.shutdown = make(chan struct{})
	go m.run()
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
		return fmt.Errorf("event manager %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("event manager %w", subsystems.ErrSubSystemNotStarted)
	}
	close(m.shutdown)
	return nil
}

func (m *Manager) run() {
	ticker := time.NewTicker(m.sleepDelay)
	select {
	case <-m.shutdown:
		return
	case <-ticker.C:
		total, executed := m.getEventCounter()
		if total > 0 && executed != total {
			m.m.Lock()
			for i := range m.events {
				m.executeEvent(i)
			}
			m.m.Unlock()
		}
	}
}

func (m *Manager) executeEvent(i int) {
	if !m.events[i].Executed {
		if m.verbose {
			log.Debugf(log.EventMgr, "Events: Processing event %s.\n", m.events[i].String())
		}
		err := m.checkEventCondition(&m.events[i])
		if err != nil {
			msg := fmt.Sprintf(
				"Events: ID: %d triggered on %s successfully [%v]\n", m.events[i].ID,
				m.events[i].Exchange, m.events[i].String(),
			)
			log.Infoln(log.EventMgr, msg)
			m.comms.PushEvent(base.Event{Type: "event", Message: msg})
			m.events[i].Executed = true
		} else {
			if m.verbose {
				log.Debugf(log.EventMgr, "%v", err)
			}
		}
	}
}

// Add adds an event to the Events chain and returns an index/eventID
// and an error
func (m *Manager) Add(exchange, item string, condition EventConditionParams, p currency.Pair, a asset.Item, action string) (int64, error) {
	if m == nil {
		return 0, fmt.Errorf("event manager %w", subsystems.ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return 0, fmt.Errorf("event manager %w", subsystems.ErrSubSystemNotStarted)
	}
	err := m.isValidEvent(exchange, item, condition, action)
	if err != nil {
		return 0, err
	}
	evt := Event{
		Exchange:  exchange,
		Item:      item,
		Condition: condition,
		Pair:      p,
		Asset:     a,
		Action:    action,
		Executed:  false,
	}
	m.m.Lock()
	if len(m.events) > 0 {
		evt.ID = int64(len(m.events) + 1)
	}
	m.events = append(m.events, evt)
	m.m.Unlock()

	return evt.ID, nil
}

// Remove deletes and event by its ID
func (m *Manager) Remove(eventID int64) bool {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return false
	}
	m.m.Lock()
	defer m.m.Unlock()
	for i := range m.events {
		if m.events[i].ID == eventID {
			m.events = append(m.events[:i], m.events[i+1:]...)
			return true
		}
	}
	return false
}

// getEventCounter displays the amount of total events on the chain and the
// events that have been executed.
func (m *Manager) getEventCounter() (total, executed int) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return 0, 0
	}
	m.m.Lock()
	defer m.m.Unlock()
	total = len(m.events)
	for i := range m.events {
		if m.events[i].Executed {
			executed++
		}
	}
	return total, executed
}

// checkEventCondition will check the event structure to see if there is a condition
// met
func (m *Manager) checkEventCondition(e *Event) error {
	if m == nil {
		return fmt.Errorf("event manager %w", subsystems.ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("event manager %w", subsystems.ErrSubSystemNotStarted)
	}
	if e == nil {
		return errNilEvent
	}
	if e.Item == ItemPrice {
		return e.processTicker()
	}
	return e.processOrderbook()
}

// isValidEvent checks the actions to be taken and returns an error if incorrect
func (m *Manager) isValidEvent(exchange, item string, condition EventConditionParams, action string) error {
	exchange = strings.ToUpper(exchange)
	item = strings.ToUpper(item)
	action = strings.ToUpper(action)

	if !m.isValidExchange(exchange) {
		return errExchangeDisabled
	}

	if !isValidItem(item) {
		return errInvalidItem
	}

	if !isValidCondition(condition.Condition) {
		return errInvalidCondition
	}

	if item == ItemPrice {
		if condition.Price <= 0 {
			return errInvalidCondition
		}
	}

	if item == ItemOrderbook {
		if condition.OrderbookAmount <= 0 {
			return errInvalidCondition
		}
	}

	if strings.Contains(action, ",") {
		a := strings.Split(action, ",")

		if a[0] != ActionSMSNotify {
			return errInvalidAction
		}
	} else if action != ActionConsolePrint && action != ActionTest {
		return errInvalidAction
	}

	return nil
}

// isValidExchange validates the exchange
func (m *Manager) isValidExchange(exchangeName string) bool {
	return m.exchangeManager.GetExchangeByName(exchangeName) != nil
}

// isValidCondition validates passed in condition
func isValidCondition(condition string) bool {
	switch condition {
	case ConditionGreaterThan, ConditionGreaterThanOrEqual, ConditionLessThan, ConditionLessThanOrEqual, ConditionIsEqual:
		return true
	}
	return false
}

// isValidItem validates passed in Item
func isValidItem(item string) bool {
	item = strings.ToUpper(item)
	switch item {
	case ItemPrice, ItemOrderbook:
		return true
	}
	return false
}
