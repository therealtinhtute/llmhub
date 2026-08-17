package updater

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// helperDir holds probe-helper binaries built once per test run.
var (
	helperDir string
	helperMu  sync.Mutex
	helpers   = map[string]string{} // version -> binary path
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "updater-probehelper-*")
	if err != nil {
		os.Exit(1)
	}
	helperDir = dir
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// buildHelper compiles testdata/probehelper with buildinfo.Version set to
// version, mirroring the real binary's `version --short` contract. key
// "slow" builds the init-hanging variant for timeout tests.
func buildHelper(t *testing.T, version string) string {
	t.Helper()
	helperMu.Lock()
	defer helperMu.Unlock()
	if p, ok := helpers[version]; ok {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(wd))
	bin := filepath.Join(helperDir, "probehelper-"+version)
	args := []string{"build", "-o", bin,
		"-ldflags", "-X github.com/therealtinhtute/llmhub/internal/buildinfo.Version=" + version}
	if version == "slow" {
		args = append(args, "-tags", "probehelper_slow")
	}
	args = append(args, "./internal/updater/testdata/probehelper")
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building probe helper %s: %v\n%s", version, err, out)
	}
	helpers[version] = bin
	return bin
}

func TestProbeExactStdout(t *testing.T) {
	helper := buildHelper(t, "v9.9.9")
	if err := Probe(helper, "v9.9.9", 5*time.Second); err != nil {
		t.Fatalf("Probe with exact version: %v", err)
	}
	if err := Probe(helper, "v9.9.10", 5*time.Second); err == nil {
		t.Fatal("Probe accepted mismatched stdout")
	}
}

func TestProbeTimeout(t *testing.T) {
	helper := buildHelper(t, "slow")
	if err := Probe(helper, "v9.9.9", 200*time.Millisecond); err == nil {
		t.Fatal("Probe accepted a non-starting candidate")
	}
}

func TestProbeSanitizedEnv(t *testing.T) {
	// The helper exits 7 when runtime-config or database env leaks through;
	// a successful probe proves the environment was sanitized.
	t.Setenv("PGSTORE_DSN", "ambient-dsn-value")
	t.Setenv("LLMHUB_INIT_CONFIG_YAML", "ambient-config-value")
	helper := buildHelper(t, "v9.9.9")
	if err := Probe(helper, "v9.9.9", 5*time.Second); err != nil {
		t.Fatalf("Probe with ambient DB/config env: %v", err)
	}
}

func TestProbeIsolatedWorkingDir(t *testing.T) {
	helper := buildHelper(t, "v9.9.9")

	// Negative control: run the helper directly with a non-empty cwd; it must
	// refuse (exit 8), proving its isolation check is live.
	busy := t.TempDir()
	if err := os.WriteFile(filepath.Join(busy, "config.yaml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(helper, "version", "--short")
	cmd.Dir = busy
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	if err := cmd.Run(); err == nil {
		t.Fatal("helper accepted a non-isolated working directory")
	}

	// Probe must run from its own empty temp dir even though the test's own
	// working directory is full of files.
	if err := Probe(helper, "v9.9.9", 5*time.Second); err != nil {
		t.Fatalf("Probe did not isolate its working directory: %v", err)
	}
}
