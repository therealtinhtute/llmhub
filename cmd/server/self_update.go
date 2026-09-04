package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/therealtinhtute/llmhub/internal/buildinfo"
	"github.com/therealtinhtute/llmhub/internal/updater"
)

// runVersion implements `llmhub version [--short]` (R1, R2): deterministic
// stdout with no banner or logging. Exit 0 success, 2 usage.
func runVersion(args []string, stdout, stderr io.Writer, version string) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	short := fs.Bool("short", false, "print only the normalized release version")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: llmhub version [--short]")
		return 2
	}
	if *short {
		fmt.Fprintln(stdout, canonicalVersion(version))
		return 0
	}
	fmt.Fprintf(stdout, "LLMHub Version: %s, Commit: %s, BuiltAt: %s\n",
		version, buildinfo.Commit, buildinfo.BuildDate)
	return 0
}

// canonicalVersion normalizes a buildinfo version exactly as the updater
// probe expects, falling back to the raw string (e.g. "dev") when it is not
// a semver release.
func canonicalVersion(v string) string {
	if c, err := updater.CanonicalVersion(v); err == nil {
		return c
	}
	return v
}

// stampedWriter prefixes every Write with an RFC 3339 timestamp. It wraps
// only the update command's output; `version` output must stay byte-exact
// because the updater probe compares it.
type stampedWriter struct {
	w io.Writer
}

func (s stampedWriter) Write(p []byte) (int, error) {
	return s.w.Write(append([]byte(time.Now().Format(time.RFC3339)+" "), p...))
}

// runSelfUpdate implements `llmhub update [--check | rollback]` (R1).
// Without --check it discovers, verifies, probes, and stages the latest stable
// release into ${DATA_DIR}/update/; with --check it only reports availability.
// `update rollback` restores <target>.previous (root only). Staging never
// touches the installed target. Every update log line is prefixed with a
// timestamp so operator actions are traceable; `version` stays deterministic
// because the updater probe compares its exact stdout. Exit 0 success,
// 1 failure, 2 usage.
func runSelfUpdate(args []string, stdout, stderr io.Writer, engine *updater.Engine) int {
	stdout = stampedWriter{w: stdout}
	stderr = stampedWriter{w: stderr}
	if len(args) > 0 && args[0] == "rollback" {
		return updater.RollbackEntry(args[1:], stdout, stderr, newApplyConfig())
	}
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "report whether an update is available without staging")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: llmhub update [--check | rollback]")
		return 2
	}
	ctx := context.Background()
	if *check {
		return updateCheck(ctx, stdout, stderr, engine)
	}
	return updateStage(ctx, stdout, stderr, engine)
}

func updateCheck(ctx context.Context, stdout, stderr io.Writer, engine *updater.Engine) int {
	res, err := engine.Check(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "update check failed: %v\n", err)
		return 1
	}
	switch res.Decision {
	case updater.DecisionUpdateAvailable:
		fmt.Fprintf(stdout, "update available: %s (current %s)\n", res.Latest, res.Current)
	case updater.DecisionUpToDate:
		fmt.Fprintf(stdout, "up to date (%s)\n", res.Current)
	case updater.DecisionDowngradeRefused:
		fmt.Fprintf(stdout, "running %s is newer than stable %s\n", res.Current, res.Latest)
	}
	return 0
}

func updateStage(ctx context.Context, stdout, stderr io.Writer, engine *updater.Engine) int {
	manifest, err := engine.StageLatest(ctx)
	switch {
	case err == nil:
		fmt.Fprintf(stdout, "staged %s to %s/update (sha256 %s)\n",
			manifest.Version, engine.DataDir, manifest.Digest)
		return 0
	case errors.Is(err, updater.ErrUpToDate):
		fmt.Fprintf(stdout, "already up to date (%s)\n", engine.Version)
		return 0
	default:
		fmt.Fprintf(stderr, "update failed: %v\n", err)
		return 1
	}
}

// newUpdateEngine resolves ${DATA_DIR} (install-local.sh convention; the
// systemd unit runs with WorkingDirectory=${DATA_DIR}) and builds the
// unprivileged staging engine.
func newUpdateEngine() *updater.Engine {
	return updater.NewEngine(updater.NewClient(), resolveDataDir(), buildinfo.Version)
}

// newApplyConfig resolves the root apply step's paths: the installed binary
// being replaced (os.Executable()), the staging data directory, and the
// root-owned boot-marker directory.
func newApplyConfig() updater.ApplyConfig {
	target, err := os.Executable()
	if err != nil {
		target = ""
	}
	return updater.ApplyConfig{
		DataDir:         resolveDataDir(),
		Target:          target,
		InstalledVersion: buildinfo.Version,
		Client:          updater.NewClient(),
		MarkerDir:       resolveMarkerDir(),
	}
}

// resolveMarkerDir returns ${LLMHUB_MARKER_DIR} or the install-local.sh
// default root-owned marker directory, always outside ${DATA_DIR} (R10).
func resolveMarkerDir() string {
	if dir, ok := lookupEnvTrimmed("LLMHUB_MARKER_DIR", "llmhub_marker_dir"); ok {
		return dir
	}
	return "/var/lib/llmhub-apply"
}

// markBooted records a healthy-start token inside ${DATA_DIR}/update. The
// root apply step reads it at the next start to tell a completed swap apart
// from a boot loop (R10). Best-effort: without it, the next apply treats the
// swap as unconfirmed and reverts.
func markBooted() {
	updateDir := filepath.Join(resolveDataDir(), updater.UpdateDirName)
	if err := os.MkdirAll(updateDir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(updateDir, ".booted"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(buildinfo.Version + "\n")
	_ = f.Close()
}

// resolveDataDir returns ${DATA_DIR} (install-local.sh convention; the
// systemd unit runs with WorkingDirectory=${DATA_DIR}), defaulting to the
// working directory.
func resolveDataDir() string {
	if dataDir, ok := lookupEnvTrimmed("DATA_DIR", "data_dir"); ok {
		return dataDir
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// dispatchEarlyCommand runs positional commands ahead of the startup banner,
// server flag parsing, and Postgres loading (R1). ok=false continues into
// ordinary server startup.
func dispatchEarlyCommand(args []string) (code int, ok bool) {
	if len(args) == 0 {
		return 0, false
	}
	switch args[0] {
	case "version":
		return runVersion(args[1:], os.Stdout, os.Stderr, buildinfo.Version), true
	case "update":
		return runSelfUpdate(args[1:], os.Stdout, os.Stderr, newUpdateEngine()), true
	case "apply-staged-update":
		return updater.ApplyEntry(args[1:], os.Stdout, os.Stderr, newApplyConfig()), true
	case "init-db-from-env":
		return runInitDBFromEnv(args[1:]), true
	case "migrate-local-to-db":
		return runMigrateLocalToDB(args[1:]), true
	}
	return 0, false
}
