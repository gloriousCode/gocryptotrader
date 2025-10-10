package funding

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrFundsNotFound used when funds are requested but the funding is not found in the manager
	ErrFundsNotFound = errors.New("funding not found")
	// ErrAlreadyExists used when a matching item or pair is already in the funding manager
	ErrAlreadyExists = errors.New("funding already exists")
	// ErrUSDTrackingDisabled used when attempting to track USD values when disabled
	ErrUSDTrackingDisabled = errors.New("USD tracking disabled")

	errCannotAllocate             = errors.New("cannot allocate funds")
	errZeroAmountReceived         = errors.New("amount received less than or equal to zero")
	errNegativeAmountReceived     = errors.New("received negative decimal")
	errNotEnoughFunds             = errors.New("not enough funds")
	errCannotTransferToSameFunds  = errors.New("cannot send funds to self")
	errTransferMustBeSameCurrency = errors.New("cannot transfer to different currency")
	errCannotMatchTrackingToItem  = errors.New("cannot match tracking data to funding items")
	errNotFutures                 = errors.New("item linking collateral currencies must be a futures asset")
	errExchangeManagerRequired    = errors.New("exchange manager required")
)

// IFundingManager limits funding usage for portfolio event handling
type IFundingManager interface {
	Reset() error
	IsUsingExchangeLevelFunding() bool
	GetFundingForEvent(common.Event) (IFundingPair, error)
	Transfer(udecimal.Decimal, *Item, *Item, bool) error
	GenerateReport() (*Report, error)
	AddUSDTrackingData(*kline.DataFromKline) error
	CreateSnapshot(time.Time) error
	USDTrackingDisabled() bool
	Liquidate(common.Event) error
	GetAllFunding() ([]BasicItem, error)
	UpdateCollateralForEvent(common.Event, bool) error
	UpdateAllCollateral(isLive, hasUpdateFunding bool) error
	UpdateFundingFromLiveData(hasUpdatedFunding bool) error
	HasFutures() bool
	HasExchangeBeenLiquidated(handler common.Event) bool
	RealisePNL(receivingExchange string, receivingAsset asset.Item, receivingCurrency currency.Code, realisedPNL udecimal.Decimal) error
	SetFunding(string, asset.Item, *account.Balance, bool) error
}

// IFundingTransferer allows for funding amounts to be transferred
// implementation can be swapped for live transferring
type IFundingTransferer interface {
	IsUsingExchangeLevelFunding() bool
	Transfer(udecimal.Decimal, *Item, *Item, bool) error
	GetFundingForEvent(common.Event) (IFundingPair, error)
	HasExchangeBeenLiquidated(handler common.Event) bool
}

// IFundingReader is a simple interface of
// IFundingManager for readonly access at portfolio
// manager
type IFundingReader interface {
	GetFundingForEvent(common.Event) (IFundingPair, error)
	GetAllFunding() []BasicItem
}

// IFundingPair allows conversion into various
// funding interfaces
type IFundingPair interface {
	FundReader() IFundReader
	FundReserver() IFundReserver
	FundReleaser() IFundReleaser
}

// IFundReader can read
// either collateral or pair details
type IFundReader interface {
	GetPairReader() (IPairReader, error)
	GetCollateralReader() (ICollateralReader, error)
}

// IFundReserver limits funding usage for portfolio event handling
type IFundReserver interface {
	IFundReader
	CanPlaceOrder(order.Side) bool
	Reserve(udecimal.Decimal, order.Side) error
}

// IFundReleaser can read
// or release pair or collateral funds
type IFundReleaser interface {
	IFundReader
	PairReleaser() (IPairReleaser, error)
	CollateralReleaser() (ICollateralReleaser, error)
}

// IPairReader is used to limit pair funding functions
// to readonly
type IPairReader interface {
	BaseInitialFunds() udecimal.Decimal
	QuoteInitialFunds() udecimal.Decimal
	BaseAvailable() udecimal.Decimal
	QuoteAvailable() udecimal.Decimal
}

