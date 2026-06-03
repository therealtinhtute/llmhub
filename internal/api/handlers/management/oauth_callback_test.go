package management

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
)

func TestPostOAuthCallbackUsesSessionStoreWithoutAuthDir(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	state := "codex-test-state"
	RegisterOAuthSession(state, "codex")
	t.Cleanup(func() { CompleteOAuthSession(state) })

	authDir := filepath.Join(t.TempDir(), "synthetic-auths")
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, nil)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(
		http.MethodPost,
		"/v0/management/oauth-callback",
		bytes.NewBufferString(`{"provider":"codex","state":"codex-test-state","code":"callback-code"}`),
	)
	ginCtx.Request.Header.Set("Content-Type", "application/json")

	h.PostOAuthCallback(ginCtx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload, err := WaitOAuthCallbackForPendingSession("codex", state, time.Second)
	if err != nil {
		t.Fatalf("expected callback payload, got error: %v", err)
	}
	if got := payload["code"]; got != "callback-code" {
		t.Fatalf("expected code callback-code, got %q", got)
	}
	if got := payload["state"]; got != state {
		t.Fatalf("expected state %q, got %q", state, got)
	}
}
