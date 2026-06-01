package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
)

func TestBuildKiroPayloadFromOpenAI_MessagesToolsImagesAndSuffixes(t *testing.T) {
	payload := []byte(`{
		"model":"claude-sonnet-4.5-thinking-agentic",
		"reasoning_effort":"high",
		"messages":[
			{"role":"system","content":"system prompt"},
			{"role":"user","content":[{"type":"text","text":"hello"},{"type":"image_url","image_url":{"url":"data:image/png;base64,abc123"}}]},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.txt\"}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"file body"}
		],
		"tools":[{"type":"function","function":{"name":"read_file","description":"Read a file","parameters":{"type":"object","additionalProperties":false,"required":[]}}}]
	}`)

	body, err := buildKiroPayloadFromOpenAI(payload, "claude-sonnet-4.5-thinking-agentic")
	if err != nil {
		t.Fatalf("buildKiroPayloadFromOpenAI() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	state := got["conversationState"].(map[string]any)
	current := state["currentMessage"].(map[string]any)["userInputMessage"].(map[string]any)
	if current["modelId"] != "CLAUDE_SONNET_4_5" {
		t.Fatalf("modelId = %#v, want CLAUDE_SONNET_4_5", current["modelId"])
	}
	content := current["content"].(string)
	if !strings.Contains(content, "<thinking_mode>enabled</thinking_mode>") {
		t.Fatalf("current content missing thinking tag: %q", content)
	}
	if !strings.Contains(content, "chunked file edits") {
		t.Fatalf("current content missing agentic hint: %q", content)
	}
	ctx := current["userInputMessageContext"].(map[string]any)
	if len(ctx["tools"].([]any)) != 1 {
		t.Fatalf("tools = %#v, want one tool", ctx["tools"])
	}
	if len(ctx["toolResults"].([]any)) != 1 {
		t.Fatalf("toolResults = %#v, want one result", ctx["toolResults"])
	}
	history := state["history"].([]any)
	firstUser := history[0].(map[string]any)["userInputMessage"].(map[string]any)
	if !strings.Contains(firstUser["content"].(string), "system prompt") {
		t.Fatalf("first history content = %q, want system prompt", firstUser["content"])
	}
	if len(firstUser["images"].([]any)) != 1 {
		t.Fatalf("images = %#v, want one image", firstUser["images"])
	}
}

func TestKiroExecutorExecute_HeadersBodyAndEventStream(t *testing.T) {
	var sawAuth, sawTarget string
	var sawBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawTarget = r.Header.Get("X-Amz-Target")
		if r.URL.Path != kiroGeneratePath {
			t.Fatalf("path = %s, want %s", r.URL.Path, kiroGeneratePath)
		}
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		_, _ = io.WriteString(w, `{"content":"hel"}{"content":"lo"}`)
	}))
	defer server.Close()

	exec := NewKiroExecutor(&config.Config{})
	resp, err := exec.Execute(context.Background(), testKiroAuth(server.URL, "access-1"), cliproxyexecutor.Request{
		Model:   "claude-sonnet-4.5",
		Payload: []byte(`{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":"hi"}]}`),
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai")})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if sawAuth != "Bearer access-1" {
		t.Fatalf("Authorization = %q, want bearer token", sawAuth)
	}
	if sawTarget != kiroAMZTarget {
		t.Fatalf("X-Amz-Target = %q, want %q", sawTarget, kiroAMZTarget)
	}
	if sawBody["conversationState"] == nil {
		t.Fatalf("request body missing conversationState: %#v", sawBody)
	}
	if sawBody["profileArn"] != "arn:aws:codewhisperer:us-east-1:123456789012:profile/ABC" {
		t.Fatalf("profileArn = %#v, want imported profile ARN", sawBody["profileArn"])
	}
	if !bytes.Contains(resp.Payload, []byte(`"content":"hello"`)) {
		t.Fatalf("response payload = %s, want content hello", resp.Payload)
	}
}

func TestKiroExecutorExecute_RefreshesAfter401(t *testing.T) {
	var generateCalls int
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"accessToken":"access-new","refreshToken":"refresh-new","expiresIn":3600}`)
	}))
	defer refreshServer.Close()
	generateServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		generateCalls++
		if generateCalls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `unauthorized`)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-new" {
			t.Fatalf("retry Authorization = %q, want refreshed token", got)
		}
		_, _ = io.WriteString(w, `{"content":"ok"}`)
	}))
	defer generateServer.Close()

	auth := testKiroAuth(generateServer.URL, "access-old")
	auth.Metadata["refresh_token"] = "refresh-old"
	auth.Metadata["refresh_url"] = refreshServer.URL
	exec := NewKiroExecutor(&config.Config{})
	resp, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "auto",
		Payload: []byte(`{"model":"auto","messages":[{"role":"user","content":"hi"}]}`),
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai")})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if generateCalls != 2 {
		t.Fatalf("generate calls = %d, want 2", generateCalls)
	}
	if !bytes.Contains(resp.Payload, []byte(`"content":"ok"`)) {
		t.Fatalf("response payload = %s, want ok", resp.Payload)
	}
}

func TestKiroExecutorExecuteStream_EmitsOpenAISSEAndDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"content":"hello"}`)
	}))
	defer server.Close()

	exec := NewKiroExecutor(&config.Config{})
	result, err := exec.ExecuteStream(context.Background(), testKiroAuth(server.URL, "access-1"), cliproxyexecutor.Request{
		Model:   "auto",
		Payload: []byte(`{"model":"auto","messages":[{"role":"user","content":"hi"}],"stream":true}`),
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai"), Stream: true})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	var all bytes.Buffer
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error: %v", chunk.Err)
		}
		all.Write(chunk.Payload)
	}
	if !strings.Contains(all.String(), `"content":"hello"`) {
		t.Fatalf("stream = %s, want hello chunk", all.String())
	}
	if !strings.Contains(all.String(), "data: [DONE]") {
		t.Fatalf("stream = %s, want DONE", all.String())
	}
}

