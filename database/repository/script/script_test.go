package script

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/volatiletech/null"
)

var (
	verbose = true
)

func TestMain(m *testing.M) {
	if verbose {
		testhelpers.EnableVerboseTestOutput()
	}

	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
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

func TestScript(t *testing.T) {
	testCases := []struct {
		name   string
		config *database.Config
		runner func()
		closer func() error
		output interface{}
	}{
		{
			"SQLite-Write",
			&database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			writeScript,
			testhelpers.CloseDatabase,
			nil,
		},
		{
			"Postgres-Write",
			testhelpers.PostgresTestDatabase,
			writeScript,
			nil,
			nil,
		},
	}

	for x := range testCases {
		test := testCases[x]
		t.Run(test.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			err := testhelpers.ConnectToDatabase(test.config)
			if err != nil {
				t.Fatal(err)
			}

			if test.runner != nil {
				test.runner()
			}

			if test.closer != nil {
				err = test.closer()
				if err != nil {
					t.Log(err)
				}
			}
		})
	}
}

func writeScript() {
	var wg sync.WaitGroup
	for x := 0; x < 20; x++ {
		wg.Add(1)

		go func(z int) {
			log.Printf("Doing %v", z)
			defer wg.Done()
			test := fmt.Sprintf("test-%v", z)
			var data null.Bytes
			Event(test, test, test, data, test, test, time.Now())
		}(x)
	}
	wg.Wait()
}
