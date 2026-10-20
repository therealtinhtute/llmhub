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
			reg := GlobalModelRegistry()
			reg.UnregisterClient(authID)
			t.Cleanup(func() { reg.UnregisterClient(authID) })

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
				reg.RegisterClient(authID, "codex", []*ModelInfo{{ID: "gpt-5", Name: "gpt-5"}})
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
			modelsPresent := len(reg.GetModelsForClient(authID)) > 0
			if present != deleteBeforeSave || modelsPresent != deleteBeforeSave {
				t.Fatalf("after delete: auth present=%t, models present=%t; want both %t", present, modelsPresent, deleteBeforeSave)
			}
		})
	}
}

func TestHandleAuthUpdates_FileDeleteSnapshotUncertain(t *testing.T) {
	for _, tc := range []struct {
		name        string
		watcher     *WatcherWrapper
		wantSurvive bool
	}{
		{name: "unavailable", watcher: &WatcherWrapper{}, wantSurvive: true},
		{name: "empty", watcher: &WatcherWrapper{snapshotAuthFile: func(string) ([]*coreauth.Auth, error) { return nil, nil }}, wantSurvive: true},
		{name: "invalid_ids", watcher: &WatcherWrapper{snapshotAuthFile: func(string) ([]*coreauth.Auth, error) {
			return []*coreauth.Auth{nil, {}}, nil
		}}, wantSurvive: true},
		{name: "error_with_partial_result", watcher: &WatcherWrapper{snapshotAuthFile: func(string) ([]*coreauth.Auth, error) {
			return []*coreauth.Auth{{ID: "other-auth"}}, fmt.Errorf("simulated parse failure")
		}}, wantSurvive: true},
		{name: "file_exists_without_id", watcher: &WatcherWrapper{snapshotAuthFile: func(string) ([]*coreauth.Auth, error) {
			return []*coreauth.Auth{{ID: "other-auth", Provider: "codex"}}, nil
		}}, wantSurvive: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := coreauth.WithSkipPersist(context.Background())
			path := filepath.Join(t.TempDir(), "auth.json")
			if errWriteFile := os.WriteFile(path, []byte(`{"type":"codex"}`), 0o600); errWriteFile != nil {
				t.Fatal(errWriteFile)
			}
			authID := "codex-uncertain-" + tc.name
			reg := GlobalModelRegistry()
			reg.UnregisterClient(authID)
			t.Cleanup(func() { reg.UnregisterClient(authID) })

			manager := coreauth.NewManager(nil, nil, nil)
			auth := &coreauth.Auth{
				ID:         authID,
				Provider:   "codex",
				Status:     coreauth.StatusActive,
				Attributes: map[string]string{coreauth.AttributePath: path},
			}
			if _, errReg := manager.Register(ctx, auth); errReg != nil {
				t.Fatal(errReg)
			}
			reg.RegisterClient(authID, "codex", []*ModelInfo{{ID: "gpt-5", Name: "gpt-5"}})

			service := &Service{
				cfg:         &config.Config{},
				coreManager: manager,
				watcher:     tc.watcher,
			}
			deletion := watcher.AuthUpdate{Action: watcher.AuthUpdateActionDelete, ID: authID}
			deletion.SetFileRemoval(true)
			service.handleAuthUpdate(ctx, deletion)

			_, present := manager.GetByID(authID)
			modelsPresent := len(reg.GetModelsForClient(authID)) > 0
			if present != tc.wantSurvive || modelsPresent != tc.wantSurvive {
				t.Fatalf("after delete: auth present=%t, models present=%t; want both %t", present, modelsPresent, tc.wantSurvive)
			}
		})
	}
}

