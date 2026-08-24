package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	_ "github.com/therealtinhtute/llmhub/internal/translator"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestXAIExecutionSessionIDUsesDerivedFallback(t *testing.T) {
	id := xaiExecutionSessionID(cliproxyexecutor.Request{}, cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:xai-root",
	}})
	if id == "" {
		t.Fatal("expected derived session UUID")
	}
	if repeated := xaiExecutionSessionID(cliproxyexecutor.Request{}, cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:xai-root",
	}}); repeated != id {
		t.Fatalf("derived session UUID is not stable: first=%q repeated=%q", id, repeated)
	}

	explicit := xaiExecutionSessionID(cliproxyexecutor.Request{}, cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.ExecutionSessionMetadataKey: "explicit-session",
		cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:xai-root",
	}})
	if explicit != "explicit-session" {
		t.Fatalf("explicit execution session = %q, want explicit-session", explicit)
	}
}

func TestXAIExecutorExecuteShapesResponsesRequest(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotGrokConvID string
	var gotOriginator string
	var gotAccountID string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotGrokConvID = r.Header.Get("x-grok-conv-id")
		gotOriginator = r.Header.Get("Originator")
		gotAccountID = r.Header.Get("Chatgpt-Account-Id")
		var errRead error
		gotBody, errRead = io.ReadAll(r.Body)
		if errRead != nil {
			t.Fatalf("read body: %v", errRead)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":0,\"status\":\"completed\",\"model\":\"grok-4.3\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "xai-auth",
		Provider: "xai",
		Attributes: map[string]string{
			"base_url":  server.URL,
			"auth_kind": "oauth",
		},
		Metadata: map[string]any{
			"access_token": "xai-token",
			"email":        "user@example.com",
		},
	}

	_, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.3",
		Payload: []byte(`{"model":"grok-4.3","input":[{"type":"reasoning","summary":[{"type":"summary_text","text":"test"}],"content":null,"encrypted_content":null},{"type":"reasoning","summary":[{"type":"summary_text","text":"second"}]},{"role":"user","content":"hello"}],"include":["reasoning.encrypted_content"],"reasoning":{"effort":"high"},"tools":[{"type":"tool_search"},{"type":"image_generation"},{"type":"custom","name":"apply_patch"},{"type":"custom","name":"custom_lookup"},{"type":"function","name":"lookup"},{"type":"web_search","external_web_access":true,"search_content_types":["text","image"]},{"type":"namespace","name":"codex_app","description":"Tools in the codex_app namespace.","tools":[{"type":"function","name":"automation_update"},{"type":"custom","name":"namespace_custom"},{"type":"tool_search"}]}]}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       false,
		Metadata: map[string]any{
			cliproxyexecutor.ExecutionSessionMetadataKey: "conv-xai-1",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gotPath != "/responses" {
		t.Fatalf("path = %q, want /responses", gotPath)
	}
	if gotAuth != "Bearer xai-token" {
		t.Fatalf("Authorization = %q, want Bearer xai-token", gotAuth)
	}
	if gotGrokConvID != "conv-xai-1" {
		t.Fatalf("x-grok-conv-id = %q, want conv-xai-1", gotGrokConvID)
	}
	if gotOriginator != "" {
		t.Fatalf("Originator = %q, want empty", gotOriginator)
	}
	if gotAccountID != "" {
		t.Fatalf("Chatgpt-Account-Id = %q, want empty", gotAccountID)
	}
	if gjson.GetBytes(gotBody, "prompt_cache_key").String() != "conv-xai-1" {
		t.Fatalf("prompt_cache_key missing from body: %s", string(gotBody))
	}
	if !gjson.GetBytes(gotBody, "stream").Bool() {
		t.Fatalf("stream = false, want true; body=%s", string(gotBody))
	}
	if gjson.GetBytes(gotBody, "reasoning.effort").String() != "high" {
		t.Fatalf("reasoning.effort = %q, want high; body=%s", gjson.GetBytes(gotBody, "reasoning.effort").String(), string(gotBody))
	}
	if gjson.GetBytes(gotBody, "input.0.content").Exists() {
		t.Fatalf("input.0.content exists, want removed; body=%s", string(gotBody))
	}
	if gjson.GetBytes(gotBody, "input.0.encrypted_content").Exists() {
		t.Fatalf("input.0.encrypted_content exists, want removed; body=%s", string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "input.0.summary.0.text").String(); got != "test" {
		t.Fatalf("input.0.summary.0.text = %q, want test; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "input.0.summary.1.text").String(); got != "second" {
		t.Fatalf("input.0.summary.1.text = %q, want second; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "input.1.role").String(); got != "user" {
		t.Fatalf("input.1.role = %q, want user; body=%s", got, string(gotBody))
	}
	if gjson.GetBytes(gotBody, "input.2").Exists() {
		t.Fatalf("input.2 exists, want consecutive reasoning item merged; body=%s", string(gotBody))
	}
	tools := gjson.GetBytes(gotBody, "tools").Array()
	if len(tools) != 5 {
		t.Fatalf("tools length = %d, want 5; body=%s", len(tools), string(gotBody))
	}
	foundAutomationUpdate := false
	foundNamespaceCustom := false
	for i, tool := range tools {
		toolType := tool.Get("type").String()
		if toolType == "image_generation" {
			t.Fatalf("tools.%d.type = image_generation, want removed; body=%s", i, string(gotBody))
		}
		if toolType != "function" && toolType != "web_search" {
			t.Fatalf("tools.%d.type = %q, want function or web_search; body=%s", i, toolType, string(gotBody))
		}
		if toolType == "function" && !tool.Get("parameters").Exists() {
			t.Fatalf("tools.%d.parameters missing for xAI function tool; body=%s", i, string(gotBody))
		}
		if got := tool.Get("name").String(); got == "apply_patch" {
			t.Fatalf("tools.%d.name = apply_patch, want removed; body=%s", i, string(gotBody))
		}
		switch tool.Get("name").String() {
		case "codex_app__automation_update":
			foundAutomationUpdate = true
		case "codex_app__namespace_custom":
			foundNamespaceCustom = true
		}
		if toolType == "web_search" {
			if tool.Get("external_web_access").Exists() {
				t.Fatalf("tools.%d.external_web_access exists, want removed; body=%s", i, string(gotBody))
			}
			if got := tool.Get("search_content_types.1").String(); got != "image" {
				t.Fatalf("tools.%d.search_content_types missing image entry; body=%s", i, string(gotBody))
			}
		}
	}
	if !foundAutomationUpdate {
		t.Fatalf("namespace function tool was not moved to top-level tools; body=%s", string(gotBody))
	}
	if !foundNamespaceCustom {
		t.Fatalf("namespace custom tool was not moved to top-level tools; body=%s", string(gotBody))
	}
	for _, include := range gjson.GetBytes(gotBody, "include").Array() {
		if include.String() == "reasoning.encrypted_content" {
			t.Fatalf("xai request must not ask for encrypted reasoning content: %s", string(gotBody))
		}
	}
}

func TestXAIExecutorOmitsUnsupportedReasoningEffort(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errRead error
		gotBody, errRead = io.ReadAll(r.Body)
		if errRead != nil {
			t.Fatalf("read body: %v", errRead)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":0,\"status\":\"completed\",\"model\":\"grok-4\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}]}}\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider: "xai",
		Attributes: map[string]string{
			"base_url":  server.URL,
			"auth_kind": "oauth",
		},
		Metadata: map[string]any{"access_token": "xai-token"},
	}

	_, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4",
		Payload: []byte(`{"model":"grok-4","input":"hello","reasoning":{"effort":"high"}}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       false,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gjson.GetBytes(gotBody, "reasoning").Exists() {
		t.Fatalf("unsupported xAI model must omit reasoning key: %s", string(gotBody))
	}
}

func TestXAIExecutorAppliesThinkingSuffix(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errRead error
		gotBody, errRead = io.ReadAll(r.Body)
		if errRead != nil {
			t.Fatalf("read body: %v", errRead)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":0,\"status\":\"completed\",\"model\":\"grok-4.3\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}]}}\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider: "xai",
		Attributes: map[string]string{
			"base_url":  server.URL,
			"auth_kind": "oauth",
		},
		Metadata: map[string]any{"access_token": "xai-token"},
	}

	_, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.3(low)",
		Payload: []byte(`{"model":"grok-4.3","input":"hello"}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       false,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := gjson.GetBytes(gotBody, "model").String(); got != "grok-4.3" {
		t.Fatalf("model = %q, want grok-4.3; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "reasoning.effort").String(); got != "low" {
		t.Fatalf("reasoning.effort = %q, want low; body=%s", got, string(gotBody))
	}
}

func TestXAIExecutorExecuteStreamFiltersToolSearchTool(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errRead error
		gotBody, errRead = io.ReadAll(r.Body)
		if errRead != nil {
			t.Fatalf("read body: %v", errRead)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":0,\"status\":\"completed\",\"model\":\"grok-4.3\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}]}}\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}

	result, err := exec.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.3",
		Payload: []byte(`{"model":"grok-4.3","input":[{"type":"reasoning","summary":[{"type":"summary_text","text":"test"}],"content":null,"encrypted_content":null},{"type":"reasoning","summary":[{"type":"summary_text","text":"second"}]},{"role":"user","content":"hello"},{"type":"reasoning","summary":[{"type":"summary_text","text":"separate"}]}],"tools":[{"type":"tool_search"},{"type":"image_generation"},{"type":"custom","name":"apply_patch"},{"type":"custom","name":"custom_lookup"},{"type":"function","name":"lookup"},{"type":"web_search","external_web_access":true,"search_content_types":["text","image"]},{"type":"namespace","name":"codex_app","description":"Tools in the codex_app namespace.","tools":[{"type":"function","name":"automation_update"},{"type":"custom","name":"namespace_custom"},{"type":"tool_search"}]}]}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       true,
	})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error = %v", chunk.Err)
		}
	}

	tools := gjson.GetBytes(gotBody, "tools").Array()
	if len(tools) != 5 {
		t.Fatalf("tools length = %d, want 5; body=%s", len(tools), string(gotBody))
	}
	if gjson.GetBytes(gotBody, "input.0.content").Exists() {
		t.Fatalf("input.0.content exists, want removed; body=%s", string(gotBody))
	}
	if gjson.GetBytes(gotBody, "input.0.encrypted_content").Exists() {
		t.Fatalf("input.0.encrypted_content exists, want removed; body=%s", string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "input.0.summary.0.text").String(); got != "test" {
		t.Fatalf("input.0.summary.0.text = %q, want test; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "input.0.summary.1.text").String(); got != "second" {
		t.Fatalf("input.0.summary.1.text = %q, want second; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "input.1.role").String(); got != "user" {
		t.Fatalf("input.1.role = %q, want user; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "input.2.summary.0.text").String(); got != "separate" {
		t.Fatalf("input.2.summary.0.text = %q, want separate; body=%s", got, string(gotBody))
	}
	foundAutomationUpdate := false
	foundNamespaceCustom := false
	for i, tool := range tools {
		toolType := tool.Get("type").String()
		if toolType == "image_generation" {
			t.Fatalf("tools.%d.type = image_generation, want removed; body=%s", i, string(gotBody))
		}
		if toolType != "function" && toolType != "web_search" {
			t.Fatalf("tools.%d.type = %q, want function or web_search; body=%s", i, toolType, string(gotBody))
		}
		if toolType == "function" && !tool.Get("parameters").Exists() {
			t.Fatalf("tools.%d.parameters missing for xAI function tool; body=%s", i, string(gotBody))
		}
		if got := tool.Get("name").String(); got == "apply_patch" {
			t.Fatalf("tools.%d.name = apply_patch, want removed; body=%s", i, string(gotBody))
		}
		switch tool.Get("name").String() {
		case "codex_app__automation_update":
			foundAutomationUpdate = true
		case "codex_app__namespace_custom":
			foundNamespaceCustom = true
		}
		if toolType == "web_search" {
			if tool.Get("external_web_access").Exists() {
				t.Fatalf("tools.%d.external_web_access exists, want removed; body=%s", i, string(gotBody))
			}
			if got := tool.Get("search_content_types.1").String(); got != "image" {
				t.Fatalf("tools.%d.search_content_types missing image entry; body=%s", i, string(gotBody))
			}
		}
	}
	if !foundAutomationUpdate {
		t.Fatalf("namespace function tool was not moved to top-level tools; body=%s", string(gotBody))
	}
	if !foundNamespaceCustom {
		t.Fatalf("namespace custom tool was not moved to top-level tools; body=%s", string(gotBody))
	}
}

func TestXAIExecutorAdditionalToolsNamespaceCustomToolDeclarationRoundTrip(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errRead error
		gotBody, errRead = io.ReadAll(r.Body)
		if errRead != nil {
			t.Fatalf("read body: %v", errRead)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","created_at":0,"status":"completed","model":"grok-4.3","output":[{"id":"fc_1","type":"function_call","call_id":"call_1","name":"mcp__stable","arguments":"patch"}]}}` + "\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}
	resp, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.3",
		Payload: []byte(`{"model":"grok-4.3","input":[{"role":"user","content":"hello"},{"type":"additional_tools","tools":[{"type":"namespace","name":"plugin","tools":[{"type":"custom","name":"mcp__stable"}]},{"type":"function","name":"extra"}]}],"tools":[{"type":"namespace","name":"app","tools":[{"type":"function","name":"lookup"},{"type":"function","name":"app__ready"}]}],"tool_choice":{"type":"allowed_tools","mode":"auto","tools":[{"type":"function","namespace":"app","name":"lookup"},{"type":"custom","namespace":"plugin","name":"mcp__stable"}]}}`),
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatOpenAIResponse})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gjson.GetBytes(gotBody, `input.#(type=="additional_tools")`).Exists() {
		t.Fatalf("additional_tools input was not promoted: %s", gotBody)
	}
	if got := gjson.GetBytes(gotBody, `tools.#(name=="app__lookup").name`).String(); got != "app__lookup" {
		t.Fatalf("namespaced function effective name = %q; body=%s", got, gotBody)
	}
	if got := gjson.GetBytes(gotBody, `tools.#(name=="app__ready").name`).String(); got != "app__ready" {
		t.Fatalf("already-qualified name = %q, want app__ready; body=%s", got, gotBody)
	}
	if strings.Contains(string(gotBody), "app__app__ready") {
		t.Fatalf("already-qualified tool was double-qualified: %s", gotBody)
	}
	if got := gjson.GetBytes(gotBody, `tools.#(name=="mcp__stable").name`).String(); got != "mcp__stable" {
		t.Fatalf("mcp name = %q, want byte-stable; body=%s", got, gotBody)
	}
	if got := gjson.GetBytes(gotBody, `tools.#(name=="mcp__stable").type`).String(); got != "function" {
		t.Fatalf("promoted custom tool type = %q, want function; body=%s", got, gotBody)
	}
	if got := gjson.GetBytes(gotBody, `tools.#(name=="extra").name`).String(); got != "extra" {
		t.Fatalf("promoted function missing; body=%s", gotBody)
	}
	if got := gjson.GetBytes(gotBody, "tool_choice.tools.0.name").String(); got != "app__lookup" {
		t.Fatalf("function choice name = %q, want app__lookup; body=%s", got, gotBody)
	}
	if got := gjson.GetBytes(gotBody, "tool_choice.tools.1.name").String(); got != "mcp__stable" {
		t.Fatalf("custom choice name = %q, want mcp__stable; body=%s", got, gotBody)
	}
	for index := range gjson.GetBytes(gotBody, "tool_choice.tools").Array() {
		if gjson.GetBytes(gotBody, fmt.Sprintf("tool_choice.tools.%d.namespace", index)).Exists() {
			t.Fatalf("choice %d retained namespace; body=%s", index, gotBody)
		}
		if got := gjson.GetBytes(gotBody, fmt.Sprintf("tool_choice.tools.%d.type", index)).String(); got != "function" {
			t.Fatalf("choice %d type = %q, want function; body=%s", index, got, gotBody)
		}
	}

	if got := gjson.GetBytes(resp.Payload, "output.0.type").String(); got != "custom_tool_call" {
		t.Fatalf("response tool type = %q, want custom_tool_call; payload=%s", got, resp.Payload)
	}
	if got := gjson.GetBytes(resp.Payload, "output.0.namespace").String(); got != "plugin" {
		t.Fatalf("response namespace = %q, want plugin; payload=%s", got, resp.Payload)
	}
	if got := gjson.GetBytes(resp.Payload, "output.0.name").String(); got != "mcp__stable" {
		t.Fatalf("response name = %q, want mcp__stable; payload=%s", got, resp.Payload)
	}
	if got := gjson.GetBytes(resp.Payload, "output.0.input").String(); got != "patch" {
		t.Fatalf("response input = %q, want patch; payload=%s", got, resp.Payload)
	}
	if gjson.GetBytes(resp.Payload, "output.0.arguments").Exists() {
		t.Fatalf("custom response retained arguments: %s", resp.Payload)
	}
}

func TestXAIExecutorToolNameCollisionRejectedBeforeNetwork(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}
	_, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.3",
		Payload: []byte(`{"model":"grok-4.3","input":[{"role":"user","content":"hello"},{"type":"additional_tools","tools":[{"type":"namespace","name":"team","tools":[{"type":"custom","name":"lookup"}]}]}],"tools":[{"type":"function","name":"team__lookup"}]}`),
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatOpenAIResponse})
	if err == nil {
		t.Fatal("Execute() error = nil, want tool_name_collision")
	}
	if requestCount != 0 {
		t.Fatalf("network requests = %d, want 0", requestCount)
	}
	statusProvider, ok := err.(interface{ StatusCode() int })
	if !ok || statusProvider.StatusCode() != http.StatusBadRequest {
		t.Fatalf("status = %v, want 400; err=%v", statusProvider, err)
	}
	if got := gjson.Get(err.Error(), "error.type").String(); got != "invalid_request_error" {
		t.Fatalf("error type = %q, want invalid_request_error; err=%v", got, err)
	}
	if got := gjson.Get(err.Error(), "error.code").String(); got != "tool_name_collision" {
		t.Fatalf("error code = %q, want tool_name_collision; err=%v", got, err)
	}
}

