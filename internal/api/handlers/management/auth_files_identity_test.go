package management

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestPatchAuthFileFieldsTargetsStableIDWhenNamesCollide(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &pathlessMemoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)
	registerAuthFileIdentityFixture(t, manager, "claude/auth.json", "auth.json", "claude")
	registerAuthFileIdentityFixture(t, manager, "codex/auth.json", "auth.json", "codex")
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
	h.tokenStore = store

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"id":"codex/auth.json","name":"auth.json","note":"codex-only"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PatchAuthFileFields(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	claudeAuth, _ := manager.GetByID("claude/auth.json")
	codexAuth, _ := manager.GetByID("codex/auth.json")
	if got, _ := claudeAuth.Metadata["note"].(string); got != "" {
		t.Fatalf("claude note = %q, want untouched", got)
	}
	if got, _ := codexAuth.Metadata["note"].(string); got != "codex-only" {
		t.Fatalf("codex note = %q, want codex-only", got)
	}
	if _, ok := codexAuth.Metadata["id"]; ok {
		t.Fatalf("selector id leaked into metadata: %#v", codexAuth.Metadata)
	}
}

func TestPatchAuthFileStatusTargetsAuthIndexWhenNamesCollide(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &pathlessMemoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)
	claudeAuth := registerAuthFileIdentityFixture(t, manager, "claude/auth.json", "auth.json", "claude")
	codexAuth := registerAuthFileIdentityFixture(t, manager, "codex/auth.json", "auth.json", "codex")
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
	h.tokenStore = store

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/status", strings.NewReader(`{"name":"auth.json","auth_index":"`+codexAuth.Index+`","disabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PatchAuthFileStatus(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	updatedClaude, _ := manager.GetByID(claudeAuth.ID)
	updatedCodex, _ := manager.GetByID(codexAuth.ID)
	if updatedClaude.Disabled {
		t.Fatalf("claude auth was disabled by colliding name")
	}
	if !updatedCodex.Disabled || updatedCodex.Status != coreauth.StatusDisabled {
		t.Fatalf("codex auth status = disabled %v status %q, want disabled", updatedCodex.Disabled, updatedCodex.Status)
	}
}

func TestDeleteAuthFileTargetsStableIDWhenNamesCollide(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &pathlessMemoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)
	registerAuthFileIdentityFixture(t, manager, "claude-auth-id", "auth.json", "claude")
	registerAuthFileIdentityFixture(t, manager, "codex-auth-id", "auth.json", "codex")
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
	h.tokenStore = store

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodDelete, "/v0/management/auth-files?name=codex-auth-id", nil)
	ctx.Request = req

	h.DeleteAuthFile(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if _, ok := manager.GetByID("claude-auth-id"); !ok {
		t.Fatalf("claude auth was deleted by colliding name")
	}
	if _, ok := manager.GetByID("codex-auth-id"); ok {
		t.Fatalf("codex auth still exists after targeted delete")
	}
}

func registerAuthFileIdentityFixture(t *testing.T, manager *coreauth.Manager, id, name, provider string) *coreauth.Auth {
	t.Helper()
	auth, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:       id,
		FileName: name,
		Provider: provider,
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"source": "postgres",
		},
		Metadata: map[string]any{
			"type": provider,
		},
	})
	if err != nil {
		t.Fatalf("failed to register %s auth: %v", provider, err)
	}
	return auth
}
