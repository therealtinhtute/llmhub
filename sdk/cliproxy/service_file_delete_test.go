package cliproxy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/watcher"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestAuthIDMatches(t *testing.T) {
	for _, tc := range []struct {
		name       string
		snapshotID string
		id         string
		wantExact  bool
		wantFolded bool
	}{
		{name: "identical", snapshotID: "account.json", id: "account.json", wantExact: true, wantFolded: true},
		{name: "windows_store_case", snapshotID: "account.json", id: "Account.json", wantFolded: true},
		{name: "different_ids", snapshotID: "account.json", id: "other.json"},
	} {
		for _, foldCase := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fold_case=%t", tc.name, foldCase), func(t *testing.T) {
				want := tc.wantExact
				if foldCase {
					want = tc.wantFolded
				}
				if got := authIDMatches(tc.snapshotID, tc.id, foldCase); got != want {
					t.Fatalf("authIDMatches(%q, %q, %t)=%t, want %t", tc.snapshotID, tc.id, foldCase, got, want)
				}
			})
		}
	}
}

func TestHandleAuthUpdates_FileDeleteOrdering(t *testing.T) {
	for _, deleteBeforeSave := range []bool{true, false} {
		t.Run(fmt.Sprintf("delete_before_save=%t", deleteBeforeSave), func(t *testing.T) {
			ctx := coreauth.WithSkipPersist(context.Background())
			authID := "codex-file-delete-ordering.json"
			path := filepath.Join(t.TempDir(), authID)
			manager := coreauth.NewManager(nil, nil, nil)
			auth := &coreauth.Auth{
				ID:       authID,
				Provider: "codex",
				Status:   coreauth.StatusActive,
				Attributes: map[string]string{
					coreauth.AttributePath: path,
				},
			}
			w, errNewWatcher := watcher.NewWatcher("", filepath.Dir(path), nil)
			if errNewWatcher != nil {
				t.Fatal(errNewWatcher)
			}
			w.SetConfig(&config.Config{AuthDir: filepath.Dir(path)})
			t.Cleanup(func() {
				_ = w.Stop()
			})
			wrapper := &WatcherWrapper{snapshotAuthFile: w.SnapshotAuthFile}
			service := &Service{cfg: &config.Config{AuthDir: filepath.Dir(path)}, coreManager: manager, watcher: wrapper}

			save := func() {
				t.Helper()
				if errWriteFile := os.WriteFile(path, []byte(`{"type":"codex"}`), 0o600); errWriteFile != nil {
					t.Fatalf("write auth file: %v", errWriteFile)
				}
				if _, errRegister := manager.Register(ctx, auth.Clone()); errRegister != nil {
					t.Fatalf("register auth: %v", errRegister)
				}
			}

			observeDelete := func() watcher.AuthUpdate {
				t.Helper()
				if errRemove := os.Remove(path); errRemove != nil {
					t.Fatalf("remove auth file: %v", errRemove)
				}
				deletion := watcher.AuthUpdate{Action: watcher.AuthUpdateActionDelete, ID: authID}
				deletion.SetFileRemoval(true)
				return deletion
			}

			save()
			var deletion watcher.AuthUpdate
			if deleteBeforeSave {
				deletion = observeDelete()
				// Re-create file on disk before delivering deletion
				save()
			} else {
				deletion = observeDelete()
			}

			service.handleAuthUpdate(ctx, deletion)
			_, present := manager.GetByID(authID)
			if present != deleteBeforeSave {
				t.Fatalf("after delete: auth present=%t, want %t", present, deleteBeforeSave)
			}
		})
	}
}

func TestHandleAuthUpdates_FileDeleteScope(t *testing.T) {
	for _, tc := range []struct {
		name        string
		attributes  map[string]string
		unmarked    bool
		wantPresent bool
	}{
		{name: "file_path", attributes: map[string]string{coreauth.AttributePath: "file"}, wantPresent: true},
		{name: "file_source", attributes: map[string]string{coreauth.AttributeSource: "file"}, wantPresent: true},
		{name: "postgres", attributes: map[string]string{coreauth.AttributePath: "file", coreauth.AttributeSourceBackend: coreauth.AuthSourcePostgres}, wantPresent: true},
		{name: "file_unmarked", attributes: map[string]string{coreauth.AttributePath: "file"}, unmarked: true},
		{name: "postgres_no_local_path", attributes: map[string]string{coreauth.AttributeSourceBackend: coreauth.AuthSourcePostgres}},
		{name: "config", attributes: map[string]string{coreauth.AttributePath: "file", coreauth.AttributeSource: "config:codex[0]"}},
	} {
		for _, filePresent := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/file_present=%t", tc.name, filePresent), func(t *testing.T) {
				ctx := coreauth.WithSkipPersist(context.Background())
				authID := "codex-file-delete-scope-" + tc.name
				path := filepath.Join(t.TempDir(), "auth.json")
				if errWriteFile := os.WriteFile(path, []byte(`{"type":"codex"}`), 0o600); errWriteFile != nil {
					t.Fatalf("write auth file: %v", errWriteFile)
				}
				attributes := make(map[string]string, len(tc.attributes))
				for key, value := range tc.attributes {
					if value == "file" {
						value = path
					}
					attributes[key] = value
				}
				manager := coreauth.NewManager(nil, nil, nil)
				auth := &coreauth.Auth{
					ID:         authID,
					Provider:   "codex",
					Status:     coreauth.StatusActive,
					Attributes: attributes,
				}
				if _, errReg := manager.Register(ctx, auth); errReg != nil {
					t.Fatal(errReg)
				}
				if !filePresent {
					_ = os.Remove(path)
				}
				service := &Service{
					cfg:         &config.Config{},
					coreManager: manager,
					watcher: &WatcherWrapper{
						snapshotAuthFile: func(p string) ([]*coreauth.Auth, error) {
							if !filePresent {
								return nil, os.ErrNotExist
							}
							return []*coreauth.Auth{{ID: authID, Provider: "codex"}}, nil
						},
					},
				}
				deletion := watcher.AuthUpdate{Action: watcher.AuthUpdateActionDelete, ID: authID}
				if !tc.unmarked {
					deletion.SetFileRemoval(true)
				}
				service.handleAuthUpdate(ctx, deletion)

				wantPresent := tc.wantPresent && filePresent && !tc.unmarked
				_, present := manager.GetByID(authID)
				if present != wantPresent {
					t.Fatalf("present=%t, want=%t", present, wantPresent)
				}
			})
		}
	}
}
