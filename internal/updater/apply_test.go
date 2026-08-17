package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// applyTestClient serves release metadata for tag whose checksums.txt matches
// asset bytes, plus the asset itself.
func applyTestClient(t *testing.T, tag string, asset []byte) *Client {
	t.Helper()
	sums := sha256.Sum256(asset)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/tags/" + tag:
			host := "https://" + r.Host
			fmt.Fprintf(w, `{"tag_name":%q,"assets":[
				{"name":"llmhub-linux-amd64","browser_download_url":%q,"size":%d},
				{"name":"checksums.txt","browser_download_url":%q,"size":%d}]}`,
				tag, host+"/bin", len(asset), host+"/sums", 512)
		case "/bin":
			_, _ = w.Write(asset)
		case "/sums":
			fmt.Fprintf(w, "%x  llmhub-linux-amd64\n", sums)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return testClient(t, srv)
}

// stageApplyFixture writes a staged candidate at dataDir/update and returns
// its path.
func stageApplyFixture(t *testing.T, dataDir string, binBytes []byte, version string) string {
	t.Helper()
	updateDir := filepath.Join(dataDir, updateDirName)
	if err := os.MkdirAll(updateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(updateDir, stagedBinaryName)
	if err := os.WriteFile(staged, binBytes, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteStagedManifest(updateDir, StagedManifest{Version: version, Digest: "x"}); err != nil {
		t.Fatal(err)
	}
	return staged
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestApplyInstallsNewerBinary(t *testing.T) {
	requireSupportedPlatform(t)
	oldBin := readBytes(t, buildHelper(t, "v1.0.0"))
	newBin := readBytes(t, buildHelper(t, "v9.9.9"))

	cfg := ApplyConfig{
		DataDir:         t.TempDir(),
		MarkerDir:       t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v1.0.0",
		Client:          applyTestClient(t, "v9.9.9", newBin),
	}
	if err := os.WriteFile(cfg.Target, oldBin, 0o755); err != nil {
		t.Fatal(err)
	}
	stageApplyFixture(t, cfg.DataDir, newBin, "v9.9.9")

	var stdout, stderr bytes.Buffer
	oldEUID := processEUID
	processEUID = func() int { return 0 }
	defer func() { processEUID = oldEUID }()
	if code := ApplyEntry(nil, &stdout, &stderr, cfg); code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, stderr.String())
	}

	if got := readBytes(t, cfg.Target); !bytes.Equal(got, newBin) {
		t.Fatal("target was not replaced with the staged binary")
	}
	info, err := os.Stat(cfg.Target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("target mode %v is not executable", info.Mode())
	}
	if got := readBytes(t, cfg.Target+".previous"); !bytes.Equal(got, oldBin) {
		t.Fatal("replaced binary was not retained as <target>.previous")
	}
}

func TestApplyReverifyForged(t *testing.T) {
	requireSupportedPlatform(t)
	oldBin := readBytes(t, buildHelper(t, "v1.0.0"))
	realBin := readBytes(t, buildHelper(t, "v9.9.9"))

	cfg := ApplyConfig{
		DataDir:         t.TempDir(),
		MarkerDir:       t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v1.0.0",
		Client:          applyTestClient(t, "v9.9.9", realBin),
	}
	if err := os.WriteFile(cfg.Target, oldBin, 0o755); err != nil {
		t.Fatal(err)
	}
	// Forged staged file: valid name, wrong bytes.
	stageApplyFixture(t, cfg.DataDir, []byte("not the real binary"), "v9.9.9")

	if err := Apply(context.Background(), cfg); err == nil {
		t.Fatal("Apply accepted a forged staged binary")
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target changed after a forged staged binary")
	}
	if _, err := os.Stat(cfg.Target + ".previous"); !os.IsNotExist(err) {
		t.Fatal("<target>.previous must not exist after a failed apply")
	}
}

func TestApplyReverifyWrongTag(t *testing.T) {
	requireSupportedPlatform(t)
	oldBin := readBytes(t, buildHelper(t, "v1.0.0"))
	// Manifest claims v8.8.8 and the checksums for v8.8.8 match the v8.8.8
	// helper, but the staged file is the v9.9.9 helper: mismatch.
	v8 := readBytes(t, buildHelper(t, "v8.8.8"))
	v9 := readBytes(t, buildHelper(t, "v9.9.9"))

	cfg := ApplyConfig{
		DataDir:         t.TempDir(),
		MarkerDir:       t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v1.0.0",
		Client:          applyTestClient(t, "v8.8.8", v8),
	}
	if err := os.WriteFile(cfg.Target, oldBin, 0o755); err != nil {
		t.Fatal(err)
	}
	stageApplyFixture(t, cfg.DataDir, v9, "v8.8.8")

	if err := Apply(context.Background(), cfg); err == nil {
		t.Fatal("Apply accepted a staged manifest/tag mismatch")
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target changed after a forged manifest")
	}
}

func TestApplyNetworkFailure(t *testing.T) {
	requireSupportedPlatform(t)
	oldBin := readBytes(t, buildHelper(t, "v1.0.0"))

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := ApplyConfig{
		DataDir:         t.TempDir(),
		MarkerDir:       t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v1.0.0",
		Client:          testClient(t, srv),
	}
	if err := os.WriteFile(cfg.Target, oldBin, 0o755); err != nil {
		t.Fatal(err)
	}
	stageApplyFixture(t, cfg.DataDir, oldBin, "v9.9.9")

	var stdout, stderr bytes.Buffer
	oldEUID := processEUID
	processEUID = func() int { return 0 }
	defer func() { processEUID = oldEUID }()
	if code := ApplyEntry(nil, &stdout, &stderr, cfg); code != 0 {
		t.Fatalf("network failure must exit 0, got %d (stderr: %s)", code, stderr.String())
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target changed after a network failure")
	}
}

func TestApplyDowngrade(t *testing.T) {
	requireSupportedPlatform(t)
	oldBin := readBytes(t, buildHelper(t, "v2.0.0"))
	stagedBin := readBytes(t, buildHelper(t, "v1.0.0"))

	cfg := ApplyConfig{
		DataDir:         t.TempDir(),
		MarkerDir:       t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v2.0.0",
		Client:          applyTestClient(t, "v1.0.0", stagedBin),
	}
	if err := os.WriteFile(cfg.Target, oldBin, 0o755); err != nil {
		t.Fatal(err)
	}
	stageApplyFixture(t, cfg.DataDir, stagedBin, "v1.0.0")

	var stdout, stderr bytes.Buffer
	oldEUID := processEUID
	processEUID = func() int { return 0 }
	defer func() { processEUID = oldEUID }()
	if code := ApplyEntry(nil, &stdout, &stderr, cfg); code != 0 {
		t.Fatalf("downgrade must exit 0, got %d", code)
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target changed on a downgrade")
	}
}

func TestApplyNonRoot(t *testing.T) {
	requireSupportedPlatform(t)
	oldBin := readBytes(t, buildHelper(t, "v1.0.0"))

	cfg := ApplyConfig{
		DataDir:         t.TempDir(),
		MarkerDir:       t.TempDir(),
		Target:          filepath.Join(t.TempDir(), "llmhub"),
		InstalledVersion: "v1.0.0",
		Client:          applyTestClient(t, "v9.9.9", oldBin),
	}
	if err := os.WriteFile(cfg.Target, oldBin, 0o755); err != nil {
		t.Fatal(err)
	}
	stageApplyFixture(t, cfg.DataDir, oldBin, "v9.9.9")

	var stdout, stderr bytes.Buffer
	oldEUID := processEUID
	processEUID = func() int { return 1000 }
	defer func() { processEUID = oldEUID }()
	if code := ApplyEntry(nil, &stdout, &stderr, cfg); code != 0 {
		t.Fatalf("non-root run must exit 0, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("must run as root")) {
		t.Fatalf("stderr = %q, want root-required message", stderr.String())
	}
	if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
		t.Fatal("target changed during a non-root run")
	}
	if _, err := os.Stat(cfg.Target + ".previous"); !os.IsNotExist(err) {
		t.Fatal("<target>.previous must not exist after a non-root run")
	}
}

func TestApplyExitCode(t *testing.T) {
	// Every skip/failure path exits 0 and leaves the target untouched. The
	// per-path assertions live in the dedicated tests above; this one
	// verifies the exit-code contract across the failure family.
	for name, tc := range map[string]struct {
		client    *Client
		installed string
		stagedVer string
		stagedBin []byte
	}{
		"forged": {
			client:    applyTestClient(t, "v9.9.9", readBytes(t, buildHelper(t, "v9.9.9"))),
			installed: "v1.0.0",
			stagedVer: "v9.9.9",
			stagedBin: []byte("garbage"),
		},
		"downgrade": {
			client:    applyTestClient(t, "v1.0.0", readBytes(t, buildHelper(t, "v1.0.0"))),
			installed: "v2.0.0",
			stagedVer: "v1.0.0",
			stagedBin: readBytes(t, buildHelper(t, "v1.0.0")),
		},
	} {
		t.Run(name, func(t *testing.T) {
			requireSupportedPlatform(t)
			oldBin := []byte("installed-binary")
			cfg := ApplyConfig{
				DataDir:         t.TempDir(),
				MarkerDir:       t.TempDir(),
				Target:          filepath.Join(t.TempDir(), "llmhub"),
				InstalledVersion: tc.installed,
				Client:          tc.client,
			}
			if err := os.WriteFile(cfg.Target, oldBin, 0o755); err != nil {
				t.Fatal(err)
			}
			stageApplyFixture(t, cfg.DataDir, tc.stagedBin, tc.stagedVer)

			var stdout, stderr bytes.Buffer
			oldEUID := processEUID
			processEUID = func() int { return 0 }
			defer func() { processEUID = oldEUID }()
			if code := ApplyEntry(nil, &stdout, &stderr, cfg); code != 0 {
				t.Fatalf("exit %d, want 0", code)
			}
			if got := readBytes(t, cfg.Target); !bytes.Equal(got, oldBin) {
				t.Fatal("target changed on a failure path")
			}
		})
	}
}
