package testhelpers

import (
	"os"
	"path/filepath"
	"reflect"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
)

var (
	// TempDir temp folder for sqlite database
	TempDir string
	// PostgresTestDatabase postgresql database config details
	PostgresTestDatabase *database.Config
	// MigrationDir default folder for migration's
	MigrationDir = filepath.Join("..", "..", "migrations")
)

// GetConnectionDetails returns connection details for CI or test db instances
func GetConnectionDetails() *database.Config {
	_, exists := os.LookupEnv("TRAVIS")
	if exists {
		return &database.Config{
			Enabled: true,
			Driver:  "postgres",
			ConnectionDetails: drivers.ConnectionDetails{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "",
				Database: "gct_dev_ci",
				SSLMode:  "",
			},
		}
	}

	_, exists = os.LookupEnv("APPVEYOR")
	if exists {
		return &database.Config{
			Enabled: true,
			Driver:  "postgres",
			ConnectionDetails: drivers.ConnectionDetails{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "Password12!",
				Database: "gct_dev_ci",
				SSLMode:  "",
			},
		}
	}

	return &database.Config{
		Enabled:           true,
		Driver:            "postgres",
		ConnectionDetails: drivers.ConnectionDetails{
			// Host:     "",
			// Port:     5432,
			// Username: "",
			// Password: "",
			// Database: "",
			// SSLMode:  "",
		},
	}
}

// ConnectToDatabase opens connection to database and returns pointer to instance of database.DB
func ConnectToDatabase(conn *database.Config) (dbConn *database.Instance, err error) {
	database.SetDBConfig(conn)
	if conn.Driver == database.DBPostgreSQL {
		err = database.Connect(conn.Driver)
		if err != nil {
			return nil, err
		}
	} else if conn.Driver == database.DBSQLite3 || conn.Driver == database.DBSQLite {
		database.SetDBPath(TempDir)
		err = database.Connect(conn.Driver)
		if err != nil {
			return nil, err
		}
	}

	err = database.RunGooseCommand("up", MigrationDir, "")
	if err != nil {
		return nil, err
	}

	return
}

// CloseDatabase closes database connection
func CloseDatabase(conn *database.Instance) (err error) {
	err = database.Close()
	if err != nil {
		return err
	}
	return nil
}

// CheckValidConfig checks if database connection details are empty
func CheckValidConfig(config *drivers.ConnectionDetails) bool {
	return !reflect.DeepEqual(drivers.ConnectionDetails{}, *config)
}

// EnableVerboseTestOutput enables debug output for SQL queries
func EnableVerboseTestOutput() {
	c := log.GenDefaultSettings()
	log.SetConfig(&c)
	log.SetupGlobalLogger()

	DBLogger := database.Logger{}
	boil.DebugMode = true
	boil.DebugWriter = DBLogger
}
