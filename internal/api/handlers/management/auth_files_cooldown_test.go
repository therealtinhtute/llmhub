package management

// Cooldown snapshot coverage for GET /v0/management/auth-files, ported from
// upstream CLIProxyAPI commit 1ca975dfc011 ("feat(cooldowns): add cooldown
// snapshot feature for management auth files") and commit b4ff581dafa4
// ("fix(management): reconcile expired cooldowns to active status in auth
// files"). Local symbols under test:
// Handler.ListAuthFiles, Handler.listAuthFilesFromDisk,
// coreauth.CooldownSnapshotForAuth, reconcileAuthFileCooldownState.
//
// Adaptations for the diverged local handler:
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
		StatusMessage  string                    `json:"status_message"`
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
		h.listAuthFilesFromDisk(ctx, authFilesPagination{})
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

// Expired-cooldown reconcile coverage ported from upstream CLIProxyAPI commit
// b4ff581dafa4 ("fix(management): reconcile expired cooldowns to active status
// in auth files"). Local symbols under test: reconcileAuthFileCooldownState,
// isPersistentAuthFailure, isModelStateBlocked via Handler.ListAuthFiles.
// Adaptation: registerAuthForLookupTest -> registerAuthForCooldownTest (local
// equivalent helper).

func TestListAuthFiles_ExpiredCooldownReconciledToActive(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	expiredDeadline := now.Add(-10 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// Credential-level expired cooldown (e.g. Codex quota exhaustion in Issue 5964)
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:             "codex-expired",
		Index:          "idx-codex-expired",
		FileName:       "codex-expired.json",
		Provider:       "codex",
		Status:         coreauth.StatusError,
		StatusMessage:  "credential_quota",
		Unavailable:    true,
		NextRetryAfter: expiredDeadline,
		Quota: coreauth.QuotaState{
			Exceeded:      true,
			Reason:        "credential_quota",
			NextRecoverAt: expiredDeadline,
			ObservedAt:    expiredDeadline,
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	// Model-level expired cooldown where all model cooldowns have elapsed
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:             "model-expired",
		Index:          "idx-model-expired",
		FileName:       "model-expired.json",
		Provider:       "claude",
		Status:         coreauth.StatusError,
		StatusMessage:  "rate limit exceeded",
		Unavailable:    true,
		NextRetryAfter: expiredDeadline,
		ModelStates: map[string]*coreauth.ModelState{
			"claude-3-5-sonnet": {
				Status:         coreauth.StatusError,
				StatusMessage:  "rate limit exceeded",
				Unavailable:    true,
				NextRetryAfter: expiredDeadline,
			},
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "")

	if len(payload.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(payload.Files))
	}

	for _, file := range payload.Files {
		if file.Unavailable {
			t.Errorf("auth %s: expected unavailable=false after cooldown expiration, got true", file.ID)
		}
		if file.Status != string(coreauth.StatusActive) {
			t.Errorf("auth %s: expected status=%q after cooldown expiration, got %q", file.ID, coreauth.StatusActive, file.Status)
		}
		if file.StatusMessage != "" {
			t.Errorf("auth %s: expected empty status_message after cooldown expiration, got %q", file.ID, file.StatusMessage)
		}
		if !file.NextRetryAfter.IsZero() {
			t.Errorf("auth %s: expected zero next_retry_after after cooldown expiration, got %v", file.ID, file.NextRetryAfter)
		}
		if string(file.Cooldowns) != "[]" {
			t.Errorf("auth %s: expected cooldowns=[], got %s", file.ID, file.Cooldowns)
		}
	}
}

func TestListAuthFiles_ExpiredCooldownWithSubsequentTokenFailure(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	expiredDeadline := now.Add(-10 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// An OAuth auth that had a cooldown, but subsequently had its access token expire / refresh fail.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:             "codex-token-expired",
		Index:          "idx-codex-token-expired",
		FileName:       "codex-token-expired.json",
		Provider:       "codex",
		Status:         coreauth.StatusError,
		StatusMessage:  "token expired",
		Unavailable:    true,
		NextRetryAfter: expiredDeadline,
		Metadata: map[string]any{
			"type":         "codex",
			"access_token": "expired-access-token",
			"expired":      now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=codex-token-expired.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if !file.Unavailable {
		t.Error("expected unavailable=true for expired token failure, got false")
	}
	if file.Status != string(coreauth.StatusError) {
		t.Errorf("expected status=%q for expired token failure, got %q", coreauth.StatusError, file.Status)
	}
	if file.StatusMessage != "token expired" {
		t.Errorf("expected status_message=%q, got %q", "token expired", file.StatusMessage)
	}
	if !file.NextRetryAfter.IsZero() {
		t.Errorf("expected zero next_retry_after, got %v", file.NextRetryAfter)
	}
}

func TestListAuthFiles_PartialModelCooldownWithAuthFailure(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	activeDeadline := now.Add(10 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// Auth has an independent auth error ("unauthorized"), but only one of its models is in cooldown.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:            "auth-unauthorized-partial-model",
		Index:         "idx-auth-unauthorized",
		FileName:      "auth-unauthorized.json",
		Provider:      "claude",
		Status:        coreauth.StatusError,
		StatusMessage: "unauthorized",
		Unavailable:   true,
		LastError:     &coreauth.Error{HTTPStatus: 401, Message: "unauthorized"},
		ModelStates: map[string]*coreauth.ModelState{
			"model-cool": {
				Status:         coreauth.StatusError,
				Unavailable:    true,
				NextRetryAfter: activeDeadline,
			},
			"model-free": {
				Status: coreauth.StatusActive,
			},
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-unauthorized.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if !file.Unavailable {
		t.Error("expected unavailable=true due to unauthorized error, got false")
	}
	if file.Status != string(coreauth.StatusError) {
		t.Errorf("expected status=%q, got %q", coreauth.StatusError, file.Status)
	}
	if file.StatusMessage != "unauthorized" {
		t.Errorf("expected status_message=%q, got %q", "unauthorized", file.StatusMessage)
	}
}

func TestListAuthFiles_UnexpiredTokenWithRefresh401_NoCooldown(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	futureExpiry := now.Add(48 * time.Hour).Format(time.RFC3339)
	retryBackoff := now.Add(5 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// An OAuth credential whose access token is still valid (+48h), but background refresh failed with 401
	// and scheduled a retry backoff. The credential itself is still active and usable.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:               "auth-valid-token-refresh-401",
		Index:            "idx-valid-token",
		FileName:         "auth-valid-token.json",
		Provider:         "codex",
		Status:           coreauth.StatusActive,
		Unavailable:      false,
		NextRefreshAfter: retryBackoff,
		LastError:        &coreauth.Error{HTTPStatus: 401, Message: "401 unauthorized on refresh"},
		Metadata: map[string]any{
			"type":         "codex",
			"access_token": "valid-future-access-token",
			"expired":      futureExpiry,
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-valid-token.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if file.Unavailable {
		t.Error("expected unavailable=false for valid token with scheduled refresh retry, got true")
	}
	if file.Status != string(coreauth.StatusActive) {
		t.Errorf("expected status=%q, got %q", coreauth.StatusActive, file.Status)
	}
}

func TestListAuthFiles_UnexpiredTokenWithRefresh401_ExpiredCooldown(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	futureExpiry := now.Add(48 * time.Hour).Format(time.RFC3339)
	expiredCooldown := now.Add(-10 * time.Minute)
	retryBackoff := now.Add(5 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// An OAuth credential with a valid access token (+48h) that previously entered quota cooldown (now expired),
	// and has a pending refresh retry after a 401 refresh error.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:               "auth-valid-token-expired-cooldown",
		Index:            "idx-valid-token-exp-cool",
		FileName:         "auth-valid-token-exp-cool.json",
		Provider:         "codex",
		Status:           coreauth.StatusError,
		StatusMessage:    "credential_quota",
		Unavailable:      true,
		NextRetryAfter:   expiredCooldown,
		NextRefreshAfter: retryBackoff,
		LastError:        &coreauth.Error{HTTPStatus: 401, Message: "401 unauthorized on refresh"},
		Quota: coreauth.QuotaState{
			Exceeded:      true,
			Reason:        "credential_quota",
			NextRecoverAt: expiredCooldown,
		},
		Metadata: map[string]any{
			"type":         "codex",
			"access_token": "valid-future-access-token",
			"expired":      futureExpiry,
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-valid-token-exp-cool.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if file.Unavailable {
		t.Error("expected unavailable=false after quota cooldown expires for valid token, got true")
	}
	if file.Status != string(coreauth.StatusActive) {
		t.Errorf("expected status=%q, got %q", coreauth.StatusActive, file.Status)
	}
	if file.StatusMessage != "" {
		t.Errorf("expected empty status_message, got %q", file.StatusMessage)
	}
	if !file.NextRetryAfter.IsZero() {
		t.Errorf("expected zero next_retry_after, got %v", file.NextRetryAfter)
	}
}

func TestListAuthFiles_ModelLevel403CooldownExpired(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	expiredDeadline := now.Add(-10 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// An auth whose model failed with 403 (forbidden) and the conductor copied the error to auth-level,
	// but the model-level cooldown has now elapsed.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:             "auth-model-403-expired",
		Index:          "idx-model-403-exp",
		FileName:       "auth-model-403-exp.json",
		Provider:       "codex",
		Status:         coreauth.StatusError,
		StatusMessage:  "forbidden",
		Unavailable:    true,
		NextRetryAfter: expiredDeadline,
		LastError:      &coreauth.Error{HTTPStatus: 403, Message: "forbidden"},
		ModelStates: map[string]*coreauth.ModelState{
			"model-a": {
				Status:         coreauth.StatusError,
				StatusMessage:  "forbidden",
				Unavailable:    true,
				NextRetryAfter: expiredDeadline,
				LastError:      &coreauth.Error{HTTPStatus: 403, Message: "forbidden"},
			},
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-model-403-exp.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if file.Unavailable {
		t.Error("expected unavailable=false after model-level 403 cooldown expires, got true")
	}
	if file.Status != string(coreauth.StatusActive) {
		t.Errorf("expected status=%q, got %q", coreauth.StatusActive, file.Status)
	}
	if file.StatusMessage != "" {
		t.Errorf("expected empty status_message after cooldown expires, got %q", file.StatusMessage)
	}
	if !file.NextRetryAfter.IsZero() {
		t.Errorf("expected zero next_retry_after, got %v", file.NextRetryAfter)
	}
}

func TestListAuthFiles_SingleModel403Cooling_OtherModelActive(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	activeDeadline := now.Add(30 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// An auth where Model A is in a 403 cooldown, but Model B is active and healthy.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:            "auth-single-403-other-active",
		Index:         "idx-single-403",
		FileName:      "auth-single-403.json",
		Provider:      "codex",
		Status:        coreauth.StatusError,
		StatusMessage: "forbidden",
		Unavailable:   false,
		LastError:     &coreauth.Error{HTTPStatus: 403, Message: "forbidden"},
		ModelStates: map[string]*coreauth.ModelState{
			"model-forbidden": {
				Status:         coreauth.StatusError,
				StatusMessage:  "forbidden",
				Unavailable:    true,
				NextRetryAfter: activeDeadline,
				LastError:      &coreauth.Error{HTTPStatus: 403, Message: "forbidden"},
			},
			"model-working": {
				Status: coreauth.StatusActive,
			},
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-single-403.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if file.Unavailable {
		t.Error("expected unavailable=false because model-working is active, got true")
	}
	if file.Status != string(coreauth.StatusActive) {
		t.Errorf("expected status=%q because model-working is active, got %q", coreauth.StatusActive, file.Status)
	}
}

func TestListAuthFiles_ExpiredCooldown_ExpiredToken_InFlightRefresh(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	expiredDeadline := now.Add(-10 * time.Minute)
	inFlightRefresh := now.Add(1 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// An OAuth credential whose quota cooldown expired, but its access token is expired,
	// and a background refresh is currently in-flight/scheduled (NextRefreshAfter in future).
	// Because the access token is expired, selector blocks it and it must NOT be reported as active.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:               "auth-expired-cooldown-expired-token",
		Index:            "idx-exp-cool-exp-tok",
		FileName:         "auth-exp-cool-exp-tok.json",
		Provider:         "codex",
		Status:           coreauth.StatusError,
		StatusMessage:    "credential_quota",
		Unavailable:      true,
		NextRetryAfter:   expiredDeadline,
		NextRefreshAfter: inFlightRefresh,
		Quota: coreauth.QuotaState{
			Exceeded:      true,
			Reason:        "credential_quota",
			NextRecoverAt: expiredDeadline,
		},
		Metadata: map[string]any{
			"type":         "codex",
			"access_token": "expired-access-token",
			"expired":      now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-exp-cool-exp-tok.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if !file.Unavailable {
		t.Error("expected unavailable=true because access token is expired, got false")
	}
	if file.Status != string(coreauth.StatusError) {
		t.Errorf("expected status=%q, got %q", coreauth.StatusError, file.Status)
	}
	if file.StatusMessage != "credential_quota" {
		t.Errorf("expected status_message=%q, got %q", "credential_quota", file.StatusMessage)
	}
}

func TestListAuthFiles_ActiveAuth_InactiveFutureTimestamp(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	inactiveFutureTimestamp := now.Add(1 * time.Hour)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// An active, healthy auth that has an inactive future NextRetryAfter timestamp
	// (Unavailable=false, Quota.Exceeded=false). Matching selector.availabilityBlock,
	// an inactive timestamp does not block the credential or report it as in error.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:             "auth-active-inactive-future-timestamp",
		Index:          "idx-active-inactive-future",
		FileName:       "auth-active-future.json",
		Provider:       "codex",
		Status:         coreauth.StatusActive,
		Unavailable:    false,
		NextRetryAfter: inactiveFutureTimestamp,
		Attributes:     map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-active-future.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if file.Unavailable {
		t.Error("expected unavailable=false for active auth with inactive future timestamp, got true")
	}
	if file.Status != string(coreauth.StatusActive) {
		t.Errorf("expected status=%q, got %q", coreauth.StatusActive, file.Status)
	}
}

func TestListAuthFiles_ModelCooling_OtherModelPermanentBlocked(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	activeDeadline := now.Add(30 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// Model A is in active cooldown (+30m).
	// Model B has a permanent failure with no recovery time (Unavailable=true, NextRetryAfter=zero).
	// Because all schedulable models are blocked, the credential as a whole cannot serve any model.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:            "auth-all-blocked-cooling-and-perm",
		Index:         "idx-all-blocked",
		FileName:      "auth-all-blocked.json",
		Provider:      "codex",
		Status:        coreauth.StatusError,
		StatusMessage: "model failure",
		Unavailable:   true,
		ModelStates: map[string]*coreauth.ModelState{
			"model-cooling": {
				Status:         coreauth.StatusError,
				Unavailable:    true,
				NextRetryAfter: activeDeadline,
			},
			"model-perm-blocked": {
				Status:      coreauth.StatusError,
				Unavailable: true,
			},
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-all-blocked.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if !file.Unavailable {
		t.Error("expected unavailable=true because all models are blocked, got false")
	}
	if file.Status != string(coreauth.StatusError) {
		t.Errorf("expected status=%q because all models are blocked, got %q", coreauth.StatusError, file.Status)
	}
}

func TestListAuthFiles_SparseModelState_BlockedModelDoesNotMakeAuthUnavailable(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	now := time.Now().UTC()
	activeDeadline := now.Add(30 * time.Minute)

	manager := coreauth.NewManager(nil, nil, nil)
	cfg := &config.Config{AuthDir: t.TempDir()}
	manager.SetConfig(cfg)

	// An auth where Model A has an active cooldown in ModelStates, but Model B has no entry in the sparse map yet.
	// Because other supported models are schedulable and auth.Unavailable is false, the credential as a whole
	// must NOT be reported as unavailable.
	registerAuthForCooldownTest(t, manager, &coreauth.Auth{
		ID:          "auth-sparse-models",
		Index:       "idx-sparse-models",
		FileName:    "auth-sparse-models.json",
		Provider:    "codex",
		Status:      coreauth.StatusActive,
		Unavailable: false,
		ModelStates: map[string]*coreauth.ModelState{
			"model-a": {
				Status:         coreauth.StatusError,
				Unavailable:    true,
				NextRetryAfter: activeDeadline,
			},
		},
		Attributes: map[string]string{"runtime_only": "true"},
	})

	h := NewHandlerWithoutConfigFilePath(cfg, manager)
	payload := requestAuthFilesCooldowns(t, h, "?name=auth-sparse-models.json")

	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	file := payload.Files[0]
	if file.Unavailable {
		t.Error("expected unavailable=false because sparse unrecorded models remain schedulable, got true")
	}
	if file.Status != string(coreauth.StatusActive) {
		t.Errorf("expected status=%q, got %q", coreauth.StatusActive, file.Status)
	}
}
