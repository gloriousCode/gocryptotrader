package diagnostics

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		path string
		want error
	}{
		{name: "valid", path: "testdata/valid.json"},
		{name: "invalid version", path: "testdata/invalid.json", want: ErrInvalidSnapshot},
		{name: "missing", path: "testdata/missing.json", want: os.ErrNotExist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			snapshot, err := Read(tc.path)
			if tc.want != nil {
				assert.ErrorIs(t, err, tc.want, "invalid evidence should retain its category")
				assert.Nil(t, snapshot, "invalid evidence should not be returned")
				return
			}
			require.NoError(t, err, "saved valid evidence must be readable")
			assert.Equal(t, "test-run", snapshot.RunID, "run identity should be retained")
		})
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
	t.Run("atomic round trip and owner-only permissions", func(t *testing.T) {
		t.Parallel()
		file, err := os.CreateTemp("testdata", "test-runtime-")
		require.NoError(t, err, "the durable test journal must be allocated")
		path := file.Name()
		require.NoError(t, file.Close(), "test journal handle must close")
		t.Cleanup(func() { _ = os.Remove(path) })
		snapshot := &Snapshot{Version: 1, RunID: "test-roundtrip", StartedAt: time.Now().UTC(), CapturedAt: time.Now().UTC(), State: "running", Views: map[string]any{"runtime_health": map[string]any{"scope": "recorder"}}}
		require.NoError(t, Write(path, snapshot), "a complete snapshot must publish")
		read, err := Read(path)
		require.NoError(t, err, "the published record must be valid")
		assert.Equal(t, snapshot, read, "the journal should preserve all observed fields")
		info, err := os.Stat(path)
		require.NoError(t, err, "published journal must exist")
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "journal should be owner-only")
		_, err = os.Stat(path + ".next-" + snapshot.RunID)
		assert.ErrorIs(t, err, os.ErrNotExist, "the atomic-write file should be consumed")
	})
	t.Run("invalid and oversized evidence", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, Write("testdata/unused", nil), ErrInvalidSnapshot, "nil snapshots should not write")
		now := time.Now().UTC()
		snapshot := &Snapshot{Version: 1, RunID: "oversized", StartedAt: now, CapturedAt: now, State: "running", Views: map[string]any{"value": strings.Repeat("a", maxSnapshotBytes)}}
		assert.ErrorIs(t, Write("testdata/unused", snapshot), ErrInvalidSnapshot, "oversized evidence should not write")
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		edit func(*Snapshot)
	}{
		{name: "version", edit: func(s *Snapshot) { s.Version = 2 }},
		{name: "path in identity", edit: func(s *Snapshot) { s.RunID = "../outside" }},
		{name: "empty identity", edit: func(s *Snapshot) { s.RunID = "" }},
		{name: "unknown state", edit: func(s *Snapshot) { s.State = "unknown" }},
		{name: "missing views", edit: func(s *Snapshot) { s.Views = nil }},
		{name: "backwards capture", edit: func(s *Snapshot) { s.CapturedAt = s.StartedAt.Add(-time.Second) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			now := time.Now().UTC()
			snapshot := &Snapshot{Version: 1, RunID: "test", StartedAt: now, CapturedAt: now, State: "running", Views: map[string]any{}}
			require.NoError(t, validate(snapshot), "the baseline snapshot must be valid")
			tc.edit(snapshot)
			assert.ErrorIs(t, validate(snapshot), ErrInvalidSnapshot, "invalid evidence should be rejected")
		})
	}
}
