package management

// Tests for the auth-file refresh endpoint ported from upstream CLIProxyAPI
// commit 60e5b8bd432e ("feat(management): add endpoint to refresh auth files").
// Local symbols under test: Handler.RefreshAuthFiles, Handler.lookupAuthFile,
// coreauth.Manager.ForceRefreshAll, coreauth.Manager.ForceRefreshAuth.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type refreshRecordExecutor struct {
	provider   string
	refreshCnt atomic.Int32
}

func (e *refreshRecordExecutor) Identifier() string {
	return e.provider
}

func (e *refreshRecordExecutor) Refresh(ctx context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	e.refreshCnt.Add(1)
	if auth.Metadata == nil {
		auth.Metadata = make(map[string]any)
	}
	auth.Metadata["access_token"] = "refreshed-token"
	auth.Metadata["refresh_token"] = "refresh-token"
	auth.Metadata["expires_in"] = int64(3600)
	auth.Metadata["expired"] = time.Now().Add(time.Hour).Format(time.RFC3339)
	return auth, nil
}

func (e *refreshRecordExecutor) Execute(ctx context.Context, auth *coreauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e *refreshRecordExecutor) ExecuteStream(ctx context.Context, auth *coreauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return nil, nil
}

func (e *refreshRecordExecutor) CountTokens(ctx context.Context, auth *coreauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e *refreshRecordExecutor) HttpRequest(ctx context.Context, auth *coreauth.Auth, req *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestRefreshAuthFiles_AllAndSpecific(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authDir := t.TempDir()

	fileA := filepath.Join(authDir, "antigravity-1.json")
	fileB := filepath.Join(authDir, "antigravity-2.json")
	_ = os.WriteFile(fileA, []byte(`{"type":"antigravity","refresh_token":"ref-1","access_token":"old-1"}`), 0o600)
	_ = os.WriteFile(fileB, []byte(`{"type":"antigravity","refresh_token":"ref-2","access_token":"old-2"}`), 0o600)

	manager := coreauth.NewManager(nil, nil, nil)
	exec := &refreshRecordExecutor{provider: "antigravity"}
	manager.RegisterExecutor(exec)

	auth1 := &coreauth.Auth{
		ID:       "antigravity-1.json",
		Provider: "antigravity",
		Status:   coreauth.StatusActive,
		Metadata: map[string]any{"type": "antigravity", "refresh_token": "ref-1", "access_token": "old-1"},
	}
	auth2 := &coreauth.Auth{
		ID:          "antigravity-2.json",
		Provider:    "antigravity",
		Status:      coreauth.StatusError,
		Unavailable: true,
		LastError:   &coreauth.Error{Message: "unauthorized"},
		Metadata:    map[string]any{"type": "antigravity", "refresh_token": "ref-2", "access_token": "old-2"},
	}
	_, _ = manager.Register(context.Background(), auth1)
	_, _ = manager.Register(context.Background(), auth2)

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, manager)

	engine := gin.New()
	engine.POST("/auth-files/refresh", h.RefreshAuthFiles)

	// 1. Refresh all
	req := httptest.NewRequest(http.MethodPost, "/auth-files/refresh?all=true", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, got %v", resp)
	}

	// Wait briefly for refresh to execute
	time.Sleep(50 * time.Millisecond)

	if cnt := exec.refreshCnt.Load(); cnt < 2 {
		t.Fatalf("expected at least 2 refreshes, got %d", cnt)
	}

	// 2. Auth2 was in StatusError, now should be active/recovering
	a2, exists := manager.GetByID("antigravity-2.json")
	if !exists || a2.Status == coreauth.StatusError {
		t.Fatalf("expected auth2 status to be recovered from error, got %+v", a2)
	}

	// 3. Refresh single file by name
	prevCnt := exec.refreshCnt.Load()
	reqSingle := httptest.NewRequest(http.MethodPost, "/auth-files/refresh?name=antigravity-1.json", nil)
	wSingle := httptest.NewRecorder()
	engine.ServeHTTP(wSingle, reqSingle)

	if wSingle.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", wSingle.Code, wSingle.Body.String())
	}
	if newCnt := exec.refreshCnt.Load(); newCnt != prevCnt+1 {
		t.Fatalf("expected cnt to increment by 1, was %d now %d", prevCnt, newCnt)
	}

	// 4. Refresh nonexistent file
	reqMissing := httptest.NewRequest(http.MethodPost, "/auth-files/refresh?name=nonexistent.json", nil)
	wMissing := httptest.NewRecorder()
	engine.ServeHTTP(wMissing, reqMissing)

	if wMissing.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", wMissing.Code, wMissing.Body.String())
	}

	// 5. Refresh via chunked JSON request body (ContentLength = -1)
	chunkedBody := strings.NewReader(`{"name":"antigravity-1.json"}`)
	reqChunked := httptest.NewRequest(http.MethodPost, "/auth-files/refresh", chunkedBody)
	reqChunked.Header.Set("Content-Type", "application/json")
	reqChunked.TransferEncoding = []string{"chunked"}
	reqChunked.ContentLength = -1
	wChunked := httptest.NewRecorder()
	engine.ServeHTTP(wChunked, reqChunked)

	if wChunked.Code != http.StatusOK {
		t.Fatalf("expected chunked request status 200, got %d: %s", wChunked.Code, wChunked.Body.String())
	}

	// 6. Malformed JSON request body returns 400
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/auth-files/refresh", strings.NewReader(`{invalid`))
	reqBadJSON.Header.Set("Content-Type", "application/json")
	wBadJSON := httptest.NewRecorder()
	engine.ServeHTTP(wBadJSON, reqBadJSON)

	if wBadJSON.Code != http.StatusBadRequest {
		t.Fatalf("expected malformed JSON to return 400, got %d", wBadJSON.Code)
	}
}