func TestHandleAuthUpdates_FileDeleteScope(t *testing.T) {
	for _, tc := range []struct {
		name        string
		attributes  map[string]string
		unmarked    bool
		staleAuth   bool
		wantPresent bool
	}{
		{name: "file_path", attributes: map[string]string{coreauth.AttributePath: "file"}, wantPresent: true},
		{name: "file_source", attributes: map[string]string{coreauth.AttributeSource: "file"}, wantPresent: true},
		{name: "git", attributes: map[string]string{coreauth.AttributePath: "file", coreauth.AttributeSourceBackend: coreauth.AuthSourceGit}, wantPresent: true},
		{name: "objectstore", attributes: map[string]string{coreauth.AttributePath: "file", coreauth.AttributeSourceBackend: coreauth.AuthSourceObjectStore}, wantPresent: true},
		{name: "postgres", attributes: map[string]string{coreauth.AttributePath: "file", coreauth.AttributeSourceBackend: coreauth.AuthSourcePostgres}, wantPresent: true},
		{name: "file_unmarked", attributes: map[string]string{coreauth.AttributePath: "file"}, unmarked: true},
		{name: "postgres_no_local_path", attributes: map[string]string{coreauth.AttributeSourceBackend: coreauth.AuthSourcePostgres}},
		{name: "remote_without_local_path", attributes: map[string]string{coreauth.AttributeSource: "file", coreauth.AttributeSourceBackend: coreauth.AuthSourceGit}},
		{name: "runtime_only", attributes: map[string]string{coreauth.AttributePath: "file", coreauth.AttributeRuntimeOnly: "true"}},
		{name: "config", attributes: map[string]string{coreauth.AttributePath: "file", coreauth.AttributeSource: "config:codex[0]"}},
		{name: "stale_auth_delete", attributes: map[string]string{coreauth.AttributePath: "file"}, staleAuth: true, wantPresent: true},
	} {
		for _, filePresent := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/file_present=%t", tc.name, filePresent), func(t *testing.T) {
				ctx := coreauth.WithSkipPersist(context.Background())
				authID := "codex-scope-" + tc.name
				reg := GlobalModelRegistry()
				reg.UnregisterClient(authID)
				t.Cleanup(func() { reg.UnregisterClient(authID) })

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
				registered, errReg := manager.Register(ctx, auth)
				if errReg != nil {
					t.Fatal(errReg)
				}
				reg.RegisterClient(authID, "codex", []*ModelInfo{{ID: "gpt-5", Name: "gpt-5"}})

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
				if tc.staleAuth {
					// Advance generation in manager so existing.Generation > 1
					updated, errUp := manager.Update(ctx, registered)
					if errUp != nil {
						t.Fatal(errUp)
					}
					staleAuthCopy := updated.Clone()
					staleAuthCopy.Generation = 1
					deletion.Auth = staleAuthCopy
				}
				service.handleAuthUpdate(ctx, deletion)

				var wantPresent bool
				if tc.staleAuth {
					wantPresent = true
				} else {
					wantPresent = tc.wantPresent && filePresent && !tc.unmarked
				}

				_, present := manager.GetByID(authID)
				modelsPresent := len(reg.GetModelsForClient(authID)) > 0
				if present != wantPresent || modelsPresent != wantPresent {
					t.Fatalf("present=%t, modelsPresent=%t; want %t", present, modelsPresent, wantPresent)
				}
			})
		}
	}
}

func TestApplyCoreAuthAddOrUpdate_StaleUpdateFallsThroughToCompleteModelRegistration(t *testing.T) {
	ctx := coreauth.WithSkipPersist(context.Background())
	authID := "stale-update-falls-through.json"
	path := filepath.Join(t.TempDir(), authID)
	if errWrite := os.WriteFile(path, []byte(`{"type":"codex","prefix":"pfx-latest"}`), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}

	reg := GlobalModelRegistry()
	reg.UnregisterClient(authID)
	t.Cleanup(func() { reg.UnregisterClient(authID) })

	manager := coreauth.NewManager(nil, nil, nil)
	initial := &coreauth.Auth{
		ID:         authID,
		Provider:   "codex",
		Prefix:     "pfx-latest",
		Status:     coreauth.StatusActive,
		Attributes: map[string]string{coreauth.AttributePath: path},
	}
	initial.BindFileSnapshot(path, []byte(`{"type":"codex","prefix":"pfx-latest"}`))
	registered, errReg := manager.Register(ctx, initial)
	if errReg != nil {
		t.Fatal(errReg)
	}

	service := &Service{
		cfg:         &config.Config{},
		coreManager: manager,
	}

	// Create a stale incoming snapshot referencing old pfx-stale
	staleIncoming := &coreauth.Auth{
		ID:         authID,
		Provider:   "codex",
		Prefix:     "pfx-stale",
		Status:     coreauth.StatusActive,
		Attributes: map[string]string{coreauth.AttributePath: path},
	}
	staleIncoming.BindFileSnapshot(path, []byte(`{"type":"codex","prefix":"pfx-stale"}`))

	// applyCoreAuthAddOrUpdate with stale snapshot: manager will reject mutation (gen unchanged),
	// but service falls through with auth = updated to complete model registration
	service.applyCoreAuthAddOrUpdate(ctx, staleIncoming)

	current, ok := manager.GetByID(authID)
	if !ok || current == nil {
		t.Fatal("auth must remain registered")
	}
	if current.Prefix != "pfx-latest" || current.Generation != registered.Generation {
		t.Fatalf("stale snapshot mutated live manager state: got prefix=%s gen=%d, want prefix=pfx-latest gen=%d", current.Prefix, current.Generation, registered.Generation)
	}
}
