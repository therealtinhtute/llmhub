package executor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	"github.com/therealtinhtute/llmhub/sdk/cliproxy/executionregistry"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
// (internal/runtime/executor/antigravity_home_model_capabilities_test.go).
type antigravityHomeModelCapabilityDispatcher struct {
	baseURL string
	model   string
}

func (*antigravityHomeModelCapabilityDispatcher) HeartbeatOK() bool { return true }

func (d *antigravityHomeModelCapabilityDispatcher) RPopAuth(context.Context, string, string, http.Header, int) ([]byte, error) {
	model := d.model
	if model == "" {
		model = "gemini-3.8-flash-high"
	}
	return json.Marshal(map[string]any{
		"model":    model,
		"provider": "antigravity",
		"model_info": map[string]any{
			"id":             model,
			"type":           "gemini",
			"context_length": 1048576,
			"thinking": map[string]any{
				"levels": []string{"low", "medium", "high"},
			},
			"user_defined": false,
		},
		"auth": cliproxyauth.Auth{
			ID:       "home-antigravity-3.8-auth",
			Provider: "antigravity",
			Status:   cliproxyauth.StatusActive,
			Attributes: map[string]string{
				"auth_kind": "oauth",
				"base_url":  d.baseURL,
			},
			Metadata: map[string]any{
				"access_token": "token",
				"project_id":   "project-1",
				"expired":      time.Now().Add(time.Hour).Format(time.RFC3339),
			},
		},
	})
}

func (*antigravityHomeModelCapabilityDispatcher) AbortAmbiguousDispatch() {}

func newAntigravityHomeCapabilityManager(t *testing.T, serverURL, model string) *cliproxyauth.Manager {
	t.Helper()
	cfg := &config.Config{RequestRetry: 1}
	cfg.Home.Enabled = true
	manager := cliproxyauth.NewManager(nil, nil, nil)
	manager.SetConfig(cfg)
	manager.RegisterExecutor(NewAntigravityExecutor(cfg))
	manager.PublishHomeDispatch(&antigravityHomeModelCapabilityDispatcher{baseURL: serverURL, model: model}, executionregistry.New(), 1)
	return manager
}

func executeAntigravityHomeRequest(t *testing.T, manager *cliproxyauth.Manager, model string) []byte {
	t.Helper()
	payload := []byte(`{"model":"` + model + `","reasoning_effort":"medium","messages":[{"role":"user","content":"hello"}]}`)
	_, errExecute := manager.Execute(context.Background(), []string{"antigravity"}, cliproxyexecutor.Request{
		Model:   model,
		Payload: payload,
	}, cliproxyexecutor.Options{
		SourceFormat:    sdktranslator.FormatOpenAI,
		ResponseFormat:  sdktranslator.FormatOpenAI,
		OriginalRequest: payload,
	})
	if errExecute != nil {
		t.Fatalf("Execute() error = %v", errExecute)
	}
	return nil
}

// TestAntigravityHomeModelThinkingLevelRemainsLevel is the faithful port of the
// upstream test: the home-dispatched model_info carries level capabilities, so
// reasoning_effort=medium must remain a thinkingLevel, not a budget.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
func TestAntigravityHomeModelThinkingLevelRemainsLevel(t *testing.T) {
	requestBodies := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			http.Error(w, errRead.Error(), http.StatusInternalServerError)
			return
		}
		requestBodies <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1,"totalTokenCount":2}}}`)
	}))
	t.Cleanup(server.Close)

	manager := newAntigravityHomeCapabilityManager(t, server.URL, "gemini-3.8-flash-high")
	executeAntigravityHomeRequest(t, manager, "gemini-3.8-flash-high")

	upstreamBody := <-requestBodies
	if got := gjson.GetBytes(upstreamBody, "request.generationConfig.thinkingConfig.thinkingLevel").String(); got != "medium" {
		t.Fatalf("thinkingLevel = %q, want medium; body=%s", got, upstreamBody)
	}
	if budget := gjson.GetBytes(upstreamBody, "request.generationConfig.thinkingConfig.thinkingBudget"); budget.Exists() {
		t.Fatalf("thinkingBudget should be absent, got %s; body=%s", budget.Raw, upstreamBody)
	}
}

// TestAntigravityHomeModelThinkingLevelForUnknownModel is a fork-specific
// companion: the dispatched model is absent from the local static catalog, so
// without the attached model_info the user-defined fallback would rewrite the
// level into a thinkingBudget.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
func TestAntigravityHomeModelThinkingLevelForUnknownModel(t *testing.T) {
	requestBodies := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			http.Error(w, errRead.Error(), http.StatusInternalServerError)
			return
		}
		requestBodies <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1,"totalTokenCount":2}}}`)
	}))
	t.Cleanup(server.Close)

	const model = "gemini-9.9-flash-ultra"
	manager := newAntigravityHomeCapabilityManager(t, server.URL, model)
	executeAntigravityHomeRequest(t, manager, model)

	upstreamBody := <-requestBodies
	if got := gjson.GetBytes(upstreamBody, "request.generationConfig.thinkingConfig.thinkingLevel").String(); got != "medium" {
		t.Fatalf("thinkingLevel = %q, want medium; body=%s", got, upstreamBody)
	}
	if budget := gjson.GetBytes(upstreamBody, "request.generationConfig.thinkingConfig.thinkingBudget"); budget.Exists() {
		t.Fatalf("thinkingBudget should be absent, got %s; body=%s", budget.Raw, upstreamBody)
	}
}
