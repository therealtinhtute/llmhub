package codexlive

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/client/codex/live"
	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type liveSettingsStore struct {
	settings runtimecontrol.Settings
}

func (s *liveSettingsStore) LoadRuntimeSettings(context.Context) (runtimecontrol.Settings, error) {
	return s.settings, nil
}

func (s *liveSettingsStore) SaveRuntimeSettings(context.Context, int64, runtimecontrol.Settings) (runtimecontrol.Settings, error) {
	return runtimecontrol.Settings{}, nil
}

type liveHTTPExecutor struct {
	requestBody    string
	authorization  string
	protocolHeader string
}

func (e *liveHTTPExecutor) Identifier() string { return "codex" }
func (e *liveHTTPExecutor) Execute(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}
func (e *liveHTTPExecutor) ExecuteStream(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return nil, nil
}
func (e *liveHTTPExecutor) Refresh(context.Context, *coreauth.Auth) (*coreauth.Auth, error) {
	return nil, nil
}
func (e *liveHTTPExecutor) CountTokens(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}
func (e *liveHTTPExecutor) PrepareRequest(req *http.Request, auth *coreauth.Auth) error {
	req.Header.Set("Authorization", "Bearer prepared")
	return nil
}
func (e *liveHTTPExecutor) HttpRequest(_ context.Context, _ *coreauth.Auth, req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	e.requestBody = string(body)
	e.authorization = req.Header.Get("Authorization")
	e.protocolHeader = req.Header.Get("OpenAI-Alpha")
	return &http.Response{
		StatusCode: http.StatusCreated,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"Location":     []string{"/backend-api/codex/realtime/calls/call-123"},
		},
		Body: io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}, nil
}

func TestCreateCallDisabledByRuntimeSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := coreauth.NewManager(nil, nil, nil)
	handler := New(manager, &liveSettingsStore{settings: runtimecontrol.DefaultSettings()})

	router := gin.New()
	router.POST("/backend-api/codex/realtime/calls", handler.CreateCall)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/backend-api/codex/realtime/calls", strings.NewReader(`{"session":{"model":"gpt-live-1-codex"}}`))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestCreateCallForwardsPreparedRequestAndStoresSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	executor := &liveHTTPExecutor{}
	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(executor)
	_, _ = manager.Register(context.Background(), &coreauth.Auth{ID: "codex-auth", Provider: "codex"})
	settings := runtimecontrol.DefaultSettings()
	settings.CodexLive.Enabled = true
	handler := New(manager, &liveSettingsStore{settings: settings})

	router := gin.New()
	router.POST("/backend-api/codex/realtime/calls", handler.CreateCall)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/backend-api/codex/realtime/calls", strings.NewReader(`{"session":{"model":"gpt-live-custom"}}`))
	req.Header.Set("OpenAI-Alpha", "quicksilver=v2")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if executor.authorization != "Bearer prepared" || executor.protocolHeader != "quicksilver=v2" {
		t.Fatalf("forwarded headers authorization=%q protocol=%q", executor.authorization, executor.protocolHeader)
	}
	if !strings.Contains(executor.requestBody, "gpt-live-custom") {
		t.Fatalf("upstream body = %s", executor.requestBody)
	}
	if rec.Header().Get("Location") != "/backend-api/codex/realtime/calls/call-123" || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("response headers = %#v", rec.Header())
	}
}

func TestSidebandRequiresStoredSessionBeforeMediaPhase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := runtimecontrol.DefaultSettings()
	settings.CodexLive.Enabled = true
	handler := New(coreauth.NewManager(nil, nil, nil), &liveSettingsStore{settings: settings})

	router := gin.New()
	router.GET("/backend-api/codex/live/:call_id", handler.Sideband)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backend-api/codex/live/call-123", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing session status = %d, want %d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	handler.sessions.Put("call-123", liveSession("codex-auth"))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("stored session status = %d, want %d body=%s", rec.Code, http.StatusNotImplemented, rec.Body.String())
	}
}

func liveSession(authID string) live.Session {
	return live.Session{AuthID: authID, Model: "gpt-live-1-codex"}
}
