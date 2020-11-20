package config

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
)

// Config defines what is in an individual strategy config
type Config struct {
	StrategyToLoad   string           `json:"strategy"`
	ExchangeSettings ExchangeSettings `json:"exchange-settings"`
	// Unsupported so far, but will move to having multiple currencies
	ExchangeSettingsButWithPassionAndLust []ExchangeSettings `json:"lustful-exchange-settings,omitempty"`
	// data source definitions:
	APIData           *APIData               `json:"api-data,omitempty"`
	DatabaseData      *DatabaseData          `json:"database-data,omitempty"`
	LiveData          *LiveData              `json:"live-data,omitempty"`
	CSVData           *CSVData               `json:"csv-data,omitempty"`
	StrategySettings  map[string]interface{} `json:"strategy-settings"`
	PortfolioSettings PortfolioSettings      `json:"portfolio"`
}

// PortfolioSettings act as a global protector for strategies
// these settings will override ExchangeSettings that go against it
// and assess the bigger picture
type PortfolioSettings struct {
	DiversificationSomething float64  `json:"diversification-something"`
	Leverage                 Leverage `json:"leverage"`
	BuySide                  MinMax   `json:"buy-side"`
	SellSide                 MinMax   `json:"sell-side"`
}

type Leverage struct {
	CanUseLeverage  bool    `json:"can-use-leverage"`
	MaximumLeverage float64 `json:"maximum-leverage"`
}

type MinMax struct {
	MinimumSize  float64 `json:"minimum-size"` // will not place an order if under this amount
	MaximumSize  float64 `json:"maximum-size"` // can only place an order up to this amount
	MaximumTotal float64 `json:"maximum-total"`
}

// ExchangeSettings stores pair based variables
// It contains rules about the specific currency pair
// you wish to trade with
type ExchangeSettings struct {
	Name  string `json:"exchange-name"`
	Asset string `json:"asset"`
	Base  string `json:"base"`
	Quote string `json:"quote"`

	InitialFunds float64 `json:"initial-funds"`

	Leverage Leverage `json:"leverage"`
	BuySide  MinMax   `json:"buy-side"`
	SellSide MinMax   `json:"sell-side"`

	MakerFee float64 `json:"maker-fee-override"`
	TakerFee float64 `json:"taker-fee-override"`
}

// APIData defines all fields to configure API based data
type APIData struct {
	DataType  string        `json:"data-type"`
	Interval  time.Duration `json:"interval"`
	StartDate time.Time     `json:"start-date"`
	EndDate   time.Time     `json:"end-date"`
}

// CSVData defines all fields to configure CSV based data
type CSVData struct {
	DataType string        `json:"data-type"`
	Interval time.Duration `json:"interval"`
	FullPath string        `json:"full-path"`
}

// DatabaseData defines all fields to configure database based data
type DatabaseData struct {
	DataType       string           `json:"data-type"`
	Interval       time.Duration    `json:"interval"`
	StartDate      time.Time        `json:"start-date"`
	EndDate        time.Time        `json:"end-date"`
	ConfigOverride *database.Config `json:"config-override"`
}

// LiveData defines all fields to configure live data
type LiveData struct {
	Interval            time.Duration `json:"interval"`
	DataType            string        `json:"data-type"`
	APIKeyOverride      string        `json:"api-key-override"`
	APISecretOverride   string        `json:"api-secret-override"`
	APIClientIDOverride string        `json:"api-client-id-override"`
	API2FAOverride      string        `json:"api-2fa-override"`
	RealOrders          bool          `json:"fake-orders"`
}
