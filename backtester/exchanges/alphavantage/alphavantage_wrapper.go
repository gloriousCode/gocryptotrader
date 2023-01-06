package alphavantage

import (
	"context"
	"fmt"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"sync"
	"time"
)

type LemonExchange struct {
	exchange.Base
}

// GetDefaultConfig returns a default exchange config
func (l *LemonExchange) GetDefaultConfig() (*config.Exchange, error) {
	l.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = l.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = l.BaseCurrencies

	err := l.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if l.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = l.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (l *LemonExchange) SetDefaults() {
	l.Name = "ZB"
	l.Enabled = true
	l.Verbose = true
	l.API.CredentialsValidator.RequiresKey = true
	l.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	err := l.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	l.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
				MultiChainDeposits:  true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				CancelOrder:            true,
				SubmitOrder:            true,
				MessageCorrelation:     true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.ThreeMin.Word():   true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.SixHour.Word():    true,
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
					kline.ThreeDay.Word():   true,
					kline.OneWeek.Word():    true,
				},
				ResultLimit: 1000,
			},
		},
	}

	l.Requester, err = request.New(l.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(&request.BasicLimit{}))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	l.API.Endpoints = l.NewEndpoints()
	err = l.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: baseURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	l.Websocket = stream.New()
	l.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	l.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup sets user configuration
func (l *LemonExchange) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		l.SetEnabled(false)
		return nil
	}
	err = l.SetupDefaults(exch)
	if err != nil {
		return err
	}
	return nil
}

// Start starts the ZB go routine
func (l *LemonExchange) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the ZB wrapper
func (l *LemonExchange) Run() {
	if l.Verbose {
		l.PrintEnabledPairs()
	}

	if !l.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := l.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", l.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (l *LemonExchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	return nil, nil
	//markets, err := l.GetMarkets(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//
	//pairs := make([]currency.Pair, len(markets))
	//var target int
	//for key := range markets {
	//	var pair currency.Pair
	//	pair, err = currency.NewPairFromString(key)
	//	if err != nil {
	//		return nil, err
	//	}
	//	pairs[target] = pair
	//	target++
	//}
	//return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (l *LemonExchange) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := l.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	return l.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (l *LemonExchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	return common.ErrFunctionNotSupported

}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LemonExchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return nil, common.ErrFunctionNotSupported

}

// FetchTicker returns the ticker for a currency pair
func (l *LemonExchange) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	return nil, common.ErrFunctionNotSupported

}

// FetchOrderbook returns orderbook base on the currency pair
func (l *LemonExchange) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	return nil, common.ErrFunctionNotSupported

}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LemonExchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	return nil, common.ErrFunctionNotSupported

}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// ZB exchange
func (l *LemonExchange) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	return account.Holdings{}, common.ErrFunctionNotSupported
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (l *LemonExchange) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	return account.Holdings{}, common.ErrFunctionNotSupported
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LemonExchange) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (l *LemonExchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (l *LemonExchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return nil, nil
	//var err error
	//p, err = l.FormatExchangeCurrency(p, assetType)
	//if err != nil {
	//	return nil, err
	//}
	//var tradeData TradeHistory
	//tradeData, err = l.GetTrades(ctx, p.String())
	//if err != nil {
	//	return nil, err
	//}
	//resp := make([]trade.Data, len(tradeData))
	//for i := range tradeData {
	//	var side order.Side
	//	side, err = order.StringToOrderSide(tradeData[i].Type)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	resp[i] = trade.Data{
	//		Exchange:     l.Name,
	//		TID:          strconv.FormatInt(tradeData[i].Tid, 10),
	//		CurrencyPair: p,
	//		AssetType:    assetType,
	//		Side:         side,
	//		Price:        tradeData[i].Price,
	//		Amount:       tradeData[i].Amount,
	//		Timestamp:    time.Unix(tradeData[i].Date, 0),
	//	}
	//}
	//
	//err = l.AddTradesToBuffer(resp...)
	//if err != nil {
	//	return nil, err
	//}
	//
	//sort.Sort(trade.ByDate(resp))
	//return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (l *LemonExchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (l *LemonExchange) SubmitOrder(ctx context.Context, o *order.Submit) (*order.SubmitResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LemonExchange) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *LemonExchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	return common.ErrFunctionNotSupported

}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (l *LemonExchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *LemonExchange) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrFunctionNotSupported

}