// TestRefreshAuthFiles_AuthIndexLookup covers the name+auth_index branch of
// Handler.lookupAuthFile added with upstream 60e5b8bd432e: both fields must
// match the same auth, and a mismatched pair must not resolve.
func TestRefreshAuthFiles_AuthIndexLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := coreauth.NewManager(nil, nil, nil)
	exec := &refreshRecordExecutor{provider: "codex"}
	manager.RegisterExecutor(exec)

	// Distinct file names give distinct stable auth indices (local indexSeed
	// derives from file path).
	authA := &coreauth.Auth{
		ID:       "codex-a.json",
		FileName: "codex-a.json",
		Provider: "codex",
		Status:   coreauth.StatusActive,
		Metadata: map[string]any{"type": "codex", "refresh_token": "ref-a"},
	}
	authB := &coreauth.Auth{
		ID:       "codex-b.json",
		FileName: "codex-b.json",
		Provider: "codex",
		Status:   coreauth.StatusActive,
		Metadata: map[string]any{"type": "codex", "refresh_token": "ref-b"},
	}
	regA, errRegister := manager.Register(context.Background(), authA)
	if errRegister != nil {
		t.Fatalf("register authA: %v", errRegister)
	}
	if _, errRegister = manager.Register(context.Background(), authB); errRegister != nil {
		t.Fatalf("register authB: %v", errRegister)
	}
	indexA := strings.TrimSpace(regA.EnsureIndex())

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
	engine := gin.New()
	engine.POST("/auth-files/refresh", h.RefreshAuthFiles)

	// Both name and auth_index match authA: resolves and refreshes exactly once.
	prevCnt := exec.refreshCnt.Load()
	req := httptest.NewRequest(http.MethodPost, "/auth-files/refresh?name=codex-a.json&auth_index="+indexA, nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		OK   bool           `json:"ok"`
		Auth *coreauth.Auth `json:"auth"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.OK || resp.Auth == nil || resp.Auth.ID != "codex-a.json" {
		t.Fatalf("expected refreshed auth codex-a.json, got %+v", resp)
	}
	if newCnt := exec.refreshCnt.Load(); newCnt != prevCnt+1 {
		t.Fatalf("expected one refresh, was %d now %d", prevCnt, newCnt)
	}

	// Mismatched name/auth_index pair must not resolve (404), matching upstream
	// matchesAuthFileLookup semantics.
	reqMiss := httptest.NewRequest(http.MethodPost, "/auth-files/refresh?name=other.json&auth_index="+indexA, nil)
	wMiss := httptest.NewRecorder()
	engine.ServeHTTP(wMiss, reqMiss)
	if wMiss.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for mismatched pair, got %d: %s", wMiss.Code, wMiss.Body.String())
	}

	// Missing name and no all=true is a 400.
	reqBad := httptest.NewRequest(http.MethodPost, "/auth-files/refresh", nil)
	wBad := httptest.NewRecorder()
	engine.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 without name/all, got %d: %s", wBad.Code, wBad.Body.String())
	}
}
