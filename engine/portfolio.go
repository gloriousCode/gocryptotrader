package engine

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// vars for the fund manager package
var (
	PortfolioSleepDelay = time.Minute
)

type portfolioManager struct {
	started  int32
	stopped  int32
	shutdown chan struct{}
}

func (p *portfolioManager) Started() bool {
	return atomic.LoadInt32(&p.started) == 1
}

func (p *portfolioManager) Start(bot *Engine) error {
	if atomic.AddInt32(&p.started, 1) != 1 {
		return errors.New("portfolio manager already started")
	}

	log.Debugln(log.PortfolioMgr, "Portfolio manager starting...")
	bot.Portfolio = &portfolio.Portfolio
	bot.Portfolio.Seed(bot.Config.Portfolio)
	p.shutdown = make(chan struct{})
	portfolio.Verbose = IsBotVerbose()

	go p.run(bot)
	return nil
}
func (p *portfolioManager) Stop() error {
	if atomic.AddInt32(&p.stopped, 1) != 1 {
		return errors.New("portfolio manager is already stopped")
	}

	log.Debugln(log.PortfolioMgr, "Portfolio manager shutting down...")
	close(p.shutdown)
	return nil
}

func (p *portfolioManager) run(bot *Engine) {
	log.Debugln(log.PortfolioMgr, "Portfolio manager started.")
	bot.ServicesWG.Add(1)
	tick := time.NewTicker(PortfolioSleepDelay)
	defer func() {
		atomic.CompareAndSwapInt32(&p.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&p.started, 1, 0)
		tick.Stop()
		bot.ServicesWG.Done()
		log.Debugf(log.PortfolioMgr, "Portfolio manager shutdown.")
	}()

	p.processPortfolio(bot)
	for {
		select {
		case <-p.shutdown:
			return
		case <-tick.C:
			p.processPortfolio(bot)
		}
	}
}

func (p *portfolioManager) processPortfolio(bot *Engine) {
	pf := portfolio.GetPortfolio()
	data := pf.GetPortfolioGroupedCoin()
	for key, value := range data {
		err := pf.UpdatePortfolio(value, key)
		if err != nil {
			log.Errorf(log.PortfolioMgr,
				"PortfolioWatcher error %s for currency %s\n",
				err,
				key)
			continue
		}

		log.Debugf(log.PortfolioMgr,
			"Portfolio manager: Successfully updated address balance for %s address(es) %s\n",
			key,
			value)
	}
	SeedExchangeAccountInfo(bot.GetAllEnabledExchangeAccountInfo().Data)
}
