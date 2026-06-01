package management

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestListAuthFiles_IncludesPathlessPostgresAuth(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &pathlessMemoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:       "postgres-user.json",
		FileName: "postgres-user.json",
		Provider: "codex",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"source": "postgres",
		},
		Metadata: map[string]any{
			"type":  "codex",
			"email": "user@example.com",
		},
	}); err != nil {
		t.Fatalf("failed to register auth: %v", err)
	}

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
	h.tokenStore = store

	entry := firstAuthFileEntry(t, h)
	if got := entry["name"]; got != "postgres-user.json" {
		t.Fatalf("expected postgres auth file name, got %#v", got)
	}
	if got := entry["source"]; got != "postgres" {
		t.Fatalf("expected postgres source, got %#v", got)
	}
	if _, ok := entry["path"]; ok {
		t.Fatalf("expected pathless postgres entry, got path %#v", entry["path"])
	}
	if got := entry["email"]; got != "user@example.com" {
		t.Fatalf("expected email from metadata, got %#v", got)
	}
}

func TestUploadAuthFile_PathlessStorePersistsAndLists(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &pathlessMemoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
	h.tokenStore = store

	fileName := "uploaded-postgres-user.json"
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(
		http.MethodPost,
		"/v0/management/auth-files?name="+url.QueryEscape(fileName),
		bytes.NewBufferString(`{"type":"codex","email":"uploaded@example.com"}`),
	)
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	h.UploadAuthFile(ginCtx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected upload status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if _, ok := store.items[fileName]; !ok {
		t.Fatalf("expected uploaded auth to be saved in pathless store")
	}

	entry := firstAuthFileEntry(t, h)
	if got := entry["name"]; got != fileName {
		t.Fatalf("expected uploaded file name, got %#v", got)
	}
	if got := entry["source"]; got != "postgres" {
		t.Fatalf("expected postgres source, got %#v", got)
	}
	if got := entry["email"]; got != "uploaded@example.com" {
		t.Fatalf("expected uploaded email, got %#v", got)
	}
}
