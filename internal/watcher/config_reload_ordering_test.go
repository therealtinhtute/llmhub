package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
)

func newIssue6631ConfigFile(t *testing.T) (configPath, authDir string) {
	t.Helper()
	tmpDir := t.TempDir()
	authDir = filepath.Join(tmpDir, "auth")
	if errMkdir := os.MkdirAll(authDir, 0o755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	return filepath.Join(tmpDir, "config.yaml"), authDir
}

// writeIssue6631Config returns an error instead of failing the test so reload
// callbacks running on other goroutines can use it.
func writeIssue6631Config(configPath, authDir string, requestRetry int) error {
	raw := fmt.Sprintf("auth-dir: %q\nrequest-retry: %d\n", filepath.ToSlash(authDir), requestRetry)
	return os.WriteFile(configPath, []byte(raw), 0o600)
}

func TestReloadConfigIfChangedReloadsWriteLandingDuringReload_Issue6631(t *testing.T) {
	configPath, authDir := newIssue6631ConfigFile(t)
	if errWrite := writeIssue6631Config(configPath, authDir, 1); errWrite != nil {
		t.Fatal(errWrite)
	}

	var applied []int
	var errSave error
	w := &Watcher{
		configPath: configPath,
		authDir:    authDir,
		reloadCallback: func(cfg *config.Config) {
			applied = append(applied, cfg.RequestRetry)
			if len(applied) == 1 {
				// A management save lands while this reload is still applying.
				errSave = writeIssue6631Config(configPath, authDir, 2)
			}
		},
	}

	w.ReloadConfigIfChanged()
	if errSave != nil {
		t.Fatal(errSave)
	}
	// The reload requested by that save must apply it.
	w.ReloadConfigIfChanged()

	if want := []int{1, 2}; !reflect.DeepEqual(applied, want) {
		t.Fatalf("applied request-retry values = %v, want %v", applied, want)
	}
}
