package eventmanager

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/engine"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

const (
	testExchange = "Bitstamp"
)

func addValidEvent() (int64, error) {
	return Add(testExchange,
		ItemPrice,
		EventConditionParams{Condition: ConditionGreaterThan, Price: 1},
		currency.NewPair(currency.BTC, currency.USD),
		asset.Spot,
		"SMS,test")
}

func TestAdd(t *testing.T) {
	bot := engine.CreateTestBot(t)
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}
	_, err := Add("", "", EventConditionParams{}, currency.Pair{}, "", "")
	if err == nil {
		t.Error("should err on invalid params")
	}

	_, err = addValidEvent()
	if err != nil {
		t.Error("unexpected result", err)
	}

	_, err = addValidEvent()
	if err != nil {
		t.Error("unexpected result", err)
	}

	if len(Events) != 2 {
		t.Error("2 events should be stored")
	}
}

func TestRemove(t *testing.T) {
	bot := engine.CreateTestBot(t)
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}
	id, err := addValidEvent()
	if err != nil {
		t.Error("unexpected result", err)
	}

	if s := Remove(id); !s {
		t.Error("unexpected result")
	}

	if s := Remove(id); s {
		t.Error("unexpected result")
	}
}

func TestGetEventCounter(t *testing.T) {
	bot := engine.CreateTestBot(t)
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}
	_, err := addValidEvent()
	if err != nil {
		t.Error("unexpected result", err)
	}

	n, e := GetEventCounter()
	if n == 0 || e > 0 {
		t.Error("unexpected result")
	}

	Events[0].Executed = true
	n, e = GetEventCounter()
	if n == 0 || e == 0 {
		t.Error("unexpected result")
	}
}

func TestExecuteAction(t *testing.T) {
	t.Parallel()
	bot := engine.CreateTestBot(t)
	if engine.Bot == nil {
		engine.Bot = bot
	}
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}

	var e Event
	if r := e.ExecuteAction(); !r {
		t.Error("unexpected result")
	}

	e.Action = "SMS,test"
	if r := e.ExecuteAction(); !r {
		t.Error("unexpected result")
	}

	e.Action = "SMS,ALL"
	if r := e.ExecuteAction(); !r {
		t.Error("unexpected result")
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	e := Event{
		Exchange: testExchange,
		Item:     ItemPrice,
		Condition: EventConditionParams{
			Condition: ConditionGreaterThan,
			Price:     1,
		},
		Pair:   currency.NewPair(currency.BTC, currency.USD),
		Asset:  asset.Spot,
		Action: "SMS,ALL",
	}

	if r := e.String(); r != "If the BTCUSD [SPOT] PRICE on Bitstamp meets the following {> 1 false false 0} then SMS,ALL." {
		t.Error("unexpected result")
	}
}

func TestProcessTicker(t *testing.T) {
	e := Event{
		Exchange: testExchange,
		Pair:     currency.NewPair(currency.BTC, currency.USD),
		Asset:    asset.Spot,
		Condition: EventConditionParams{
			Condition: ConditionGreaterThan,
			Price:     1,
		},
	}

	// now populate it with a 0 entry
	tick := ticker.Price{
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		ExchangeName: e.Exchange,
		AssetType:    e.Asset,
	}
	if err := ticker.ProcessTicker(&tick); err != nil {
		t.Fatal("unexpected result:", err)
	}
	if r := e.processTicker(false); r {
		t.Error("unexpected result")
	}

	// now populate it with a number > 0
	tick.Last = 1337
	if err := ticker.ProcessTicker(&tick); err != nil {
		t.Fatal("unexpected result:", err)
	}
	if r := e.processTicker(false); !r {
		t.Error("unexpected result")
	}
}

func TestProcessCondition(t *testing.T) {
	t.Parallel()
	var e Event
	tester := []struct {
		Condition      string
		Actual         float64
		Threshold      float64
		ExpectedResult bool
	}{
		{ConditionGreaterThan, 1, 2, false},
		{ConditionGreaterThan, 2, 1, true},
		{ConditionGreaterThanOrEqual, 1, 2, false},
		{ConditionGreaterThanOrEqual, 2, 1, true},
		{ConditionIsEqual, 1, 1, true},
		{ConditionIsEqual, 1, 2, false},
		{ConditionLessThan, 1, 2, true},
		{ConditionLessThan, 2, 1, false},
		{ConditionLessThanOrEqual, 1, 2, true},
		{ConditionLessThanOrEqual, 2, 1, false},
	}
	for x := range tester {
		e.Condition.Condition = tester[x].Condition
		if r := e.shouldProcessEvent(tester[x].Actual, tester[x].Threshold); r != tester[x].ExpectedResult {
			t.Error("unexpected result")
		}
	}
}

