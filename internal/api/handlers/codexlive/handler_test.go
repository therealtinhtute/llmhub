package codexlive

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

func TestSidebandRequiresStoredSessionAndWebsocketUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := runtimecontrol.DefaultSettings()
	settings.CodexLive.Enabled = true
	handler := New(coreauth.NewManager(nil, nil, nil), &liveSettingsStore{settings: settings})

	router := gin.New()
	router.GET("/backend-api/codex/live/:call_id", handler.Sideband)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/backend-api/codex/live/call-123", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUpgradeRequired {
		t.Fatalf("non-websocket status = %d, want %d body=%s", rec.Code, http.StatusUpgradeRequired, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/backend-api/codex/live/call-123", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-Websocket-Version", "13")
	req.Header.Set("Sec-Websocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing session status = %d, want %d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestHandleSidebandPinsAuthAndRelaysBidirectionally(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	upstreamGot := make(chan string, 1)
	upstreamAuth := make(chan string, 1)
	upstreamPath := make(chan string, 1)
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath <- r.URL.RequestURI()
		upstreamAuth <- r.Header.Get("Authorization")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade upstream websocket: %v", err)
			return
		}
		defer conn.Close()
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read upstream websocket: %v", err)
			return
		}
		upstreamGot <- string(payload)
		if err = conn.WriteMessage(msgType, []byte("from-upstream")); err != nil {
			t.Errorf("write upstream websocket: %v", err)
		}
	}))
	defer upstreamServer.Close()
	oldBaseURL := sidebandBaseURL
	sidebandBaseURL = func() string { return "ws" + strings.TrimPrefix(upstreamServer.URL, "http") }
	defer func() { sidebandBaseURL = oldBaseURL }()

	executor := &liveHTTPExecutor{}
	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(executor)
	_, _ = manager.Register(context.Background(), &coreauth.Auth{ID: "codex-auth", Provider: "codex"})
	settings := runtimecontrol.DefaultSettings()
	settings.CodexLive.Enabled = true
	handler := New(manager, &liveSettingsStore{settings: settings})
	handler.sessions.Put("call-123", liveSession("codex-auth"))

	router := gin.New()
	router.GET("/backend-api/codex/live/:call_id", handler.Sideband)
	server := httptest.NewServer(router)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/backend-api/codex/live/call-123", nil)
	if err != nil {
		t.Fatalf("dial downstream websocket: %v", err)
	}
	defer conn.Close()
	if err = conn.WriteMessage(websocket.TextMessage, []byte("from-client")); err != nil {
		t.Fatalf("write downstream websocket: %v", err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read downstream websocket: %v", err)
	}
	if string(payload) != "from-upstream" {
		t.Fatalf("downstream payload = %q", payload)
	}
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))

	select {
	case got := <-upstreamGot:
		if got != "from-client" {
			t.Fatalf("upstream payload = %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for upstream payload")
	}
	select {
	case got := <-upstreamAuth:
		if got != "Bearer prepared" {
			t.Fatalf("upstream authorization = %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for upstream auth")
	}
	select {
	case got := <-upstreamPath:
		if got != "/live/call-123" {
			t.Fatalf("upstream path = %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for upstream path")
	}
	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, ok := handler.sessions.Peek("call-123"); !ok {
			break
		}
		select {
		case <-deadline:
			t.Fatal("session still stored after sideband relay closed")
		case <-ticker.C:
		}
	}
}

func liveSession(authID string) live.Session {
	return live.Session{AuthID: authID, Model: "gpt-live-1-codex", Resources: &live.SessionResources{}}
}
