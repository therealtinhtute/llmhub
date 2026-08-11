package cliproxy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/nativeproviders"
	"github.com/therealtinhtute/llmhub/internal/watcher"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

type fakeRuntimeStore struct {
	configBytes []byte
	version     int64
	authVersion string
	auths       []*coreauth.Auth
	records     map[string][]nativeproviders.ResourceRecord
}

func (s *fakeRuntimeStore) LoadConfigBytes(context.Context) ([]byte, error) { return s.configBytes, nil }
func (s *fakeRuntimeStore) CurrentVersion(context.Context) (int64, error)   { return s.version, nil }
func (s *fakeRuntimeStore) AuthVersion(context.Context) (string, error)     { return s.authVersion, nil }
func (s *fakeRuntimeStore) List(context.Context) ([]*coreauth.Auth, error)  { return s.auths, nil }

func (s *fakeRuntimeStore) ListNativeProviderResources(_ context.Context, provider string) ([]nativeproviders.ResourceRecord, error) {
	return s.records[provider], nil
}
func (s *fakeRuntimeStore) SaveNativeProviderResource(context.Context, string, string, []byte) error {
	return nil
}
func (s *fakeRuntimeStore) DeleteNativeProviderResource(context.Context, string, string) error {
	return nil
}

func openCodeRecordPayload(id string) []byte {
	payload, _ := json.Marshal(nativeproviders.OpenCodeResource{
		ID:      id,
		Enabled: true,
		APIKey:  "sk-test",
		Models:  []nativeproviders.Model{{Name: "gpt-5", DisplayName: "GPT-5"}},
	})
	return payload
}

func TestStorageWatcherSynthesizesNativeProviderAuths(t *testing.T) {
	store := &fakeRuntimeStore{
		configBytes: []byte("host: 127.0.0.1\nport: 9090\n"),
		version:     1,
		authVersion: "a",
		records:     map[string][]nativeproviders.ResourceRecord{},
	}
	w := &storageWatcher{store: store}
	cfg, err := config.ParseConfigBytes(store.configBytes)
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	w.setConfig(cfg)

	queue := make(chan watcher.AuthUpdate, 16)
	w.setUpdateQueue(queue)

	// First poll establishes the baseline (no dispatch on first poll).
	if err := w.poll(context.Background()); err != nil {
		t.Fatalf("baseline poll() error = %v", err)
	}
	drainUpdates(t, queue)

	// Resource created after startup. AuthVersion stays "a": native provider
	// records live outside the auth versioning, so the synthesized diff must not
	// rely on the version gate to dispatch Add.
	store.records[nativeproviders.ProviderOpenCode] = []nativeproviders.ResourceRecord{
		{Provider: nativeproviders.ProviderOpenCode, ID: "opencode-1", Payload: openCodeRecordPayload("opencode-1")},
	}
	if err := w.poll(context.Background()); err != nil {
		t.Fatalf("poll() error = %v", err)
	}
	got := drainUpdates(t, queue)
	var compatAdd *watcher.AuthUpdate
	for _, u := range got {
		if u.Action == watcher.AuthUpdateActionAdd && u.Auth != nil && u.Auth.Attributes["compat_name"] == "opencode:opencode-1" {
			compatAdd = &u
		}
	}
	if compatAdd == nil {
		t.Fatalf("expected Add update for opencode auth, got %+v", got)
	}
	if compatAdd.Auth.Provider != "opencode:opencode-1" {
		t.Errorf("Provider = %q, want %q", compatAdd.Auth.Provider, "opencode:opencode-1")
	}

	// Another poll with no changes must not re-emit updates for the unchanged
	// synthesized auth (timestamps are zeroed, so it compares equal).
	if err := w.poll(context.Background()); err != nil {
		t.Fatalf("third poll() error = %v", err)
	}
	for _, u := range drainUpdates(t, queue) {
		if u.Auth != nil && u.Auth.Attributes["compat_name"] == "opencode:opencode-1" {
			t.Errorf("synthesized auth re-emitted update on unchanged config: %+v", u)
		}
	}
}

func TestStorageWatcherDispatchesDeleteWhenResourceGone(t *testing.T) {
	store := &fakeRuntimeStore{
		configBytes: []byte("host: 127.0.0.1\nport: 9090\n"),
		version:     1,
		authVersion: "a",
		records: map[string][]nativeproviders.ResourceRecord{
			nativeproviders.ProviderOpenCode: {
				{Provider: nativeproviders.ProviderOpenCode, ID: "opencode-1", Payload: openCodeRecordPayload("opencode-1")},
			},
		},
	}
	w := &storageWatcher{store: store}
	cfg, err := config.ParseConfigBytes(store.configBytes)
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	w.setConfig(cfg)

	queue := make(chan watcher.AuthUpdate, 16)
	w.setUpdateQueue(queue)

	if err := w.poll(context.Background()); err != nil {
		t.Fatalf("baseline poll() error = %v", err)
	}
	drainUpdates(t, queue)

	// Resource removed from the store (AuthVersion unchanged): next poll must
	// dispatch Delete for the synthesized auth.
	store.records[nativeproviders.ProviderOpenCode] = nil
	if err := w.poll(context.Background()); err != nil {
		t.Fatalf("second poll() error = %v", err)
	}
	got := drainUpdates(t, queue)
	for _, u := range got {
		if u.Action == watcher.AuthUpdateActionDelete && u.ID != "" {
			return // synthesized auth removed with its resource
		}
	}
	t.Fatalf("expected Delete for synthesized opencode auth, got %+v", got)
}

func drainUpdates(t *testing.T, queue <-chan watcher.AuthUpdate) []watcher.AuthUpdate {
	t.Helper()
	var out []watcher.AuthUpdate
	for {
		select {
		case u := <-queue:
			out = append(out, u)
		case <-time.After(50 * time.Millisecond):
			return out
		}
	}
}
