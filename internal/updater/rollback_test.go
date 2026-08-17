package updater

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rollbackFixture(t *testing.T) (ApplyConfig, []byte) {
	t.Helper()
	newBin := readBytes(t, buildHelper(t, "v9.9.9"))
	oldBin := readBytes(t, buildHelper(t, "v1.0.0"))
	cfg := ApplyConfig{
		DataDir:         t.TempDir(),
		MarkerDir:       t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v9.9.9",
	}
	if err := os.WriteFile(cfg.Target, newBin, 0o755); err != nil {
		t.Fatal(err)
	}
	// Simulate the state right after a completed swap: new binary installed,
	// old one retained as <target>.previous, boot marker written.
	if err := os.WriteFile(cfg.Target+".previous", oldBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeMarker(markerPath(cfg), "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	return cfg, oldBin
}

func runRollbackAsRoot(t *testing.T, cfg ApplyConfig) (int, string, string) {
	t.Helper()
	oldEUID := processEUID
	processEUID = func() int { return 0 }
	defer func() { processEUID = oldEUID }()
	var stdout, stderr bytes.Buffer
	code := RollbackEntry(nil, &stdout, &stderr, cfg)
	return code, stdout.String(), stderr.String()
}

func TestRollback(t *testing.T) {
	requireSupportedPlatform(t)
	cfg, oldBin := rollbackFixture(t)

	code, stdout, stderr := runRollbackAsRoot(t, cfg)
	if code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, stderr)
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target was not restored from <target>.previous")
	}
	if got := readBytes(t, cfg.Target+".previous"); !bytes.Equal(got, oldBin) {
		t.Fatal("rollback must never overwrite <target>.previous")
	}
	if fileExists(markerPath(cfg)) {
		t.Fatal("rollback must clear the boot marker so the next apply starts fresh")
	}
	if !strings.Contains(stdout, "rolled back") {
		t.Fatalf("stdout = %q, want rollback confirmation", stdout)
	}
}

func TestRollbackMissingBackup(t *testing.T) {
	requireSupportedPlatform(t)
	newBin := readBytes(t, buildHelper(t, "v9.9.9"))
	cfg := ApplyConfig{
		DataDir:         t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v9.9.9",
	}
	if err := os.WriteFile(cfg.Target, newBin, 0o755); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runRollbackAsRoot(t, cfg)
	if code != 0 {
		t.Fatalf("missing backup must exit 0, got %d", code)
	}
	if !strings.Contains(stderr, "no previous binary") {
		t.Fatalf("stderr = %q, want no-previous-binary message", stderr)
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, newBin) {
		t.Fatal("target changed with no backup present")
	}
}

func TestRollbackPaths(t *testing.T) {
	requireSupportedPlatform(t)
	cfg, oldBin := rollbackFixture(t)
	// A decoy binary in an unrelated directory must never be touched: rollback
	// only ever restores the resolved target's own <target>.previous.
	decoyDir := t.TempDir()
	decoy := filepath.Join(decoyDir, "llmhub")
	if err := os.WriteFile(decoy, []byte("decoy"), 0o755); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runRollbackAsRoot(t, cfg)
	if code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, stderr)
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target not restored")
	}
	if got := readBytes(t, decoy); !bytes.Equal(got, []byte("decoy")) {
		t.Fatal("rollback touched an unrelated path")
	}
}

func TestRollbackNonRoot(t *testing.T) {
	requireSupportedPlatform(t)
	cfg, _ := rollbackFixture(t)

	oldEUID := processEUID
	processEUID = func() int { return 1000 }
	defer func() { processEUID = oldEUID }()
	var stdout, stderr bytes.Buffer
	if code := RollbackEntry(nil, &stdout, &stderr, cfg); code != 0 {
		t.Fatalf("non-root run must exit 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "must run as root") {
		t.Fatalf("stderr = %q, want root-required message", stderr.String())
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, readBytes(t, buildHelper(t, "v9.9.9"))) {
		t.Fatal("target changed during a non-root run")
	}
}

func TestRollbackFailure(t *testing.T) {
	requireSupportedPlatform(t)
	cfg, _ := rollbackFixture(t)
	// Make the target directory unwritable so the atomic restore fails; the
	// failure must surface as exit 1, unlike the apply path's exit 0.
	if err := os.Chmod(filepath.Dir(cfg.Target), 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(cfg.Target), 0o755)

	code, _, stderr := runRollbackAsRoot(t, cfg)
	if code != 1 {
		t.Fatalf("restore failure must exit 1, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "rollback failed") {
		t.Fatalf("stderr = %q, want failure message", stderr)
	}
}
