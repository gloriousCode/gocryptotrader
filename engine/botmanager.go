package engine

import (
	"errors"
	"sync"
	"sync/atomic"

	gctlog "github.com/thrasher-corp/gocryptotrader/log"
)

// BotManager is a nice little storage spot
// for the engine and have some basic protections
type BotManager struct {
	bot       *Engine
	isLoaded  int32
	isStarted int32
	sync.RWMutex
}

// LoadBot will place the bot into the bot manager after some basic checks
func (bm *BotManager) LoadBot(b *Engine) error {
	bm.Lock()
	defer bm.Unlock()
	if atomic.LoadInt32(&bm.isLoaded) == 1 {
		return errors.New("bot already loaded")
	}
	if b == nil {
		return errors.New("will not load nil bots")
	}
	bm.bot = b
	if b.ServicesWG == nil {
		b.ServicesWG = new(sync.WaitGroup)
	}
	atomic.StoreInt32(&bm.isLoaded, 1)
	return nil
}

func (bm *BotManager) StartBot() error {
	if atomic.LoadInt32(&bm.isLoaded) == 0 {
		return errors.New("bot not yet loaded")
	}
	if atomic.LoadInt32(&bm.isStarted) == 1 {
		return errors.New("bot already started")
	}
	PrintSettings(&bm.bot.Settings)
	if err := bm.bot.Start(); err != nil {
		gctlog.Errorf(gctlog.Global, "Unable to start bot engine. Error: %s\n", err)
		return err
	}
	atomic.StoreInt32(&bm.isStarted, 1)

	return nil
}

func (bm *BotManager) StopBot() error {
	bm.Lock()
	defer bm.Unlock()
	if atomic.LoadInt32(&bm.isLoaded) == 0 {
		return errors.New("no bot loaded")
	}
	if atomic.LoadInt32(&bm.isStarted) == 0 {
		return errors.New("no bot started")
	}
	bm.bot.Stop()
	atomic.StoreInt32(&bm.isStarted, 0)

	return nil
}

// UnloadBot will remove the bot by stopping it, then killing it
func (bm *BotManager) UnloadBot() error {
	bm.Lock()
	defer bm.Unlock()
	if atomic.LoadInt32(&bm.isLoaded) == 0 {
		return errors.New("no bot loaded")
	}
	if atomic.LoadInt32(&bm.isStarted) == 1 {
		return errors.New("cannot unload running bot")
	}
	bm.bot = nil
	atomic.StoreInt32(&bm.isLoaded, 0)

	return nil
}

// IsBotVerbose is a quick check for verbosity
// so you don't need to load the bot, then check to do
// a println
func IsBotVerbose() bool {
	bm.RLock()
	defer bm.RUnlock()
	if atomic.LoadInt32(&bm.isLoaded) == 0 {
		return false
	}
	if bm.bot == nil {
		return false
	}
	isVerbose := bm.bot.Settings.Verbose
	return isVerbose
}

// GetBot will return the bot pointer with some basic checks
func (bm *BotManager) GetBot() (*Engine, error) {
	bm.RLock()
	defer bm.RUnlock()
	if atomic.LoadInt32(&bm.isLoaded) == 0 {
		return nil, errors.New("no bot loaded")
	}
	return bm.bot, nil
}
