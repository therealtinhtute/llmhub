package responses

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func parseClaudeResponsesSSEEvent(t *testing.T, chunk []byte) (string, gjson.Result) {
	t.Helper()

	var event string
	var data string
	for _, line := range strings.Split(string(chunk), "\n") {
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		}
	}
	if data == "" {
		t.Fatalf("SSE chunk has no data line: %s", string(chunk))
	}

	return event, gjson.Parse(data)
}

func TestConvertClaudeResponseToOpenAIResponses_ThinkingIncludesSignature(t *testing.T) {
	signature := "claude_sig_123"
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_123","usage":{"input_tokens":1,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"internal "}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"reasoning"}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"` + signature + `"}}`),
		[]byte(`data: {"type":"content_block_stop","index":0}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	var outputs [][]byte
	for _, chunk := range chunks {
		outputs = append(outputs, ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-test", nil, nil, chunk, &param)...)
	}

	var reasoningDone gjson.Result
	var completed gjson.Result
	for _, output := range outputs {
		event, data := parseClaudeResponsesSSEEvent(t, output)
		switch event {
		case "response.output_item.done":
			if data.Get("item.type").String() == "reasoning" {
				reasoningDone = data
			}
		case "response.completed":
			completed = data
		}
	}

	if !reasoningDone.Exists() {
		t.Fatal("expected reasoning output_item.done event")
	}
	if got := reasoningDone.Get("item.encrypted_content").String(); got != signature {
		t.Fatalf("reasoning encrypted_content = %q, want %q", got, signature)
	}
	if got := reasoningDone.Get("item.summary.0.text").String(); got != "internal reasoning" {
		t.Fatalf("reasoning summary text = %q", got)
	}
	if got := completed.Get("response.output.0.encrypted_content").String(); got != signature {
		t.Fatalf("completed reasoning encrypted_content = %q, want %q", got, signature)
	}
	if got := completed.Get("response.output.0.summary.0.text").String(); got != "internal reasoning" {
		t.Fatalf("completed reasoning summary text = %q", got)
	}
}

func TestConvertClaudeResponseToOpenAIResponsesNonStream_ThinkingIncludesSignature(t *testing.T) {
	signature := "claude_sig_nonstream"
	raw := []byte(strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"msg_nonstream","usage":{"input_tokens":1,"output_tokens":0}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"nonstream reasoning"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"` + signature + `"}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"message_stop"}`,
	}, "\n"))

	out := ConvertClaudeResponseToOpenAIResponsesNonStream(context.Background(), "claude-test", nil, nil, raw, nil)
	root := gjson.ParseBytes(out)

	if got := root.Get("output.0.encrypted_content").String(); got != signature {
		t.Fatalf("non-stream reasoning encrypted_content = %q, want %q", got, signature)
	}
	if got := root.Get("output.0.summary.0.text").String(); got != "nonstream reasoning" {
		t.Fatalf("non-stream reasoning summary text = %q", got)
	}
}

