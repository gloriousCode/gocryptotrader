package websocket

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
)

// OrderbookInvalidation tells downstream stores to retire copied books belonging
// to a disconnected native stream. At is the local invalidation boundary; a
// queued snapshot inserted before it cannot restore the old connection's book.
type OrderbookInvalidation struct {
	Pairs []key.PairAsset
	At    time.Time
}