// GetOrderInfo returns order information based on order ID
func (l *LemonExchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrFunctionNotSupported
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *LemonExchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported

}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LemonExchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LemonExchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LemonExchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (l *LemonExchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (l *LemonExchange) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	return nil, common.ErrFunctionNotSupported

}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (l *LemonExchange) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	return nil, common.ErrFunctionNotSupported

}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (l *LemonExchange) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	return common.ErrFunctionNotSupported
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (l *LemonExchange) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, nil
	//ret, err := l.validateCandlesRequest(p, a, start, end, interval)
	//if err != nil {
	//	return kline.Item{}, err
	//}
	//
	//p, err = l.FormatExchangeCurrency(p, a)
	//if err != nil {
	//	return kline.Item{}, err
	//}
	//
	//klineParams := KlinesRequestParams{
	//	Type:   l.FormatExchangeKlineInterval(interval),
	//	Symbol: p.String(),
	//	Since:  start.UnixMilli(),
	//	Size:   int64(l.Features.Enabled.Kline.ResultLimit),
	//}
	//var candles KLineResponse
	//candles, err = l.GetSpotKline(ctx, klineParams)
	//if err != nil {
	//	return kline.Item{}, err
	//}
	//
	//for x := range candles.Data {
	//	if candles.Data[x].KlineTime.Before(start) || candles.Data[x].KlineTime.After(end) {
	//		continue
	//	}
	//	ret.Candles = append(ret.Candles, kline.Candle{
	//		Time:   candles.Data[x].KlineTime,
	//		Open:   candles.Data[x].Open,
	//		High:   candles.Data[x].High,
	//		Low:    candles.Data[x].Low,
	//		Close:  candles.Data[x].Close,
	//		Volume: candles.Data[x].Volume,
	//	})
	//}
	//
	//ret.SortCandlesByTimestamp(false)
	//return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (l *LemonExchange) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, nil
	//ret, err := l.validateCandlesRequest(p, a, start, end, interval)
	//if err != nil {
	//	return kline.Item{}, err
	//}
	//
	//p, err = l.FormatExchangeCurrency(p, a)
	//if err != nil {
	//	return kline.Item{}, err
	//}
	//
	//startTime := start
	//lines:
	//for {
	//	klineParams := KlinesRequestParams{
	//		Type:   l.FormatExchangeKlineInterval(interval),
	//		Symbol: p.String(),
	//		Since:  startTime.UnixMilli(),
	//		Size:   int64(l.Features.Enabled.Kline.ResultLimit),
	//	}
	//
	//	candles, err := l.GetSpotKline(ctx, klineParams)
	//	if err != nil {
	//		return kline.Item{}, err
	//	}
	//
	//	for x := range candles.Data {
	//		if candles.Data[x].KlineTime.Before(start) || candles.Data[x].KlineTime.After(end) {
	//			continue
	//		}
	//		if startTime.Equal(candles.Data[x].KlineTime) {
	//			// no new data has been sent
	//			break allKlines
	//		}
	//		ret.Candles = append(ret.Candles, kline.Candle{
	//			Time:   candles.Data[x].KlineTime,
	//			Open:   candles.Data[x].Open,
	//			High:   candles.Data[x].High,
	//			Low:    candles.Data[x].Low,
	//			Close:  candles.Data[x].Close,
	//			Volume: candles.Data[x].Volume,
	//		})
	//		if x == len(candles.Data)-1 {
	//			startTime = candles.Data[x].KlineTime
	//		}
	//	}
	//	if len(candles.Data) != int(l.Features.Enabled.Kline.ResultLimit) {
	//		break allKlines
	//	}
	//}
	//
	//ret.SortCandlesByTimestamp(false)
	//return ret, nil
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (l *LemonExchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	return nil, common.ErrFunctionNotSupported
}
