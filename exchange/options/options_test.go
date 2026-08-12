package options

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// TestNormalisedOptionsTypes validates core fields required by optionsmm data contracts.
func TestNormalisedOptionsTypes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 13, 9, 0, 0, 0, time.UTC)
	pair := currency.NewBTCUSD()

	t.Run("option includes market and greek data", func(t *testing.T) {
		t.Parallel()

		op := Option{
			ExchangeName:      "okx",
			Pair:              pair,
			AssetType:         asset.Options,
			InstrumentID:      "BTC-USD-260626-100000-C",
			LastUpdated:       now,
			ExchangeTimestamp: now.Add(-100 * time.Millisecond),
			ReceivedAt:        now,
			Sequence:          12,
			Delta:             0.21,
			Gamma:             0.03,
			Vega:              0.9,
			Theta:             -0.12,
			MarkIV:            0.45,
			Bid:               120,
			Ask:               122,
			BidSize:           4.2,
			AskSize:           3.7,
			MarkPrice:         121,
			IndexPrice:        95500,
			UnderlyingPrice:   95620,
			LastTradePrice:    121.2,
			LastTradeSize:     0.5,
			LastTradeAt:       now.Add(-200 * time.Millisecond),
			OpenInterest:      110,
			Volume24h:         984,
		}

		require.Equal(t, "okx", op.ExchangeName)
		require.Equal(t, pair, op.Pair)
		require.Equal(t, asset.Options, op.AssetType)
		require.Equal(t, "BTC-USD-260626-100000-C", op.InstrumentID)
		require.Equal(t, now, op.LastUpdated)
		require.Equal(t, now.Add(-100*time.Millisecond), op.ExchangeTimestamp)
		require.Equal(t, now, op.ReceivedAt)
		require.Equal(t, int64(12), op.Sequence)
		require.Equal(t, 0.21, op.Delta)
		require.Equal(t, 0.03, op.Gamma)
		require.Equal(t, 0.9, op.Vega)
		require.Equal(t, -0.12, op.Theta)
		require.Equal(t, 0.45, op.MarkIV)
		require.Equal(t, 120.0, op.Bid)
		require.Equal(t, 122.0, op.Ask)
		require.Equal(t, 4.2, op.BidSize)
		require.Equal(t, 3.7, op.AskSize)
		require.Equal(t, 121.0, op.MarkPrice)
		require.Equal(t, 95500.0, op.IndexPrice)
		require.Equal(t, 95620.0, op.UnderlyingPrice)
		require.Equal(t, 121.2, op.LastTradePrice)
		require.Equal(t, 0.5, op.LastTradeSize)
		require.Equal(t, now.Add(-200*time.Millisecond), op.LastTradeAt)
		require.Equal(t, 110.0, op.OpenInterest)
		require.Equal(t, 984.0, op.Volume24h)
	})

	t.Run("trade captures execution metadata", func(t *testing.T) {
		t.Parallel()

		tr := Trade{
			ExchangeName:      "okx",
			Pair:              pair,
			AssetType:         asset.Options,
			InstrumentID:      "BTC-USD-260626-100000-C",
			TradeID:           "abc-123",
			Side:              order.Buy,
			Price:             121.5,
			Size:              1.25,
			ExchangeTimestamp: now.Add(-50 * time.Millisecond),
			ReceivedAt:        now,
			Sequence:          33,
		}

		require.Equal(t, "abc-123", tr.TradeID)
		require.Equal(t, "okx", tr.ExchangeName)
		require.Equal(t, pair, tr.Pair)
		require.Equal(t, asset.Options, tr.AssetType)
		require.Equal(t, "BTC-USD-260626-100000-C", tr.InstrumentID)
		require.Equal(t, order.Buy, tr.Side)
		require.Equal(t, 121.5, tr.Price)
		require.Equal(t, 1.25, tr.Size)
		require.Equal(t, now.Add(-50*time.Millisecond), tr.ExchangeTimestamp)
		require.Equal(t, now, tr.ReceivedAt)
		require.Equal(t, int64(33), tr.Sequence)
	})

	t.Run("orderbook supports snapshot and sequencing", func(t *testing.T) {
		t.Parallel()

		ob := Orderbook{
			ExchangeName:      "okx",
			Pair:              pair,
			AssetType:         asset.Options,
			InstrumentID:      "BTC-USD-260626-100000-C",
			IsSnapshot:        true,
			Bids:              []OrderbookLevel{{Price: 120, Amount: 2.0}},
			Asks:              []OrderbookLevel{{Price: 122, Amount: 1.8}},
			ExchangeTimestamp: now.Add(-25 * time.Millisecond),
			ReceivedAt:        now,
			Sequence:          101,
			PrevSequence:      100,
		}

		require.True(t, ob.IsSnapshot)
		require.Equal(t, "okx", ob.ExchangeName)
		require.Equal(t, pair, ob.Pair)
		require.Equal(t, asset.Options, ob.AssetType)
		require.Equal(t, "BTC-USD-260626-100000-C", ob.InstrumentID)
		require.Len(t, ob.Bids, 1)
		require.Len(t, ob.Asks, 1)
		require.Equal(t, 120.0, ob.Bids[0].Price)
		require.Equal(t, 2.0, ob.Bids[0].Amount)
		require.Equal(t, 122.0, ob.Asks[0].Price)
		require.Equal(t, 1.8, ob.Asks[0].Amount)
		require.Equal(t, now.Add(-25*time.Millisecond), ob.ExchangeTimestamp)
		require.Equal(t, now, ob.ReceivedAt)
		require.Equal(t, int64(100), ob.PrevSequence)
		require.Equal(t, int64(101), ob.Sequence)
	})
}
