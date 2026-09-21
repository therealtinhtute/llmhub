package executor

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	coreusage "github.com/therealtinhtute/llmhub/sdk/cliproxy/usage"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

type kimiRoundTripperFunc func(req *http.Request) (*http.Response, error)

func (f kimiRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNormalizeKimiToolMessageLinks_UsesCallIDFallback(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","tool_calls":[{"id":"list_directory:1","type":"function","function":{"name":"list_directory","arguments":"{}"}}]},
			{"role":"tool","call_id":"list_directory:1","content":"[]"}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	got := gjson.GetBytes(out, "messages.1.tool_call_id").String()
	if got != "list_directory:1" {
		t.Fatalf("messages.1.tool_call_id = %q, want %q", got, "list_directory:1")
	}
}

func TestNormalizeKimiToolMessageLinks_InferSinglePendingID(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","tool_calls":[{"id":"call_123","type":"function","function":{"name":"read_file","arguments":"{}"}}]},
			{"role":"tool","content":"file-content"}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	got := gjson.GetBytes(out, "messages.1.tool_call_id").String()
	if got != "call_123" {
		t.Fatalf("messages.1.tool_call_id = %q, want %q", got, "call_123")
	}
}

func TestNormalizeKimiToolMessageLinks_AmbiguousMissingIDIsNotInferred(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","tool_calls":[
				{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}},
				{"id":"call_2","type":"function","function":{"name":"read_file","arguments":"{}"}}
			]},
			{"role":"tool","content":"result-without-id"}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	if gjson.GetBytes(out, "messages.1.tool_call_id").Exists() {
		t.Fatalf("messages.1.tool_call_id should be absent for ambiguous case, got %q", gjson.GetBytes(out, "messages.1.tool_call_id").String())
	}
}

func TestNormalizeKimiToolMessageLinks_PreservesExistingToolCallID(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"call_1","call_id":"different-id","content":"result"}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	got := gjson.GetBytes(out, "messages.1.tool_call_id").String()
	if got != "call_1" {
		t.Fatalf("messages.1.tool_call_id = %q, want %q", got, "call_1")
	}
}

func TestNormalizeKimiToolMessageLinks_InheritsPreviousReasoningForAssistantToolCalls(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","content":"plan","reasoning_content":"previous reasoning"},
			{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}}]}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	got := gjson.GetBytes(out, "messages.1.reasoning_content").String()
	if got != "previous reasoning" {
		t.Fatalf("messages.1.reasoning_content = %q, want %q", got, "previous reasoning")
	}
}

func TestNormalizeKimiToolMessageLinks_InsertsFallbackReasoningWhenMissing(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}}]}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	reasoning := gjson.GetBytes(out, "messages.0.reasoning_content")
	if !reasoning.Exists() {
		t.Fatalf("messages.0.reasoning_content should exist")
	}
	if reasoning.String() != "[reasoning unavailable]" {
		t.Fatalf("messages.0.reasoning_content = %q, want %q", reasoning.String(), "[reasoning unavailable]")
	}
}

func TestNormalizeKimiToolMessageLinks_UsesContentAsReasoningFallback(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","content":[{"type":"text","text":"first line"},{"type":"text","text":"second line"}],"tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}}]}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	got := gjson.GetBytes(out, "messages.0.reasoning_content").String()
	if got != "first line\nsecond line" {
		t.Fatalf("messages.0.reasoning_content = %q, want %q", got, "first line\nsecond line")
	}
}

func TestNormalizeKimiToolMessageLinks_ReplacesEmptyReasoningContent(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","content":"assistant summary","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}}],"reasoning_content":""}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	got := gjson.GetBytes(out, "messages.0.reasoning_content").String()
	if got != "assistant summary" {
		t.Fatalf("messages.0.reasoning_content = %q, want %q", got, "assistant summary")
	}
}

func TestNormalizeKimiToolMessageLinks_PreservesExistingAssistantReasoning(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}}],"reasoning_content":"keep me"}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	got := gjson.GetBytes(out, "messages.0.reasoning_content").String()
	if got != "keep me" {
		t.Fatalf("messages.0.reasoning_content = %q, want %q", got, "keep me")
	}
}

