package gemini

import (
	"context"
	"testing"

	"github.com/tidwall/gjson"
)

// Ported from upstream CLIProxyAPI commit 4dce5f3a2b9a ("ignore null and empty
// finish reasons in openai to gemini response"): null/empty finish_reason must
// not emit a Gemini finishReason.
func TestConvertOpenAIResponseToGeminiStream_NullFinishReasonIgnored(t *testing.T) {
	var param any

	// Chunk 1: Contentless delta with explicit finish_reason: null
	chunk1 := []byte(`{"choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`)
	out1 := ConvertOpenAIResponseToGemini(context.Background(), "gpt-test", nil, nil, chunk1, &param)
	for i, chunk := range out1 {
		if fr := gjson.GetBytes(chunk, "candidates.0.finishReason"); fr.Exists() && fr.String() != "" {
			t.Fatalf("chunk1[%d] unexpectedly set finishReason = %q on non-final chunk; payload=%s", i, fr.String(), chunk)
		}
	}

	// Chunk 2: Contentless delta with finish_reason: "" (empty string should not be treated as stop)
	chunk2 := []byte(`{"choices":[{"index":0,"delta":{},"finish_reason":""}]}`)
	out2 := ConvertOpenAIResponseToGemini(context.Background(), "gpt-test", nil, nil, chunk2, &param)
	for i, chunk := range out2 {
		if fr := gjson.GetBytes(chunk, "candidates.0.finishReason"); fr.Exists() && fr.String() != "" {
			t.Fatalf("chunk2[%d] unexpectedly set finishReason = %q on contentless chunk with empty finish_reason; payload=%s", i, fr.String(), chunk)
		}
	}

	// Chunk 3: Final chunk with finish_reason: "stop"
	chunk3 := []byte(`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
	out3 := ConvertOpenAIResponseToGemini(context.Background(), "gpt-test", nil, nil, chunk3, &param)
	if len(out3) == 0 {
		t.Fatalf("expected output for final chunk, got 0 chunks")
	}
	if got := gjson.GetBytes(out3[len(out3)-1], "candidates.0.finishReason").String(); got != "STOP" {
		t.Fatalf("expected finishReason STOP on final chunk, got %q", got)
	}
}

// Ported from upstream CLIProxyAPI commit 4dce5f3a2b9a: the non-streaming
// converter must also ignore null/empty finish_reason.
func TestConvertOpenAIResponseToGeminiNonStream_NullFinishReasonIgnored(t *testing.T) {
	testCases := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "null finish_reason",
			payload: []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":null}]}`),
		},
		{
			name:    "empty finish_reason",
			payload: []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":""}]}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := ConvertOpenAIResponseToGeminiNonStream(context.Background(), "gpt-test", nil, nil, tc.payload, nil)
			if fr := gjson.GetBytes(out, "candidates.0.finishReason"); fr.Exists() && fr.String() != "" {
				t.Fatalf("unexpectedly set finishReason = %q for %s; payload=%s", fr.String(), tc.name, out)
			}
		})
	}

	out := ConvertOpenAIResponseToGeminiNonStream(context.Background(), "gpt-test", nil, nil,
		[]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}]}`), nil)
	if got := gjson.GetBytes(out, "candidates.0.finishReason").String(); got != "STOP" {
		t.Fatalf("expected finishReason STOP, got %q; payload=%s", got, out)
	}
}
