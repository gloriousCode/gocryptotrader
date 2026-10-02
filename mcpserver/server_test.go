package mcpserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type diagnosticBackend struct {
	gctrpc.UnimplementedGoCryptoTraderServiceServer
	err       error
	lastQuery *gctrpc.GetDiagnosticLogsRequest
}

func (b *diagnosticBackend) GetInfo(ctx context.Context, _ *gctrpc.GetInfoRequest) (*gctrpc.GetInfoResponse, error) {
	if b.err != nil {
		return nil, b.err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &gctrpc.GetInfoResponse{Uptime: "1h"}, nil
}

func (b *diagnosticBackend) GetExchanges(_ context.Context, r *gctrpc.GetExchangesRequest) (*gctrpc.GetExchangesResponse, error) {
	if r.Enabled {
		return &gctrpc.GetExchangesResponse{Exchanges: "Kraken"}, b.err
	}
	return &gctrpc.GetExchangesResponse{Exchanges: "Kraken,GateIO"}, b.err
}

func (b *diagnosticBackend) GetExchangeInfo(_ context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GetExchangeInfoResponse, error) {
	return &gctrpc.GetExchangeInfoResponse{Name: r.Exchange, Enabled: true, HttpProxy: "https://user:secret@localhost"}, b.err
}

func (b *diagnosticBackend) WebsocketGetInfo(_ context.Context, r *gctrpc.WebsocketGetInfoRequest) (*gctrpc.WebsocketGetInfoResponse, error) {
	return &gctrpc.WebsocketGetInfoResponse{Exchange: r.Exchange, Enabled: true, RunningUrl: "wss://secret", ProxyAddress: "secret"}, b.err
}

func (b *diagnosticBackend) GetDiagnosticLogs(_ context.Context, r *gctrpc.GetDiagnosticLogsRequest) (*gctrpc.GetDiagnosticLogsResponse, error) {
	b.lastQuery = r
	return &gctrpc.GetDiagnosticLogsResponse{Enabled: true, Matched: 7, Groups: []*gctrpc.DiagnosticLogGroup{{Message: "failed", Count: 7}}}, b.err
}

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("invalid backend", func(t *testing.T) {
		t.Parallel()
		_, err := New(nil, time.Second, false)
		require.ErrorIs(t, err, errBackendRequired, "backend must be provided")
	})
	t.Run("invalid timeout", func(t *testing.T) {
		t.Parallel()
		_, err := New(&diagnosticBackend{}, 0, false)
		require.ErrorIs(t, err, errBackendRequired, "timeout must be positive")
	})
	t.Run("protocol discovery and execution", func(t *testing.T) {
		t.Parallel()
		server, err := New(&diagnosticBackend{}, time.Second, false)
		require.NoError(t, err, "server must initialise")
		serverTransport, clientTransport := mcp.NewInMemoryTransports()
		serverSession, err := server.Connect(t.Context(), serverTransport, nil)
		require.NoError(t, err, "server must connect")
		defer func() { assert.NoError(t, serverSession.Close(), "server should close") }()
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		session, err := client.Connect(t.Context(), clientTransport, nil)
		require.NoError(t, err, "client must initialise MCP")
		defer func() { assert.NoError(t, session.Close(), "client should close") }()
		tools, err := session.ListTools(t.Context(), nil)
		require.NoError(t, err, "tools must be discoverable")
		require.Len(t, tools.Tools, 13, "only curated tools must be exposed")
		for _, tool := range tools.Tools {
			assert.True(t, tool.Annotations.ReadOnlyHint, "tools should declare read-only behaviour")
		}
		for _, name := range []string{"get_info", "get_exchanges", "inspect_exchange", "get_recent_logs", "get_log_summary", "get_ticker", "get_orderbook", "get_recent_trades", "get_latest_funding_rate", "get_orders", "get_order", "get_account_balances", "get_futures_positions", "SubmitOrder"} {
			result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: map[string]any{"exchange": "Kraken", "asset": "futures", "base": "BTC", "quote": "USD", "order_id": "order"}})
			if name == "SubmitOrder" {
				require.Error(t, err, "unregistered trading methods must fail")
				continue
			}
			require.NoError(t, err, "read-only tool must execute")
			assert.False(t, result.IsError, "read-only tool should succeed")
			assert.NotEmpty(t, result.Content, "tool should return evidence")
		}
	})
}

