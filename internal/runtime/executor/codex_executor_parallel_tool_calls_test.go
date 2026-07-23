package executor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func TestCodexDeclarationCollisionFailsBeforeNetworkIO(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	executor := NewCodexExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{"base_url": server.URL, "api_key": "test"}}
	request := cliproxyexecutor.Request{
		Model: "gpt-5.4",
		Payload: []byte(`{
			"model":"gpt-5.4",
			"input":"hello",
			"tools":[
				{"type":"function","name":"repo__read"},
				{"type":"namespace","name":"repo","tools":[{"type":"function","name":"read"}]}
			]
		}`),
	}
	opts := cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-response")}

	_, err := executor.Execute(context.Background(), auth, request, opts)
	if err == nil {
		t.Fatal("Execute() error = nil, want tool_name_collision")
	}
	status, ok := err.(interface{ StatusCode() int })
	if !ok || status.StatusCode() != http.StatusBadRequest {
		t.Fatalf("Execute() error = %T %v, want HTTP 400", err, err)
	}
	if got := gjson.Get(err.Error(), "error.type").String(); got != "invalid_request_error" {
		t.Fatalf("error.type = %q, want invalid_request_error: %v", got, err)
	}
	if got := gjson.Get(err.Error(), "error.code").String(); got != "tool_name_collision" {
		t.Fatalf("error.code = %q, want tool_name_collision: %v", got, err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("network requests = %d, want 0", got)
	}

	_, err = executor.ExecuteStream(context.Background(), auth, request, opts)
	if err == nil || gjson.Get(err.Error(), "error.code").String() != "tool_name_collision" {
		t.Fatalf("ExecuteStream() error = %v, want tool_name_collision", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("network requests after ExecuteStream = %d, want 0", got)
	}
}

func TestCodexExecutorDeclarationTableNormalizesAndRestoresNonStreamCalls(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[{"id":"fc_1","type":"function_call","call_id":"call_read","name":"repo__read","arguments":"{}"},{"id":"fc_2","type":"function_call","call_id":"call_patch","name":"repo__patch","arguments":"{\"input\":\"diff\"}"}]}}` + "\n\n"))
	}))
	defer server.Close()

	executor := NewCodexExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{"base_url": server.URL, "api_key": "test"}}
	payload := []byte(`{
		"model":"gpt-5.4",
		"tools":[{"type":"namespace","name":"repo","tools":[
			{"type":"function","name":"read","parameters":{"type":"object","properties":{"path":{"type":"string"}}}},
			{"type":"custom","name":"patch"},
			{"type":"function","name":"mcp__stable"}
		]}],
		"input":[{"type":"additional_tools","tools":[{"type":"function","name":"shell"}]},{"type":"message","role":"user","content":"hello"}]
	}`)

	response, err := executor.Execute(context.Background(), auth, cliproxyexecutor.Request{Model: "gpt-5.4", Payload: payload}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-response")})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	wantOutboundNames := []string{"repo__read", "repo__patch", "mcp__stable", "shell"}
	for i, want := range wantOutboundNames {
		if got := gjson.GetBytes(gotBody, "tools."+string(rune('0'+i))+".name").String(); got != want {
			t.Fatalf("outbound tools[%d].name = %q, want %q: %s", i, got, want, gotBody)
		}
	}
	if got := gjson.GetBytes(gotBody, "input.#").Int(); got != 1 {
		t.Fatalf("outbound input count = %d, want 1: %s", got, gotBody)
	}
	if got := gjson.GetBytes(response.Payload, "output.0.name").String(); got != "read" {
		t.Fatalf("function name = %q, want read: %s", got, response.Payload)
	}
	if got := gjson.GetBytes(response.Payload, "output.0.namespace").String(); got != "repo" {
		t.Fatalf("function namespace = %q, want repo: %s", got, response.Payload)
	}
	if got := gjson.GetBytes(response.Payload, "output.1.type").String(); got != "custom_tool_call" {
		t.Fatalf("custom type = %q, want custom_tool_call: %s", got, response.Payload)
	}
	if got := gjson.GetBytes(response.Payload, "output.1.name").String(); got != "patch" {
		t.Fatalf("custom name = %q, want patch: %s", got, response.Payload)
	}
	if got := gjson.GetBytes(response.Payload, "output.1.namespace").String(); got != "repo" {
		t.Fatalf("custom namespace = %q, want repo: %s", got, response.Payload)
	}
	if got := gjson.GetBytes(response.Payload, "output.1.input").String(); got != "diff" {
		t.Fatalf("custom input = %q, want diff: %s", got, response.Payload)
	}
}

func TestCodexExecutorDeclarationTableRestoresCustomToolStreamingLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_patch","name":"patch","arguments":""}}`,
			`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"input\":\"diff\"}"}`,
			`data: {"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":"{\"input\":\"diff\"}"}`,
			`data: {"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_patch","name":"patch","arguments":"{\"input\":\"diff\"}"}}`,
			`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[]}}`,
		}, "\n\n") + "\n\n"))
	}))
	defer server.Close()

	executor := NewCodexExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{"base_url": server.URL, "api_key": "test"}}
	payload := []byte(`{"model":"gpt-5.4","input":"hello","tools":[{"type":"custom","name":"patch"}]}`)
	result, err := executor.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{Model: "gpt-5.4", Payload: payload}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-response"), Stream: true})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}

	var all bytes.Buffer
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error = %v", chunk.Err)
		}
		all.Write(chunk.Payload)
	}
	got := all.String()
	for _, want := range []string{
		`"type":"response.custom_tool_call_input.delta"`,
		`"type":"response.custom_tool_call_input.done"`,
		`"type":"custom_tool_call"`,
		`"item_id":"ctc_1"`,
		`"id":"ctc_1"`,
		`"input":"diff"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stream missing %s: %s", want, got)
		}
	}
	for _, unwanted := range []string{
		`"type":"response.function_call_arguments.delta"`,
		`"type":"response.function_call_arguments.done"`,
		`"item_id":"fc_1"`,
		`"id":"fc_1"`,
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("stream retained %s: %s", unwanted, got)
		}
	}
}

func TestCodexExecutorDeclarationTableRestoresFragmentedCustomToolStreamingLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_patch","name":"patch","arguments":""}}`,
			`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"in"}`,
			`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"put\":\"d"}`,
			`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"if"}`,
			`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"f\"}"}`,
			`data: {"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":"{\"input\":\"diff\"}"}`,
			`data: {"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_patch","name":"patch","arguments":"{\"input\":\"diff\"}"}}`,
			`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[{"id":"fc_1","type":"function_call","call_id":"call_patch","name":"patch","arguments":"{\"input\":\"diff\"}"}]}}`,
		}, "\n\n") + "\n\n"))
	}))
	defer server.Close()

	executor := NewCodexExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{"base_url": server.URL, "api_key": "test"}}
	payload := []byte(`{"model":"gpt-5.4","input":"hello","tools":[{"type":"custom","name":"patch"}]}`)
	result, err := executor.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{Model: "gpt-5.4", Payload: payload}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-response"), Stream: true})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}

	var all bytes.Buffer
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error = %v", chunk.Err)
		}
		all.Write(chunk.Payload)
	}
	if got := collectCustomToolInputDeltas(all.String()); got != "diff" {
		t.Fatalf("custom input deltas = %q, want diff: %s", got, all.String())
	}
	if strings.Contains(collectCustomToolInputDeltas(all.String()), `{"input"`) {
		t.Fatalf("custom input deltas leaked wrapper JSON: %s", all.String())
	}
}

