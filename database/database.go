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

// SetDBConfig ensures the config is set in a thread safe fashion
func SetDBConfig(cfg *Config) {
	db.mu.Lock()
	db.config = cfg
	db.mu.Unlock()
}

// SetDBPath ensures the path is set in a thread safe fashion
func SetDBPath(path string) {
	db.mu.Lock()
	db.dataPath = path
	db.mu.Unlock()
}

// SupportedDrivers slice of supported database driver types
func SupportedDrivers() []string {
	return []string{DBSQLite, DBSQLite3, DBPostgreSQL}
}

// CheckConnection checks the status of the db connection
func CheckConnection() bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.sql == nil {
		return false
	}
	err := db.sql.Ping()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Database connection error: %v\n", err)
		db.isConnected = false
	}

	if !db.isConnected {
		log.Info(log.DatabaseMgr, "Database connection reestablished")
		db.isConnected = true
	}
	return db.isConnected
}

// SetConnectionStatus sets the connection status in a thread safe fashion
func SetConnectionStatus(status bool) {
	db.mu.Lock()
	db.isConnected = status
	db.mu.Unlock()
}

// Connect will connect to the database depending on the driver provided
func Connect(driver string) (err error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	switch driver {
	case DBSQLite, DBSQLite3:
		databaseFullLocation := filepath.Join(db.dataPath, db.config.Database)
		dbConn, err := sql.Open("sqlite3", databaseFullLocation)
		if err != nil {
			return err
		}
		db.sql = dbConn
		db.sql.SetMaxOpenConns(1)

	case DBPostgreSQL:
		if db.config.SSLMode == "" {
			db.config.SSLMode = "disable"
		}

		configDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			db.config.Username,
			db.config.Password,
			db.config.Host,
			db.config.Port,
			db.config.Database,
			db.config.SSLMode)
		dbConn, err := sql.Open(DBPostgreSQL, configDSN)
		if err != nil {
			return err
		}
		err = dbConn.Ping()
		if err != nil {
			return err
		}

		db.sql = dbConn
		db.sql.SetMaxOpenConns(2)
		db.sql.SetMaxIdleConns(1)
		db.sql.SetConnMaxLifetime(time.Hour)
	default:
		return errors.New(DBInvalidDriver)
	}
	return err
}

// RunGooseCommand will run a goose command against the database
func RunGooseCommand(command, migrationDir, args string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return goose.Run(command, db.sql, GetSQLDialect(), migrationDir, args)
}

// GetSQLDialect returns current SQL Dialect based on enabled driver
func GetSQLDialect() string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	switch db.config.Driver {
	case "sqlite", "sqlite3":
		return DBSQLite3
	case "psql", "postgres", "postgresql":
		return DBPostgreSQL
	}
	return DBInvalidDriver
}

// Executor sadly returns the db.SQL obj
func Executor() *sql.DB {
	return db.sql
}

// BeginTransaction returns a transaction without exposing db.SQL
func BeginTransaction() (*sql.Tx, error) {
	return db.sql.BeginTx(context.Background(), nil)
}

// Close closes the connection
func Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.sql.Close()
}
