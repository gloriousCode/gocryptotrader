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
	APIData           *APIData          `json:"api-data,omitempty"`
	DatabaseData      *DatabaseData     `json:"database-data,omitempty"`
	LiveData          *LiveData         `json:"live-data,omitempty"`
	CSVData           *CSVData          `json:"csv-data,omitempty"`
	PortfolioSettings PortfolioSettings `json:"portfolio"`
}

// PortfolioSettings act as a global protector for strategies
// these settings will override ExchangeSettings that go against it
// and assess the bigger picture
type PortfolioSettings struct {
	DiversificationSomething float64 `json:"diversification-something"`

	CanUseLeverage  bool    `json:"can-use-leverage"`
	MaximumLeverage float64 `json:"maximum-leverage"`

	MinimumBuySize float64 `json:"minimum-buy-size"` // will not place an order if under this amount
	MaximumBuySize float64 `json:"maximum-buy-size"` // can only place an order up to this amount
	DefaultBuySize float64 `json:"default-buy-size"`

	MinimumSellSize float64 `json:"minimum-sell-size"` // will not sell an order if under this amount
	MaximumSellSize float64 `json:"maximum-sell-size"` // can only sell an order up to this amount
	DefaultSellSize float64 `json:"default-sell-size"`
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

	MinimumBuySize float64 `json:"minimum-buy-size"` // will not place an order if under this amount
	MaximumBuySize float64 `json:"maximum-buy-size"` // can only place an order up to this amount
	DefaultBuySize float64 `json:"default-buy-size"`

	MinimumSellSize float64 `json:"minimum-sell-size"` // will not sell an order if under this amount
	MaximumSellSize float64 `json:"maximum-sell-size"` // can only sell an order up to this amount
	DefaultSellSize float64 `json:"default-sell-size"`

	CanUseLeverage  bool    `json:"can-use-leverage"`
	MaximumLeverage float64 `json:"maximum-leverage"`

	MakerFee float64 `json:"-"`
	TakerFee float64 `json:"-"`
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
	APIKeyOverride      string        `json:"api-key-override"`
	APISecretOverride   string        `json:"api-secret-override"`
	APIClientIDOverride string        `json:"api-client-id-override"`
	API2FAOverride      string        `json:"api-2fa-override"`
	RealOrders          bool          `json:"fake-orders"`
}
