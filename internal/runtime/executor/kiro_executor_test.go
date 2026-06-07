package executor

import (
	"bytes"
	"context"
	"encoding/binary"
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

func TestBuildKiroPayloadFromOpenAI_MessagesToolsAndSuffixes(t *testing.T) {
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
	if current["modelId"] != "claude-sonnet-4.5" {
		t.Fatalf("modelId = %#v, want claude-sonnet-4.5", current["modelId"])
	}
	content := current["content"].(string)
	if !strings.Contains(content, "<thinking_mode>enabled</thinking_mode>") {
		t.Fatalf("current content missing thinking tag: %q", content)
	}
	if !strings.Contains(content, "<max_thinking_length>16000</max_thinking_length>") {
		t.Fatalf("current content missing max thinking tag: %q", content)
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
	toolSchema := ctx["tools"].([]any)[0].(map[string]any)["toolSpecification"].(map[string]any)["inputSchema"].(map[string]any)["json"].(map[string]any)
	if got := toolSchema["type"]; got != "object" {
		t.Fatalf("tool schema type = %#v, want object", got)
	}
	if _, ok := toolSchema["properties"].(map[string]any); !ok {
		t.Fatalf("tool schema properties = %#v, want object map", toolSchema["properties"])
	}
	if required, ok := toolSchema["required"].([]any); !ok || len(required) != 0 {
		t.Fatalf("tool schema required = %#v, want empty array", toolSchema["required"])
	}
	history := state["history"].([]any)
	firstUser := history[0].(map[string]any)["userInputMessage"].(map[string]any)
	if !strings.Contains(firstUser["content"].(string), "system prompt") {
		t.Fatalf("first history content = %q, want system prompt", firstUser["content"])
	}
	if _, ok := firstUser["images"]; ok {
		t.Fatalf("images = %#v, want images omitted on Kiro path", firstUser["images"])
	}
	inferenceConfig := got["inferenceConfig"].(map[string]any)
	if inferenceConfig["maxTokens"].(float64) != 32000 {
		t.Fatalf("maxTokens = %#v, want 32000", inferenceConfig["maxTokens"])
	}
}

func TestBuildKiroPayloadFromOpenAI_RejectsLongToolDefinitionName(t *testing.T) {
	longName := "mcp__GitHub__check_if_a_repository_is_starred_by_the_authenticated_user"
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[{"role":"user","content":"hi"}],
		"tools":[{"type":"function","function":{"name":"` + longName + `","description":"tool","parameters":{"type":"object"}}}]
	}`)

	_, err := buildKiroPayloadFromOpenAI(payload, "claude-sonnet-4.5")
	if err == nil {
		t.Fatal("buildKiroPayloadFromOpenAI() error = nil, want bad request")
	}
	status, ok := err.(interface{ StatusCode() int })
	if !ok {
		t.Fatalf("error %T does not expose StatusCode()", err)
	}
	if got := status.StatusCode(); got != http.StatusBadRequest {
		t.Fatalf("StatusCode() = %d, want %d", got, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), longName) {
		t.Fatalf("error = %q, want long tool name", err.Error())
	}
	if !strings.Contains(err.Error(), "64") {
		t.Fatalf("error = %q, want Kiro limit detail", err.Error())
	}
}

func TestBuildKiroPayloadFromOpenAI_RejectsLongAssistantToolCallName(t *testing.T) {
	longName := "mcp__GitHub__check_if_a_person_is_followed_by_the_authenticated_user"
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"` + longName + `","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"done"}
		]
	}`)

	_, err := buildKiroPayloadFromOpenAI(payload, "claude-sonnet-4.5")
	if err == nil {
		t.Fatal("buildKiroPayloadFromOpenAI() error = nil, want bad request")
	}
	status, ok := err.(interface{ StatusCode() int })
	if !ok {
		t.Fatalf("error %T does not expose StatusCode()", err)
	}
	if got := status.StatusCode(); got != http.StatusBadRequest {
		t.Fatalf("StatusCode() = %d, want %d", got, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), longName) {
		t.Fatalf("error = %q, want long tool call name", err.Error())
	}
}