// ICollateralReader is used to read data from
// collateral pairs
type ICollateralReader interface {
	ContractCurrency() currency.Code
	CollateralCurrency() currency.Code
	InitialFunds() udecimal.Decimal
	AvailableFunds() udecimal.Decimal
	CurrentHoldings() udecimal.Decimal
}

// IPairReleaser limits funding usage for exchange event handling
type IPairReleaser interface {
	IPairReader
	IncreaseAvailable(udecimal.Decimal, order.Side) error
	Release(udecimal.Decimal, udecimal.Decimal, order.Side) error
	Liquidate()
}

// ICollateralReleaser limits funding usage for exchange event handling
type ICollateralReleaser interface {
	ICollateralReader
	UpdateContracts(order.Side, udecimal.Decimal) error
	TakeProfit(contracts, positionReturns udecimal.Decimal) error
	ReleaseContracts(udecimal.Decimal) error
	Liquidate()
}

// FundManager is the benevolent holder of all funding levels across all
// currencies used in the backtester
type FundManager struct {
	usingExchangeLevelFunding bool
	disableUSDTracking        bool
	items                     []*Item
	exchangeManager           *engine.ExchangeManager
	verbose                   bool
}

// Item holds funding data per currency item
type Item struct {
	exchange          string
	asset             asset.Item
	currency          currency.Code
	initialFunds      udecimal.Decimal
	available         udecimal.Decimal
	reserved          udecimal.Decimal
	transferFee       udecimal.Decimal
	pairedWith        *Item
	trackingCandles   *kline.DataFromKline
	snapshot          map[int64]ItemSnapshot
	isCollateral      bool
	isLiquidated      bool
	appendedViaAPI    bool
	collateralCandles map[currency.Code]kline.DataFromKline
}

// SpotPair holds two currencies that are associated with each other
type SpotPair struct {
	base  *Item
	quote *Item
}

// CollateralPair consists of a currency pair for a futures contract
// and associates it with an addition collateral pair to take funding from
type CollateralPair struct {
	currentDirection *order.Side
	contract         *Item
	collateral       *Item
}

// BasicItem is a representation of Item
type BasicItem struct {
	Exchange     string
	Asset        asset.Item
	Currency     currency.Code
	InitialFunds udecimal.Decimal
	Available    udecimal.Decimal
	Reserved     udecimal.Decimal
	USDPrice     udecimal.Decimal
}

// Report holds all funding data for result reporting
type Report struct {
	DisableUSDTracking        bool
	UsingExchangeLevelFunding bool
	Items                     []ReportItem
	USDTotalsOverTime         []ItemSnapshot
	InitialFunds              udecimal.Decimal
	FinalFunds                udecimal.Decimal
}

// ReportItem holds reporting fields
type ReportItem struct {
	Exchange             string
	Asset                asset.Item
	Currency             currency.Code
	TransferFee          udecimal.Decimal
	InitialFunds         udecimal.Decimal
	FinalFunds           udecimal.Decimal
	USDInitialFunds      udecimal.Decimal
	USDInitialCostForOne udecimal.Decimal
	USDFinalFunds        udecimal.Decimal
	USDFinalCostForOne   udecimal.Decimal
	Snapshots            []ItemSnapshot
	USDPairCandle        *kline.DataFromKline
	Difference           udecimal.Decimal
	ShowInfinite         bool
	IsCollateral         bool
	AppendedViaAPI       bool
	PairedWith           currency.Code
}

// ItemSnapshot holds USD values to allow for tracking
// across backtesting results
type ItemSnapshot struct {
	Time          time.Time
	Available     udecimal.Decimal
	USDClosePrice udecimal.Decimal
	USDValue      udecimal.Decimal
	Breakdown     []CurrencyContribution
}

// CurrencyContribution helps breakdown how a USD value
// determines its number
type CurrencyContribution struct {
	Currency        currency.Code
	USDContribution udecimal.Decimal
}
