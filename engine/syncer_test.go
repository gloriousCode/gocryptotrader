package engine

import (
	"testing"
	"time"
)

func TestNewCurrencyPairSyncer(t *testing.T) {
	bot := createTestBot(t)
	bot.Settings.DisableExchangeAutoPairUpdates = true
	bot.Settings.Verbose = true
	bot.Settings.EnableExchangeWebsocketSupport = true
	bot.SetupExchanges()
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

	bot.ExchangeCurrencyPairManager.Start(bot.GetExchanges())
	time.Sleep(time.Second * 15)
	bot.ExchangeCurrencyPairManager.Stop()
}
