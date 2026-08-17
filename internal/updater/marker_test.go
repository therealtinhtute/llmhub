package updater

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// markerFixture returns an ApplyConfig whose target is a v1.0.0 helper
// binary, plus the fixture's data dir and marker dir.
func markerFixture(t *testing.T, stagedBin []byte, stagedVer string) (ApplyConfig, string, []byte) {
	t.Helper()
	oldBin := readBytes(t, buildHelper(t, "v1.0.0"))
	dataDir := t.TempDir()
	cfg := ApplyConfig{
		DataDir:         dataDir,
		MarkerDir:       t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v1.0.0",
		Client:          applyTestClient(t, "v9.9.9", stagedBin),
	}
	if err := os.WriteFile(cfg.Target, oldBin, 0o755); err != nil {
		t.Fatal(err)
	}
	stageApplyFixture(t, dataDir, stagedBin, stagedVer)
	return cfg, dataDir, oldBin
}

func runApplyAsRoot(t *testing.T, cfg ApplyConfig) int {
	t.Helper()
	oldEUID := processEUID
	processEUID = func() int { return 0 }
	defer func() { processEUID = oldEUID }()
	var stdout, stderr bytes.Buffer
	return ApplyEntry(nil, &stdout, &stderr, cfg)
}

func TestBootMarkerWrittenOnSwap(t *testing.T) {
	requireSupportedPlatform(t)
	newBin := readBytes(t, buildHelper(t, "v9.9.9"))
	cfg, _, _ := markerFixture(t, newBin, "v9.9.9")
	// A stale token from a prior healthy start must not survive a new cycle.
	if err := os.WriteFile(bootedTokenPath(cfg), []byte("v0.9.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Simulate a healthy previous start so the pre-swap marker check is a
	// fresh cycle.
	if code := runApplyAsRoot(t, cfg); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}

	marker := markerPath(cfg)
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("boot marker not written: %v", err)
	}
	if strings.TrimSpace(string(data)) != "v1.0.0" {
		t.Fatalf("marker records %q, want outgoing v1.0.0", strings.TrimSpace(string(data)))
	}
	if fileExists(bootedTokenPath(cfg)) {
		t.Fatal("booted token must be cleared when a new swap cycle starts")
	}
}

func TestMarkerCleared(t *testing.T) {
	requireSupportedPlatform(t)
	newBin := readBytes(t, buildHelper(t, "v9.9.9"))
	cfg, _, _ := markerFixture(t, newBin, "v9.9.9")

	// A previous swap whose boot completed: marker + booted token present.
	if err := writeMarker(markerPath(cfg), "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bootedTokenPath(cfg), []byte("v1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := runApplyAsRoot(t, cfg); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	// A healthy previous boot unblocks the apply: the old marker+token pair
	// is consumed and a fresh cycle starts (marker re-written for the swap
	// about to happen, outgoing = installed).
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, newBin) {
		t.Fatal("healthy marker cycle must still apply the staged candidate")
	}
	if data, err := os.ReadFile(markerPath(cfg)); err != nil {
		t.Fatalf("fresh marker missing after healthy apply: %v", err)
	} else if strings.TrimSpace(string(data)) != "v1.0.0" {
		t.Fatalf("marker records %q, want outgoing v1.0.0", strings.TrimSpace(string(data)))
	}
	if fileExists(bootedTokenPath(cfg)) {
		t.Fatal("booted token must be cleared when a new swap cycle starts")
	}
}

func TestMarkerRevert(t *testing.T) {
	requireSupportedPlatform(t)
	newBin := readBytes(t, buildHelper(t, "v9.9.9"))
	cfg, _, oldBin := markerFixture(t, newBin, "v9.9.9")

	// The previous swap crashed: marker present, no booted token, and the
	// pre-swap backup exists.
	if err := writeMarker(markerPath(cfg), "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(cfg.Target+".previous", cfg.Target, 0o755); err != nil {
		t.Fatal(err)
	}

	if code := runApplyAsRoot(t, cfg); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target must be reverted to <target>.previous")
	}
	if fileExists(markerPath(cfg)) {
		t.Fatal("marker must be cleared after a revert")
	}
}

func TestMarkerRevertMissingBackup(t *testing.T) {
	requireSupportedPlatform(t)
	newBin := readBytes(t, buildHelper(t, "v9.9.9"))
	cfg, _, oldBin := markerFixture(t, newBin, "v9.9.9")

	if err := writeMarker(markerPath(cfg), "v1.0.0"); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	oldEUID := processEUID
	processEUID = func() int { return 0 }
	defer func() { processEUID = oldEUID }()
	if code := ApplyEntry(nil, &stdout, &stderr, cfg); code != 0 {
		t.Fatalf("missing backup must exit 0, got %d", code)
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target must be left untouched when no backup exists")
	}
	if !fileExists(markerPath(cfg)) {
		t.Fatal("marker must survive a failed revert so new applies stay blocked")
	}
}

func TestMarkerUnwritable(t *testing.T) {
	requireSupportedPlatform(t)
	cfg, dataDir, _ := markerFixture(t, readBytes(t, buildHelper(t, "v9.9.9")), "v9.9.9")

	// The marker directory must be outside ${DATA_DIR}, while the healthy
	// boot token the server writes lives inside ${DATA_DIR}/update.
	if rel, err := filepath.Rel(dataDir, markerPath(cfg)); err == nil && !strings.HasPrefix(rel, "..") {
		t.Fatalf("marker %q lives inside DATA_DIR %q", markerPath(cfg), dataDir)
	}
	if rel, err := filepath.Rel(filepath.Join(dataDir, updateDirName), bootedTokenPath(cfg)); err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("booted token %q must live inside %s/update", bootedTokenPath(cfg), dataDir)
	}

	if code := runApplyAsRoot(t, cfg); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	info, err := os.Stat(markerPath(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("marker mode %v must be owner-only", info.Mode())
	}
}
