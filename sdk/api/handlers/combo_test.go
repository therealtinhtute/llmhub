package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	coreexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdkconfig "github.com/therealtinhtute/llmhub/sdk/config"
)

// comboTestExecutor is a per-provider fake executor; failures are keyed by
// upstream model id (the combo candidate's model).
type comboTestExecutor struct {
	id string

	mu                sync.Mutex
	executeModels     []string
	streamModels      []string
	executeErrors     map[string]error
	streamFirstErrors map[string]error
	streamPayloads    map[string][]coreexecutor.StreamChunk
}

func (e *comboTestExecutor) Identifier() string { return e.id }

func (e *comboTestExecutor) Execute(_ context.Context, _ *coreauth.Auth, req coreexecutor.Request, _ coreexecutor.Options) (coreexecutor.Response, error) {
	e.mu.Lock()
	e.executeModels = append(e.executeModels, req.Model)
	err := e.executeErrors[req.Model]
	e.mu.Unlock()
	if err != nil {
		return coreexecutor.Response{}, err
	}
	return coreexecutor.Response{Payload: []byte(req.Model)}, nil
}

func (e *comboTestExecutor) ExecuteStream(_ context.Context, _ *coreauth.Auth, req coreexecutor.Request, _ coreexecutor.Options) (*coreexecutor.StreamResult, error) {
	e.mu.Lock()
	e.streamModels = append(e.streamModels, req.Model)
	err := e.streamFirstErrors[req.Model]
	payloadChunks, hasCustomChunks := e.streamPayloads[req.Model]
	chunks := append([]coreexecutor.StreamChunk(nil), payloadChunks...)
	e.mu.Unlock()
	ch := make(chan coreexecutor.StreamChunk, max(1, len(chunks)))
	if err != nil {
		ch <- coreexecutor.StreamChunk{Err: err}
		close(ch)
		return &coreexecutor.StreamResult{Chunks: ch}, nil
	}
	if !hasCustomChunks {
		ch <- coreexecutor.StreamChunk{Payload: []byte(req.Model)}
	} else {
		for _, chunk := range chunks {
			ch <- chunk
		}
	}
	close(ch)
	return &coreexecutor.StreamResult{Chunks: ch}, nil
}

func (e *comboTestExecutor) Refresh(_ context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *comboTestExecutor) CountTokens(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{}, &coreauth.Error{Code: "not_implemented", Message: "CountTokens not implemented"}
}

func (e *comboTestExecutor) HttpRequest(context.Context, *coreauth.Auth, *http.Request) (*http.Response, error) {
	return nil, &coreauth.Error{Code: "not_implemented", Message: "HttpRequest not implemented", HTTPStatus: http.StatusNotImplemented}
}

func (e *comboTestExecutor) ExecuteModels() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.executeModels...)
}

func (e *comboTestExecutor) StreamModels() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.streamModels...)
}

// comboRetryAfterError wraps a coreauth.Error with a provider retry hint, the
// same shape HomeConcurrencyBusyError exposes.
type comboRetryAfterError struct {
	cause      *coreauth.Error
	retryAfter time.Duration
}

func (e *comboRetryAfterError) Error() string              { return e.cause.Error() }
func (e *comboRetryAfterError) StatusCode() int            { return e.cause.StatusCode() }
func (e *comboRetryAfterError) RetryAfter() *time.Duration { return &e.retryAfter }

// comboFixture wires two candidate providers (claude, openrouter) behind a
// fallback combo named "daily", each with its own auth and executor.
func comboFixture(t *testing.T) (*coreauth.Manager, *comboTestExecutor, *comboTestExecutor) {
	t.Helper()
	claude := &comboTestExecutor{id: "claude"}
	openrouter := &comboTestExecutor{id: "openrouter"}

	m := coreauth.NewManager(nil, nil, nil)
	m.SetRetryConfig(0, 0, 0)
	cfg := configComboFixture()
	m.SetConfig(cfg)
	m.RegisterExecutor(claude)
	m.RegisterExecutor(openrouter)

	reg := registry.GetGlobalRegistry()
	for _, provider := range []string{"claude", "openrouter"} {
		auth := &coreauth.Auth{
			ID:       "auth-" + provider,
			Provider: provider,
			Status:   coreauth.StatusActive,
			Attributes: map[string]string{
				"api_key": "test-key",
			},
		}
		if _, err := m.Register(context.Background(), auth); err != nil {
			t.Fatalf("register %s auth: %v", provider, err)
		}
		reg.RegisterClient(auth.ID, provider, []*registry.ModelInfo{{ID: providerModel(provider)}})
		t.Cleanup(func() { reg.UnregisterClient(auth.ID) })
	}
	return m, claude, openrouter
}