func TestCall(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, tool            string
		input                 ToolInput
		backendErr, errorWant error
	}{
		{name: "ticker", tool: "get_ticker", input: ToolInput{Exchange: "Kraken", Asset: "spot", Base: "BTC", Quote: "USD"}},
		{name: "orderbook", tool: "get_orderbook", input: ToolInput{Exchange: "Kraken", Asset: "spot", Base: "BTC", Quote: "USD", Depth: 1}},
		{name: "trades", tool: "get_recent_trades", input: ToolInput{Exchange: "Kraken", Asset: "spot", Base: "BTC", Quote: "USD", Limit: 1}},
		{name: "funding", tool: "get_latest_funding_rate", input: ToolInput{Exchange: "Kraken", Asset: "futures", Base: "BTC", Quote: "USD"}},
		{name: "orders", tool: "get_orders", input: ToolInput{Exchange: "Kraken", Asset: "spot", Base: "BTC", Quote: "USD", Offset: 1, Limit: 1}},
		{name: "order", tool: "get_order", input: ToolInput{Exchange: "Kraken", Asset: "spot", Base: "BTC", Quote: "USD", OrderID: "order", Limit: 1}},
		{name: "balances", tool: "get_account_balances", input: ToolInput{Exchange: "Kraken", Asset: "spot", Offset: 1, Limit: 1}},
		{name: "positions", tool: "get_futures_positions", input: ToolInput{Exchange: "Kraken", Asset: "futures", Base: "BTC", Quote: "USD", UnderlyingBase: "BTC", UnderlyingQuote: "USD"}},
		{name: "spot funding rejected", tool: "get_latest_funding_rate", input: ToolInput{Exchange: "Kraken", Asset: "spot", Base: "BTC", Quote: "USD"}, errorWant: errInvalidInput},
		{name: "missing market", tool: "get_ticker", input: ToolInput{Exchange: "Kraken", Asset: "spot"}, errorWant: errInvalidInput},
		{name: "invalid asset", tool: "get_ticker", input: ToolInput{Exchange: "Kraken", Asset: "invalid", Base: "BTC", Quote: "USD"}, errorWant: errInvalidInput},
		{name: "too much depth", tool: "get_orderbook", input: ToolInput{Exchange: "Kraken", Asset: "spot", Base: "BTC", Quote: "USD", Depth: 101}, errorWant: errInvalidInput},
		{name: "missing order ID", tool: "get_order", input: ToolInput{Exchange: "Kraken", Asset: "spot", Base: "BTC", Quote: "USD"}, errorWant: errInvalidInput},
		{name: "missing account scope", tool: "get_account_balances", input: ToolInput{Asset: "spot"}, errorWant: errInvalidInput},
		{name: "invalid account asset", tool: "get_account_balances", input: ToolInput{Exchange: "Kraken", Asset: "invalid"}, errorWant: errInvalidInput},
		{name: "incomplete underlying", tool: "get_futures_positions", input: ToolInput{Exchange: "Kraken", Asset: "futures", Base: "BTC", Quote: "USD", UnderlyingBase: "BTC"}, errorWant: errInvalidInput},
		{name: "info", tool: "get_info"},
		{name: "exchanges", tool: "get_exchanges", input: ToolInput{Enabled: true}},
		{name: "inspection", tool: "inspect_exchange", input: ToolInput{Exchange: "Kraken"}},
		{name: "missing exchange", tool: "inspect_exchange", errorWant: errInvalidInput},
		{name: "logs", tool: "get_recent_logs", input: ToolInput{Since: "2026-10-01T00:00:00Z", Until: "2026-10-02T00:00:00Z", Severity: "ERROR", Limit: 1}},
		{name: "summary", tool: "get_log_summary"},
		{name: "bad time", tool: "get_recent_logs", input: ToolInput{Since: "yesterday"}, errorWant: errInvalidInput},
		{name: "unsupported method", tool: "Shutdown", errorWant: errInvalidInput},
		{name: "sanitised backend failure", tool: "get_info", backendErr: status.Error(codes.Unavailable, "password=secret"), errorWant: errRPCFailed},
		{name: "inspect backend failure", tool: "inspect_exchange", input: ToolInput{Exchange: "Kraken"}, backendErr: status.Error(codes.Unavailable, "password=secret"), errorWant: errRPCFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			backend := &diagnosticBackend{err: tc.backendErr}
			a := &adapter{backend: backend, timeout: time.Second}
			result, _, err := a.call(t.Context(), tc.tool, &tc.input)
			if tc.errorWant != nil {
				require.ErrorIs(t, err, tc.errorWant, "invalid query must fail")
				assert.NotContains(t, err.Error(), "secret", "errors should not expose backend text")
				return
			}
			require.NoError(t, err, "valid query must succeed")
			require.Len(t, result.Content, 1, "tool must return evidence")
			text, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok, "content must be text")
			assert.NotContains(t, text.Text, "secret", "inspection should omit credential-bearing URLs")
			if tc.tool == "get_orderbook" || tc.tool == "get_recent_trades" || tc.tool == "get_account_balances" || tc.tool == "get_orders" || tc.tool == "get_order" {
				var evidence map[string]json.RawMessage
				require.NoError(t, json.Unmarshal([]byte(text.Text), &evidence), "response must be valid JSON")
				switch tc.tool {
				case "get_orderbook":
					var book gctrpc.OrderbookResponse
					require.NoError(t, protojson.Unmarshal(evidence["data"], &book), "book must decode")
					assert.Len(t, book.Bids, 1, "depth should be bounded")
					assert.JSONEq(t, `true`, string(evidence["truncated"]), "omitted depth should be explicit")
				case "get_recent_trades":
					var trades gctrpc.SavedTradesResponse
					require.NoError(t, protojson.Unmarshal(evidence["data"], &trades), "trades must decode")
					require.Len(t, trades.Trades, 1, "trades must be bounded")
					assert.Equal(t, "newer", trades.Trades[0].TradeId, "most recent trade should be retained")
				case "get_account_balances":
					var balances gctrpc.GetAccountBalancesResponse
					require.NoError(t, protojson.Unmarshal(evidence["data"], &balances), "balances must decode")
					require.Len(t, balances.Accounts, 1, "balance page must retain the correct account")
					require.Len(t, balances.Accounts[0].Currencies, 1, "balance page must respect limit")
					assert.Equal(t, "USDT", balances.Accounts[0].Currencies[0].Currency, "offset should select the next currency")
					assert.JSONEq(t, `true`, string(evidence["has_more"]), "balance coverage should be explicit")
				case "get_orders":
					var orders gctrpc.GetOrdersResponse
					require.NoError(t, protojson.Unmarshal(evidence["data"], &orders), "orders must decode")
					require.Len(t, orders.Orders, 1, "order page must respect limit")
					assert.Equal(t, "second", orders.Orders[0].Id, "offset should select the next order")
				case "get_order":
					var order gctrpc.OrderDetails
					require.NoError(t, protojson.Unmarshal(evidence["data"], &order), "order must decode")
					assert.Len(t, order.Trades, 1, "fill history should be bounded")
				}
			}
			if backend.lastQuery != nil {
				assert.Equal(t, tc.tool == "get_log_summary", backend.lastQuery.Summary, "summary selection should be explicit")
				assert.Equal(t, tc.input.Limit, backend.lastQuery.Limit, "limits should reach backend")
			}
		})
	}
	t.Run("cancelled request", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, _, err := (&adapter{backend: &diagnosticBackend{}, timeout: time.Second}).call(ctx, "get_info", &ToolInput{})
		require.ErrorIs(t, err, errRPCFailed, "cancelled query must fail")
	})
	t.Run("bounded response", func(t *testing.T) {
		t.Parallel()
		backend := &largeBackend{}
		_, _, err := (&adapter{backend: backend, timeout: time.Second}).call(t.Context(), "get_info", &ToolInput{})
		require.ErrorIs(t, err, errResponseTooLarge, "large result must fail")
	})
}