func TestNormalizeKimiToolMessageLinks_RepairsIDsAndReasoningTogether(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}}],"reasoning_content":"r1"},
			{"role":"tool","call_id":"call_1","content":"[]"},
			{"role":"assistant","tool_calls":[{"id":"call_2","type":"function","function":{"name":"read_file","arguments":"{}"}}]},
			{"role":"tool","call_id":"call_2","content":"file"}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	if got := gjson.GetBytes(out, "messages.1.tool_call_id").String(); got != "call_1" {
		t.Fatalf("messages.1.tool_call_id = %q, want %q", got, "call_1")
	}
	if got := gjson.GetBytes(out, "messages.3.tool_call_id").String(); got != "call_2" {
		t.Fatalf("messages.3.tool_call_id = %q, want %q", got, "call_2")
	}
	if got := gjson.GetBytes(out, "messages.2.reasoning_content").String(); got != "r1" {
		t.Fatalf("messages.2.reasoning_content = %q, want %q", got, "r1")
	}
}

func TestNormalizeKimiToolMessageLinks_DropsEmptyAssistantWithoutToolLink(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"user","content":"start"},
			{"role":"assistant","content":""},
			{"role":"assistant","content":"   "},
			{"role":"assistant","content":"","tool_calls":null},
			{"role":"assistant","content":[{"type":"text","text":"  "}]},
			{"role":"assistant"},
			{"role":"assistant","content":"keep"},
			{"role":"user","content":"next"}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	messages := gjson.GetBytes(out, "messages").Array()
	if len(messages) != 3 {
		t.Fatalf("messages length = %d, want 3, raw = %s", len(messages), gjson.GetBytes(out, "messages").Raw)
	}
	if got := messages[0].Get("content").String(); got != "start" {
		t.Fatalf("messages.0.content = %q, want %q", got, "start")
	}
	if got := messages[1].Get("content").String(); got != "keep" {
		t.Fatalf("messages.1.content = %q, want %q", got, "keep")
	}
	if got := messages[2].Get("content").String(); got != "next" {
		t.Fatalf("messages.2.content = %q, want %q", got, "next")
	}
}