func TestBuildKiroPayloadFromOpenAI_NormalizesSchemaDefaults(t *testing.T) {
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[{"role":"user","content":"hi"}],
		"tools":[{"type":"function","function":{"name":"read_file","parameters":{"properties":{"path":{"type":"string"}}}}}]
	}`)

	body, err := buildKiroPayloadFromOpenAI(payload, "claude-sonnet-4.5")
	if err != nil {
		t.Fatalf("buildKiroPayloadFromOpenAI() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	current := got["conversationState"].(map[string]any)["currentMessage"].(map[string]any)["userInputMessage"].(map[string]any)
	ctx := current["userInputMessageContext"].(map[string]any)
	toolSchema := ctx["tools"].([]any)[0].(map[string]any)["toolSpecification"].(map[string]any)["inputSchema"].(map[string]any)["json"].(map[string]any)
	if got := toolSchema["type"]; got != "object" {
		t.Fatalf("tool schema type = %#v, want object", got)
	}
	if _, ok := toolSchema["properties"].(map[string]any); !ok {
		t.Fatalf("tool schema properties = %#v, want object map", toolSchema["properties"])
	}
	if required, ok := toolSchema["required"].([]any); !ok || len(required) != 0 {
		t.Fatalf("tool schema required = %#v, want empty array", toolSchema["required"])
	}
}

func TestBuildKiroPayloadFromOpenAI_MergesAdjacentUserHistoryTurns(t *testing.T) {
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[
			{"role":"user","content":"First question"},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.txt\"}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"file body"},
			{"role":"user","content":"Follow-up after tool"},
			{"role":"assistant","content":"Second answer"},
			{"role":"user","content":"Final user turn"}
		]
	}`)

	body, err := buildKiroPayloadFromOpenAI(payload, "claude-sonnet-4.5")
	if err != nil {
		t.Fatalf("buildKiroPayloadFromOpenAI() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	state := got["conversationState"].(map[string]any)
	history := state["history"].([]any)
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2 after flattening completed tool turn", len(history))
	}
	mergedUser := history[0].(map[string]any)["userInputMessage"].(map[string]any)
	content := mergedUser["content"].(string)
	if !strings.Contains(content, "Tool results:") || !strings.Contains(content, "[read_file] file body") {
		t.Fatalf("merged user content = %q, want flattened narrated tool results", content)
	}
	if _, ok := mergedUser["userInputMessageContext"]; ok {
		t.Fatalf("merged user context = %#v, want historical tool results flattened", mergedUser["userInputMessageContext"])
	}
	current := state["currentMessage"].(map[string]any)["userInputMessage"].(map[string]any)
	if got := current["content"].(string); !strings.Contains(got, "Final user turn") {
		t.Fatalf("current content = %q, want final user turn only", got)
	}
}

