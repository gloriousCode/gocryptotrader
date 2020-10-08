package engine

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/sqlboiler/boil"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type databaseManager struct {
	started  int32
	stopped  int32
	shutdown chan struct{}
}

func (a *databaseManager) Started() bool {
	return atomic.LoadInt32(&a.started) == 1
}

func (a *databaseManager) Start() (err error) {
	if atomic.AddInt32(&a.started, 1) != 1 {
		return errors.New("database manager already started")
	}

	defer func() {
		if err != nil {
			atomic.CompareAndSwapInt32(&a.started, 1, 0)
		}
	}()

	log.Debugln(log.DatabaseMgr, "Database manager starting...")

	a.shutdown = make(chan struct{})
	bot, err := Bot()
	if err != nil {
		return err
	}
	if bot.Config.Database.Enabled {
		if bot.Config.Database.Driver == database.DBPostgreSQL {
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to host %s/%s utilising %s driver\n",
				bot.Config.Database.Host,
				bot.Config.Database.Database,
				bot.Config.Database.Driver)
		} else if bot.Config.Database.Driver == database.DBSQLite ||
			bot.Config.Database.Driver == database.DBSQLite3 {
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to %s utilising %s driver\n",
				bot.Config.Database.Database,
				bot.Config.Database.Driver)
		}
		err = database.Connect(bot.Config.Database.Driver)
		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}

		database.SetConnectionStatus(true)

		DBLogger := database.Logger{}
		if bot.Config.Database.Verbose {
			boil.DebugMode = true
			boil.DebugWriter = DBLogger
		}

		go a.run()
		return nil
	}

	return errors.New("database support disabled")
}

func (a *databaseManager) Stop() error {
	if atomic.LoadInt32(&a.started) == 0 {
		return errors.New("database manager not started")
	}

	if atomic.AddInt32(&a.stopped, 1) != 1 {
		return errors.New("database manager is already stopping")
	}

	err := database.Close()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to close database: %v", err)
	}

	close(a.shutdown)
	return nil
}

func (a *databaseManager) run() {
	log.Debugln(log.DatabaseMgr, "Database manager started.")
	bot, err := Bot()
	if err != nil {
		return
	}
	bot.ServicesWG.Add(1)

	t := time.NewTicker(time.Second * 2)

	defer func() {
		t.Stop()
		atomic.CompareAndSwapInt32(&a.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&a.started, 1, 0)
		bot.ServicesWG.Done()
		log.Debugln(log.DatabaseMgr, "Database manager shutdown.")
	}()

	for {
		select {
		case <-a.shutdown:
			return
		case <-t.C:
			a.checkConnection()
		}
	}
}

func (a *databaseManager) checkConnection() {
	database.CheckConnection()
}
