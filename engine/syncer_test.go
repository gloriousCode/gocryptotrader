package engine

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestNewCurrencyPairSyncer(t *testing.T) {
	t.Skip()
	bot, _ := Bot()
	if bot == nil {
		bot = new(Engine)
	}
	bot.Config = &config.Cfg
	err := bot.Config.LoadConfig("", true)
	if err != nil {
		t.Fatalf("TestNewExchangeSyncer: Failed to load config: %s", err)
	}

	bot.Settings.DisableExchangeAutoPairUpdates = true
	bot.Settings.Verbose = true
	bot.Settings.EnableExchangeWebsocketSupport = true

	bot.SetupExchanges()

	if err != nil {
		t.Log("failed to start exchange syncer")
	}

	bot.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(CurrencyPairSyncerConfig{
		SyncTicker:       true,
		SyncOrderbook:    false,
		SyncTrades:       false,
		SyncContinuously: false,
	})
	if err != nil {
		t.Errorf("NewCurrencyPairSyncer failed: err %s", err)
	}

	bot.ExchangeCurrencyPairManager.Start()
	time.Sleep(time.Second * 15)
	bot.ExchangeCurrencyPairManager.Stop()
}