func TestBuildKiroPayloadFromOpenAI_SynthesizesToolsFromHistory(t *testing.T) {
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[
			{"role":"user","content":"First question"},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"edit_file","arguments":"{\"path\":\"a.txt\"}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"edited"},
			{"role":"user","content":"Continue"}
		]
	}`)

	body, err := buildKiroPayloadFromOpenAI(payload, "claude-sonnet-4.5")
	if err != nil {
		t.Fatalf("buildKiroPayloadFromOpenAI() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	current := got["conversationState"].(map[string]any)["currentMessage"].(map[string]any)["userInputMessage"].(map[string]any)
	ctx := current["userInputMessageContext"].(map[string]any)
	tools := ctx["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want synthesized tool", len(tools))
	}
	spec := tools[0].(map[string]any)["toolSpecification"].(map[string]any)
	if got := spec["name"]; got != "edit_file" {
		t.Fatalf("tool name = %#v, want edit_file", got)
	}
	schema := spec["inputSchema"].(map[string]any)["json"].(map[string]any)
	if required, ok := schema["required"].([]any); !ok || len(required) != 0 {
		t.Fatalf("synthesized required = %#v, want empty array", schema["required"])
	}
}

func TestBuildKiroRequest_ClaudeSourceSynthesizesToolsFromHistory(t *testing.T) {
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[
			{"role":"user","content":[{"type":"text","text":"First question"}]},
			{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"edit_file","input":{"path":"a.txt"}}]},
			{"role":"user","content":[
				{"type":"tool_result","tool_use_id":"call_1","content":[{"type":"text","text":"edited"}]},
				{"type":"text","text":"Continue"}
			]}
		],
		"stream":false
	}`)

	body, _, _, err := buildKiroRequest(cliproxyexecutor.Request{
		Model:   "claude-sonnet-4.5",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("claude")}, false)
	if err != nil {
		t.Fatalf("buildKiroRequest() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	current := got["conversationState"].(map[string]any)["currentMessage"].(map[string]any)["userInputMessage"].(map[string]any)
	ctx := current["userInputMessageContext"].(map[string]any)
	tools := ctx["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}
	if got := tools[0].(map[string]any)["toolSpecification"].(map[string]any)["name"]; got != "edit_file" {
		t.Fatalf("tool name = %#v, want edit_file", got)
	}
}

func TestBuildKiroRequest_ClaudeSourceDropsCurrentImages(t *testing.T) {
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[
			{"role":"user","content":[
				{"type":"text","text":"Inspect this image"},
				{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+cqRsAAAAASUVORK5CYII="}}
			]}
		],
		"tools":[
			{"name":"inspect_image","description":"Inspect image metadata","input_schema":{"properties":{"target":{"type":"string"}}}}
		],
		"stream":false
	}`)

	body, _, _, err := buildKiroRequest(cliproxyexecutor.Request{
		Model:   "claude-sonnet-4.5",
		Payload: payload,
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("claude")}, false)
	if err != nil {
		t.Fatalf("buildKiroRequest() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	current := got["conversationState"].(map[string]any)["currentMessage"].(map[string]any)["userInputMessage"].(map[string]any)
	if _, ok := current["images"]; ok {
		t.Fatalf("images = %#v, want images omitted on Kiro path", current["images"])
	}
}

func TestBuildKiroPayloadFromOpenAI_StripsHistoricalStructuredToolTurns(t *testing.T) {
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[
			{"role":"user","content":"First question"},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"edit_file","arguments":"{\"path\":\"a.txt\"}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"edited 1"},
			{"role":"user","content":"Continue 1"},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_2","type":"function","function":{"name":"edit_file","arguments":"{\"path\":\"b.txt\"}"}}]},
			{"role":"tool","tool_call_id":"call_2","content":"edited 2"}
		]
	}`)

	body, err := buildKiroPayloadFromOpenAI(payload, "claude-sonnet-4.5")
	if err != nil {
		t.Fatalf("buildKiroPayloadFromOpenAI() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	state := got["conversationState"].(map[string]any)
	history := state["history"].([]any)
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2 after stripping completed tool turn", len(history))
	}
	firstUser := history[0].(map[string]any)["userInputMessage"].(map[string]any)
	content := firstUser["content"].(string)
	if !strings.Contains(content, "Tool results:") {
		t.Fatalf("history user content = %q, want narrated tool results", content)
	}
	if !strings.Contains(content, "[edit_file] edited 1") {
		t.Fatalf("history user content = %q, want tool attribution", content)
	}
	if _, ok := firstUser["userInputMessageContext"]; ok {
		t.Fatalf("history user context = %#v, want historical tool results flattened", firstUser["userInputMessageContext"])
	}
	lastAssistant := history[1].(map[string]any)["assistantResponseMessage"].(map[string]any)
	if len(lastAssistant["toolUses"].([]any)) != 1 {
		t.Fatalf("active assistant toolUses = %#v, want one active tool turn", lastAssistant["toolUses"])
	}
	current := state["currentMessage"].(map[string]any)["userInputMessage"].(map[string]any)
	ctx := current["userInputMessageContext"].(map[string]any)
	if len(ctx["toolResults"].([]any)) != 1 {
		t.Fatalf("current toolResults = %#v, want active current tool result preserved", ctx["toolResults"])
	}
	if got := ctx["toolResults"].([]any)[0].(map[string]any)["toolUseId"]; got != "call_2" {
		t.Fatalf("current toolUseId = %#v, want call_2", got)
	}
}

func TestBuildKiroPayloadFromOpenAI_FlattensOrphanCurrentToolResults(t *testing.T) {
	payload := []byte(`{
		"model":"claude-sonnet-4.5",
		"messages":[
			{"role":"user","content":"First question"},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"edit_file","arguments":"{\"path\":\"a.txt\"}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"edited 1"},
			{"role":"assistant","content":"Done with that tool"},
			{"role":"tool","tool_call_id":"call_1","content":"edited 1 retry"}
		]
	}`)

	body, err := buildKiroPayloadFromOpenAI(payload, "claude-sonnet-4.5")
	if err != nil {
		t.Fatalf("buildKiroPayloadFromOpenAI() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	state := got["conversationState"].(map[string]any)
	current := state["currentMessage"].(map[string]any)["userInputMessage"].(map[string]any)
	content := current["content"].(string)
	if !strings.Contains(content, "Tool results:") {
		t.Fatalf("current content = %q, want orphan tool result flattened into content", content)
	}
	if !strings.Contains(content, "[edit_file] edited 1 retry") {
		t.Fatalf("current content = %q, want flattened tool attribution", content)
	}
	if ctx, ok := current["userInputMessageContext"].(map[string]any); ok {
		if _, hasResults := ctx["toolResults"]; hasResults {
			t.Fatalf("current toolResults = %#v, want orphan tool result removed from structured context", ctx["toolResults"])
		}
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

func TestKiroExecutorExecute_ParsesAWSEventStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		_, _ = w.Write(kiroTestEventFrame("assistantResponseEvent", `{"content":"hello"}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"read_file","input":{"path":"a.txt"}}`))
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
	if !bytes.Contains(resp.Payload, []byte(`"content":"hello"`)) {
		t.Fatalf("response payload = %s, want content hello", resp.Payload)
	}
	if !bytes.Contains(resp.Payload, []byte(`"tool_calls"`)) {
		t.Fatalf("response payload = %s, want tool_calls", resp.Payload)
	}
	if !bytes.Contains(resp.Payload, []byte(`"arguments":"{\"path\":\"a.txt\"}"`)) {
		t.Fatalf("response payload = %s, want JSON string tool arguments", resp.Payload)
	}
}

func TestKiroExecutorExecute_CollapsesFragmentedToolUseEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":{}}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":""}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":"{\"own"}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":"er\": "}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":"\"therealtinhtute\""}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":", "}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":"\"repo\""}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":": \"llmhub\"}"}`))
		_, _ = w.Write(kiroTestEventFrame("toolUseEvent", `{"toolUseId":"tool_1","name":"repo_starred","input":{}}`))
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

	var payload struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(resp.Payload, &payload); err != nil {
		t.Fatalf("unmarshal response payload: %v", err)
	}
	if len(payload.Choices) != 1 {
		t.Fatalf("choices = %d, want 1", len(payload.Choices))
	}
	toolCalls := payload.Choices[0].Message.ToolCalls
	if len(toolCalls) != 1 {
		t.Fatalf("tool_calls = %d, want 1; payload=%s", len(toolCalls), resp.Payload)
	}
	if got := toolCalls[0].Function.Name; got != "repo_starred" {
		t.Fatalf("tool name = %q, want repo_starred", got)
	}
	var args map[string]string
	if err := json.Unmarshal([]byte(toolCalls[0].Function.Arguments), &args); err != nil {
		t.Fatalf("unmarshal tool arguments: %v (raw=%q)", err, toolCalls[0].Function.Arguments)
	}
	if got := args["owner"]; got != "therealtinhtute" {
		t.Fatalf("owner = %q, want therealtinhtute", got)
	}
	if got := args["repo"]; got != "llmhub" {
		t.Fatalf("repo = %q, want llmhub", got)
	}
}

func TestKiroExecutorExecute_RefreshesAfter401(t *testing.T) {
	var generateCalls int
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"accessToken":"access-new","refreshToken":"refresh-new","profileArn":"arn:aws:codewhisperer:us-east-1:123456789012:profile/NEW","expiresIn":3600}`)
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
			_, _ = io.WriteString(w, `{"accessToken":"access-new","refreshToken":"refresh-new","profileArn":"arn:aws:codewhisperer:us-east-1:123456789012:profile/NEW","expiresIn":3600}`)
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
	if got := refreshed.Metadata["profile_arn"]; got != "arn:aws:codewhisperer:us-east-1:123456789012:profile/NEW" {
		t.Fatalf("refreshed profile_arn = %#v", got)
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

func kiroTestEventFrame(eventType, payload string) []byte {
	headers := kiroTestHeader(":event-type", eventType)
	payloadBytes := []byte(payload)
	totalLength := 12 + len(headers) + len(payloadBytes) + 4
	frame := make([]byte, totalLength)
	binary.BigEndian.PutUint32(frame[0:4], uint32(totalLength))
	binary.BigEndian.PutUint32(frame[4:8], uint32(len(headers)))
	copy(frame[12:], headers)
	copy(frame[12+len(headers):], payloadBytes)
	return frame
}

func kiroTestHeader(name, value string) []byte {
	out := make([]byte, 1+len(name)+1+2+len(value))
	offset := 0
	out[offset] = byte(len(name))
	offset++
	copy(out[offset:], name)
	offset += len(name)
	out[offset] = 7
	offset++
	binary.BigEndian.PutUint16(out[offset:offset+2], uint16(len(value)))
	offset += 2
	copy(out[offset:], value)
	return out
}