type largeBackend struct{ diagnosticBackend }

func (*largeBackend) GetInfo(context.Context, *gctrpc.GetInfoRequest) (*gctrpc.GetInfoResponse, error) {
	return &gctrpc.GetInfoResponse{Uptime: strings.Repeat("x", 512*1024)}, nil
}

func (b *diagnosticBackend) GetTicker(context.Context, *gctrpc.GetTickerRequest) (*gctrpc.TickerResponse, error) {
	return &gctrpc.TickerResponse{Last: 123, LastUpdated: 1000}, b.err
}

func (b *diagnosticBackend) GetOrderbook(context.Context, *gctrpc.GetOrderbookRequest) (*gctrpc.OrderbookResponse, error) {
	book := &gctrpc.OrderbookResponse{LastUpdated: 1000}
	for range 30 {
		book.Bids = append(book.Bids, &gctrpc.OrderbookItem{Price: 100, Amount: 1})
		book.Asks = append(book.Asks, &gctrpc.OrderbookItem{Price: 101, Amount: 1})
	}
	return book, b.err
}

func (b *diagnosticBackend) GetRecentTrades(context.Context, *gctrpc.GetSavedTradesRequest) (*gctrpc.SavedTradesResponse, error) {
	return &gctrpc.SavedTradesResponse{Trades: []*gctrpc.SavedTrades{{Timestamp: "2026-10-01 01:00:00 UTC", TradeId: "older"}, {Timestamp: "2026-10-02 01:00:00 UTC", TradeId: "newer"}}}, b.err
}

