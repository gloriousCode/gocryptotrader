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

func (a *databaseManager) Start(cfg database.Config) (err error) {
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
	if cfg.Enabled {
		if cfg.Driver == database.DBPostgreSQL {
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to host %s/%s utilising %s driver\n",
				cfg.Host,
				cfg.Database,
				cfg.Driver)
		} else if cfg.Driver == database.DBSQLite ||
			cfg.Driver == database.DBSQLite3 {
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to %s utilising %s driver\n",
				cfg.Database,
				cfg.Driver)
		}
		dbManager, err := database.GetDBManager()
		if err != nil {
			return fmt.Errorf("failed to setup database: %v Some features that utilise a database will be unavailable", err)
		}
		err = dbManager.Connect(cfg.Driver)
		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}

		dbManager.SetConnectionStatus(true)

		DBLogger := database.Logger{}
		if cfg.Verbose {
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
	dbManager, err := database.GetDBManager()
	if err != nil {
		return err
	}
	err = dbManager.Close()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to close database: %v", err)
	}

	close(a.shutdown)
	return nil
}

func (a *databaseManager) run() {
	err := AddToServiceWG(1)
	if err != nil {
		return
	}
	log.Debugln(log.DatabaseMgr, "Database manager started.")

	t := time.NewTicker(time.Second * 2)

	defer func() {
		t.Stop()
		atomic.CompareAndSwapInt32(&a.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&a.started, 1, 0)
		err = CompleteServiceWG(1)
		if err != nil {
			return
		}
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
	dbManager, err := database.GetDBManager()
	if err != nil {
		return
	}
	dbManager.CheckConnection()
}
