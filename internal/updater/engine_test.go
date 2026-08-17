package updater

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func requireSupportedPlatform(t *testing.T) {
	t.Helper()
	switch runtime.GOOS {
	case "linux", "darwin":
	default:
		t.Skipf("self-update unsupported on %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		t.Skipf("self-update unsupported on %s", runtime.GOARCH)
	}
}

// stageServer serves a release whose binary asset is binBytes and whose
// checksums.txt matches it, tagged tag.
func stageServer(t *testing.T, binBytes []byte, tag string) *httptest.Server {
	t.Helper()
	sums := sha256.Sum256(binBytes)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case latestPath:
			host := "https://" + r.Host
			fmt.Fprintf(w, `{"tag_name":%q,"assets":[
				{"name":"llmhub-linux-amd64","browser_download_url":%q,"size":%d},
				{"name":"checksums.txt","browser_download_url":%q,"size":%d}]}`,
				tag, host+"/bin", len(binBytes), host+"/sums", 512)
		case "/bin":
			_, _ = w.Write(binBytes)
		case "/sums":
			fmt.Fprintf(w, "%x  llmhub-linux-amd64\n", sums)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// updateDirContents returns the non-hidden entries of dataDir/update.
func updateDirContents(t *testing.T, dataDir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dataDir, updateDirName))
	if err != nil {
		t.Fatalf("reading update dir: %v", err)
	}
	return entries
}

func TestStageValidCandidateStagesTwoFiles(t *testing.T) {
	requireSupportedPlatform(t)
	helper := buildHelper(t, "v9.9.9")
	bin, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	srv := stageServer(t, bin, "v9.9.9")
	dataDir := t.TempDir()
	engine := NewEngine(testClient(t, srv), dataDir, "v1.0.0")

	got, err := engine.StageLatest(context.Background())
	if err != nil {
		t.Fatalf("StageLatest: %v", err)
	}

	wantSum := fmt.Sprintf("%x", sha256.Sum256(bin))
	if got.Version != "v9.9.9" || got.Digest != wantSum {
		t.Fatalf("manifest = %+v, want version v9.9.9 digest %s", got, wantSum)
	}

	entries := updateDirContents(t, dataDir)
	if len(entries) != 2 {
		t.Fatalf("update dir has %d entries, want exactly 2: %v", len(entries), entries)
	}
	staged := filepath.Join(dataDir, updateDirName, stagedBinaryName)
	info, err := os.Stat(staged)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("llmhub.staged mode %v is not executable", info.Mode())
	}
	stagedBytes, err := os.ReadFile(staged)
	if err != nil {
		t.Fatal(err)
	}
	if string(stagedBytes) != string(bin) {
		t.Fatal("staged binary differs from downloaded bytes")
	}

	manifest, err := ReadStagedManifest(filepath.Join(dataDir, updateDirName))
	if err != nil {
		t.Fatal(err)
	}
	if manifest != got {
		t.Fatalf("staged.json = %+v, want %+v", manifest, got)
	}
}

func TestDownloadCorruptedChecksumFailsAndLeavesNoResidue(t *testing.T) {
	requireSupportedPlatform(t)
	// Garbage bytes whose checksum will never match the served checksums.txt.
	srv := stageServer(t, []byte("not the real binary"), "v9.9.9")
	dataDir := t.TempDir()
	engine := NewEngine(testClient(t, srv), dataDir, "v1.0.0")

	if _, err := engine.StageLatest(context.Background()); err == nil {
		t.Fatal("StageLatest accepted a checksum mismatch")
	}
	if entries := updateDirContents(t, dataDir); len(entries) != 0 {
		t.Fatalf("update dir must be empty after failure, found %v", entries)
	}
}

func TestStageWrongVersionFails(t *testing.T) {
	requireSupportedPlatform(t)
	// Candidate reports v8.8.8 while the release claims v9.9.9: checksum is
	// valid, probe stdout must not match.
	helper := buildHelper(t, "v8.8.8")
	bin, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	srv := stageServer(t, bin, "v9.9.9")
	dataDir := t.TempDir()
	engine := NewEngine(testClient(t, srv), dataDir, "v1.0.0")

	if _, err := engine.StageLatest(context.Background()); err == nil {
		t.Fatal("StageLatest accepted a candidate reporting the wrong version")
	}
	if entries := updateDirContents(t, dataDir); len(entries) != 0 {
		t.Fatalf("update dir must be empty after failure, found %v", entries)
	}
}

func TestStageProbeTimeoutLeavesNoResidue(t *testing.T) {
	requireSupportedPlatform(t)
	bin, err := os.ReadFile(buildHelper(t, "slow"))
	if err != nil {
		t.Fatal(err)
	}
	srv := stageServer(t, bin, "v9.9.9")
	dataDir := t.TempDir()
	engine := NewEngine(testClient(t, srv), dataDir, "v1.0.0")

	old := probeTimeout
	probeTimeout = 300 * time.Millisecond
	defer func() { probeTimeout = old }()

	if _, err := engine.StageLatest(context.Background()); err == nil {
		t.Fatal("StageLatest accepted a non-starting candidate")
	}
	if entries := updateDirContents(t, dataDir); len(entries) != 0 {
		t.Fatalf("update dir must be empty after failure, found %v", entries)
	}
}

func TestStageUpToDateFails(t *testing.T) {
	srv := stageServer(t, nil, "v9.9.9")
	engine := NewEngine(testClient(t, srv), t.TempDir(), "v9.9.9")
	if _, err := engine.StageLatest(context.Background()); !errors.Is(err, ErrUpToDate) {
		t.Fatalf("err = %v, want ErrUpToDate", err)
	}
}

func TestStageDowngradeRefusedFails(t *testing.T) {
	srv := stageServer(t, nil, "v9.9.9")
	engine := NewEngine(testClient(t, srv), t.TempDir(), "v10.0.0")
	if _, err := engine.StageLatest(context.Background()); !errors.Is(err, ErrDowngradeRefused) {
		t.Fatalf("err = %v, want ErrDowngradeRefused", err)
	}
}

func TestStageDevelopmentBuildFails(t *testing.T) {
	engine := NewEngine(NewClient(), t.TempDir(), "dev")
	if _, err := engine.StageLatest(context.Background()); !errors.Is(err, ErrDevelopmentBuild) {
		t.Fatalf("err = %v, want ErrDevelopmentBuild", err)
	}
}

func TestStagedManifestRoundTrip(t *testing.T) {
	updateDir := t.TempDir()
	want := StagedManifest{Version: "v9.9.9", Digest: "abcd"}
	if err := WriteStagedManifest(updateDir, want); err != nil {
		t.Fatalf("WriteStagedManifest: %v", err)
	}
	got, err := ReadStagedManifest(updateDir)
	if err != nil {
		t.Fatalf("ReadStagedManifest: %v", err)
	}
	if got != want {
		t.Fatalf("manifest = %+v, want %+v", got, want)
	}
	entries, err := os.ReadDir(updateDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != stagedManifestName {
		t.Fatalf("update dir contains %v, want only staged.json", entries)
	}
}
