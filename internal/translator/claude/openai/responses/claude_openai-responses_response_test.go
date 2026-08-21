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