func TestNormalizeKimiToolMessageLinks_PreservesAssistantWithToolLinkOrReasoning(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_directory","arguments":"{}"}}]},
			{"role":"assistant","content":"","function_call":{"name":"legacy_call","arguments":"{}"}},
			{"role":"assistant","content":"","reasoning_content":"thought"},
			{"role":"assistant","content":[{"type":"text","text":" visible "}]}
		]
	}`)

	out, err := normalizeKimiToolMessageLinks(body)
	if err != nil {
		t.Fatalf("normalizeKimiToolMessageLinks() error = %v", err)
	}

	messages := gjson.GetBytes(out, "messages").Array()
	if len(messages) != 4 {
		t.Fatalf("messages length = %d, want 4, raw = %s", len(messages), gjson.GetBytes(out, "messages").Raw)
	}
	if !messages[0].Get("tool_calls").Exists() {
		t.Fatalf("messages.0.tool_calls should exist")
	}
	if !messages[1].Get("function_call").Exists() {
		t.Fatalf("messages.1.function_call should exist")
	}
	if got := messages[2].Get("reasoning_content").String(); got != "thought" {
		t.Fatalf("messages.2.reasoning_content = %q, want %q", got, "thought")
	}
	if got := messages[3].Get("content.0.text").String(); got != " visible " {
		t.Fatalf("messages.3.content.0.text = %q, want %q", got, " visible ")
	}
}

func TestKimiExecutorResponsesPassthrough(t *testing.T) {
	var upstreamURL string
	var upstreamBody []byte
	var authHeader string

	ctx := context.WithValue(context.Background(), "cliproxy.roundtripper", kimiRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		upstreamURL = req.URL.String()
		authHeader = req.Header.Get("Authorization")
		var errRead error
		upstreamBody, errRead = io.ReadAll(req.Body)
		if errRead != nil {
			return nil, errRead
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"resp_123","object":"response","status":"completed","model":"k3","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello world"}]}],"usage":{"total_tokens":10,"input_tokens":6,"output_tokens":4}}`,
			)),
		}, nil
	}))

	executor := NewKimiExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Attributes: map[string]string{},
		Metadata:   map[string]any{"access_token": "test-kimi-key"},
	}

	payload := []byte(`{
		"model":"kimi-k3",
		"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]
	}`)

	resp, err := executor.Execute(ctx, auth, cliproxyexecutor.Request{
		Model:   "kimi-k3",
		Payload: payload,
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if upstreamURL != "https://api.kimi.com/coding/v1/responses" {
		t.Fatalf("upstreamURL = %q, want %q", upstreamURL, "https://api.kimi.com/coding/v1/responses")
	}
	if authHeader != "Bearer test-kimi-key" {
		t.Fatalf("Authorization = %q, want Bearer test-kimi-key", authHeader)
	}
	if gotModel := gjson.GetBytes(upstreamBody, "model").String(); gotModel != "k3" {
		t.Fatalf("upstreamBody model = %q, want k3", gotModel)
	}
	if gotText := gjson.GetBytes(resp.Payload, "output.0.content.0.text").String(); gotText != "hello world" {
		t.Fatalf("response output text = %q, want hello world", gotText)
	}
}

func TestKimiExecutorResponsesStreamPassthrough(t *testing.T) {
	var upstreamURL string
	var upstreamBody []byte
	var streamHeader string

	ctx := context.WithValue(context.Background(), "cliproxy.roundtripper", kimiRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		upstreamURL = req.URL.String()
		streamHeader = req.Header.Get("Accept")
		var errRead error
		upstreamBody, errRead = io.ReadAll(req.Body)
		if errRead != nil {
			return nil, errRead
		}
		sseData := "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_stream\",\"status\":\"in_progress\",\"service_tier\":\"default\",\"model\":\"k3\"}}\n\n" +
			"event: response.output_item.added\ndata: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[]}}\n\n" +
			"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"output_index\":0,\"content_index\":0,\"delta\":\"hello stream\"}\n\n" +
			"event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_stream\",\"status\":\"completed\",\"usage\":{\"total_tokens\":12,\"input_tokens\":5,\"output_tokens\":7}}}\n\n"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(sseData)),
		}, nil
	}))

	executor := NewKimiExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Attributes: map[string]string{},
		Metadata:   map[string]any{"access_token": "test-kimi-key"},
	}

	payload := []byte(`{
		"model":"kimi-k3",
		"stream":true,
		"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]
	}`)

	result, err := executor.ExecuteStream(ctx, auth, cliproxyexecutor.Request{
		Model:   "kimi-k3",
		Payload: payload,
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Stream:       true,
	})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}

	if upstreamURL != "https://api.kimi.com/coding/v1/responses" {
		t.Fatalf("upstreamURL = %q, want %q", upstreamURL, "https://api.kimi.com/coding/v1/responses")
	}
	if streamHeader != "text/event-stream" {
		t.Fatalf("Accept header = %q, want text/event-stream", streamHeader)
	}
	if gotModel := gjson.GetBytes(upstreamBody, "model").String(); gotModel != "k3" {
		t.Fatalf("upstreamBody model = %q, want k3", gotModel)
	}

	var chunks []string
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error = %v", chunk.Err)
		}
		chunks = append(chunks, string(chunk.Payload))
	}
	combined := strings.Join(chunks, "")
	if !strings.Contains(combined, "event: response.created") {
		t.Fatalf("stream chunks missing response.created: %s", combined)
	}
	if !strings.Contains(combined, "event: response.completed") {
		t.Fatalf("stream chunks missing response.completed: %s", combined)
	}
	if !strings.Contains(combined, "hello stream") {
		t.Fatalf("stream chunks missing expected text: %s", combined)
	}
}