func (b *diagnosticBackend) GetLatestFundingRate(context.Context, *gctrpc.GetLatestFundingRateRequest) (*gctrpc.GetLatestFundingRateResponse, error) {
	return &gctrpc.GetLatestFundingRateResponse{Rate: &gctrpc.FundingData{Exchange: "Kraken"}}, b.err
}

func (b *diagnosticBackend) GetOrders(context.Context, *gctrpc.GetOrdersRequest) (*gctrpc.GetOrdersResponse, error) {
	return &gctrpc.GetOrdersResponse{Orders: []*gctrpc.OrderDetails{{Id: "first", Trades: []*gctrpc.TradeHistory{{Id: "fill"}}}, {Id: "second"}}}, b.err
}

func (b *diagnosticBackend) GetOrder(_ context.Context, r *gctrpc.GetOrderRequest) (*gctrpc.OrderDetails, error) {
	return &gctrpc.OrderDetails{Id: r.OrderId, Trades: []*gctrpc.TradeHistory{{Id: "first"}, {Id: "second"}}}, b.err
}

func (b *diagnosticBackend) GetAccountBalances(_ context.Context, r *gctrpc.GetAccountBalancesRequest) (*gctrpc.GetAccountBalancesResponse, error) {
	return &gctrpc.GetAccountBalancesResponse{Exchange: r.Exchange, Accounts: []*gctrpc.Account{{Id: "spot", Currencies: []*gctrpc.AccountCurrencyInfo{{Currency: "BTC", TotalValue: 2}, {Currency: "USDT", TotalValue: 100}}}, {Id: "margin", Currencies: []*gctrpc.AccountCurrencyInfo{{Currency: "USDT", TotalValue: 10, Borrowed: 5}}}}}, b.err
}

func (b *diagnosticBackend) GetFuturesPositionsSummary(_ context.Context, r *gctrpc.GetFuturesPositionsSummaryRequest) (*gctrpc.GetFuturesPositionsSummaryResponse, error) {
	return &gctrpc.GetFuturesPositionsSummaryResponse{Exchange: r.Exchange, Asset: r.Asset, Pair: r.Pair}, b.err
}
