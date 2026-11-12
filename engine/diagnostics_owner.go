package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var errDiagnosticOwned = errors.New("diagnostic directory is owned by another process")

// acquireDiagnosticOwner only reclaims an owner known to have exited.
func acquireDiagnosticOwner(directory, runID string) (*diagnosticOwner, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create diagnostic directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open diagnostic directory: %w", err)
	}
	defer root.Close()
	path := filepath.Join(directory, "owner.lock")
	token := strconv.Itoa(os.Getpid()) + " " + runID
	var reclaim *os.File
	var reclaimPath string
	defer func() {
		if reclaim != nil {
			_ = reclaim.Close()
			_ = root.Remove(reclaimPath)
		}
	}()
	for range 2 {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_, writeErr := file.WriteString(token)
			closeErr := file.Close()
			if writeErr != nil || closeErr != nil {
				_ = os.Remove(path)
				return nil, fmt.Errorf("record diagnostic owner: %w", errors.Join(writeErr, closeErr))
			}
			return &diagnosticOwner{Path: path, Token: token}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("claim diagnostic journal: %w", err)
		}
		if reclaim != nil {
			return nil, errDiagnosticOwned
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) > 256 {
			return nil, errDiagnosticOwned
		}
		fields := strings.Fields(string(data))
		if len(fields) != 2 || fields[1] == "" || len(fields[1]) > 128 || strings.Trim(fields[1], "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-") != "" {
			return nil, errDiagnosticOwned
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 {
			return nil, errDiagnosticOwned
		}
		if !diagnosticProcessExited(pid) {
			return nil, errDiagnosticOwned
		}
		oldTokenHash := sha256.Sum256(data)
		reclaimPath = ".reclaim-" + hex.EncodeToString(oldTokenHash[:])
		reclaim, err = root.OpenFile(reclaimPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, errDiagnosticOwned
		}
		current, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(current, data) {
			return nil, errDiagnosticOwned
		}
		if err := os.Remove(path); err != nil {
			return nil, errDiagnosticOwned
		}
	}
	return nil, errDiagnosticOwned
}

// release refuses to remove a newer owner's claim.
func (owner *diagnosticOwner) release() {
	if owner == nil {
		return
	}
	if data, err := os.ReadFile(owner.Path); err == nil && string(data) == owner.Token {
		_ = os.Remove(owner.Path)
	}
}