func TestXAIExecutorXSearchLifecycleUsesExactDeclarationIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		frames := []string{
			`event: response.output_item.added\ndata: {"type":"response.output_item.added","sequence_number":1,"output_index":0,"item":{"id":"internal_1","type":"custom_tool_call","call_id":"xs_call_internal","name":"x_keyword_search","input":"hidden"}}\n\n`,
			`event: response.custom_tool_call_input.delta\ndata: {"type":"response.custom_tool_call_input.delta","sequence_number":2,"output_index":0,"item_id":"internal_1","delta":"hidden"}\n\n`,
			`event: response.output_item.done\ndata: {"type":"response.output_item.done","sequence_number":3,"output_index":0,"item":{"id":"internal_1","type":"custom_tool_call","call_id":"xs_call_internal","name":"x_keyword_search","input":"hidden"}}\n\n`,
			`event: response.output_item.added\ndata: {"type":"response.output_item.added","sequence_number":4,"output_index":1,"item":{"id":"client_1","type":"function_call","call_id":"call_client","name":"search__x_keyword_search","arguments":""}}\n\n`,
			`event: response.function_call_arguments.delta\ndata: {"type":"response.function_call_arguments.delta","sequence_number":5,"output_index":1,"item_id":"client_1","delta":"{\"q\":\"x\"}"}\n\n`,
			`event: response.function_call_arguments.done\ndata: {"type":"response.function_call_arguments.done","sequence_number":6,"output_index":1,"item_id":"client_1","arguments":"{\"q\":\"x\"}"}\n\n`,
			`event: response.output_item.done\ndata: {"type":"response.output_item.done","sequence_number":7,"output_index":1,"item":{"id":"client_1","type":"function_call","call_id":"call_client","name":"search__x_keyword_search","arguments":"{\"q\":\"x\"}"}}\n\n`,
			`event: response.completed\ndata: {"type":"response.completed","sequence_number":8,"response":{"id":"resp_1","object":"response","created_at":0,"status":"completed","model":"grok-4.3","output":[{"id":"internal_1","type":"custom_tool_call","call_id":"xs_call_internal","name":"x_keyword_search","input":"hidden"},{"id":"client_1","type":"function_call","call_id":"call_client","name":"search__x_keyword_search","arguments":"{\"q\":\"x\"}"}]}}\n\n`,
		}
		for _, frame := range frames {
			_, _ = w.Write([]byte(strings.ReplaceAll(frame, `\n`, "\n")))
		}
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}
	result, err := exec.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.3",
		Payload: []byte(`{"model":"grok-4.3","input":"hello","tools":[{"type":"namespace","name":"search","tools":[{"type":"custom","name":"x_keyword_search"}]}]}`),
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatOpenAIResponse, Stream: true})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	var streamed bytes.Buffer
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error = %v", chunk.Err)
		}
		streamed.Write(chunk.Payload)
	}
	output := streamed.String()
	for _, hidden := range []string{"internal_1", "xs_call_internal", `"input":"hidden"`} {
		if strings.Contains(output, hidden) {
			t.Fatalf("stream retained internal x_search trace %q: %s", hidden, output)
		}
	}
	for _, want := range []string{
		`"id":"client_1"`,
		`"call_id":"call_client"`,
		`"namespace":"search"`,
		`"name":"x_keyword_search"`,
		`"type":"custom_tool_call"`,
		`"type":"response.custom_tool_call_input.delta"`,
		`"type":"response.custom_tool_call_input.done"`,
		`"input":"{\"q\":\"x\"}"`,
		`"output_index":0`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stream missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, `"output_index":1`) {
		t.Fatalf("stream retained uncompact output index: %s", output)
	}
	completedMarker := `"type":"response.completed"`
	completedIndex := strings.LastIndex(output, completedMarker)
	if completedIndex < 0 {
		t.Fatalf("stream missing completed event: %s", output)
	}
	completed := output[completedIndex:]
	if strings.Count(completed, `"call_id":"call_client"`) != 1 || strings.Contains(completed, "xs_call_internal") {
		t.Fatalf("completed output lifecycle was not filtered exactly: %s", completed)
	}
}

