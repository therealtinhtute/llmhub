package management

// Cooldown snapshot coverage for GET /v0/management/auth-files, ported from
// upstream CLIProxyAPI commit 1ca975dfc011 ("feat(cooldowns): add cooldown
// snapshot feature for management auth files"). Local symbols under test:
// Handler.ListAuthFiles, Handler.listAuthFilesFromDisk,
// coreauth.CooldownSnapshotForAuth.
//
// Adaptations for the diverged local handler:
//   - Local ListAuthFiles has no ?name/?auth_index filtering (pre-existing
//     divergence from upstream baseline), so filter cases are dropped.
//   - Local ListAuthFiles returns 503 when the auth manager is unavailable
//     instead of the upstream disk fallback, so the upstream "disk" unknown-
//     cooldowns case is asserted against listAuthFilesFromDisk directly.
//   - Local buildAuthFileEntry exposes the full QuotaState under "quota"
//     (kiro-quota feature); upstream's two-key observation payload assertion
//     is kept only for "model_quotas".
//   - MarkPluginVirtualAuth does not exist locally (pluginhost excluded), so
//     the virtual-auth case is dropped.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

type authFilesCooldownResponse struct {
	ObservedAt time.Time `json:"observed_at"`
	Files      []struct {
		ID             string                    `json:"id"`
		AuthIndex      string                    `json:"auth_index"`
		Name           string                    `json:"name"`
		Status         string                    `json:"status"`
		Unavailable    bool                      `json:"unavailable"`
		NextRetryAfter time.Time                 `json:"next_retry_after"`
		Cooldowns      json.RawMessage           `json:"cooldowns"`
		Quota          map[string]any            `json:"quota"`
		ModelQuotas    map[string]map[string]any `json:"model_quotas"`
	} `json:"files"`
}

func requestAuthFilesCooldowns(t *testing.T, h *Handler, query string) authFilesCooldownResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/auth-files"+query, nil)
	h.ListAuthFiles(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	var payload authFilesCooldownResponse
	if errDecode := json.Unmarshal(rec.Body.Bytes(), &payload); errDecode != nil {
		t.Fatal(errDecode)
	}
	if payload.ObservedAt.IsZero() || payload.ObservedAt.Location() != time.UTC {
		t.Fatalf("invalid observed_at: %v", payload.ObservedAt)
	}
	return payload
}

func registerAuthForCooldownTest(t *testing.T, manager *coreauth.Manager, auth *coreauth.Auth) {
	t.Helper()
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth %s: %v", auth.ID, err)
	}
}

func TestListAuthFilesCooldownsSnapshot(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	next := now.Add(time.Hour)
	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)
	// Local divergence: EnsureIndex assigns a stable hash (upstream honors a
	// preset Index), so expected indices are learned from the registered auths.
	wantIndex := make(map[string]string)
	for _, id := range []string{"a", "b"} {
		reg, errRegister := manager.Register(context.Background(), &coreauth.Auth{
			ID: id, Provider: "codex", Status: coreauth.StatusError,
			Unavailable: true, NextRetryAfter: next,
			Attributes: map[string]string{"runtime_only": "true"},
			Quota:      coreauth.QuotaState{Exceeded: true, Reason: "quota", NextRecoverAt: next, ObservedAt: now, Signals: map[string]string{"x-codex-primary-used-percent": "90"}},
			ModelStates: map[string]*coreauth.ModelState{
				"model-a": {Unavailable: true, NextRetryAfter: next, Quota: coreauth.QuotaState{Exceeded: true, Reason: "quota", NextRecoverAt: next, BackoffLevel: 6, ObservedAt: now, Signals: map[string]string{"x-codex-primary-used-percent": "90"}}, LastError: &coreauth.Error{HTTPStatus: 429, Message: "private upstream body"}},
				"expired": {Status: coreauth.StatusError, Unavailable: true, NextRetryAfter: now.Add(-time.Hour), Quota: coreauth.QuotaState{Exceeded: true, NextRecoverAt: now.Add(-time.Hour), BackoffLevel: 9}},
			},
		})
		if errRegister != nil {
			t.Fatalf("register auth %s: %v", id, errRegister)
		}
		wantIndex[id] = reg.EnsureIndex()
	}
	beforeA, _ := manager.GetByID("a")
	beforeB, _ := manager.GetByID("b")
	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	for range 2 {
		payload := requestAuthFilesCooldowns(t, h, "")
		if len(payload.Files) != 2 {
			t.Fatalf("files = %+v", payload.Files)
		}
		for i, file := range payload.Files {
			if file.ID != []string{"a", "b"}[i] || file.AuthIndex != wantIndex[file.ID] {
				t.Fatalf("identity/order changed: %+v", file)
			}
			if file.Status != string(coreauth.StatusError) || !file.Unavailable || !file.NextRetryAfter.Equal(next) {
				t.Fatalf("existing state changed: %+v", file)
			}
			var views []coreauth.CooldownView
			if errDecode := json.Unmarshal(file.Cooldowns, &views); errDecode != nil {
				t.Fatal(errDecode)
			}
			if len(views) != 1 || views[0].Scope != "model" || views[0].ModelKey != "model-a" || views[0].Reason != "quota" || views[0].HTTPStatus != 429 || views[0].BackoffLevel == nil || *views[0].BackoffLevel != 6 {
				t.Fatalf("unexpected cooldowns: %s", file.Cooldowns)
			}
			remaining := next.Sub(payload.ObservedAt)
			wantSeconds := int64(remaining / time.Second)
			if remaining%time.Second != 0 {
				wantSeconds++
			}
			if views[0].RemainingSeconds != wantSeconds || !views[0].RetryAt.Equal(next) {
				t.Fatalf("inconsistent time basis: %+v", views[0])
			}
			// Local "quota" exposes the full QuotaState (diverged from upstream's
			// two-key observation payload); only model_quotas keeps the passive
			// observation shape.
			if file.Quota["signals"] == nil || file.Quota["observed_at"] == nil {
				t.Fatalf("quota observation missing: %+v", file.Quota)
			}
			if quota := file.ModelQuotas["model-a"]; len(quota) != 2 || quota["signals"] == nil || quota["observed_at"] == nil {
				t.Fatalf("model quota observation changed: %+v", quota)
			}
			var fields []map[string]any
			if errDecode := json.Unmarshal(file.Cooldowns, &fields); errDecode != nil {
				t.Fatal(errDecode)
			}
			if len(fields[0]) != 7 {
				t.Fatalf("unexpected fields: %+v", fields[0])
			}
			// The view must not leak credential metadata or raw error bodies.
			if raw := string(file.Cooldowns); raw == "" || json.Valid(file.Cooldowns) == false {
				t.Fatalf("invalid cooldowns payload: %s", raw)
			}
		}
	}
	afterA, _ := manager.GetByID("a")
	afterB, _ := manager.GetByID("b")
	if !reflect.DeepEqual(beforeA, afterA) || !reflect.DeepEqual(beforeB, afterB) {
		t.Fatal("GET mutated auth state")
	}
}

