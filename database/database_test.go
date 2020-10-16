package database

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/database/drivers"
)

func TestGetSQLDialect(t *testing.T) {
	testCases := []struct {
		driver         string
		expectedReturn string
	}{
		{
			"postgresql",
			DBPostgreSQL,
		},
		{
			"sqlite",
			DBSQLite3,
		},
		{
			"sqlite3",
			DBSQLite3,
		},
		{
			"invalid",
			DBInvalidDriver,
		},
	}
	for x := range testCases {
		test := testCases[x]

		t.Run(test.driver, func(t *testing.T) {
			dbManager, err := GetDBManager()
			if err != nil {
				t.Error(err)
			}

			err = dbManager.SetDBConfig(&Config{
				ConnectionDetails: drivers.ConnectionDetails{
					Database: test.driver,
				},
				Driver: test.driver,
			})
			if err != nil {
				t.Fatal(err)
			}
			ret := dbManager.GetSQLDialect()
			if ret != test.expectedReturn {
				t.Fatalf("unexpected return: %v", ret)
			}
		})
	}
}