func TestConvertClaudeResponseToOpenAIResponsesNonStream_PreservesContentBlockOrder(t *testing.T) {
	raw := []byte(strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"msg_nonstream_order","usage":{"input_tokens":1,"output_tokens":0}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"thinking","thinking":""}}`,
		`data: {"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"call_order","name":"exec_command","input":{}}}`,
		`data: {"type":"content_block_start","index":3,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"thinking_delta","thinking":"plan"}}`,
		`data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\"cmd\":\"pwd\"}"}}`,
		`data: {"type":"content_block_delta","index":3,"delta":{"type":"text_delta","text":"done"}}`,
		`data: {"type":"content_block_stop","index":1}`,
		`data: {"type":"content_block_stop","index":2}`,
		`data: {"type":"content_block_stop","index":3}`,
		`data: {"type":"content_block_start","index":4,"content_block":{"type":"thinking","thinking":""}}`,
		`data: {"type":"content_block_delta","index":4,"delta":{"type":"thinking_delta","thinking":"more"}}`,
		`data: {"type":"content_block_stop","index":4}`,
		`data: {"type":"message_stop"}`,
	}, "\n"))

	root := gjson.ParseBytes(ConvertClaudeResponseToOpenAIResponsesNonStream(context.Background(), "claude-test", nil, nil, raw, nil))
	wantTypes := []string{"message", "reasoning", "function_call", "message", "reasoning"}
	if got := root.Get("output.#").Int(); got != int64(len(wantTypes)) {
		t.Fatalf("non-stream output count = %d, want %d", got, len(wantTypes))
	}
	for index, wantType := range wantTypes {
		if got := root.Get(fmt.Sprintf("output.%d.type", index)).String(); got != wantType {
			t.Fatalf("non-stream output.%d.type = %q, want %q", index, got, wantType)
		}
	}
	if got := root.Get("output.0.content.0.text").String(); got != "" {
		t.Fatalf("empty text block content = %q, want empty string", got)
	}
	if got := root.Get("output.1.summary.0.text").String(); got != "plan" {
		t.Fatalf("first reasoning text = %q, want %q", got, "plan")
	}
	if got := root.Get("output.2.call_id").String(); got != "call_order" {
		t.Fatalf("function call id = %q, want %q", got, "call_order")
	}
	if got := root.Get("output.2.arguments").String(); got != `{"cmd":"pwd"}` {
		t.Fatalf("function call arguments = %q, want %q", got, `{"cmd":"pwd"}`)
	}
	if got := root.Get("output.3.content.0.text").String(); got != "done" {
		t.Fatalf("second message text = %q, want %q", got, "done")
	}
	if got := root.Get("output.4.summary.0.text").String(); got != "more" {
		t.Fatalf("second reasoning text = %q, want %q", got, "more")
	}
	if got := root.Get("usage.output_tokens_details.reasoning_tokens").Int(); got != 2 {
		t.Fatalf("reasoning tokens = %d, want 2", got)
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbols
// claudeResponsesTerminalState and claudeResponsesOutputStatus via
// ConvertClaudeResponseToOpenAIResponses.
func TestConvertClaudeResponseToOpenAIResponses_MaxTokensEmitsIncomplete(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_max_tokens","usage":{"input_tokens":10,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"unfinished reasoning"}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig_max_tokens"}}`),
		[]byte(`data: {"type":"content_block_stop","index":0}`),
		[]byte(`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	var incomplete gjson.Result
	for _, chunk := range chunks {
		for _, output := range ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-fable-5-1", nil, nil, chunk, &param) {
			event, data := parseClaudeResponsesSSEEvent(t, output)
			if event == "response.completed" {
				t.Fatalf("max_tokens response emitted response.completed: %s", output)
			}
			if event == "response.incomplete" {
				incomplete = data
			}
		}
	}

	if !incomplete.Exists() {
		t.Fatal("expected response.incomplete event")
	}
	if got := incomplete.Get("response.status").String(); got != "incomplete" {
		t.Fatalf("response.status = %q, want incomplete", got)
	}
	if got := incomplete.Get("response.incomplete_details.reason").String(); got != "max_output_tokens" {
		t.Fatalf("incomplete reason = %q, want max_output_tokens", got)
	}
	if got := incomplete.Get("response.output.0.type").String(); got != "reasoning" {
		t.Fatalf("response.output.0.type = %q, want reasoning; response=%s", got, incomplete.Raw)
	}
	if got := incomplete.Get("response.output.0.status").String(); got != "incomplete" {
		t.Fatalf("response.output.0.status = %q, want incomplete", got)
	}
	if got := incomplete.Get("response.output.0.summary.0.text").String(); got != "unfinished reasoning" {
		t.Fatalf("reasoning summary = %q, want unfinished reasoning", got)
	}
	if got := incomplete.Get("response.usage.output_tokens").Int(); got != 64000 {
		t.Fatalf("output_tokens = %d, want 64000", got)
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbol
// ConvertClaudeResponseToOpenAIResponsesNonStream.
func TestConvertClaudeResponseToOpenAIResponsesNonStream_MaxTokensPreservesPartialText(t *testing.T) {
	raw := []byte(strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"msg_partial","usage":{"input_tokens":10,"output_tokens":0}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial answer"}}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`,
		`data: {"type":"message_stop"}`,
	}, "\n"))

	out := ConvertClaudeResponseToOpenAIResponsesNonStream(context.Background(), "claude-fable-5-1", nil, nil, raw, nil)
	root := gjson.ParseBytes(out)
	if got := root.Get("status").String(); got != "incomplete" {
		t.Fatalf("status = %q, want incomplete; response=%s", got, out)
	}
	if got := root.Get("incomplete_details.reason").String(); got != "max_output_tokens" {
		t.Fatalf("incomplete reason = %q, want max_output_tokens", got)
	}
	if got := root.Get("output.0.status").String(); got != "incomplete" {
		t.Fatalf("output.0.status = %q, want incomplete", got)
	}
	if got := root.Get("output.0.content.0.text").String(); got != "partial answer" {
		t.Fatalf("partial text = %q, want partial answer", got)
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbol finalizeFuncItem via
// ConvertClaudeResponseToOpenAIResponses.
func TestConvertClaudeResponseToOpenAIResponses_MaxTokensWithToolCallAndTextEmitsIncomplete(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_tool_incomplete","usage":{"input_tokens":10,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"calling tool"}}`),
		[]byte(`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"call_inc_1","name":"get_weather","input":{}}}`),
		[]byte(`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"city\":\"San"}}`),
		[]byte(`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	var incomplete gjson.Result
	var funcDone gjson.Result
	for _, chunk := range chunks {
		for _, output := range ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-fable-5-1", nil, nil, chunk, &param) {
			event, data := parseClaudeResponsesSSEEvent(t, output)
			if event == "response.output_item.done" && data.Get("item.type").String() == "function_call" {
				funcDone = data
			}
			if event == "response.incomplete" {
				incomplete = data
			}
		}
	}

	if !funcDone.Exists() {
		t.Fatal("expected function_call response.output_item.done event")
	}
	if got := funcDone.Get("item.status").String(); got != "incomplete" {
		t.Fatalf("funcDone item.status = %q, want incomplete", got)
	}
	if got := funcDone.Get("item.arguments").String(); got != "{\"city\":\"San" {
		t.Fatalf("funcDone item.arguments = %q, want '{\"city\":\"San'", got)
	}

	if !incomplete.Exists() {
		t.Fatal("expected response.incomplete event")
	}
	if got := incomplete.Get("response.status").String(); got != "incomplete" {
		t.Fatalf("response.status = %q, want incomplete", got)
	}
	if got := incomplete.Get("response.incomplete_details.reason").String(); got != "max_output_tokens" {
		t.Fatalf("incomplete reason = %q, want max_output_tokens", got)
	}
	if got := incomplete.Get("response.output.0.status").String(); got != "completed" {
		t.Fatalf("output.0.status = %q, want completed", got)
	}
	if got := incomplete.Get("response.output.0.content.0.text").String(); got != "calling tool" {
		t.Fatalf("output.0 text = %q, want 'calling tool'", got)
	}
	if got := incomplete.Get("response.output.1.status").String(); got != "incomplete" {
		t.Fatalf("output.1.status = %q, want incomplete", got)
	}
	if got := incomplete.Get("response.output.1.type").String(); got != "function_call" {
		t.Fatalf("output.1.type = %q, want function_call", got)
	}
	if got := incomplete.Get("response.output.1.arguments").String(); got != "{\"city\":\"San" {
		t.Fatalf("output.1.arguments = %q, want '{\"city\":\"San'", got)
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbol
// finalizeWebSearchWithStatus via ConvertClaudeResponseToOpenAIResponses.
func TestConvertClaudeResponseToOpenAIResponses_MaxTokensWithWebSearchEmitsIncomplete(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_ws_incomplete","usage":{"input_tokens":10,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"server_tool_use","id":"srv_1","name":"web_search"}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"query\":\"golang\"}"}}`),
		[]byte(`data: {"type":"content_block_stop","index":0}`),
		[]byte(`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	var incomplete gjson.Result
	var wsDone gjson.Result
	for _, chunk := range chunks {
		for _, output := range ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-fable-5-1", nil, nil, chunk, &param) {
			event, data := parseClaudeResponsesSSEEvent(t, output)
			if event == "response.output_item.done" && data.Get("item.type").String() == "web_search_call" {
				wsDone = data
			}
			if event == "response.incomplete" {
				incomplete = data
			}
		}
	}

	if !wsDone.Exists() {
		t.Fatal("expected web_search_call response.output_item.done event")
	}
	if got := wsDone.Get("item.status").String(); got != "incomplete" {
		t.Fatalf("wsDone item.status = %q, want incomplete", got)
	}
	if !incomplete.Exists() {
		t.Fatal("expected response.incomplete event")
	}
	if got := incomplete.Get("response.output.0.status").String(); got != "incomplete" {
		t.Fatalf("response.output.0.status = %q, want incomplete", got)
	}

	// Non-stream
	raw := []byte(strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"msg_ws_incomplete","usage":{"input_tokens":10,"output_tokens":0}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"server_tool_use","id":"srv_1","name":"web_search"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"query\":\"golang\"}"}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`,
		`data: {"type":"message_stop"}`,
	}, "\n"))
	nonStreamOut := ConvertClaudeResponseToOpenAIResponsesNonStream(context.Background(), "claude-fable-5-1", nil, nil, raw, nil)
	nsRoot := gjson.ParseBytes(nonStreamOut)
	if got := nsRoot.Get("status").String(); got != "incomplete" {
		t.Fatalf("ns status = %q, want incomplete", got)
	}
	if got := nsRoot.Get("output.0.status").String(); got != "incomplete" {
		t.Fatalf("ns output.0.status = %q, want incomplete", got)
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbol finalizeReasoningItem
// via ConvertClaudeResponseToOpenAIResponses.
func TestConvertClaudeResponseToOpenAIResponses_MaxTokensReasoningWithoutBlockStop(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_reasoning_nostop","usage":{"input_tokens":10,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"partial thought before cut"}}`),
		[]byte(`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	var incomplete gjson.Result
	var rsDone gjson.Result
	for _, chunk := range chunks {
		for _, output := range ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-fable-5-1", nil, nil, chunk, &param) {
			event, data := parseClaudeResponsesSSEEvent(t, output)
			if event == "response.output_item.done" && data.Get("item.type").String() == "reasoning" {
				rsDone = data
			}
			if event == "response.incomplete" {
				incomplete = data
			}
		}
	}

	if !rsDone.Exists() {
		t.Fatal("expected reasoning response.output_item.done event")
	}
	if got := rsDone.Get("item.status").String(); got != "incomplete" {
		t.Fatalf("rsDone item.status = %q, want incomplete", got)
	}
	if !incomplete.Exists() {
		t.Fatal("expected response.incomplete event")
	}
	if got := incomplete.Get("response.output.0.status").String(); got != "incomplete" {
		t.Fatalf("response.output.0.status = %q, want incomplete", got)
	}
	if got := incomplete.Get("response.output.0.type").String(); got != "reasoning" {
		t.Fatalf("response.output.0.type = %q, want reasoning", got)
	}
	if got := incomplete.Get("response.output.0.summary.0.text").String(); got != "partial thought before cut" {
		t.Fatalf("summary = %q, want 'partial thought before cut'", got)
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbol finalizeFuncItem via
// ConvertClaudeResponseToOpenAIResponses.
func TestConvertClaudeResponseToOpenAIResponses_MaxTokensWithToolBlockStopEmitsIncomplete(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_tool_blockstop","usage":{"input_tokens":10,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call_stop_1","name":"do_work","input":{}}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"step\":1}"}}`),
		[]byte(`data: {"type":"content_block_stop","index":0}`),
		[]byte(`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	var incomplete gjson.Result
	var funcDone gjson.Result
	for _, chunk := range chunks {
		for _, output := range ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-fable-5-1", nil, nil, chunk, &param) {
			event, data := parseClaudeResponsesSSEEvent(t, output)
			if event == "response.output_item.done" && data.Get("item.type").String() == "function_call" {
				funcDone = data
			}
			if event == "response.incomplete" {
				incomplete = data
			}
		}
	}

	if !funcDone.Exists() {
		t.Fatal("expected function_call response.output_item.done event")
	}
	if got := funcDone.Get("item.status").String(); got != "incomplete" {
		t.Fatalf("funcDone item.status = %q, want incomplete", got)
	}
	if !incomplete.Exists() {
		t.Fatal("expected response.incomplete event")
	}
	if got := incomplete.Get("response.output.0.status").String(); got != "incomplete" {
		t.Fatalf("output.0.status = %q, want incomplete", got)
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbol
// finalizeWebSearchWithStatus via ConvertClaudeResponseToOpenAIResponses.
func TestConvertClaudeResponseToOpenAIResponses_MaxTokensWithWebSearchResultsEmitsIncomplete(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_ws_res_incomplete","usage":{"input_tokens":10,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"server_tool_use","id":"srv_2","name":"web_search"}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"query\":\"golang\"}"}}`),
		[]byte(`data: {"type":"content_block_stop","index":0}`),
		[]byte(`data: {"type":"content_block_start","index":1,"content_block":{"type":"web_search_tool_result","tool_use_id":"srv_2","content":[{"type":"web_search_result","title":"Go","url":"https://golang.org"}]}}`),
		[]byte(`data: {"type":"content_block_stop","index":1}`),
		[]byte(`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	var incomplete gjson.Result
	var wsDone gjson.Result
	for _, chunk := range chunks {
		for _, output := range ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-fable-5-1", nil, nil, chunk, &param) {
			event, data := parseClaudeResponsesSSEEvent(t, output)
			if event == "response.output_item.done" && data.Get("item.type").String() == "web_search_call" {
				wsDone = data
			}
			if event == "response.incomplete" {
				incomplete = data
			}
		}
	}

	if !wsDone.Exists() {
		t.Fatal("expected web_search_call response.output_item.done event")
	}
	if got := wsDone.Get("item.status").String(); got != "incomplete" {
		t.Fatalf("wsDone item.status = %q, want incomplete", got)
	}
	if !incomplete.Exists() {
		t.Fatal("expected response.incomplete event")
	}
	if got := incomplete.Get("response.output.0.status").String(); got != "incomplete" {
		t.Fatalf("response.output.0.status = %q, want incomplete", got)
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbols
// finalizeReasoningDeltas and finalizeReasoningItem via
// ConvertClaudeResponseToOpenAIResponses.
func TestConvertClaudeResponseToOpenAIResponses_ReasoningEventsNotDuplicated(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_rs_once","usage":{"input_tokens":10,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"full thought"}}`),
		[]byte(`data: {"type":"content_block_stop","index":0}`),
		[]byte(`data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":10}}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	counts := make(map[string]int)
	for _, chunk := range chunks {
		for _, output := range ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-fable-5-1", nil, nil, chunk, &param) {
			event, _ := parseClaudeResponsesSSEEvent(t, output)
			counts[event]++
		}
	}

	if counts["response.reasoning_summary_text.done"] != 1 {
		t.Fatalf("reasoning_summary_text.done count = %d, want 1", counts["response.reasoning_summary_text.done"])
	}
	if counts["response.reasoning_summary_part.done"] != 1 {
		t.Fatalf("reasoning_summary_part.done count = %d, want 1", counts["response.reasoning_summary_part.done"])
	}
	if counts["response.output_item.done"] != 1 {
		t.Fatalf("output_item.done count = %d, want 1", counts["response.output_item.done"])
	}
}

// Ported from upstream CLIProxyAPI commit ba2cdea3b919 (handle incomplete status
// and terminal state on max_tokens); covers local symbol finalizeFuncItem via
// ConvertClaudeResponseToOpenAIResponses.
func TestConvertClaudeResponseToOpenAIResponses_EmptyFunctionArgsConsistentOnTruncation(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"type":"message_start","message":{"id":"msg_empty_args_trunc","usage":{"input_tokens":10,"output_tokens":0}}}`),
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call_empty","name":"get_info","input":{}}}`),
		[]byte(`data: {"type":"content_block_stop","index":0}`),
		[]byte(`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":64000}}`),
		[]byte(`data: {"type":"message_stop"}`),
	}

	var param any
	var funcDone gjson.Result
	var argsDone gjson.Result
	var incomplete gjson.Result
	for _, chunk := range chunks {
		for _, output := range ConvertClaudeResponseToOpenAIResponses(context.Background(), "claude-fable-5-1", nil, nil, chunk, &param) {
			event, data := parseClaudeResponsesSSEEvent(t, output)
			if event == "response.function_call_arguments.done" {
				argsDone = data
			}
			if event == "response.output_item.done" && data.Get("item.type").String() == "function_call" {
				funcDone = data
			}
			if event == "response.incomplete" {
				incomplete = data
			}
		}
	}

	if !argsDone.Exists() {
		t.Fatal("expected response.function_call_arguments.done event")
	}
	if !funcDone.Exists() {
		t.Fatal("expected function_call response.output_item.done event")
	}
	if got := argsDone.Get("arguments").String(); got != "" {
		t.Fatalf("argsDone arguments = %q, want empty string", got)
	}
	if got := funcDone.Get("item.arguments").String(); got != "" {
		t.Fatalf("funcDone arguments = %q, want empty string", got)
	}
	if got := funcDone.Get("item.status").String(); got != "incomplete" {
		t.Fatalf("funcDone item.status = %q, want incomplete", got)
	}
	if !incomplete.Exists() {
		t.Fatal("expected response.incomplete event")
	}
	if got := incomplete.Get("response.output.0.arguments").String(); got != "" {
		t.Fatalf("response.output.0.arguments = %q, want empty string", got)
	}
}
