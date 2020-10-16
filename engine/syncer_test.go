package engine

import (
	"testing"
	"time"
)

func TestNewCurrencyPairSyncer(t *testing.T) {
	t.Skip()
	t.Parallel()
	bot := createTestBot(t)
	bot.Settings.DisableExchangeAutoPairUpdates = true
	bot.Settings.Verbose = true
	bot.Settings.EnableExchangeWebsocketSupport = true
	var err error
	bot.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(CurrencyPairSyncerConfig{
		SyncTicker:       true,
		SyncOrderbook:    false,
		SyncTrades:       false,
		SyncContinuously: false,
	})
	if err != nil {
		t.Errorf("NewCurrencyPairSyncer failed: err %s", err)
	}
	bot.ExchangeCurrencyPairManager.Start(bot)
	time.Sleep(time.Second * 5)
	bot.ExchangeCurrencyPairManager.Stop()
}
