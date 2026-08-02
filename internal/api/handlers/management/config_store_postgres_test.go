package management

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
)

type recordingConfigStore struct {
	data      []byte
	saveErr   error
	saveCount int
}

func (s *recordingConfigStore) LoadConfigBytes(context.Context) ([]byte, error) {
	return append([]byte(nil), s.data...), nil
}

func (s *recordingConfigStore) SaveConfig(_ context.Context, data []byte) (int64, error) {
	if s.saveErr != nil {
		return 0, s.saveErr
	}
	s.saveCount++
	s.data = append([]byte(nil), data...)
	return int64(s.saveCount), nil
}

func TestPutCodexKeys_UsesConfigStore(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &recordingConfigStore{}
	h := NewHandlerWithoutConfigFilePath(&config.Config{}, nil)
	h.SetConfigStore(store)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(
		http.MethodPut,
		"/v0/management/codex-api-key",
		bytes.NewBufferString(`[{"api-key":"codex-key","base-url":"https://codex.example.com"}]`),
	)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PutCodexKeys(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if store.saveCount != 1 {
		t.Fatalf("expected config store save count 1, got %d", store.saveCount)
	}
	if !bytes.Contains(store.data, []byte("codex-api-key")) {
		t.Fatalf("expected saved config to contain codex-api-key, got %s", string(store.data))
	}
}

func TestPutCodexKeys_ConfigStoreSaveErrorReturnsFailure(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	h := NewHandlerWithoutConfigFilePath(&config.Config{}, nil)
	h.SetConfigStore(&recordingConfigStore{saveErr: errors.New("database unavailable")})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(
		http.MethodPut,
		"/v0/management/codex-api-key",
		bytes.NewBufferString(`[{"api-key":"codex-key","base-url":"https://codex.example.com"}]`),
	)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PutCodexKeys(ctx)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func TestPutDebug_ConfigChangeHookRunsOutsideLock(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &recordingConfigStore{}
	h := NewHandlerWithoutConfigFilePath(&config.Config{}, nil)
	h.SetConfigStore(store)
	h.SetConfigChangeHook(func(cfg *config.Config) {
		h.SetConfig(cfg)
	})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPut, "/v0/management/debug", bytes.NewBufferString(`{"value":true}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	done := make(chan struct{})
	go func() {
		h.PutDebug(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("PutDebug deadlocked while invoking config change hook")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !h.cfg.Debug {
		t.Fatal("expected debug flag to be persisted in handler config")
	}
}

func TestPutConfigYAML_ConfigChangeHookRunsOutsideLock(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &recordingConfigStore{}
	h := NewHandlerWithoutConfigFilePath(&config.Config{}, nil)
	h.SetConfigStore(store)
	h.SetConfigChangeHook(func(cfg *config.Config) {
		h.SetConfig(cfg)
	})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPut, "/v0/management/config.yaml", bytes.NewBufferString("debug: true\n"))
	req.Header.Set("Content-Type", "application/yaml")
	ctx.Request = req

	done := make(chan struct{})
	go func() {
		h.PutConfigYAML(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("PutConfigYAML deadlocked while invoking config change hook")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !h.cfg.Debug {
		t.Fatal("expected config yaml write to update handler config")
	}
}

func TestPutStaticProviderCredentialWeights_RejectInvalidValues(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		path string
		body string
		call func(*Handler, *gin.Context)
	}{
		{
			name: "gemini",
			path: "/v0/management/gemini-api-key",
			body: `[{"api-key":"gemini-key","weight":-1}]`,
			call: func(h *Handler, c *gin.Context) { h.PutGeminiKeys(c) },
		},
		{
			name: "claude",
			path: "/v0/management/claude-api-key",
			body: `[{"api-key":"claude-key","weight":-1}]`,
			call: func(h *Handler, c *gin.Context) { h.PutClaudeKeys(c) },
		},
		{
			name: "codex",
			path: "/v0/management/codex-api-key",
			body: `[{"api-key":"codex-key","base-url":"https://codex.example.com","weight":-1}]`,
			call: func(h *Handler, c *gin.Context) { h.PutCodexKeys(c) },
		},
		{
			name: "openai-compatible",
			path: "/v0/management/openai-compatibility",
			body: `[{"name":"compat","base-url":"https://compat.example.com","api-key-entries":[{"api-key":"compat-key"}],"weight":-1}]`,
			call: func(h *Handler, c *gin.Context) { h.PutOpenAICompat(c) },
		},
		{
			name: "vertex-compatible",
			path: "/v0/management/vertex-api-key",
			body: `[{"api-key":"vertex-key","base-url":"https://vertex.example.com","weight":-1}]`,
			call: func(h *Handler, c *gin.Context) { h.PutVertexCompatKeys(c) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &recordingConfigStore{}
			h := NewHandlerWithoutConfigFilePath(&config.Config{}, nil)
			h.SetConfigStore(store)

			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			req := httptest.NewRequest(http.MethodPut, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req

			tt.call(h, ctx)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, rec.Code, rec.Body.String())
			}
			if store.saveCount != 0 {
				t.Fatalf("expected no config store writes, got %d", store.saveCount)
			}
		})
	}
}

func TestPutCodexKeys_PersistsCredentialWeight(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &recordingConfigStore{}
	h := NewHandlerWithoutConfigFilePath(&config.Config{}, nil)
	h.SetConfigStore(store)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(
		http.MethodPut,
		"/v0/management/codex-api-key",
		bytes.NewBufferString(`[{"api-key":"codex-key","base-url":"https://codex.example.com","weight":7}]`),
	)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PutCodexKeys(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if store.saveCount != 1 {
		t.Fatalf("expected config store save count 1, got %d", store.saveCount)
	}
	if len(h.cfg.CodexKey) != 1 || h.cfg.CodexKey[0].Weight != 7 {
		t.Fatalf("expected codex weight 7, got %+v", h.cfg.CodexKey)
	}
	if !bytes.Contains(store.data, []byte("weight: 7")) {
		t.Fatalf("expected saved config to contain weight, got %s", string(store.data))
	}
}

func TestPatchCodexKey_RejectsInvalidCredentialWeightWithoutMutation(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &recordingConfigStore{}
	h := NewHandlerWithoutConfigFilePath(&config.Config{
		CodexKey: []config.CodexKey{{APIKey: "codex-key", BaseURL: "https://codex.example.com", Weight: 5}},
	}, nil)
	h.SetConfigStore(store)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(
		http.MethodPatch,
		"/v0/management/codex-api-key",
		bytes.NewBufferString(`{"index":0,"value":{"weight":1000001,"prefix":"mutated"}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PatchCodexKey(ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if got := h.cfg.CodexKey[0].Weight; got != 5 {
		t.Fatalf("expected weight to remain 5, got %d", got)
	}
	if got := h.cfg.CodexKey[0].Prefix; got != "" {
		t.Fatalf("expected prefix to remain empty, got %q", got)
	}
	if store.saveCount != 0 {
		t.Fatalf("expected no config store writes, got %d", store.saveCount)
	}
}

func TestPatchCodexKey_AllowsZeroCredentialWeight(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &recordingConfigStore{}
	h := NewHandlerWithoutConfigFilePath(&config.Config{
		CodexKey: []config.CodexKey{{APIKey: "codex-key", BaseURL: "https://codex.example.com", Weight: 5}},
	}, nil)
	h.SetConfigStore(store)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(
		http.MethodPatch,
		"/v0/management/codex-api-key",
		bytes.NewBufferString(`{"index":0,"value":{"weight":0}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PatchCodexKey(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := h.cfg.CodexKey[0].Weight; got != 0 {
		t.Fatalf("expected weight to clear to 0, got %d", got)
	}
	if store.saveCount != 1 {
		t.Fatalf("expected config store save count 1, got %d", store.saveCount)
	}
}
