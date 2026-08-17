package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/updater"
)

// updateTestClient returns a Client pointed at a TLS test server serving a
// release whose binary asset is asset and tag is tag.
func updateTestClient(t *testing.T, tag string, asset []byte) *updater.Client {
	t.Helper()
	sums := sha256.Sum256(asset)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
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
	return updater.NewClient(updater.WithHTTPClient(srv.Client()), updater.WithBaseURL(srv.URL))
}

func requireUpdatePlatform(t *testing.T) {
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

var (
	updateHelperOnce sync.Once
	updateHelperPath string
)

// buildUpdateHelper compiles the updater probe fixture with a release
// version, standing in for the real server binary during staging tests.
func buildUpdateHelper(t *testing.T) string {
	t.Helper()
	updateHelperOnce.Do(func() {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		root := filepath.Dir(filepath.Dir(wd))
		bin := filepath.Join(t.TempDir(), "probehelper")
		cmd := exec.Command("go", "build", "-o", bin,
			"-ldflags", "-X github.com/therealtinhtute/llmhub/internal/buildinfo.Version=v9.9.9",
			"./internal/updater/testdata/probehelper")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building probe helper: %v\n%s", err, out)
		}
		updateHelperPath = bin
	})
	return updateHelperPath
}

func TestVersionCommandShort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runVersion([]string{"--short"}, &stdout, &stderr, "v1.2.3"); code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stdout.String() != "v1.2.3\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "v1.2.3\n")
	}
}

func TestVersionCommandBareNormalized(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runVersion([]string{"--short"}, &stdout, &stderr, "1.2.3"); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if stdout.String() != "v1.2.3\n" {
		t.Fatalf("stdout = %q, want normalized %q", stdout.String(), "v1.2.3\n")
	}
}

func TestVersionCommandDevFallback(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runVersion([]string{"--short"}, &stdout, &stderr, "dev"); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if stdout.String() != "dev\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "dev\n")
	}
}

func TestVersionCommandFull(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runVersion(nil, &stdout, &stderr, "v1.2.3"); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	want := "LLMHub Version: v1.2.3, Commit: none, BuiltAt: unknown\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestVersionCommandUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runVersion([]string{"--bogus"}, &stdout, &stderr, "v1.2.3"); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("flag provided but not defined")) {
		t.Fatalf("stderr = %q, want flag-parse diagnostic", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runVersion([]string{"extra"}, &stdout, &stderr, "v1.2.3"); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("usage: llmhub version")) {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func TestUpdateCommandCheckAvailable(t *testing.T) {
	engine := updater.NewEngine(updateTestClient(t, "v9.9.9", nil), t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate([]string{"--check"}, &stdout, &stderr, engine); code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("update available: v9.9.9 (current v1.0.0)")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUpdateCommandCheckUpToDate(t *testing.T) {
	engine := updater.NewEngine(updateTestClient(t, "v1.0.0", nil), t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate([]string{"--check"}, &stdout, &stderr, engine); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("up to date (v1.0.0)")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUpdateCommandCheckNewerThanStable(t *testing.T) {
	engine := updater.NewEngine(updateTestClient(t, "v0.9.0", nil), t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate([]string{"--check"}, &stdout, &stderr, engine); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("running v1.0.0 is newer than stable v0.9.0")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUpdateCommandCheckDevBuild(t *testing.T) {
	engine := updater.NewEngine(updater.NewClient(), t.TempDir(), "dev")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate([]string{"--check"}, &stdout, &stderr, engine); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("development build")) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestUpdateCommandCheckServerError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	engine := updater.NewEngine(
		updater.NewClient(updater.WithHTTPClient(srv.Client()), updater.WithBaseURL(srv.URL)),
		t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate([]string{"--check"}, &stdout, &stderr, engine); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("update check failed")) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestUpdateCommandStages(t *testing.T) {
	requireUpdatePlatform(t)
	bin, err := os.ReadFile(buildUpdateHelper(t))
	if err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	engine := updater.NewEngine(updateTestClient(t, "v9.9.9", bin), dataDir, "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate(nil, &stdout, &stderr, engine); code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("staged v9.9.9")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, "update"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("update dir has %d entries, want exactly 2", len(entries))
	}
}

func TestUpdateCommandAlreadyUpToDate(t *testing.T) {
	engine := updater.NewEngine(updateTestClient(t, "v1.0.0", nil), t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate(nil, &stdout, &stderr, engine); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("already up to date (v1.0.0)")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUpdateCommandDowngradeRefused(t *testing.T) {
	engine := updater.NewEngine(updateTestClient(t, "v0.9.0", nil), t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate(nil, &stdout, &stderr, engine); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("downgrade refused")) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestUpdateCommandUsage(t *testing.T) {
	engine := updater.NewEngine(updater.NewClient(), t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate([]string{"--bogus"}, &stdout, &stderr, engine); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("flag provided but not defined")) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

// TestUpdateCommandRollback wires `update rollback` into the same parser as
// --check. The test process runs unprivileged, so the root gate reports and
// exits 0 without touching anything; the root-path restore behavior is
// covered by the updater package's TestRollback*.
func TestUpdateCommandRollback(t *testing.T) {
	engine := updater.NewEngine(updater.NewClient(), t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate([]string{"rollback"}, &stdout, &stderr, engine); code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("must run as root")) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestUpdateCommandRollbackUsage(t *testing.T) {
	engine := updater.NewEngine(updater.NewClient(), t.TempDir(), "v1.0.0")
	var stdout, stderr bytes.Buffer
	if code := runSelfUpdate([]string{"rollback", "extra"}, &stdout, &stderr, engine); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("usage: llmhub update rollback")) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEarlyCommandDispatch(t *testing.T) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = origStdout
		w.Close()
		r.Close()
	}()

	if code, ok := dispatchEarlyCommand([]string{"version", "--short"}); !ok || code != 0 {
		t.Fatalf("dispatch(version) = %d, %v; want 0, true", code, ok)
	}
	w.Close()
	out, _ := io.ReadAll(r)
	if string(out) != "dev\n" {
		t.Fatalf("version --short through dispatch = %q, want %q (test binary is dev)", out, "dev\n")
	}
}

func TestEarlyCommandDispatchUnknown(t *testing.T) {
	if _, ok := dispatchEarlyCommand([]string{"serve"}); ok {
		t.Fatal("dispatch accepted an unknown positional command")
	}
	if _, ok := dispatchEarlyCommand(nil); ok {
		t.Fatal("dispatch handled an empty argv")
	}
}

func TestExistingDBCommands(t *testing.T) {
	// -h exits 2 at flag parse time, before any Postgres access, proving the
	// commands still dispatch through the early path.
	if code, ok := dispatchEarlyCommand([]string{"init-db-from-env", "-h"}); !ok || code != 2 {
		t.Fatalf("dispatch(init-db-from-env -h) = %d, %v; want 2, true", code, ok)
	}
	if code, ok := dispatchEarlyCommand([]string{"migrate-local-to-db", "-h"}); !ok || code != 2 {
		t.Fatalf("dispatch(migrate-local-to-db -h) = %d, %v; want 2, true", code, ok)
	}
}