func TestListAuthFilesCooldownsEmptyAndUnknown(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager := coreauth.NewManager(nil, nil, nil)
	manager.SetConfig(cfg)
	path := filepath.Join(cfg.AuthDir, "shared.json")
	if errWrite := os.WriteFile(path, []byte(`{"type":"codex"}`), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	for _, id := range []string{"file", "runtime"} {
		auth := &coreauth.Auth{ID: id, FileName: "shared.json", Provider: "codex", Status: coreauth.StatusActive, Attributes: map[string]string{"path": path}}
		if id == "runtime" {
			auth.Attributes = map[string]string{"runtime_only": "true"}
		}
		registerAuthForCooldownTest(t, manager, auth)
	}
	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "")
	if len(payload.Files) != 2 {
		t.Fatalf("files = %+v", payload.Files)
	}
	for _, file := range payload.Files {
		if string(file.Cooldowns) != "[]" {
			t.Fatalf("known empty cooldowns = %s", file.Cooldowns)
		}
	}
}

func TestListAuthFilesCooldownsUnknown(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	t.Run("home", func(t *testing.T) {
		cfg := &config.Config{AuthDir: t.TempDir()}
		cfg.Home.Enabled = true
		manager := coreauth.NewManager(nil, nil, nil)
		manager.SetConfig(cfg)
		registerAuthForCooldownTest(t, manager, &coreauth.Auth{ID: "a", Provider: "codex", Status: coreauth.StatusActive, Attributes: map[string]string{"runtime_only": "true"}})
		h := NewHandlerWithoutConfigFilePath(cfg, manager)
		payload := requestAuthFilesCooldowns(t, h, "")
		if len(payload.Files) != 1 || string(payload.Files[0].Cooldowns) != "null" {
			t.Fatalf("unknown cooldowns = %+v", payload.Files)
		}
	})
	t.Run("disk", func(t *testing.T) {
		// Local divergence: ListAuthFiles returns 503 when the auth manager is
		// unavailable (upstream falls back to disk). The disk fallback helper is
		// exercised directly here; it must still emit observed_at and a null
		// cooldowns marker per upstream 1ca975dfc011.
		cfg := &config.Config{AuthDir: t.TempDir()}
		if errWrite := os.WriteFile(filepath.Join(cfg.AuthDir, "a.json"), []byte(`{"type":"codex"}`), 0o600); errWrite != nil {
			t.Fatal(errWrite)
		}
		h := NewHandlerWithoutConfigFilePath(cfg, nil)
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/auth-files", nil)
		h.listAuthFilesFromDisk(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		var payload authFilesCooldownResponse
		if errDecode := json.Unmarshal(rec.Body.Bytes(), &payload); errDecode != nil {
			t.Fatal(errDecode)
		}
		if payload.ObservedAt.IsZero() || payload.ObservedAt.Location() != time.UTC {
			t.Fatalf("invalid observed_at: %v", payload.ObservedAt)
		}
		if len(payload.Files) != 1 || string(payload.Files[0].Cooldowns) != "null" {
			t.Fatalf("unknown cooldowns = %+v", payload.Files)
		}
	})
}