func TestXAIExecutorSameNameXSearchCallIdentityFiltering(t *testing.T) {
	writeResponse := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/event-stream")
		frames := []string{
			`event: response.output_item.added\ndata: {"type":"response.output_item.added","sequence_number":1,"output_index":0,"item":{"id":"internal_same","type":"function_call","call_id":"xs_call_internal","name":"x_keyword_search","arguments":""}}\n\n`,
			`event: response.function_call_arguments.delta\ndata: {"type":"response.function_call_arguments.delta","sequence_number":2,"output_index":0,"item_id":"internal_same","delta":"internal"}\n\n`,
			`event: response.function_call_arguments.done\ndata: {"type":"response.function_call_arguments.done","sequence_number":3,"output_index":0,"item_id":"internal_same","arguments":"internal"}\n\n`,
			`event: response.output_item.done\ndata: {"type":"response.output_item.done","sequence_number":4,"output_index":0,"item":{"id":"internal_same","type":"function_call","call_id":"xs_call_internal","name":"x_keyword_search","arguments":"internal"}}\n\n`,
			`event: response.output_item.added\ndata: {"type":"response.output_item.added","sequence_number":5,"output_index":1,"item":{"id":"client_same","type":"function_call","call_id":"call_client","name":"x_keyword_search","arguments":""}}\n\n`,
			`event: response.function_call_arguments.delta\ndata: {"type":"response.function_call_arguments.delta","sequence_number":6,"output_index":1,"item_id":"client_same","delta":"{\"q\":\"visible\"}"}\n\n`,
			`event: response.function_call_arguments.done\ndata: {"type":"response.function_call_arguments.done","sequence_number":7,"output_index":1,"item_id":"client_same","arguments":"{\"q\":\"visible\"}"}\n\n`,
			`event: response.output_item.done\ndata: {"type":"response.output_item.done","sequence_number":8,"output_index":1,"item":{"id":"client_same","type":"function_call","call_id":"call_client","name":"x_keyword_search","arguments":"{\"q\":\"visible\"}"}}\n\n`,
			`event: response.output_item.added\ndata: {"type":"response.output_item.added","sequence_number":9,"output_index":2,"item":{"id":"client_namespaced","type":"function_call","call_id":"xs_call_client","namespace":"search","name":"x_keyword_search","arguments":""}}\n\n`,
			`event: response.function_call_arguments.delta\ndata: {"type":"response.function_call_arguments.delta","sequence_number":10,"output_index":2,"item_id":"client_namespaced","delta":"{\"q\":\"namespaced\"}"}\n\n`,
			`event: response.function_call_arguments.done\ndata: {"type":"response.function_call_arguments.done","sequence_number":11,"output_index":2,"item_id":"client_namespaced","arguments":"{\"q\":\"namespaced\"}"}\n\n`,
			`event: response.output_item.done\ndata: {"type":"response.output_item.done","sequence_number":12,"output_index":2,"item":{"id":"client_namespaced","type":"function_call","call_id":"xs_call_client","namespace":"search","name":"x_keyword_search","arguments":"{\"q\":\"namespaced\"}"}}\n\n`,
			`event: response.output_item.added\ndata: {"type":"response.output_item.added","sequence_number":13,"output_index":3,"item":{"id":"message_1","type":"message","role":"assistant","content":[]}}\n\n`,
			`event: response.output_item.done\ndata: {"type":"response.output_item.done","sequence_number":14,"output_index":3,"item":{"id":"message_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}}\n\n`,
			`event: response.completed\ndata: {"type":"response.completed","sequence_number":15,"response":{"id":"resp_1","object":"response","created_at":0,"status":"completed","model":"grok-4.3","output":[{"id":"internal_same","type":"function_call","call_id":"xs_call_internal","name":"x_keyword_search","arguments":"internal"},{"id":"client_same","type":"function_call","call_id":"call_client","name":"x_keyword_search","arguments":"{\"q\":\"visible\"}"},{"id":"client_namespaced","type":"function_call","call_id":"xs_call_client","namespace":"search","name":"x_keyword_search","arguments":"{\"q\":\"namespaced\"}"},{"id":"message_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}}\n\n`,
		}
		for _, frame := range frames {
			_, _ = w.Write([]byte(strings.ReplaceAll(frame, `\n`, "\n")))
		}
	}
	request := cliproxyexecutor.Request{
		Model:   "grok-4.3",
		Payload: []byte(`{"model":"grok-4.3","input":"hello","tools":[{"type":"function","name":"x_keyword_search"},{"type":"namespace","name":"search","tools":[{"type":"function","name":"x_keyword_search"}]}]}`),
	}

	t.Run("streaming", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeResponse(w)
		}))
		defer server.Close()
		auth := &cliproxyauth.Auth{Provider: "xai", Attributes: map[string]string{"base_url": server.URL}, Metadata: map[string]any{"access_token": "xai-token"}}
		result, err := NewXAIExecutor(&config.Config{}).ExecuteStream(context.Background(), auth, request, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatOpenAIResponse, Stream: true})
		if err != nil {
			t.Fatalf("ExecuteStream() error = %v", err)
		}
		var streamed bytes.Buffer
		for chunk := range result.Chunks {
			if chunk.Err != nil {
				t.Fatalf("stream chunk error = %v", chunk.Err)
			}
			streamed.Write(chunk.Payload)
		}
		output := streamed.String()
		for _, hidden := range []string{"internal_same", "xs_call_internal", `"delta":"internal"`, `"arguments":"internal"`} {
			if strings.Contains(output, hidden) {
				t.Fatalf("stream retained internal same-name lifecycle %q: %s", hidden, output)
			}
		}
		for _, visible := range []string{
			`"output_index":0,"item":{"id":"client_same"`,
			`"output_index":0,"item_id":"client_same"`,
			`"call_id":"call_client"`,
			`"type":"response.output_item.added","sequence_number":9,"output_index":1,"item":{"id":"client_namespaced"`,
			`"type":"response.function_call_arguments.delta","sequence_number":10,"output_index":1,"item_id":"client_namespaced"`,
			`"type":"response.function_call_arguments.done","sequence_number":11,"output_index":1,"item_id":"client_namespaced"`,
			`"type":"response.output_item.done","sequence_number":12,"output_index":1,"item":{"id":"client_namespaced"`,
			`"call_id":"xs_call_client"`,
			`"namespace":"search"`,
			`"output_index":2,"item":{"id":"message_1"`,
			`"type":"response.completed"`,
		} {
			if !strings.Contains(output, visible) {
				t.Fatalf("stream missing visible same-name lifecycle %q: %s", visible, output)
			}
		}
		if strings.Contains(output, `"output_index":3`) {
			t.Fatalf("stream retained uncompacted later index: %s", output)
		}
	})

	t.Run("nonstream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeResponse(w)
		}))
		defer server.Close()
		auth := &cliproxyauth.Auth{Provider: "xai", Attributes: map[string]string{"base_url": server.URL}, Metadata: map[string]any{"access_token": "xai-token"}}
		resp, err := NewXAIExecutor(&config.Config{}).Execute(context.Background(), auth, request, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatOpenAIResponse})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if strings.Contains(string(resp.Payload), "internal_same") || strings.Contains(string(resp.Payload), "xs_call_internal") {
			t.Fatalf("nonstream output retained internal same-name item: %s", resp.Payload)
		}
		if got := len(gjson.GetBytes(resp.Payload, "output").Array()); got != 3 {
			t.Fatalf("nonstream output length = %d, want 3; payload=%s", got, resp.Payload)
		}
		if got := gjson.GetBytes(resp.Payload, "output.0.call_id").String(); got != "call_client" {
			t.Fatalf("nonstream legitimate call_id = %q, want call_client; payload=%s", got, resp.Payload)
		}
		if got := gjson.GetBytes(resp.Payload, "output.1.call_id").String(); got != "xs_call_client" {
			t.Fatalf("nonstream namespaced call_id = %q, want xs_call_client; payload=%s", got, resp.Payload)
		}
		if got := gjson.GetBytes(resp.Payload, "output.1.namespace").String(); got != "search" {
			t.Fatalf("nonstream namespaced identity = %q, want search; payload=%s", got, resp.Payload)
		}
		if got := gjson.GetBytes(resp.Payload, "output.2.id").String(); got != "message_1" {
			t.Fatalf("nonstream later visible item = %q, want message_1; payload=%s", got, resp.Payload)
		}
	})
}

