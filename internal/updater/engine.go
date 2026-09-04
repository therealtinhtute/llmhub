package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Staging layout inside ${DATA_DIR}: exactly two files after a successful
// stage, and nothing executable on any failure. UpdateDirName is the staging
// subdirectory, exported for the server's healthy-start token (markBooted).
const (
	UpdateDirName      = "update"
	stagedBinaryName   = "llmhub.staged"
	stagedManifestName = "staged.json"
)

const updateDirName = UpdateDirName

// probeTimeout bounds the version probe subprocess (R8). A var so tests can
// shorten it.
var probeTimeout = 10 * time.Second

// Engine discovers, verifies, probes, and stages the latest release binary
// into ${DATA_DIR}/update/. It never touches the installed target.
type Engine struct {
	Client  *Client
	DataDir string
	Version string // running binary's buildinfo version
}

// NewEngine returns an engine staging into dataDir/update.
func NewEngine(c *Client, dataDir, version string) *Engine {
	return &Engine{Client: c, DataDir: dataDir, Version: version}
}

// CheckResult is what `update --check` reports; nothing is downloaded or
// staged. Current and Latest are canonical "v"-prefixed versions.
type CheckResult struct {
	Current  string
	Latest   string
	Decision Decision
}

// Check compares the running version with the stable channel without any
// network-side effects beyond one release-metadata request.
func (e *Engine) Check(ctx context.Context) (CheckResult, error) {
	if _, err := normalizeVersion(e.Version); err != nil {
		return CheckResult{}, ErrDevelopmentBuild
	}
	rel, err := e.Client.LatestRelease(ctx)
	if err != nil {
		return CheckResult{}, err
	}
	decision, err := Evaluate(e.Version, rel.Tag)
	if err != nil {
		return CheckResult{}, err
	}
	cur, _ := normalizeVersion(e.Version)
	latest, _ := normalizeStable(rel.Tag)
	return CheckResult{Current: cur, Latest: latest, Decision: decision}, nil
}

// StageLatest downloads, verifies, probes, and stages the latest stable
// release. On success the update directory holds exactly llmhub.staged and
// staged.json; on any failure no executable residue remains and any existing
// staged state is untouched.
func (e *Engine) StageLatest(ctx context.Context) (StagedManifest, error) {
	if _, err := AssetName(runtime.GOOS, runtime.GOARCH); err != nil {
		return StagedManifest{}, fmt.Errorf("%w: %v", ErrUnsupportedPlatform, err)
	}
	// A development build is decided locally: never even contact the network.
	if _, err := normalizeVersion(e.Version); err != nil {
		return StagedManifest{}, ErrDevelopmentBuild
	}

	rel, err := e.Client.LatestRelease(ctx)
	if err != nil {
		return StagedManifest{}, err
	}

	decision, err := Evaluate(e.Version, rel.Tag)
	if err != nil {
		return StagedManifest{}, err
	}
	switch decision {
	case DecisionUpdateAvailable:
		// fall through
	case DecisionUpToDate:
		return StagedManifest{}, ErrUpToDate
	case DecisionDowngradeRefused:
		return StagedManifest{}, ErrDowngradeRefused
	}

	name, err := AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return StagedManifest{}, fmt.Errorf("%w: %v", ErrUnsupportedPlatform, err)
	}
	asset, err := SelectAsset(rel, name)
	if err != nil {
		return StagedManifest{}, err
	}
	sumAsset, err := SelectAsset(rel, "checksums.txt")
	if err != nil {
		return StagedManifest{}, err
	}

	version, err := normalizeStable(rel.Tag)
	if err != nil {
		return StagedManifest{}, err
	}

	// Fetch and parse the checksum manifest before downloading the binary:
	// a missing or malformed manifest aborts without a multi-megabyte
	// download.
	var sums bytes.Buffer
	if err := e.Client.FetchMetadata(ctx, sumAsset.URL, &sums); err != nil {
		return StagedManifest{}, fmt.Errorf("fetching checksums.txt: %w", err)
	}
	digest, err := ParseChecksum(sums.String(), name)
	if err != nil {
		return StagedManifest{}, err
	}

	updateDir := filepath.Join(e.DataDir, updateDirName)
	if err := os.MkdirAll(updateDir, 0o755); err != nil {
		return StagedManifest{}, fmt.Errorf("creating staging directory: %w", err)
	}

	// Download inside the staging directory so the final rename is atomic and
	// nothing ever leaves ${DATA_DIR}/update/.
	tmp, err := os.CreateTemp(updateDir, ".download-*")
	if err != nil {
		return StagedManifest{}, err
	}
	// CreateTemp makes 0600 files; the probe must be able to exec the
	// candidate, and the final stage is 0755 anyway.
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return StagedManifest{}, err
	}
	fail := func(err error) (StagedManifest, error) {
		// Close may already have happened; only the removal matters.
		os.Remove(tmp.Name())
		return StagedManifest{}, err
	}

	if err := e.Client.FetchAsset(ctx, asset.URL, tmp); err != nil {
		tmp.Close()
		return fail(err)
	}
	// Seal the write descriptor before hashing and probing: exec of a file
	// with an open write handle fails with ETXTBSY on Linux.
	if err := tmp.Close(); err != nil {
		return fail(err)
	}

	sealed, err := os.Open(tmp.Name())
	if err != nil {
		return fail(err)
	}
	h := sha256.New()
	if _, err := io.Copy(h, sealed); err != nil {
		sealed.Close()
		return fail(err)
	}
	sealed.Close()
	if got := hex.EncodeToString(h.Sum(nil)); got != digest {
		return fail(fmt.Errorf("checksum mismatch for %s: got %s, want %s", name, got, digest))
	}

	// R8: only after checksum verification, run the candidate bounded.
	if err := Probe(tmp.Name(), version, probeTimeout); err != nil {
		return fail(err)
	}

	staged := filepath.Join(updateDir, stagedBinaryName)
	if err := os.Rename(tmp.Name(), staged); err != nil {
		return fail(err)
	}

	manifest := StagedManifest{Version: version, Digest: digest}
	if err := WriteStagedManifest(updateDir, manifest); err != nil {
		os.Remove(staged)
		return StagedManifest{}, err
	}
	return manifest, nil
}

// Probe runs candidate's `version --short` as a bounded subprocess in a
// sanitized environment and isolated temporary working directory, requiring
// exact normalized stdout. This is the pre-stage behavior check: a candidate
// that cannot identify itself must never become staged.
func Probe(candidate, wantVersion string, timeout time.Duration) error {
	dir, err := os.MkdirTemp("", "llmhub-probe-*")
	if err != nil {
		return fmt.Errorf("creating probe working directory: %w", err)
	}
	defer os.RemoveAll(dir)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, candidate, "version", "--short")
	cmd.Dir = dir
	// Sanitized whitelist: the probe must not see operator config, database
	// credentials, or any other ambient environment.
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("candidate probe failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	if got := strings.TrimSpace(stdout.String()); got != wantVersion {
		return fmt.Errorf("candidate version %q does not match expected %q", got, wantVersion)
	}
	return nil
}
