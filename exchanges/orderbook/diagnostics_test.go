package orderbook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepthRetrieveLimited(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		limit int
		count int
	}{
		{name: "negative", limit: -1},
		{name: "zero", limit: 0},
		{name: "one", limit: 1, count: 1},
		{name: "exact", limit: 2, count: 2},
		{name: "beyond depth", limit: 3, count: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			depth := NewDepth([16]byte{})
			now := time.Now().UTC()
			require.NoError(t, depth.LoadSnapshot(&Book{Bids: Levels{{Price: 2, Amount: 1}, {Price: 1, Amount: 1}}, Asks: Levels{{Price: 3, Amount: 1}, {Price: 4, Amount: 1}}, LastUpdated: now}), "valid snapshot must load")
			book, err := depth.RetrieveLimited(tc.limit)
			if tc.limit <= 0 {
				assert.ErrorIs(t, err, errInvalidBookDepth, "unbounded requests should be rejected")
				return
			}
			require.NoError(t, err, "bounded snapshot must succeed")
			assert.Len(t, book.Bids, tc.count, "bid depth should be bounded")
			assert.Len(t, book.Asks, tc.count, "ask depth should be bounded")
			assert.Equal(t, now, book.LastUpdated, "source timestamp should be retained")
			assert.Equal(t, float64(2), book.Bids[0].Price, "best bid should be first")
			assert.Equal(t, float64(3), book.Asks[0].Price, "best ask should be first")
			book.Bids[0].Price = 99
			again, err := depth.RetrieveLimited(1)
			require.NoError(t, err, "a second observation must succeed")
			assert.Equal(t, float64(2), again.Bids[0].Price, "returned levels should not mutate the shared book")
		})
	}
	t.Run("invalid book", func(t *testing.T) {
		t.Parallel()
		depth := NewDepth([16]byte{})
		depth.validationError = ErrOrderbookInvalid
		_, err := depth.RetrieveLimited(1)
		assert.ErrorIs(t, err, ErrOrderbookInvalid, "invalid source evidence should remain invalid")
	})
}
