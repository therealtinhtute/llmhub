package cliproxy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/therealtinhtute/llmhub/sdk/config"
)

func TestWatcherWrapperSnapshotAuthFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"type":"codex"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	wrapper, err := defaultWatcherFactory(configPath, dir, func(*config.Config) {})
	if err != nil {
		t.Fatalf("failed to create default watcher factory: %v", err)
	}
	t.Cleanup(func() {
		_ = wrapper.Stop()
	})
	wrapper.SetConfig(&config.Config{AuthDir: dir})
	auths, errSnapshot := wrapper.SnapshotAuthFile(authPath)
	if errSnapshot != nil {
		t.Fatalf("SnapshotAuthFile failed: %v", errSnapshot)
	}
	if len(auths) != 1 || auths[0].ID != "auth.json" {
		t.Fatalf("expected 1 auth with ID auth.json, got %+v", auths)
	}

	// Test nil wrapper or unset snapshotAuthFile returns error
	var nilWrapper *WatcherWrapper
	if _, err := nilWrapper.SnapshotAuthFile(authPath); err == nil {
		t.Fatal("expected error on nil WatcherWrapper SnapshotAuthFile")
	}
}
