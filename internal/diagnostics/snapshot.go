package diagnostics

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// ErrInvalidSnapshot identifies malformed, oversized or incompatible diagnostic evidence.
var ErrInvalidSnapshot = errors.New("invalid diagnostic snapshot")

const maxSnapshotBytes = 4 << 20

// Read rejects incomplete and oversized journal records without returning raw contents.
func Read(path string) (*Snapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open diagnostic journal: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSnapshotBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read diagnostic journal: %w", err)
	}
	if len(data) > maxSnapshotBytes {
		return nil, ErrInvalidSnapshot
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var snapshot *Snapshot
	if decoder.Decode(&snapshot) != nil || decoder.Decode(new(any)) != io.EOF || validate(snapshot) != nil {
		return nil, ErrInvalidSnapshot
	}
	return snapshot, nil
}

// Write publishes one complete owner-only record using an atomic same-directory rename.
func Write(path string, snapshot *Snapshot) error {
	if err := validate(snapshot); err != nil {
		return err
	}
	data, err := json.Marshal(snapshot)
	if err != nil || len(data) > maxSnapshotBytes {
		return ErrInvalidSnapshot
	}
	next := path + ".next-" + snapshot.RunID
	file, err := os.OpenFile(next, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("prepare diagnostic journal: %w", err)
	}
	defer os.Remove(next)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write diagnostic journal: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync diagnostic journal: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close diagnostic journal: %w", err)
	}
	if err := os.Rename(next, path); err != nil {
		return fmt.Errorf("publish diagnostic journal: %w", err)
	}
	return nil
}

// validate keeps identity values safe for the journal's atomic-write filenames.
func validate(snapshot *Snapshot) error {
	if snapshot == nil || snapshot.Version != 1 || snapshot.RunID == "" || len(snapshot.RunID) > 128 || strings.Trim(snapshot.RunID, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-") != "" || snapshot.StartedAt.IsZero() || snapshot.CapturedAt.Before(snapshot.StartedAt) || snapshot.Views == nil {
		return ErrInvalidSnapshot
	}
	switch snapshot.State {
	case "running", "stopping", "stopped", "unclean":
		return nil
	default:
		return ErrInvalidSnapshot
	}
}