func TestXAIExecutorExecuteStreamNormalizesReasoningTextEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_item.added\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_item.added\",\"sequence_number\":1,\"output_index\":0,\"item\":{\"id\":\"rs_1\",\"type\":\"reasoning\",\"status\":\"in_progress\",\"summary\":[]}}\n\n"))
		_, _ = w.Write([]byte("event: response.content_part.added\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.content_part.added\",\"sequence_number\":2,\"item_id\":\"rs_1\",\"output_index\":0,\"content_index\":0,\"part\":{\"type\":\"reasoning_text\",\"text\":\"\"}}\n\n"))
		_, _ = w.Write([]byte("event: response.reasoning_text.delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.reasoning_text.delta\",\"sequence_number\":3,\"item_id\":\"rs_1\",\"output_index\":0,\"content_index\":0,\"delta\":\"thinking\"}\n\n"))
		_, _ = w.Write([]byte("event: response.reasoning_text.done\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.reasoning_text.done\",\"sequence_number\":4,\"item_id\":\"rs_1\",\"output_index\":0,\"content_index\":0,\"text\":\"thinking\"}\n\n"))
		_, _ = w.Write([]byte("event: response.output_item.done\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_item.done\",\"sequence_number\":5,\"output_index\":0,\"item\":{\"id\":\"rs_1\",\"type\":\"reasoning\",\"status\":\"completed\",\"summary\":[],\"content\":[{\"type\":\"reasoning_text\",\"text\":\"thinking\"}]}}\n\n"))
		_, _ = w.Write([]byte("event: response.completed\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"sequence_number\":6,\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":0,\"status\":\"completed\",\"model\":\"grok-4.3\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}

	result, err := exec.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.3",
		Payload: []byte(`{"model":"grok-4.3","input":"hello"}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       true,
	})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}

	var streamed bytes.Buffer
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error = %v", chunk.Err)
		}
		streamed.Write(chunk.Payload)
	}
	output := streamed.String()
	if strings.Contains(output, "reasoning_text") {
		t.Fatalf("stream contains xAI reasoning_text shape: %s", output)
	}
	for _, want := range []string{
		"event: response.reasoning_summary_part.added",
		"event: response.reasoning_summary_text.delta",
		"event: response.reasoning_summary_text.done",
		"event: response.reasoning_summary_part.done",
		`"type":"response.reasoning_summary_part.added"`,
		`"type":"response.reasoning_summary_text.delta"`,
		`"type":"response.reasoning_summary_text.done"`,
		`"type":"response.reasoning_summary_part.done"`,
		`"part":{"type":"summary_text","text":"thinking"}`,
		`"summary_index":0`,
		`"summary":[{"type":"summary_text","text":"thinking"}]`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stream missing %q: %s", want, output)
		}
	}
	textDoneIndex := strings.Index(output, `"type":"response.reasoning_summary_text.done"`)
	partDoneIndex := strings.Index(output, `"type":"response.reasoning_summary_part.done"`)
	if textDoneIndex < 0 || partDoneIndex < 0 || textDoneIndex > partDoneIndex {
		t.Fatalf("reasoning done events are out of order: %s", output)
	}
}

func TestXAIExecutorExecuteNormalizesReasoningOutputForNonStreamTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_item.done\",\"sequence_number\":1,\"output_index\":0,\"item\":{\"id\":\"rs_1\",\"type\":\"reasoning\",\"status\":\"completed\",\"summary\":[],\"content\":[{\"type\":\"reasoning_text\",\"text\":\"thinking\"}]}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"sequence_number\":2,\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":0,\"status\":\"completed\",\"model\":\"grok-4.3\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}

	resp, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.3",
		Payload: []byte(`{"model":"grok-4.3","input":"hello"}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       false,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := string(resp.Payload)
	if strings.Contains(output, "reasoning_text") {
		t.Fatalf("nonstream output contains xAI reasoning_text shape: %s", output)
	}
	if !strings.Contains(output, "thinking") {
		t.Fatalf("nonstream output missing reasoning summary text: %s", output)
	}
}

func TestXAIExecutorExecuteImagesUsesImagesEndpoint(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotAccept string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		var errRead error
		gotBody, errRead = io.ReadAll(r.Body)
		if errRead != nil {
			t.Fatalf("read body: %v", errRead)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":123,"data":[{"b64_json":"AA=="}]}`))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider: "xai",
		Attributes: map[string]string{
			"base_url":  server.URL,
			"auth_kind": "oauth",
		},
		Metadata: map[string]any{"access_token": "xai-token"},
	}

	resp, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-imagine-image",
		Payload: []byte(`{"model":"grok-imagine-image","prompt":"draw"}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FromString("openai-image"),
		Metadata: map[string]any{
			cliproxyexecutor.RequestPathMetadataKey: "/v1/images/generations",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gotPath != "/images/generations" {
		t.Fatalf("path = %q, want /images/generations", gotPath)
	}
	if gotAuth != "Bearer xai-token" {
		t.Fatalf("Authorization = %q, want Bearer xai-token", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q, want application/json", gotAccept)
	}
	if string(gotBody) != `{"model":"grok-imagine-image","prompt":"draw"}` {
		t.Fatalf("body = %s", string(gotBody))
	}
	if gjson.GetBytes(resp.Payload, "data.0.b64_json").String() != "AA==" {
		t.Fatalf("payload = %s", string(resp.Payload))
	}
}

func TestXAIExecutorExecuteImagesUsesEditsEndpoint(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":123,"data":[{"url":"https://x.ai/image.png"}]}`))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}

	_, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-imagine-image",
		Payload: []byte(`{"model":"grok-imagine-image","prompt":"edit","image":{"type":"image_url","url":"https://example.com/a.png"}}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FromString("openai-image"),
		Metadata: map[string]any{
			cliproxyexecutor.RequestPathMetadataKey: "/v1/images/edits",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gotPath != "/images/edits" {
		t.Fatalf("path = %q, want /images/edits", gotPath)
	}
}

func TestXAIExecutorExecuteVideosCreate(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotAuth string
	var gotIdempotencyKey string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotIdempotencyKey = r.Header.Get("x-idempotency-key")
		var errRead error
		gotBody, errRead = io.ReadAll(r.Body)
		if errRead != nil {
			t.Fatalf("read body: %v", errRead)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"request_id":"vid_123"}`))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}

	resp, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-imagine-video",
		Payload: []byte(`{"model":"grok-imagine-video","prompt":"animate","duration":4}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FromString("openai-video"),
		Metadata: map[string]any{
			"idempotency_key": "idem-123",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/videos/generations" {
		t.Fatalf("path = %q, want /videos/generations", gotPath)
	}
	if gotAuth != "Bearer xai-token" {
		t.Fatalf("Authorization = %q, want Bearer xai-token", gotAuth)
	}
	if gotIdempotencyKey != "idem-123" {
		t.Fatalf("x-idempotency-key = %q, want idem-123", gotIdempotencyKey)
	}
	if string(gotBody) != `{"model":"grok-imagine-video","prompt":"animate","duration":4}` {
		t.Fatalf("body = %s", string(gotBody))
	}
	if gjson.GetBytes(resp.Payload, "request_id").String() != "vid_123" {
		t.Fatalf("payload = %s", string(resp.Payload))
	}
}

func TestXAIExecutorExecuteVideosRetrieve(t *testing.T) {
	var gotPath string
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"done","video":{"url":"https://vidgen.x.ai/video.mp4","duration":6},"model":"grok-imagine-video","progress":100}`))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "xai",
		Attributes: map[string]string{"base_url": server.URL},
		Metadata:   map[string]any{"access_token": "xai-token"},
	}

	resp, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-imagine-video",
		Payload: []byte(`{"request_id":"vid_123"}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FromString("openai-video"),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/videos/vid_123" {
		t.Fatalf("path = %q, want /videos/vid_123", gotPath)
	}
	if gjson.GetBytes(resp.Payload, "video.url").String() != "https://vidgen.x.ai/video.mp4" {
		t.Fatalf("payload = %s", string(resp.Payload))
	}
}

func TestXAIExecutorExecuteVideosUsesNativeEndpointFromRequestPath(t *testing.T) {
	tests := []struct {
		name        string
		requestPath string
		wantPath    string
	}{
		{
			name:        "generations",
			requestPath: "/v1/videos/generations",
			wantPath:    "/videos/generations",
		},
		{
			name:        "edits",
			requestPath: "/v1/videos/edits",
			wantPath:    "/videos/edits",
		},
		{
			name:        "extensions",
			requestPath: "/v1/videos/extensions",
			wantPath:    "/videos/extensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotMethod string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotMethod = r.Method
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"request_id":"vid_123"}`))
			}))
			defer server.Close()

			exec := NewXAIExecutor(&config.Config{})
			auth := &cliproxyauth.Auth{
				Provider:   "xai",
				Attributes: map[string]string{"base_url": server.URL},
				Metadata:   map[string]any{"access_token": "xai-token"},
			}

			_, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
				Model:   "grok-imagine-video",
				Payload: []byte(`{"model":"grok-imagine-video","prompt":"animate"}`),
			}, cliproxyexecutor.Options{
				SourceFormat: sdktranslator.FromString("openai-video"),
				Metadata: map[string]any{
					cliproxyexecutor.RequestPathMetadataKey: tt.requestPath,
				},
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if gotMethod != http.MethodPost {
				t.Fatalf("method = %q, want POST", gotMethod)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %s", gotPath, tt.wantPath)
			}
		})
	}
}

