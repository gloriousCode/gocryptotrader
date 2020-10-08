package database

import (
	"database/sql"
	"errors"
	"path/filepath"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/database/drivers"
)

// Instance holds all information for a database instance
type Instance struct {
	sql         *sql.DB
	dataPath    string
	config      *Config
	isConnected bool
}

// Config holds all database configurable options including enable/disabled & DSN settings
type Config struct {
	Enabled                   bool   `json:"enabled"`
	Verbose                   bool   `json:"verbose"`
	Driver                    string `json:"driver"`
	drivers.ConnectionDetails `json:"connectionDetails"`
}

// Manager is the external manager for database related activities
type Manager struct {
	db *Instance
	mu sync.RWMutex
}

var (
	// dm Global Database Manager
	dm Manager
	// MigrationDir which folder to look in for current migrations
	MigrationDir = filepath.Join("..", "..", "database", "migrations")
	// ErrNoDatabaseProvided error to display when no database is provided
	ErrNoDatabaseProvided = errors.New("no database provided")
	// ErrDatabaseSupportDisabled error to display when no database is provided
	ErrDatabaseSupportDisabled = errors.New("database support is disabled")
)

const (
	// DBSQLite const string for sqlite across code base
	DBSQLite = "sqlite"
	// DBSQLite3 const string for sqlite3 across code base
	DBSQLite3 = "sqlite3"
	// DBPostgreSQL const string for PostgreSQL across code base
	DBPostgreSQL = "postgres"
	// DBInvalidDriver const string for invalid driver
	DBInvalidDriver = "invalid driver"
	// DefaultSQLiteDatabase is the default sqlite3 database name to use
	DefaultSQLiteDatabase = "gocryptotrader.db"
)
