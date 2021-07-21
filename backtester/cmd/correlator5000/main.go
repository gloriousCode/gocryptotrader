package main

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type CorrelationConfig struct {
	Verbose bool       `json:"verbose"`
	Pairs   []Exchange `json:"pairs"`
}

type Exchange struct {
	Name          string                 `json:"name"`
	Enabled       bool                   `json:"enabled"`
	CurrencyPairs *currency.PairsManager `json:"currencyPairs"`
}

func main() {
	// create/load config and pairs to correlate

	// load time scale and interval

	// load data source - live or db

	// set which analytics to use

	// what are the pass results? Or should they just be left to interpretation?

	// output report to file

	// output results to database

}
