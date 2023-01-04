package config

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
)

func TestLoadBacktesterConfig(t *testing.T) {
	t.Parallel()
	cfg, err := GenerateDefaultConfig()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	testConfig, err := json.Marshal(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "test.config")
	err = file.Write(f, testConfig)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	_, err = ReadBacktesterConfigFromPath(f)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = ReadBacktesterConfigFromPath("test")
	if !errors.Is(err, common.ErrFileNotFound) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrFileNotFound)
	}
}

func TestGenerateDefaultConfig(t *testing.T) {
	t.Parallel()
	cfg, err := GenerateDefaultConfig()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !cfg.PrintLogo {
		t.Errorf("received '%v' expected '%v'", cfg.PrintLogo, true)
	}
}
