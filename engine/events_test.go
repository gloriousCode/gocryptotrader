package engine

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

const (
	testExchange = "Bitstamp"
)

func addValidEvent(bot *Engine) (int64, error) {
	return Add(bot, testExchange,
		ItemPrice,
		EventConditionParams{Condition: ConditionGreaterThan, Price: 1},
		currency.NewPair(currency.BTC, currency.USD),
		asset.Spot,
		"SMS,test")
}

func TestAdd(t *testing.T) {
	bot := createTestBot(t)
	_, err := Add(bot, testExchange, "", EventConditionParams{}, currency.Pair{}, "", "")
	if err == nil {
		t.Error("should err on invalid params")
	}

	_, err = addValidEvent(bot)
	if err != nil {
		t.Error("unexpected result", err)
	}

	_, err = addValidEvent(bot)
	if err != nil {
		t.Error("unexpected result", err)
	}

	if len(events) != 2 {
		t.Error("2 events should be stored")
	}
}

func TestRemove(t *testing.T) {
	bot := createTestBot(t)
	id, err := addValidEvent(bot)
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
	t.Parallel()
	bot := createTestBot(t)
	_, err := addValidEvent(bot)
	if err != nil {
		t.Error("unexpected result", err)
	}

	n, e := GetEventCounter()
	if n == 0 || e > 0 {
		t.Error("unexpected result")
	}

	events[0].Executed = true
	n, e = GetEventCounter()
	if n == 0 || e == 0 {
		t.Error("unexpected result")
	}
}

func TestExecuteAction(t *testing.T) {
	bot := createTestBot(t)
	err := bm.LoadBot(bot)
	if err != nil {
		t.Error(err)
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
	bot := createTestBot(t)
	bot.Settings.Verbose = true

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
	if r := e.processTicker(true); r {
		t.Error("unexpected result")
	}

	// now populate it with a number > 0
	tick.Last = 1337
	if err := ticker.ProcessTicker(&tick); err != nil {
		t.Fatal("unexpected result:", err)
	}
	if r := e.processTicker(true); !r {
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
		if r := e.processCondition(tester[x].Actual, tester[x].Threshold); r != tester[x].ExpectedResult {
			t.Error("unexpected result")
		}
	}
}

func TestProcessOrderbook(t *testing.T) {
	bot := createTestBot(t)
	bot.Settings.Verbose = true

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

	if r := e.processOrderbook(true); !r {
		t.Error("unexpected result")
	}
}

func TestCheckEventCondition(t *testing.T) {
	t.Parallel()
	e := Event{
		Item: ItemPrice,
	}
	if r := e.CheckEventCondition(true); r {
		t.Error("unexpected result")
	}

	e.Item = ItemOrderbook
	if r := e.CheckEventCondition(true); r {
		t.Error("unexpected result")
	}
}

func TestIsValidEvent(t *testing.T) {
	t.Parallel()
	bot := createTestBot(t)
	// invalid exchange name
	if err := IsValidEvent(bot, "meow", "", EventConditionParams{}, ""); err != errExchangeDisabled {
		t.Error("unexpected result:", err)
	}

	// invalid item
	if err := IsValidEvent(bot, testExchange, "", EventConditionParams{}, ""); err != errInvalidItem {
		t.Error("unexpected result:", err)
	}

	// invalid condition
	if err := IsValidEvent(bot, testExchange, ItemPrice, EventConditionParams{}, ""); err != errInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// valid condition but empty price which will still throw an errInvalidCondition
	c := EventConditionParams{
		Condition: ConditionGreaterThan,
	}
	if err := IsValidEvent(bot, testExchange, ItemPrice, c, ""); err != errInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// valid condition but empty orderbook amount will still still throw an errInvalidCondition
	if err := IsValidEvent(bot, testExchange, ItemOrderbook, c, ""); err != errInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// test action splitting, but invalid
	c.OrderbookAmount = 1337
	if err := IsValidEvent(bot, testExchange, ItemOrderbook, c, "a,meow"); err != errInvalidAction {
		t.Error("unexpected result:", err)
	}

	// check for invalid action without splitting
	if err := IsValidEvent(bot, testExchange, ItemOrderbook, c, "hi"); err != errInvalidAction {
		t.Error("unexpected result:", err)
	}

	// valid event
	if err := IsValidEvent(bot, testExchange, ItemOrderbook, c, "SMS,test"); err != nil {
		t.Error("unexpected result:", err)
	}
}

func TestIsValidExchange(t *testing.T) {
	t.Parallel()
	if s := IsValidExchange(nil, "invalidexchangerino"); s {
		t.Error("unexpected result")
	}
	bot := createTestBot(t)
	if s := IsValidExchange(bot, testExchange); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidCondition(t *testing.T) {
	t.Parallel()
	if s := IsValidCondition("invalidconditionerino"); s {
		t.Error("unexpected result")
	}
	if s := IsValidCondition(ConditionGreaterThan); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidAction(t *testing.T) {
	t.Parallel()
	if s := IsValidAction("invalidactionerino"); s {
		t.Error("unexpected result")
	}
	if s := IsValidAction(ActionSMSNotify); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidItem(t *testing.T) {
	t.Parallel()
	if s := IsValidItem("invaliditemerino"); s {
		t.Error("unexpected result")
	}
	if s := IsValidItem(ItemPrice); !s {
		t.Error("unexpected result")
	}
}
