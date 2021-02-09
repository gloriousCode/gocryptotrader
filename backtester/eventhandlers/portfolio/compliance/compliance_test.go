package compliance

import (
	"strings"
	"testing"
	"time"
)

func TestAddSnapshot(t *testing.T) {
	t.Parallel()
	m := Manager{}
	tt := time.Now()
	err := m.AddSnapshot([]SnapshotOrder{}, tt, false)
	if err != nil {
		t.Error(err)
	}

	err = m.AddSnapshot([]SnapshotOrder{}, tt, false)
	if err != nil && !strings.Contains(err.Error(), "already exists. Use force to overwrite") {
		t.Error(err)
	}

	err = m.AddSnapshot([]SnapshotOrder{}, tt, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSnapshotAtTime(t *testing.T) {
	t.Parallel()
	m := Manager{}
	tt := time.Now()
	err := m.AddSnapshot([]SnapshotOrder{
		{
			ClosePrice: 1337,
		},
	}, tt, false)
	if err != nil {
		t.Error(err)
	}
	var snappySnap Snapshot
	snappySnap, err = m.GetSnapshotAtTime(tt)
	if err != nil {
		t.Error(err)
	}
	if len(snappySnap.Orders) == 0 {
		t.Fatal("expected an order")
	}
	if snappySnap.Orders[0].ClosePrice != 1337 {
		t.Error("expected 1337")
	}
	if !snappySnap.Timestamp.Equal(tt) {
		t.Errorf("expected %v, received %v", tt, snappySnap.Timestamp)
	}

	_, err = m.GetSnapshotAtTime(time.Now().Add(time.Hour))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Error(err)
	}
}

func TestGetLatestSnapshot(t *testing.T) {
	t.Parallel()
	m := Manager{}
	snappySnap := m.GetLatestSnapshot()
	if !snappySnap.Timestamp.IsZero() {
		t.Error("expected blank snapshot")
	}
	tt := time.Now()
	err := m.AddSnapshot([]SnapshotOrder{
		{
			ClosePrice: 1337,
		},
	}, tt, false)
	if err != nil {
		t.Error(err)
	}
	err = m.AddSnapshot([]SnapshotOrder{
		{
			ClosePrice: 1337,
		},
	}, tt.Add(time.Hour), false)
	if err != nil {
		t.Error(err)
	}
	snappySnap = m.GetLatestSnapshot()
	if snappySnap.Timestamp.Equal(tt) {
		t.Errorf("expected %v", tt.Add(time.Hour))
	}
	if !snappySnap.Timestamp.Equal(tt.Add(time.Hour)) {
		t.Errorf("expected %v", tt.Add(time.Hour))
	}
}
