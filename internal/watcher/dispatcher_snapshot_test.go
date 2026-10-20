package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestWatcherFileSnapshotReformatAndDifferentDigests(t *testing.T) {
	t.Run("authEqual_differentiates_differing_file_snapshots", func(t *testing.T) {
		a := &coreauth.Auth{ID: "acc.json"}
		b := &coreauth.Auth{ID: "acc.json"}
		a.BindFileSnapshot("/path/to/acc.json", []byte(`{"type":"codex","prefix":"pfx-1"}`))
		b.BindFileSnapshot("/path/to/acc.json", []byte(`{"type":"codex","prefix":"pfx-2"}`))

		if authEqual(a, b) {
			t.Fatal("authEqual must return false when two file snapshots have different content digests")
		}
	})

	t.Run("authEqual_matches_same_canonical_file_snapshot", func(t *testing.T) {
		a := &coreauth.Auth{ID: "acc.json", Provider: "codex"}
		b := &coreauth.Auth{ID: "acc.json", Provider: "codex"}
		dataA := []byte(`{"type":"codex","prefix":"pfx-1"}`)
		dataB := []byte("{\n  \"prefix\": \"pfx-1\",\n  \"type\": \"codex\"\n}\n")
		a.BindFileSnapshot("/path/to/acc.json", dataA)
		b.BindFileSnapshot("/path/to/acc.json", dataB)

		if !authEqual(a, b) {
			t.Fatal("authEqual must return true when file snapshots share the same canonical digest")
		}
	})

	t.Run("reformatted_file_deduplicated_by_watcher", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "account.json")
		if errWrite := os.WriteFile(path, []byte(`{"type":"codex","prefix":"pfx-initial"}`), 0o600); errWrite != nil {
			t.Fatal(errWrite)
		}
		w := &Watcher{authDir: dir, config: &config.Config{AuthDir: dir}, authQueue: make(chan AuthUpdate, 10)}
		w.addOrUpdateClient(path)
		initBatch, okInit := w.nextPendingBatch(context.Background())
		if !okInit || len(initBatch) != 1 {
			t.Fatalf("expected 1 initial update, got %v", initBatch)
		}
		if !initBatch[0].Auth.HasFileSnapshot() {
			t.Fatal("expected synthesized auth to have file snapshot bound")
		}

		// Edit file to B; Watcher queues snapshot B.
		if errWrite := os.WriteFile(path, []byte(`{"type":"codex","prefix":"pfx-edited"}`), 0o600); errWrite != nil {
			t.Fatal(errWrite)
		}
		w.addOrUpdateClient(path)
		batchB, okB := w.nextPendingBatch(context.Background())
		if !okB || len(batchB) != 1 {
			t.Fatalf("expected 1 update for B, got %v", batchB)
		}

		// Reformat file on disk with whitespace and reversed key order (B').
		// Content is semantically identical so Watcher deduplicates it.
		if errWrite := os.WriteFile(path, []byte("{\n  \"prefix\": \"pfx-edited\",\n  \"type\": \"codex\"\n}\n"), 0o600); errWrite != nil {
			t.Fatal(errWrite)
		}
		w.addOrUpdateClient(path)
		if len(w.pendingOrder) != 0 {
			t.Fatalf("expected reformatted B' to be deduplicated, got pending=%v", w.pendingOrder)
		}
	})
}
