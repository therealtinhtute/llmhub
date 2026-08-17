package updater

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

// ErrNoPreviousBinary reports that <target>.previous does not exist.
var ErrNoPreviousBinary = errors.New("no previous binary to restore")

// RollbackEntry is the `update rollback` subcommand body: refuse unprivileged
// runs, then restore <target>.previous over the target. A missing backup is
// not an error (exit 0); a real restore failure exits 1 so the operator sees
// it, unlike the startup apply path which must never block boot.
func RollbackEntry(args []string, stdout, stderr io.Writer, cfg ApplyConfig) int {
	fs := flag.NewFlagSet("rollback", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: llmhub update rollback")
		return 2
	}
	if processEUID() != 0 {
		fmt.Fprintln(stderr, "update rollback must run as root; skipping")
		return 0
	}
	if err := Rollback(cfg); err != nil {
		if errors.Is(err, ErrNoPreviousBinary) {
			fmt.Fprintln(stderr, "update rollback skipped: no previous binary to restore")
			return 0
		}
		fmt.Fprintf(stderr, "update rollback failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "rolled back to %s\n", cfg.Target)
	return 0
}

// Rollback restores the pre-swap binary from <target>.previous, clearing the
// boot-loop marker so the next apply starts a fresh cycle instead of
// re-installing the staged candidate. The staged file is left in place for
// inspection. The backup itself is never overwritten.
func Rollback(cfg ApplyConfig) error {
	if cfg.Target == "" {
		return errors.New("empty install target")
	}
	backup := cfg.Target + ".previous"
	if !fileExists(backup) {
		return ErrNoPreviousBinary
	}
	if err := revertToPrevious(cfg.Target); err != nil {
		return err
	}
	_ = os.Remove(markerPath(cfg))
	_ = os.Remove(bootedTokenPath(cfg))
	return nil
}