func TestCodexExecutorPreservesRawBraceAndIncompleteCustomToolInput(t *testing.T) {
	tests := []struct {
		name      string
		arguments string
		wantInput string
	}{
		{name: "raw brace text", arguments: `{diff --git a/file b/file`, wantInput: `{diff --git a/file b/file`},
		{name: "incomplete wrapper", arguments: `{"input":"diff`, wantInput: `{"input":"diff`},
		{name: "raw JSON with extra member", arguments: `{"input":"literal","other":1}`, wantInput: `{"input":"literal","other":1}`},
		{name: "exact compatibility wrapper", arguments: `{"input":"literal"}`, wantInput: "literal"},
		{name: "leading whitespace exact wrapper", arguments: " \n { \t\"input\" : \"snow \\u96ea\" } \r", wantInput: "snow 雪"},
		{name: "null input remains raw", arguments: `{"input":null}`, wantInput: `{"input":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				var events []string
				events = append(events, `data: {"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_patch","name":"patch","arguments":""}}`)
				for i := 0; i < len(tt.arguments); i++ {
					event := []byte(`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":""}`)
					event, _ = sjson.SetBytes(event, "delta", tt.arguments[i:i+1])
					events = append(events, "data: "+string(event))
				}
				done := []byte(`{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":""}`)
				done, _ = sjson.SetBytes(done, "arguments", tt.arguments)
				events = append(events, "data: "+string(done))
				itemDone := []byte(`{"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_patch","name":"patch","arguments":""}}`)
				itemDone, _ = sjson.SetBytes(itemDone, "item.arguments", tt.arguments)
				events = append(events, "data: "+string(itemDone))
				completed := []byte(`{"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[{"id":"fc_1","type":"function_call","call_id":"call_patch","name":"patch","arguments":""}]}}`)
				completed, _ = sjson.SetBytes(completed, "response.output.0.arguments", tt.arguments)
				events = append(events, "data: "+string(completed))
				_, _ = w.Write([]byte(strings.Join(events, "\n\n") + "\n\n"))
			}))
			defer server.Close()

			executor := NewCodexExecutor(&config.Config{})
			auth := &cliproxyauth.Auth{Attributes: map[string]string{"base_url": server.URL, "api_key": "test"}}
			payload := []byte(`{"model":"gpt-5.4","input":"hello","tools":[{"type":"custom","name":"patch"}]}`)
			result, err := executor.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{Model: "gpt-5.4", Payload: payload}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-response"), Stream: true})
			if err != nil {
				t.Fatalf("ExecuteStream() error = %v", err)
			}
			var all bytes.Buffer
			for chunk := range result.Chunks {
				if chunk.Err != nil {
					t.Fatalf("stream chunk error = %v", chunk.Err)
				}
				all.Write(chunk.Payload)
			}
			assertCustomToolInputLifecycle(t, all.String(), tt.wantInput)
		})
	}
}

