package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/mod/semver"
)

// ApplyConfig carries the resolved paths for the privileged apply step.
// Target is the installed binary (os.Executable() in production; tests pass
// temporary paths).
type ApplyConfig struct {
	DataDir         string // ${DATA_DIR}; staged files live in DataDir/update
	Target          string // installed binary path
	InstalledVersion string // buildinfo.Version of the installed binary
	Client          *Client
	MarkerDir       string // root-owned boot-marker directory outside DataDir
}

// markerPath is the root-owned boot-loop marker (R10), outside ${DATA_DIR}
// so a compromised service user can neither forge nor clear it.
func markerPath(cfg ApplyConfig) string {
	return filepath.Join(cfg.MarkerDir, "apply.marker")
}

// bootedTokenPath is the service-writable healthy-start token written by the
// server inside ${DATA_DIR}/update.
func bootedTokenPath(cfg ApplyConfig) string {
	return filepath.Join(cfg.DataDir, updateDirName, ".booted")
}

// processEUID is injectable so tests can exercise the root paths.
var processEUID = func() int { return os.Geteuid() }

// ApplyEntry is the apply-staged-update subcommand body: refuse unprivileged
// runs, then install when the staged candidate re-verifies. Every skip and
// failure exits 0 after reporting, so systemd startup is never blocked (R9).
func ApplyEntry(args []string, stdout, stderr io.Writer, cfg ApplyConfig) int {
	fs := flag.NewFlagSet("apply-staged-update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: llmhub apply-staged-update")
		return 2
	}
	if processEUID() != 0 {
		fmt.Fprintln(stderr, "apply-staged-update must run as root; skipping staged update")
		return 0
	}
	if err := Apply(context.Background(), cfg); err != nil {
		fmt.Fprintf(stderr, "apply-staged-update skipped: %v\n", err)
	}
	return 0
}

// Apply installs the staged candidate over Target only after re-verifying it
// independently of the on-disk manifest (R9): the manifest merely names the
// claimed release tag; checksums.txt is re-fetched over HTTPS for that tag,
// the staged file is re-hashed against it, and the version must be strictly
// newer than the installed binary's. Every failure returns before the
// installed binary is touched.
func Apply(ctx context.Context, cfg ApplyConfig) error {
	if cfg.Target == "" {
		return errors.New("empty install target")
	}
	if cfg.Client == nil {
		return errors.New("nil re-verification client")
	}
	updateDir := filepath.Join(cfg.DataDir, updateDirName)

	// Never trusted for authenticity; only names the tag whose checksums.txt
	// must be re-fetched.
	manifest, err := ReadStagedManifest(updateDir)
	if err != nil {
		return fmt.Errorf("no staged update: %w", err)
	}
	version, err := normalizeStable(manifest.Version)
	if err != nil {
		return fmt.Errorf("staged version invalid: %w", err)
	}
	name, err := AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return fmt.Errorf("unsupported platform: %w", err)
	}

	rel, err := cfg.Client.ReleaseForTag(ctx, version)
	if err != nil {
		return fmt.Errorf("re-fetching release metadata for %s: %w", version, err)
	}
	sumAsset, err := SelectAsset(rel, "checksums.txt")
	if err != nil {
		return fmt.Errorf("re-fetching checksums for %s: %w", version, err)
	}
	var sums bytes.Buffer
	if err := cfg.Client.FetchMetadata(ctx, sumAsset.URL, &sums); err != nil {
		return fmt.Errorf("re-fetching checksums.txt for %s: %w", version, err)
	}
	digest, err := ParseChecksum(sums.String(), name)
	if err != nil {
		return fmt.Errorf("checksums.txt for %s: %w", version, err)
	}

	staged := filepath.Join(updateDir, stagedBinaryName)
	h, err := hashFile(staged)
	if err != nil {
		return fmt.Errorf("re-hashing staged binary: %w", err)
	}
	if h != digest {
		return fmt.Errorf("staged binary digest %s does not match checksums.txt digest %s", h, digest)
	}

	installed, err := normalizeVersion(cfg.InstalledVersion)
	if err != nil {
		return fmt.Errorf("installed version %q is not a release version; refusing to replace it", cfg.InstalledVersion)
	}
	if semver.Compare(version, installed) != 1 {
		return fmt.Errorf("staged %s is not strictly newer than installed %s", version, installed)
	}

	// Boot-loop guard (R10): a surviving marker means the previous swap never
	// reached a healthy start, so the previous binary is restored and nothing
	// new is applied until the marker clears.
	marker := markerPath(cfg)
	if fileExists(marker) {
		if fileExists(bootedTokenPath(cfg)) {
			// The swap's boot completed: close the cycle and apply normally.
			_ = os.Remove(marker)
			_ = os.Remove(bootedTokenPath(cfg))
		} else {
			// The previous swap never started cleanly: revert and refuse.
			backup := cfg.Target + ".previous"
			if !fileExists(backup) {
				return fmt.Errorf("boot marker present but %q is missing; leaving %q in place", backup, cfg.Target)
			}
			if err := revertToPrevious(cfg.Target); err != nil {
				return fmt.Errorf("boot marker present; revert failed: %w", err)
			}
			_ = os.Remove(marker)
			return fmt.Errorf("boot marker present: previous swap never reached a healthy start; reverted to %s", backup)
		}
	}

	// Record the outgoing version before swapping (R10), then start a fresh
	// boot cycle. A failed swap still self-heals: the next apply sees the
	// marker without a booted token and reverts to the untouched target.
	if err := writeMarker(marker, installed); err != nil {
		return fmt.Errorf("writing boot marker: %w", err)
	}
	_ = os.Remove(bootedTokenPath(cfg))

	return installOver(cfg.Target, staged)
}

// revertToPrevious restores target from <target>.previous atomically,
// without overwriting the backup itself.
func revertToPrevious(target string) error {
	backup := target + ".previous"
	tmp, err := os.CreateTemp(filepath.Dir(target), ".revert-*")
	if err != nil {
		return err
	}
	if err := copyFile(tmp.Name(), backup, 0o755); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// writeMarker records the outgoing version in the root-owned marker file.
func writeMarker(path, outgoing string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(outgoing+"\n"), 0o600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// installOver replaces target with staged, retaining the replaced binary as
// <target>.previous. All writes stay on target's filesystem; the final
// rename is atomic, so the installed binary is never missing.
func installOver(target, staged string) error {
	backup := target + ".previous"
	if err := copyFile(backup, target, 0o755); err != nil {
		return fmt.Errorf("backing up installed binary: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".apply-*")
	if err != nil {
		return err
	}
	if err := copyFile(tmp.Name(), staged, 0o755); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("staging replacement: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	// The replacement stays root-owned like the installed binary. Tests run
	// unprivileged, so ownership is only asserted when actually root.
	if processEUID() == 0 {
		_ = os.Chown(tmp.Name(), 0, 0)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("replacing installed binary: %w", err)
	}
	return nil
}

// copyFile copies src to dst with the given mode, preserving nothing else.
func copyFile(dst, src string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Chmod(mode); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

// hashFile returns the lowercase-hex SHA-256 digest of path.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
