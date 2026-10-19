package synthesizer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestFileSnapshotCannotOverwriteLaterPersistedState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "synthetic.json")
	write := func(prefix string) []byte {
		data := []byte(`{"type":"codex","prefix":"` + prefix + `"}`)
		if errWrite := os.WriteFile(path, data, 0o600); errWrite != nil {
			t.Fatal(errWrite)
		}
		return data
	}
	snapshot := func(data []byte) *coreauth.Auth {
		items, errSynthesize := SynthesizeAuthFile(&SynthesisContext{
			Config:            &config.Config{},
			AuthDir:           dir,
			Now:               time.Now(),
			IDGenerator:       NewStableIDGenerator(),
			StrictFileParsing: true,
		}, path, data)
		if errSynthesize != nil || len(items) != 1 {
			t.Fatalf("snapshot: %v count=%d", errSynthesize, len(items))
		}
		return items[0]
	}

	initialData := write("pfx-1")
	initialAuth := snapshot(initialData)
	if !initialAuth.HasFileSnapshot() {
		t.Fatal("expected synthesized auth to have file snapshot bound")
	}
	if !initialAuth.FileSnapshotMatchesDisk() {
		t.Fatal("initial snapshot should match disk")
	}

	// Later write "pfx-2" to disk
	write("pfx-2")

	// Old snapshot for "pfx-1" is now stale against disk
	if initialAuth.FileSnapshotMatchesDisk() {
		t.Fatal("stale snapshot must report FileSnapshotMatchesDisk() == false after disk edit")
	}

	// Delete file on disk
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	// Stale snapshot must report false after file deletion
	if initialAuth.FileSnapshotMatchesDisk() {
		t.Fatal("snapshot must report FileSnapshotMatchesDisk() == false after file removal")
	}
}

func TestSynthesizeFileStrictParsingSurfacesError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{"type":`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Under strict parsing, error is surfaced
	ctxStrict := &SynthesisContext{
		Config:            &config.Config{},
		AuthDir:           dir,
		Now:               time.Now(),
		IDGenerator:       NewStableIDGenerator(),
		StrictFileParsing: true,
	}
	auths, err := SynthesizeAuthFile(ctxStrict, path, []byte(`{"type":`))
	if err == nil {
		t.Fatal("expected parse error with StrictFileParsing true, got nil")
	}
	if len(auths) != 0 {
		t.Fatalf("expected 0 auths on error, got %d", len(auths))
	}

	// Without strict parsing, returns nil, nil
	ctxLenient := &SynthesisContext{
		Config:      &config.Config{},
		AuthDir:     dir,
		Now:         time.Now(),
		IDGenerator: NewStableIDGenerator(),
	}
	authsLenient, errLenient := SynthesizeAuthFile(ctxLenient, path, []byte(`{"type":`))
	if errLenient != nil {
		t.Fatalf("expected nil error with StrictFileParsing false, got %v", errLenient)
	}
	if len(authsLenient) != 0 {
		t.Fatalf("expected 0 auths on lenient error, got %d", len(authsLenient))
	}
}
