package compliance

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// AddSnapshot creates a snapshot in time of the orders placed to allow for finer detail tracking
// and to protect against anything modifying order details elsewhere
func (m *Manager) AddSnapshot(orders []SnapshotOrder, t time.Time, force bool) error {
	found := false
	for i := range m.Snapshots {
		if t.Equal(m.Snapshots[i].Timestamp) {
			found = true
			if force {
				m.Snapshots[i].Orders = orders
			} else {
				return errors.New("already exists buttts")
			}
		}
	}
	if !found {
		m.Snapshots = append(m.Snapshots, Snapshot{
			Orders:    orders,
			Timestamp: t,
		})
	}

	return nil
}

// GetSnapshotAtTime returns the snapshot of orders a t time
func (m *Manager) GetSnapshotAtTime(t time.Time) (Snapshot, error) {
	for i := range m.Snapshots {
		if t.Equal(m.Snapshots[i].Timestamp) {
			return m.Snapshots[i], nil
		}
	}
	return Snapshot{}, errors.New("not found")
}

func (m *Manager) SetInterval(i kline.Interval) {
	m.Interval = i
}

// GetLatestSnapshot returns the snapshot of t - 1 interval
func (m *Manager) GetLatestSnapshot() Snapshot {
	if len(m.Snapshots) == 0 {
		return Snapshot{}
	}

	return m.Snapshots[len(m.Snapshots)-1]
}
