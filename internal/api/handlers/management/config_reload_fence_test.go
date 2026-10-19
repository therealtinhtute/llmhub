package management

import (
	"context"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
)

// TestSetConfig_StaleSnapshotIgnored verifies that a config reload parsed from an older
// store version is ignored when the handler has already saved a newer version.
func TestSetConfig_StaleSnapshotIgnored(t *testing.T) {
	store := &recordingConfigStore{
		saveCount: 2,
		version:   2,
	}

	initialCfg := &config.Config{
		Port:          8317,
		ConfigVersion: 2,
	}
	h := NewHandlerWithoutConfigFilePath(initialCfg, nil)
	h.SetConfigStore(store)
	h.configVersion = 2

	// An older reload (ConfigVersion 1) must be ignored.
	staleCfg := &config.Config{
		Port:          8000,
		ConfigVersion: 1,
	}
	h.SetConfig(staleCfg)
	if h.cfg.Port != 8317 {
		t.Fatalf("stale SetConfig replaced config: got port=%d want 8317", h.cfg.Port)
	}

	// A same-version reload (ConfigVersion 2) is accepted.
	currentCfg := &config.Config{
		Port:          8318,
		ConfigVersion: 2,
	}
	h.SetConfig(currentCfg)
	if h.cfg.Port != 8318 {
		t.Fatalf("current SetConfig was not applied: got port=%d want 8318", h.cfg.Port)
	}

	// A newer reload (ConfigVersion 3) is accepted and advances handler configVersion.
	store.version = 3
	newerCfg := &config.Config{
		Port:          8319,
		ConfigVersion: 3,
	}
	h.SetConfig(newerCfg)
	if h.cfg.Port != 8319 {
		t.Fatalf("newer SetConfig was not applied: got port=%d want 8319", h.cfg.Port)
	}
	if h.configVersion != 3 {
		t.Fatalf("h.configVersion = %d, want 3", h.configVersion)
	}

	// An unversioned reload (ConfigVersion 0) is treated as unversioned test input and accepted.
	unversionedCfg := &config.Config{
		Port:          9000,
		ConfigVersion: 0,
	}
	h.SetConfig(unversionedCfg)
	if h.cfg.Port != 9000 {
		t.Fatalf("unversioned SetConfig was not applied: got port=%d want 9000", h.cfg.Port)
	}
}

// TestManagementSaveSerializesAgainstReload ensures a management save under h.mu
// updates the version and that an in-flight reload from before the save cannot overwrite it.
func TestManagementSaveSerializesAgainstReload(t *testing.T) {
	store := &recordingConfigStore{
		saveCount: 1,
		version:   1,
	}

	cfg := &config.Config{
		Port:          8317,
		ConfigVersion: 1,
	}
	h := NewHandlerWithoutConfigFilePath(cfg, nil)
	h.SetConfigStore(store)

	// In-flight reload captured before save
	inFlightReload := &config.Config{
		Port:          8317,
		ConfigVersion: 1,
	}

	// Management save increments store version to 2
	h.mu.Lock()
	h.cfg.Port = 8320
	_, _, ok := h.persistLocked(nil)
	h.mu.Unlock()

	if !ok {
		t.Fatal("persistLocked failed")
	}
	if h.configVersion != 2 {
		t.Fatalf("h.configVersion = %d, want 2", h.configVersion)
	}
	if v, _ := store.CurrentVersion(context.Background()); v != 2 {
		t.Fatalf("store version = %d, want 2", v)
	}

	// The in-flight reload (version 1) finishes and calls SetConfig.
	h.SetConfig(inFlightReload)

	// Port 8320 must be preserved, not overwritten by port 8317.
	if h.cfg.Port != 8320 {
		t.Fatalf("in-flight reload overwritten saved config: got port=%d, want 8320", h.cfg.Port)
	}
}