func collectCustomToolInputDeltas(stream string) string {
	var result strings.Builder
	for {
		start := strings.Index(stream, "data:")
		if start < 0 {
			break
		}
		stream = stream[start+len("data:"):]
		end := strings.Index(stream, "data:")
		data := stream
		if end >= 0 {
			data = stream[:end]
			stream = stream[end:]
		} else {
			stream = ""
		}
		data = strings.TrimSpace(data)
		if gjson.Get(data, "type").String() == "response.custom_tool_call_input.delta" {
			result.WriteString(gjson.Get(data, "delta").String())
		}
	}
	return result.String()
}

func assertCustomToolInputLifecycle(t *testing.T, stream, want string) {
	t.Helper()
	if got := collectCustomToolInputDeltas(stream); got != want {
		t.Fatalf("custom input deltas = %q, want %q: %s", got, want, stream)
	}

	var doneInput, itemDoneInput, completedInput string
	for _, part := range strings.Split(stream, "data:") {
		data := strings.TrimSpace(part)
		switch gjson.Get(data, "type").String() {
		case "response.custom_tool_call_input.done":
			doneInput = gjson.Get(data, "input").String()
		case "response.output_item.done":
			if gjson.Get(data, "item.type").String() == "custom_tool_call" {
				itemDoneInput = gjson.Get(data, "item.input").String()
			}
		case "response.completed":
			for _, output := range gjson.Get(data, "response.output").Array() {
				if output.Get("type").String() == "custom_tool_call" {
					completedInput = output.Get("input").String()
				}
			}
		}
	}
	if doneInput != want {
		t.Fatalf("custom input done = %q, want %q: %s", doneInput, want, stream)
	}
	if itemDoneInput != want {
		t.Fatalf("output_item.done input = %q, want %q: %s", itemDoneInput, want, stream)
	}
	if completedInput != want {
		t.Fatalf("response.completed input = %q, want %q: %s", completedInput, want, stream)
	}
}

func TestCodexExecutorDeclarationTableRestoresStreamingCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_read","name":"repo__read","arguments":""}}`,
			`data: {"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_read","name":"repo__read","arguments":"{}"}}`,
			`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[{"id":"fc_1","type":"function_call","call_id":"call_read","name":"repo__read","arguments":"{}"}]}}`,
		}, "\n\n") + "\n\n"))
	}))
	defer server.Close()

	executor := NewCodexExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{"base_url": server.URL, "api_key": "test"}}
	payload := []byte(`{"model":"gpt-5.4","input":"hello","tools":[{"type":"namespace","name":"repo","tools":[{"type":"function","name":"read"}]}]}`)
	result, err := executor.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{Model: "gpt-5.4", Payload: payload}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-response"), Stream: true})
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}

	var all bytes.Buffer
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error = %v", chunk.Err)
		}
		all.Write(chunk.Payload)
	}
	if !strings.Contains(all.String(), `"name":"read"`) || !strings.Contains(all.String(), `"namespace":"repo"`) {
		t.Fatalf("stream did not restore exact namespace/name: %s", all.String())
	}
	if strings.Contains(all.String(), `"name":"repo__read"`) {
		t.Fatalf("stream retained effective name: %s", all.String())
	}
}
