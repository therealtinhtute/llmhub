package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

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

// runSelfUpdate implements `llmhub update [--check]` (R1). Without --check it
// discovers, verifies, probes, and stages the latest stable release into
// ${DATA_DIR}/update/; with --check it only reports availability. It never
// touches the installed target. Exit 0 success, 1 failure, 2 usage.
func runSelfUpdate(args []string, stdout, stderr io.Writer, engine *updater.Engine) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "report whether an update is available without staging")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: llmhub update [--check]")
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
	dataDir, ok := lookupEnvTrimmed("DATA_DIR", "data_dir")
	if !ok {
		if wd, err := os.Getwd(); err == nil {
			dataDir = wd
		} else {
			dataDir = "."
		}
	}
	return updater.NewEngine(updater.NewClient(), dataDir, buildinfo.Version)
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
	case "init-db-from-env":
		return runInitDBFromEnv(args[1:]), true
	case "migrate-local-to-db":
		return runMigrateLocalToDB(args[1:]), true
	}
	return 0, false
}