func TestXAIExecutorExecuteAcceptsResponseIncomplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"id\":\"rs_1\",\"type\":\"reasoning\",\"summary\":[]}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.incomplete\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"},\"output\":[],\"usage\":{\"input_tokens\":8,\"output_tokens\":1,\"total_tokens\":9}}}\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "xai-auth",
		Provider: "xai",
		Attributes: map[string]string{
			"base_url":  server.URL,
			"auth_kind": "oauth",
		},
		Metadata: map[string]any{
			"access_token": "xai-token",
		},
	}
	resp, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.5",
		Payload: []byte(`{"model":"grok-4.5","input":"hi","max_output_tokens":1}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       false,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := gjson.GetBytes(resp.Payload, "status").String(); got != "incomplete" {
		t.Fatalf("status = %q, want incomplete; payload=%s", got, resp.Payload)
	}
	if got := gjson.GetBytes(resp.Payload, "incomplete_details.reason").String(); got != "max_output_tokens" {
		t.Fatalf("incomplete reason = %q, want max_output_tokens; payload=%s", got, resp.Payload)
	}
}

func TestXAIExecutorExecuteStreamForwardsResponseIncomplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"id\":\"rs_1\",\"type\":\"reasoning\",\"summary\":[]}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.incomplete\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"},\"output\":[],\"usage\":{\"input_tokens\":8,\"output_tokens\":1,\"total_tokens\":9}}}\n\n"))
	}))
	defer server.Close()

	exec := NewXAIExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "xai-auth",
		Provider: "xai",
		Attributes: map[string]string{
			"base_url":  server.URL,
			"auth_kind": "oauth",
		},
		Metadata: map[string]any{
			"access_token": "xai-token",
		},
	}
	result, err := exec.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "grok-4.5",
		Payload: []byte(`{"model":"grok-4.5","input":"hi","stream":true,"max_output_tokens":1}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       true,
	})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error = %v", chunk.Err)
		}
	}
}