func configComboFixture() *sdkconfig.Config {
	return &sdkconfig.Config{
		Combos: []internalconfig.ComboConfig{
			{
				Name:     "daily",
				Strategy: "fallback",
				Models:   []string{"claude/claude-opus-4-7", "openrouter/deepseek-v4:free"},
			},
		},
	}
}

func providerModel(provider string) string {
	if provider == "claude" {
		return "claude-opus-4-7"
	}
	return "deepseek-v4:free"
}

func TestExecuteWithAuthManager_ComboFallsBackOnRetryable429(t *testing.T) {
	m, claude, openrouter := comboFixture(t)
	claude.executeErrors = map[string]error{
		"claude-opus-4-7": &coreauth.Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	}
	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, m)

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", "daily", []byte(`{"model":"daily"}`), "")
	if errMsg != nil {
		t.Fatalf("unexpected error: %+v", errMsg)
	}
	if string(payload) != "deepseek-v4:free" {
		t.Fatalf("payload = %q, want %q", string(payload), "deepseek-v4:free")
	}
	gotClaude := claude.ExecuteModels()
	gotOpen := openrouter.ExecuteModels()
	if len(gotClaude) != 1 || gotClaude[0] != "claude-opus-4-7" {
		t.Fatalf("claude calls = %v, want [claude-opus-4-7]", gotClaude)
	}
	if len(gotOpen) != 1 || gotOpen[0] != "deepseek-v4:free" {
		t.Fatalf("openrouter calls = %v, want [deepseek-v4:free]", gotOpen)
	}
}

func TestExecuteWithAuthManager_ComboUnmasksNonRetryable400(t *testing.T) {
	m, claude, openrouter := comboFixture(t)
	claude.executeErrors = map[string]error{
		"claude-opus-4-7": &coreauth.Error{HTTPStatus: http.StatusBadRequest, Message: "invalid_request_error: malformed payload"},
	}
	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, m)

	_, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", "daily", []byte(`{"model":"daily"}`), "")
	if errMsg == nil {
		t.Fatal("expected error, got nil")
	}
	if errMsg.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", errMsg.StatusCode, http.StatusBadRequest)
	}
	if calls := openrouter.ExecuteModels(); len(calls) != 0 {
		t.Fatalf("openrouter calls = %v, want none (candidate must not be attempted)", calls)
	}
}

func TestExecuteWithAuthManager_ComboExhaustedReturns503WithEarliestReset(t *testing.T) {
	m, _, _ := comboFixture(t)
	// Suspend each candidate's model via MarkResult: a 429 gives a short
	// cooldown, a 401 gives 30 minutes, so the earliest reset is claude's.
	m.MarkResult(context.Background(), coreauth.Result{
		AuthID:   "auth-claude",
		Provider: "claude",
		Model:    "claude-opus-4-7",
		Success:  false,
		Error:    &coreauth.Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	})
	m.MarkResult(context.Background(), coreauth.Result{
		AuthID:   "auth-openrouter",
		Provider: "openrouter",
		Model:    "deepseek-v4:free",
		Success:  false,
		Error:    &coreauth.Error{HTTPStatus: http.StatusUnauthorized, Message: "invalid grant"},
	})
	claudeAuth, ok := m.GetByID("auth-claude")
	if !ok || claudeAuth == nil {
		t.Fatal("auth-claude not found")
	}
	earliest := claudeAuth.ModelStates["claude-opus-4-7"].NextRetryAfter
	if earliest.IsZero() {
		t.Fatal("expected claude model cooldown reset")
	}
	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, m)

	_, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", "daily", []byte(`{"model":"daily"}`), "")
	if errMsg == nil {
		t.Fatal("expected error, got nil")
	}
	if errMsg.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", errMsg.StatusCode, http.StatusServiceUnavailable)
	}
	var exhausted *coreauth.ComboExhaustedError
	if !errors.As(errMsg.Error, &exhausted) || exhausted == nil {
		t.Fatalf("error = %T, want *ComboExhaustedError", errMsg.Error)
	}
	if exhausted.Candidates != 2 {
		t.Fatalf("candidates = %d, want 2", exhausted.Candidates)
	}
	if exhausted.ResetAt.Before(earliest.Add(-2*time.Second)) || exhausted.ResetAt.After(earliest.Add(2*time.Second)) {
		t.Fatalf("reset at %v, want ≈ claude's cooldown %v", exhausted.ResetAt, earliest)
	}
	openrouterAuth, ok := m.GetByID("auth-openrouter")
	if !ok || openrouterAuth == nil {
		t.Fatal("auth-openrouter not found")
	}
	if later := openrouterAuth.ModelStates["deepseek-v4:free"].NextRetryAfter; exhausted.ResetAt.After(later) {
		t.Fatalf("reset at %v must be the earliest, openrouter resets at %v", exhausted.ResetAt, later)
	}
	wantHuman := "reset after " + earliest.Sub(time.Now()).Truncate(time.Second).String()
	if !strings.Contains(exhausted.Error(), wantHuman) {
		t.Fatalf("error string %q missing %q", exhausted.Error(), wantHuman)
	}
}

