package engine

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestLoadUnload(t *testing.T) {
	t.Parallel()
	var botManager BotManager
	err := botManager.LoadBot(nil)
	if err == nil {
		t.Error("expected error")
	}
	if err != nil && err.Error() != "will not load nil bots" {
		t.Error(err)
	}
	e1 := &Engine{Config: &config.Config{Name: "1"}}
	e2 := &Engine{Config: &config.Config{Name: "2"}}
	err = botManager.LoadBot(e1)
	if err != nil {
		t.Error(err)
	}
	// load twice
	err = botManager.LoadBot(e2)
	if err == nil {
		t.Error("expected error")
	}
	if err != nil && err.Error() != "bot already loaded" {
		t.Error(err)
	}
	// unload reload
	err = botManager.UnloadBot()
	if err != nil {
		t.Error(err)
	}
	err = botManager.LoadBot(e2)
	if err != nil {
		t.Error(err)
	}
	gb, err := botManager.GetBot()
	if err != nil {
		t.Error(err)
	}
	if gb.Config.Name != e2.Config.Name {
		t.Errorf("unknown bot loaded, expected %v, received %v", e2.Config.Name, gb.Config.Name)
	}
}

func TestStartBot(t *testing.T) {
	t.Parallel()
	// no bot test
	var botManager BotManager
	err := botManager.StartBot()
	if err == nil {
		t.Error("expected error")
	}
	if err != nil && err.Error() != "bot not yet loaded" {
		t.Error(err)
	}
	// real test
	e1 := &Engine{}
	e1.Config = &config.Config{}
	err = e1.Config.LoadConfig(config.TestFile, true)
	if err != nil {
		t.Error(err)
	}
	for i := range e1.Config.Exchanges {
		e1.Config.Exchanges[i].Enabled = false
		if e1.Config.Exchanges[i].Name == testExchange {
			e1.Config.Exchanges[i].Enabled = true
		}
	}

	err = botManager.LoadBot(e1)
	if err != nil {
		t.Error(err)
	}
	err = botManager.StartBot()
	if err != nil {
		t.Error(err)
	}
}

func TestStopBot(t *testing.T) {
	t.Parallel()
	// no bot test
	var botManager BotManager
	err := botManager.StopBot()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "no bot loaded" {
		t.Error(err)
	}
	// no bot started test
	e1 := &Engine{}
	e1.Config = &config.Config{}
	err = e1.Config.LoadConfig(config.TestFile, true)
	if err != nil {
		t.Error(err)
	}
	for i := range e1.Config.Exchanges {
		e1.Config.Exchanges[i].Enabled = false
		if e1.Config.Exchanges[i].Name == testExchange {
			e1.Config.Exchanges[i].Enabled = true
		}
	}
	err = botManager.LoadBot(e1)
	if err != nil {
		t.Error(err)
	}
	err = botManager.StopBot()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "no bot started" {
		t.Error(err)
	}
	// real test
	err = botManager.StartBot()
	if err != nil {
		t.Error(err)
	}
	err = botManager.StopBot()
	if err != nil {
		t.Error(err)
	}
}

func TestUnloadBot(t *testing.T) {
	t.Parallel()
	// no bot test
	var botManager BotManager
	err := botManager.UnloadBot()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "no bot loaded" {
		t.Error(err)
	}
	// unload while running test
	e1 := &Engine{}
	e1.Config = &config.Config{}
	err = e1.Config.LoadConfig(config.TestFile, true)
	if err != nil {
		t.Error(err)
	}
	for i := range e1.Config.Exchanges {
		e1.Config.Exchanges[i].Enabled = false
		if e1.Config.Exchanges[i].Name == testExchange {
			e1.Config.Exchanges[i].Enabled = true
		}
	}
	err = botManager.LoadBot(e1)
	if err != nil {
		t.Error(err)
	}
	err = botManager.StartBot()
	if err != nil {
		t.Error(err)
	}
	err = botManager.UnloadBot()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "cannot unload running bot" {
		t.Error(err)
	}
	// valid test
	err = botManager.StopBot()
	if err != nil {
		t.Error(err)
	}
	err = botManager.UnloadBot()
	if err != nil {
		t.Error(err)
	}
}

func TestGetBot(t *testing.T) {
	t.Parallel()
	var botManager BotManager
	b, err := botManager.GetBot()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "no bot loaded" {
		t.Error(err)
	}
	if b != nil {
		t.Error("bot should be nil")
	}
	e1 := &Engine{Config: &config.Config{Name: "1"}}
	err = botManager.LoadBot(e1)
	if err != nil {
		t.Error(err)
	}
	b, err = botManager.GetBot()
	if err != nil {
		t.Error(err)
	}
	if b.Config.Name != e1.Config.Name {
		t.Errorf("unknown bot loaded, expected %v, received %v", e1.Config.Name, b.Config.Name)
	}
}