func TestXAISupportsNativeImageGeneration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model string
		want  bool
	}{
		{model: "", want: false},
		{model: "grok-4.5", want: false},
		{model: "grok-4.3", want: false},
		{model: "grok-4", want: false},
		{model: "grok-4.20-0309-reasoning", want: false},
		{model: "grok-4.20-multi-agent-0309", want: false},
		{model: "grok-build-0.1", want: false},
		{model: "grok-composer-2.5-fast", want: false},
		{model: "grok-3-mini", want: false},
		{model: "gpt-5.6", want: false},
		{model: "grok-4.6", want: true},
		{model: "grok-4.6(high)", want: true},
		{model: "xai/grok-4.6", want: true},
		{model: "grok-4.7", want: true},
		{model: "grok-5", want: true},
		{model: "grok-5.0", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			t.Parallel()
			if got := xaiSupportsNativeImageGeneration(tt.model); got != tt.want {
				t.Fatalf("xaiSupportsNativeImageGeneration(%q) = %t, want %t", tt.model, got, tt.want)
			}
		})
	}
}

func TestNormalizeXAITools_ImageGenerationByModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       []byte
		wantKeep   bool
		wantAction string
	}{
		{
			name:     "missing model still strips",
			body:     []byte(`{"tools":[{"type":"image_generation"},{"type":"web_search"}]}`),
			wantKeep: false,
		},
		{
			name:     "grok-4.5 strips",
			body:     []byte(`{"model":"grok-4.5","tools":[{"type":"image_generation"},{"type":"web_search"}]}`),
			wantKeep: false,
		},
		{
			name:     "grok-4.20 strips despite larger minor",
			body:     []byte(`{"model":"grok-4.20-0309-reasoning","tools":[{"type":"image_generation"},{"type":"web_search"}]}`),
			wantKeep: false,
		},
		{
			name:       "grok-4.6 keeps action",
			body:       []byte(`{"model":"grok-4.6","tools":[{"type":"image_generation","action":"generate"},{"type":"web_search"}]}`),
			wantKeep:   true,
			wantAction: "generate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := normalizeXAITools(tt.body)
			tools := gjson.GetBytes(out, "tools").Array()
			foundImage := false
			foundWebSearch := false
			var imageTool gjson.Result
			for _, tool := range tools {
				switch tool.Get("type").String() {
				case "image_generation":
					foundImage = true
					imageTool = tool
				case "web_search":
					foundWebSearch = true
				}
			}
			if !foundWebSearch {
				t.Fatalf("web_search missing; body=%s", out)
			}
			if foundImage != tt.wantKeep {
				t.Fatalf("image_generation kept=%t, want %t; body=%s", foundImage, tt.wantKeep, out)
			}
			if tt.wantKeep && tt.wantAction != "" {
				if got := imageTool.Get("action").String(); got != tt.wantAction {
					t.Fatalf("image_generation.action = %q, want %q; body=%s", got, tt.wantAction, out)
				}
			}
		})
	}
}

