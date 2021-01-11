package database

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var verbose = false

func TestMain(m *testing.M) {
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = ioutil.TempDir("", "gct-temp")
	if err != nil {
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	t := m.Run()

	err = os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func TestLoadData(t *testing.T) {
	tt1 := time.Now().Add(-time.Hour)
	tt2 := time.Now()
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var err error
	engine.Bot, err = engine.NewFromSettings(&engine.Settings{}, nil)
	if err != nil {
		t.Error(err)
	}
	dbConfg := database.Config{
		Enabled: true,
		Verbose: false,
		Driver:  "sqlite",
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test",
		},
	}
	engine.Bot.Config = &config.Config{
		Database: dbConfg,
	}

	err = engine.Bot.Config.CheckConfig()
	if err != nil && verbose {
		// this loads the database config to the global database
		// the errors are unrelated and likely prone to change for reasons that
		// this test does not need to care about

		// so we only log the error if verbose
		t.Log(err)
	}
	// oldMigrate = database.MigrationDir
	database.MigrationDir = filepath.Join("..", "..", "..", "..", "migrations")
	testhelpers.MigrationDir = filepath.Join("..", "..", "..", "..", "migrations")
	_, err = testhelpers.ConnectToDatabase(&dbConfg)
	if err != nil {
		t.Error(err)
	}

	err = engine.Bot.DatabaseManager.Start(engine.Bot)
	if err != nil {
		t.Error(err)
	}

	_, err = LoadData(tt1, tt2, gctkline.FifteenMin.Duration(), exch, common.CandleStr, p, a)
	if err != nil {
		t.Error(err)
	}
}
