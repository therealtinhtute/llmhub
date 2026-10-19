package watcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestAuthDeleteFileRemovalClassification(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "accounts.json")
	w := &Watcher{
		authDir:         dir,
		config:          &config.Config{AuthDir: dir},
		authQueue:       make(chan AuthUpdate, 8),
		lastAuthHashes:  make(map[string]string),
		fileAuthsByPath: make(map[string]map[string]*coreauth.Auth),
	}
	write := func(data string) {
		t.Helper()
		if errWriteFile := os.WriteFile(path, []byte(data), 0o600); errWriteFile != nil {
			t.Fatal(errWriteFile)
		}
		w.addOrUpdateClient(path)
	}
	assertDelete := func(id string, fileRemoval bool) {
		t.Helper()
		update := w.pendingUpdates[id]
		if update.Action != AuthUpdateActionDelete || update.Auth != nil || update.IsFileRemoval() != fileRemoval || update.Revision() == 0 {
			t.Fatalf("delete %s: got %+v, want file removal=%t", id, update, fileRemoval)
		}
		if w.currentAuths[id] != nil {
			t.Fatalf("deleted auth %s is still in watcher state", id)
		}
	}

	// 1. Initial multi-auth file with two virtual projects
	write(`{"type":"gemini-cli","email":"test@example.com","project_id":"proj-a,proj-b"}`)
	idA := "accounts.json::proj-a"
	idB := "accounts.json::proj-b"
	if w.currentAuths[idA] == nil || w.currentAuths[idB] == nil {
		t.Fatalf("expected both virtual auths in current state: a=%+v b=%+v", w.currentAuths[idA], w.currentAuths[idB])
	}
	initialRevA := w.pendingUpdates[idA].Revision()

	// 2. Removing one ID from an existing file is a content change, NOT a whole-file removal.
	write(`{"type":"gemini-cli","email":"test@example.com","project_id":"proj-b,proj-c"}`)
	assertDelete(idA, false)
	if w.pendingUpdates[idA].Revision() <= initialRevA {
		t.Fatal("content delete did not advance the revision")
	}
	if _, errStat := os.Stat(path); errStat != nil {
		t.Fatalf("source file must still exist: %v", errStat)
	}

	// 3. Whole file removal on disk -> all IDs from that file marked as whole-file removal.
	if errRemove := os.Remove(path); errRemove != nil {
		t.Fatal(errRemove)
	}
	idC := "accounts.json::proj-c"
	w.removeClient(path)
	assertDelete(idB, true)
	assertDelete(idC, true)
	removalRevB := w.pendingUpdates[idB].Revision()

	// 4. Re-create file without a create event. Snapshot must read disk, not cleared state or tombstones.
	if errWriteFile := os.WriteFile(path, []byte(`{"type":"gemini-cli","email":"test@example.com","project_id":"proj-b"}`), 0o600); errWriteFile != nil {
		t.Fatal(errWriteFile)
	}
	fullSnapshot := w.SnapshotCoreAuths()
	if len(fullSnapshot) == 0 {
		t.Fatal("expected auths from re-created file in full snapshot")
	}
	foundB := false
	for _, a := range fullSnapshot {
		if a.ID == "accounts.json" || a.ID == idB {
			foundB = true
			break
		}
	}
	if !foundB {
		t.Fatalf("expected proj-b from re-created file, got %+v", fullSnapshot)
	}

	fileSnapshot, errSnapshotAuthFile := w.SnapshotAuthFile(path)
	if errSnapshotAuthFile != nil || len(fileSnapshot) == 0 {
		t.Fatalf("single-file snapshot failed: auths=%+v, err=%v", fileSnapshot, errSnapshotAuthFile)
	}

	// 5. Newer add replaces marked delete, and later content delete does not inherit file-removal mark.
	write(`{"type":"gemini-cli","email":"test@example.com","project_id":"proj-a,proj-b"}`)
	addition := w.pendingUpdates[idB]
	if addition.Action != AuthUpdateActionAdd || addition.IsFileRemoval() || addition.Revision() <= removalRevB {
		t.Fatalf("expected newer unmarked add, got %+v", addition)
	}
	write(`{"type":"gemini-cli","email":"test@example.com","project_id":"proj-a"}`)
	assertDelete(idB, false)
}

func TestAuthDeleteFullScanIsNotFileRemoval(t *testing.T) {
	for _, filePresent := range []bool{true, false} {
		name := "file_absent"
		if filePresent {
			name = "file_present"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "accounts.json")
			if filePresent {
				if errWriteFile := os.WriteFile(path, []byte(`{"type":"codex"}`), 0o600); errWriteFile != nil {
					t.Fatal(errWriteFile)
				}
			}
			auth := &coreauth.Auth{ID: "account.json", Attributes: map[string]string{coreauth.AttributePath: path}}
			w := &Watcher{currentAuths: map[string]*coreauth.Auth{auth.ID: auth}, authQueue: make(chan AuthUpdate, 1)}
			updates := w.prepareAuthUpdatesLocked(nil, false)
			if len(updates) != 1 || updates[0].Action != AuthUpdateActionDelete || updates[0].IsFileRemoval() || updates[0].Auth != nil {
				t.Fatalf("expected unmarked full-scan delete, got %+v", updates)
			}
		})
	}
}