func TestXAIExecutorPrepareKeepsNativeImageGenerationForGrok46(t *testing.T) {
	t.Parallel()

	exec := NewXAIExecutor(&config.Config{})
	prepared, err := exec.prepareResponsesRequest(context.Background(), cliproxyexecutor.Request{
		Model: "grok-4.6",
		Payload: []byte(`{
			"model":"grok-4.6",
			"input":"draw a red circle",
			"tools":[{"type":"image_generation","action":"generate"}],
			"tool_choice":{"type":"image_generation"}
		}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       false,
	}, false)
	if err != nil {
		t.Fatalf("prepareResponsesRequest() error = %v", err)
	}

	tools := gjson.GetBytes(prepared.body, "tools").Array()
	if len(tools) != 1 {
		t.Fatalf("tools length = %d, want 1; body=%s", len(tools), prepared.body)
	}
	if got := tools[0].Get("type").String(); got != "image_generation" {
		t.Fatalf("tools.0.type = %q, want image_generation; body=%s", got, prepared.body)
	}
	if got := tools[0].Get("action").String(); got != "generate" {
		t.Fatalf("tools.0.action = %q, want generate; body=%s", got, prepared.body)
	}
	choice := gjson.GetBytes(prepared.body, "tool_choice")
	if got := choice.Get("type").String(); got != "allowed_tools" {
		t.Fatalf("tool_choice.type = %q, want allowed_tools; body=%s", got, prepared.body)
	}
	if got := choice.Get("mode").String(); got != "required" {
		t.Fatalf("tool_choice.mode = %q, want required; body=%s", got, prepared.body)
	}
	allowed := choice.Get("tools").Array()
	if len(allowed) != 1 {
		t.Fatalf("tool_choice.tools length = %d, want 1; body=%s", len(allowed), prepared.body)
	}
	if got := allowed[0].Get("type").String(); got != "image_generation" {
		t.Fatalf("tool_choice.tools.0.type = %q, want image_generation; body=%s", got, prepared.body)
	}
}

// Compact-shaped requests funnel through the same preparation path locally, so
// an orphaned tool_choice must be dropped there once normalization strips the
// referenced hosted tool (87fb01b23788).
func TestXAIExecutorPrepareDropsOrphanedImageGenerationToolChoice(t *testing.T) {
	t.Parallel()

	exec := NewXAIExecutor(&config.Config{})
	prepared, err := exec.prepareResponsesRequest(context.Background(), cliproxyexecutor.Request{
		Model: "grok-4.5",
		Payload: []byte(`{
			"model":"grok-4.5",
			"input":"compact this",
			"tools":[{"type":"image_generation","action":"generate"}],
			"tool_choice":{"type":"image_generation"},
			"parallel_tool_calls":true
		}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       false,
	}, false)
	if err != nil {
		t.Fatalf("prepareResponsesRequest() error = %v", err)
	}
	if gjson.GetBytes(prepared.body, "tools").Exists() {
		t.Fatalf("tools exists in prepared body: %s", prepared.body)
	}
	if gjson.GetBytes(prepared.body, "tool_choice").Exists() {
		t.Fatalf("orphaned tool_choice leaked into prepared body: %s", prepared.body)
	}
	if gjson.GetBytes(prepared.body, "parallel_tool_calls").Exists() {
		t.Fatalf("parallel_tool_calls exists in prepared body: %s", prepared.body)
	}
}