func TestKimiExecutorResponsesCompactReturnsNotImplemented(t *testing.T) {
	executor := NewKimiExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Attributes: map[string]string{},
		Metadata:   map[string]any{"access_token": "test-kimi-key"},
	}

	_, err := executor.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "kimi-k3",
		Payload: []byte(`{"model":"kimi-k3","input":[]}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
		Alt:          "responses/compact",
	})
	if err == nil {
		t.Fatal("expected error for responses/compact")
	}
	statusErr, ok := err.(interface{ StatusCode() int })
	if !ok {
		t.Fatalf("error type = %T, want status error", err)
	}
	if statusErr.StatusCode() != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", statusErr.StatusCode(), http.StatusNotImplemented)
	}
}
func TestKimiExecutor_MappedModelDoesNotWarnWhenUpstreamServesMappedModel(t *testing.T) {
	ctx := context.WithValue(context.Background(), "cliproxy.roundtripper", kimiRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if gjson.GetBytes(body, "model").String() != "kimi-for-coding" {
			t.Fatalf("upstream request model = %q, want kimi-for-coding", gjson.GetBytes(body, "model").String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"chatcmpl-123","object":"chat.completion","model":"kimi-for-coding","choices":[{"message":{"role":"assistant","content":"hello"}}],"usage":{"total_tokens":10}}`,
			)),
		}, nil
	}))

	const alias = "kimi-mapped-no-warn-test"
	capture := &multiProviderUsageCapture{alias: alias, records: make(chan coreusage.Record, 4)}
	coreusage.RegisterNamedPlugin(t.Name(), capture)
	t.Cleanup(func() {
		coreusage.RegisterNamedPlugin(t.Name(), multiProviderNoopUsagePlugin{})
	})

	hook := new(logtest.Hook)
	log.StandardLogger().AddHook(hook)
	t.Cleanup(func() {
		log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
	})

	executor := NewKimiExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "kimi",
		Attributes: map[string]string{},
		Metadata:   map[string]any{"access_token": "test-key"},
	}

	ctx = coreusage.WithRequestedModelAlias(ctx, alias)
	payload := []byte(`{"model":"kimi-k2.7-code","messages":[{"role":"user","content":"hello"}]}`)
	resp, err := executor.Execute(ctx, auth, cliproxyexecutor.Request{
		Model:   "kimi-k2.7-code",
		Payload: payload,
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAI,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Payload) == 0 {
		t.Fatal("expected non-empty payload")
	}

	record := capture.await(t)
	if record.Model != "kimi-k2.7-code" {
		t.Fatalf("record.Model = %q, want kimi-k2.7-code (must preserve requested model)", record.Model)
	}
	if record.ResponseModel != "kimi-for-coding" {
		t.Fatalf("record.ResponseModel = %q, want kimi-for-coding", record.ResponseModel)
	}

	for _, entry := range hook.AllEntries() {
		if entry.Level == log.WarnLevel && strings.Contains(entry.Message, "upstream served model") {
			t.Fatalf("unexpected model substitution warning for intentional mapping: %s", entry.Message)
		}
	}
}

func TestKimiExecutor_WarnsWhenUpstreamServesUnexpectedModel(t *testing.T) {
	ctx := context.WithValue(context.Background(), "cliproxy.roundtripper", kimiRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"chatcmpl-123","object":"chat.completion","model":"unexpected-model-v2","choices":[{"message":{"role":"assistant","content":"hello"}}],"usage":{"total_tokens":10}}`,
			)),
		}, nil
	}))

	const alias = "kimi-unexpected-warn-test"
	capture := &multiProviderUsageCapture{alias: alias, records: make(chan coreusage.Record, 4)}
	coreusage.RegisterNamedPlugin(t.Name(), capture)
	t.Cleanup(func() {
		coreusage.RegisterNamedPlugin(t.Name(), multiProviderNoopUsagePlugin{})
	})

	hook := new(logtest.Hook)
	log.StandardLogger().AddHook(hook)
	t.Cleanup(func() {
		log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
	})

	executor := NewKimiExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider:   "kimi",
		Attributes: map[string]string{},
		Metadata:   map[string]any{"access_token": "test-key"},
	}

	ctx = coreusage.WithRequestedModelAlias(ctx, alias)
	payload := []byte(`{"model":"kimi-k2.7-code","messages":[{"role":"user","content":"hello"}]}`)
	resp, err := executor.Execute(ctx, auth, cliproxyexecutor.Request{
		Model:   "kimi-k2.7-code",
		Payload: payload,
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAI,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Payload) == 0 {
		t.Fatal("expected non-empty payload")
	}

	record := capture.await(t)
	if record.Model != "kimi-k2.7-code" {
		t.Fatalf("record.Model = %q, want kimi-k2.7-code", record.Model)
	}
	if record.ResponseModel != "unexpected-model-v2" {
		t.Fatalf("record.ResponseModel = %q, want unexpected-model-v2", record.ResponseModel)
	}

	var foundWarning bool
	for _, entry := range hook.AllEntries() {
		if entry.Level == log.WarnLevel && strings.Contains(entry.Message, "upstream served model") && strings.Contains(entry.Message, "unexpected-model-v2") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("expected substitution warning in logs for unexpected-model-v2")
	}
}