func TestProcessOrderbook(t *testing.T) {
	e := Event{
		Exchange: testExchange,
		Pair:     currency.NewPair(currency.BTC, currency.USD),
		Asset:    asset.Spot,
		Condition: EventConditionParams{
			Condition:        ConditionGreaterThan,
			CheckBidsAndAsks: true,
			OrderbookAmount:  100,
		},
	}

	// now populate it with a 0 entry
	o := orderbook.Base{
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		Bids:         []orderbook.Item{{Amount: 24, Price: 23}},
		Asks:         []orderbook.Item{{Amount: 24, Price: 23}},
		ExchangeName: e.Exchange,
		AssetType:    e.Asset,
	}
	if err := o.Process(); err != nil {
		t.Fatal("unexpected result:", err)
	}

	if r := e.processOrderbook(false); !r {
		t.Error("unexpected result")
	}
}

func TestCheckEventCondition(t *testing.T) {
	t.Parallel()
	if engine.Bot == nil {
		engine.Bot = new(engine.Engine)
	}

	e := Event{
		Item: ItemPrice,
	}
	if r := e.CheckEventCondition(false); r {
		t.Error("unexpected result")
	}

	e.Item = ItemOrderbook
	if r := e.CheckEventCondition(false); r {
		t.Error("unexpected result")
	}
}

func TestIsValidEvent(t *testing.T) {
	bot := engine.CreateTestBot(t)
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}
	// invalid exchange name
	if err := isValidEvent("meow", "", EventConditionParams{}, ""); err != ErrExchangeDisabled {
		t.Error("unexpected result:", err)
	}

	// invalid item
	if err := isValidEvent(testExchange, "", EventConditionParams{}, ""); err != ErrInvalidItem {
		t.Error("unexpected result:", err)
	}

	// invalid condition
	if err := isValidEvent(testExchange, ItemPrice, EventConditionParams{}, ""); err != ErrInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// valid condition but empty price which will still throw an ErrInvalidCondition
	c := EventConditionParams{
		Condition: ConditionGreaterThan,
	}
	if err := isValidEvent(testExchange, ItemPrice, c, ""); err != ErrInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// valid condition but empty orderbook amount will still still throw an ErrInvalidCondition
	if err := isValidEvent(testExchange, ItemOrderbook, c, ""); err != ErrInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// test action splitting, but invalid
	c.OrderbookAmount = 1337
	if err := isValidEvent(testExchange, ItemOrderbook, c, "a,meow"); err != ErrInvalidAction {
		t.Error("unexpected result:", err)
	}

	// check for invalid action without splitting
	if err := isValidEvent(testExchange, ItemOrderbook, c, "hi"); err != ErrInvalidAction {
		t.Error("unexpected result:", err)
	}

	// valid event
	if err := isValidEvent(testExchange, ItemOrderbook, c, "SMS,test"); err != nil {
		t.Error("unexpected result:", err)
	}
}

func TestIsValidExchange(t *testing.T) {
	t.Parallel()
	if s := isValidExchange("invalidexchangerino"); s {
		t.Error("unexpected result")
	}
	engine.CreateTestBot(t)
	if s := isValidExchange(testExchange); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidCondition(t *testing.T) {
	t.Parallel()
	if s := isValidCondition("invalidconditionerino"); s {
		t.Error("unexpected result")
	}
	if s := isValidCondition(ConditionGreaterThan); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidAction(t *testing.T) {
	t.Parallel()
	if s := isValidAction("invalidactionerino"); s {
		t.Error("unexpected result")
	}
	if s := isValidAction(ActionSMSNotify); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidItem(t *testing.T) {
	t.Parallel()
	if s := isValidItem("invaliditemerino"); s {
		t.Error("unexpected result")
	}
	if s := isValidItem(ItemPrice); !s {
		t.Error("unexpected result")
	}
}