func TestKiroExecutorResolveModels_LiveCatalogExpandsVariants(t *testing.T) {
	var sawAuth, sawOrigin string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != kiroListModelsPath {
			t.Fatalf("path = %s, want %s", r.URL.Path, kiroListModelsPath)
		}
		sawAuth = r.Header.Get("Authorization")
		sawOrigin = r.URL.Query().Get("origin")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"models":[
			{"modelId":"auto","modelName":"Auto","tokenLimits":{"maxInputTokens":100000}},
			{"modelId":"CLAUDE_SONNET_4_5","modelName":"Claude Sonnet 4.5","rateMultiplier":2.5,"description":"sonnet","tokenLimits":{"maxInputTokens":200000}}
		]}`)
	}))
	defer server.Close()

	auth := &cliproxyauth.Auth{
		ID:       "kiro-models",
		Provider: "kiro",
		Metadata: map[string]any{
			"access_token": "access-1",
			"models_url":   server.URL + kiroListModelsPath,
		},
	}

	models, refreshed, err := NewKiroExecutor(&config.Config{}).ResolveModels(context.Background(), auth)
	if err != nil {
		t.Fatalf("ResolveModels() error = %v", err)
	}
	if refreshed != nil {
		t.Fatalf("refreshed auth = %#v, want nil", refreshed)
	}
	if sawAuth != "Bearer access-1" {
		t.Fatalf("Authorization = %q, want bearer token", sawAuth)
	}
	if sawOrigin != "AI_EDITOR" {
		t.Fatalf("origin = %q, want AI_EDITOR", sawOrigin)
	}
	if len(models) != 6 {
		t.Fatalf("models len = %d, want 6", len(models))
	}
	ids := map[string]*registry.ModelInfo{}
	for _, model := range models {
		ids[model.ID] = model
	}
	if ids["auto-agentic"] != nil {
		t.Fatal("auto-agentic should not be generated")
	}
	if ids["auto"] == nil || ids["auto-thinking"] == nil {
		t.Fatalf("missing auto variants: %#v", ids)
	}
	thinking := ids["CLAUDE_SONNET_4_5-thinking"]
	if thinking == nil || thinking.Thinking == nil {
		t.Fatalf("missing thinking model support: %#v", thinking)
	}
	if got := ids["CLAUDE_SONNET_4_5"].DisplayName; got != "Kiro Claude Sonnet 4.5 (2.5x credit)" {
		t.Fatalf("display name = %q", got)
	}
}

func TestKiroExecutorResolveModels_RefreshesOnUnauthorized(t *testing.T) {
	var listAttempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case kiroListModelsPath:
			listAttempts++
			if listAttempts == 1 {
				http.Error(w, "expired", http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer access-new" {
				t.Fatalf("retry Authorization = %q, want refreshed token", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"models":[{"modelId":"auto","modelName":"Auto"}]}`)
		case "/refreshToken":
			_, _ = io.WriteString(w, `{"accessToken":"access-new","refreshToken":"refresh-new","expiresIn":3600}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	auth := &cliproxyauth.Auth{
		ID:       "kiro-models-refresh",
		Provider: "kiro",
		Metadata: map[string]any{
			"access_token":  "access-old",
			"refresh_token": "refresh-old",
			"models_url":    server.URL + kiroListModelsPath,
			"refresh_url":   server.URL + "/refreshToken",
		},
	}

	models, refreshed, err := NewKiroExecutor(&config.Config{}).ResolveModels(context.Background(), auth)
	if err != nil {
		t.Fatalf("ResolveModels() error = %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models len = %d, want auto base + thinking", len(models))
	}
	if refreshed == nil {
		t.Fatal("expected refreshed auth")
	}
	if got := refreshed.Metadata["access_token"]; got != "access-new" {
		t.Fatalf("refreshed access_token = %#v", got)
	}
	if got := refreshed.Metadata["refresh_token"]; got != "refresh-new" {
		t.Fatalf("refreshed refresh_token = %#v", got)
	}
	if listAttempts != 2 {
		t.Fatalf("list attempts = %d, want 2", listAttempts)
	}
}

func testKiroAuth(baseURL, accessToken string) *cliproxyauth.Auth {
	return &cliproxyauth.Auth{
		ID:       "kiro.json",
		Provider: "kiro",
		Metadata: map[string]any{
			"type":          "kiro",
			"access_token":  accessToken,
			"refresh_token": "refresh-1",
			"base_url":      baseURL,
			"profile_arn":   "arn:aws:codewhisperer:us-east-1:123456789012:profile/ABC",
		},
	}
}
