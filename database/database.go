package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/goose"
)

func GetDBManager() (*Manager, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if dm.db == nil {
		dm.db = &Instance{}
	}
	return &dm, nil
}

// SetDBConfig ensures the config is set in a thread safe fashion
func (dm *Manager) SetDBConfig(cfg *Config) error {
	if cfg.ConnectionDetails.Database == "" {
		return ErrNoDatabaseProvided
	}
	dm.mu.Lock()
	dm.db.config = cfg
	dm.mu.Unlock()
	return nil
}

// SetDBPath ensures the path is set in a thread safe fashion
func (dm *Manager) SetDBPath(path string) {
	dm.mu.Lock()
	dm.db.dataPath = path
	dm.mu.Unlock()
}

// SupportedDrivers slice of supported database driver types
func SupportedDrivers() []string {
	return []string{DBSQLite, DBSQLite3, DBPostgreSQL}
}

// CheckConnection checks the status of the db connection
func (dm *Manager) CheckConnection() bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if dm.db.sql == nil {
		return false
	}
	err := dm.db.sql.Ping()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Database connection error: %v\n", err)
		dm.db.isConnected = false
	}

	if !dm.db.isConnected {
		log.Info(log.DatabaseMgr, "Database connection reestablished")
		dm.db.isConnected = true
	}
	return dm.db.isConnected
}

// SetConnectionStatus sets the connection status in a thread safe fashion
func (dm *Manager) SetConnectionStatus(status bool) {
	dm.mu.Lock()
	dm.db.isConnected = status
	dm.mu.Unlock()
}

// Connect will connect to the database depending on the driver provided
func (dm *Manager) Connect(driver string) (err error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	switch driver {
	case DBSQLite, DBSQLite3:
		databaseFullLocation := filepath.Join(dm.db.dataPath, dm.db.config.Database)
		dbConn, err := sql.Open("sqlite3", databaseFullLocation)
		if err != nil {
			return err
		}
		dm.db.sql = dbConn
		dm.db.sql.SetMaxOpenConns(1)

	case DBPostgreSQL:
		if dm.db.config.SSLMode == "" {
			dm.db.config.SSLMode = "disable"
		}

		configDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			dm.db.config.Username,
			dm.db.config.Password,
			dm.db.config.Host,
			dm.db.config.Port,
			dm.db.config.Database,
			dm.db.config.SSLMode)
		dbConn, err := sql.Open(DBPostgreSQL, configDSN)
		if err != nil {
			return err
		}
		err = dbConn.Ping()
		if err != nil {
			return err
		}

		dm.db.sql = dbConn
		dm.db.sql.SetMaxOpenConns(2)
		dm.db.sql.SetMaxIdleConns(1)
		dm.db.sql.SetConnMaxLifetime(time.Hour)
	default:
		return errors.New(DBInvalidDriver)
	}
	return err
}

// RunGooseCommand will run a goose command against the database
func (dm *Manager) RunGooseCommand(command, migrationDir, args string) error {
	dialect := dm.GetSQLDialect()
	return goose.Run(command, dm.db.sql, dialect, migrationDir, args)
}

// GetSQLDialect returns current SQL Dialect based on enabled driver
func (dm *Manager) GetSQLDialect() string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	switch dm.db.config.Driver {
	case "sqlite", "sqlite3":
		return DBSQLite3
	case "psql", "postgres", "postgresql":
		return DBPostgreSQL
	}
	return DBInvalidDriver
}

// Executor sadly returns the db.SQL obj
func (dm *Manager) Executor() *sql.DB {
	return dm.db.sql
}

// BeginTransaction returns a transaction without exposing db.SQL
func (dm *Manager) BeginTransaction(ctx context.Context) (*sql.Tx, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.db.sql.BeginTx(ctx, nil)
}

// Close closes the connection
func (dm *Manager) Close() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.db.sql.Close()
}
