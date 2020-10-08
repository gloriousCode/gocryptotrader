package database

import "testing"

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

			dbManager.SetDBConfig(&Config{
				Driver: test.driver,
			})
			ret := dbManager.GetSQLDialect()
			if ret != test.expectedReturn {
				t.Fatalf("unexpected return: %v", ret)
			}
		})
	}
}