func TestExecuteStreamWithAuthManager_ComboBootstrapFailureSwitchesSilently(t *testing.T) {
	m, claude, openrouter := comboFixture(t)
	claude.streamFirstErrors = map[string]error{
		"claude-opus-4-7": &coreauth.Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"},
	}
	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{
		Streaming: sdkconfig.StreamingConfig{BootstrapRetries: 0},
	}, m)

	dataChan, _, errChan := handler.ExecuteStreamWithAuthManager(context.Background(), "openai", "daily", []byte(`{"model":"daily"}`), "")
	if dataChan == nil || errChan == nil {
		t.Fatal("expected non-nil channels")
	}
	var got []byte
	for chunk := range dataChan {
		got = append(got, chunk...)
	}
	for msg := range errChan {
		if msg != nil {
			t.Fatalf("unexpected error: %+v", msg)
		}
	}
	if string(got) != "deepseek-v4:free" {
		t.Fatalf("payload = %q, want %q", string(got), "deepseek-v4:free")
	}
	gotClaude := claude.StreamModels()
	gotOpen := openrouter.StreamModels()
	if len(gotClaude) != 1 || gotClaude[0] != "claude-opus-4-7" {
		t.Fatalf("claude streams = %v, want [claude-opus-4-7]", gotClaude)
	}
	if len(gotOpen) != 1 || gotOpen[0] != "deepseek-v4:free" {
		t.Fatalf("openrouter streams = %v, want [deepseek-v4:free]", gotOpen)
	}
}

func TestExecuteStreamWithAuthManager_ComboPostChunkFailureDoesNotSwitch(t *testing.T) {
	m, claude, openrouter := comboFixture(t)
	claude.streamPayloads = map[string][]coreexecutor.StreamChunk{
		"claude-opus-4-7": {
			{Payload: []byte("partial")},
			{Err: &coreauth.Error{HTTPStatus: http.StatusBadGateway, Message: "upstream closed"}},
		},
	}
	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{
		Streaming: sdkconfig.StreamingConfig{BootstrapRetries: 0},
	}, m)

	dataChan, _, errChan := handler.ExecuteStreamWithAuthManager(context.Background(), "openai", "daily", []byte(`{"model":"daily"}`), "")
	var got []byte
	for chunk := range dataChan {
		got = append(got, chunk...)
	}
	var gotStatus int
	for msg := range errChan {
		if msg != nil {
			gotStatus = msg.StatusCode
		}
	}
	if string(got) != "partial" {
		t.Fatalf("payload = %q, want %q", string(got), "partial")
	}
	if gotStatus != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", gotStatus, http.StatusBadGateway)
	}
	if calls := openrouter.StreamModels(); len(calls) != 0 {
		t.Fatalf("openrouter streams = %v, want none (post-chunk failure must not switch)", calls)
	}
}

func TestExecuteWithAuthManager_ComboSleepsOnShort502Cooldown(t *testing.T) {
	m, claude, _ := comboFixture(t)
	claude.executeErrors = map[string]error{
		"claude-opus-4-7": &comboRetryAfterError{
			cause:      &coreauth.Error{HTTPStatus: http.StatusBadGateway, Message: "upstream restarting"},
			retryAfter: 3 * time.Second,
		},
	}
	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, m)

	start := time.Now()
	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", "daily", []byte(`{"model":"daily"}`), "")
	elapsed := time.Since(start)
	if errMsg != nil {
		t.Fatalf("unexpected error: %+v", errMsg)
	}
	if string(payload) != "deepseek-v4:free" {
		t.Fatalf("payload = %q, want %q", string(payload), "deepseek-v4:free")
	}
	if elapsed < 3*time.Second {
		t.Fatalf("elapsed = %v, want >= 3s cooldown pause before switching", elapsed)
	}
}

func TestExecuteWithAuthManager_ComboSkipsLong502Cooldown(t *testing.T) {
	m, claude, _ := comboFixture(t)
	claude.executeErrors = map[string]error{
		"claude-opus-4-7": &comboRetryAfterError{
			cause:      &coreauth.Error{HTTPStatus: http.StatusBadGateway, Message: "upstream restarting"},
			retryAfter: 40 * time.Second,
		},
	}
	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, m)

	start := time.Now()
	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", "daily", []byte(`{"model":"daily"}`), "")
	elapsed := time.Since(start)
	if errMsg != nil {
		t.Fatalf("unexpected error: %+v", errMsg)
	}
	if string(payload) != "deepseek-v4:free" {
		t.Fatalf("payload = %q, want %q", string(payload), "deepseek-v4:free")
	}
	if elapsed >= 5*time.Second {
		t.Fatalf("elapsed = %v, want no wait for a 40s cooldown", elapsed)
	}
}
