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
			db.config = &Config{
				Driver: test.driver,
			}
			ret := GetSQLDialect()
			if ret != test.expectedReturn {
				t.Fatalf("unexpected return: %v", ret)
			}
		})
	}
}
