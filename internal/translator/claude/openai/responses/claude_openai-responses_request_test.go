package responses

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponsesRequestToClaude_ReasoningItemToThinkingBlock(t *testing.T) {
	signature := "claude_sig_request"
	raw := []byte(`{
		"model":"claude-test",
		"input":[
			{
				"type":"reasoning",
				"encrypted_content":"` + signature + `",
				"summary":[{"type":"summary_text","text":"internal reasoning"}]
			},
			{
				"type":"message",
				"role":"assistant",
				"content":[{"type":"output_text","text":"visible answer"}]
			},
			{
				"type":"message",
				"role":"user",
				"content":[{"type":"input_text","text":"continue"}]
			}
		]
	}`)

	out := ConvertOpenAIResponsesRequestToClaude("claude-test", raw, false)
	root := gjson.ParseBytes(out)

	assistant := root.Get("messages.0")
	if got := assistant.Get("role").String(); got != "assistant" {
		t.Fatalf("first message role = %q, want assistant. Output: %s", got, string(out))
	}
	if got := assistant.Get("content.0.type").String(); got != "thinking" {
		t.Fatalf("first content type = %q, want thinking. Output: %s", got, string(out))
	}
	if got := assistant.Get("content.0.signature").String(); got != signature {
		t.Fatalf("thinking signature = %q, want %q", got, signature)
	}
	if got := assistant.Get("content.0.thinking").String(); got != "internal reasoning" {
		t.Fatalf("thinking text = %q, want internal reasoning", got)
	}
	if got := assistant.Get("content.1.type").String(); got != "text" {
		t.Fatalf("second content type = %q, want text. Output: %s", got, string(out))
	}
	if got := assistant.Get("content.1.text").String(); got != "visible answer" {
		t.Fatalf("assistant text = %q, want visible answer", got)
	}
	if got := root.Get("messages.1.role").String(); got != "user" {
		t.Fatalf("second message role = %q, want user. Output: %s", got, string(out))
	}
}

func TestConvertOpenAIResponsesRequestToClaude_SignatureOnlyReasoningFlushesBeforeUser(t *testing.T) {
	signature := "claude_sig_only"
	raw := []byte(`{
		"model":"claude-test",
		"input":[
			{
				"type":"reasoning",
				"encrypted_content":"` + signature + `",
				"summary":[]
			},
			{
				"type":"message",
				"role":"user",
				"content":[{"type":"input_text","text":"continue"}]
			}
		]
	}`)

	out := ConvertOpenAIResponsesRequestToClaude("claude-test", raw, false)
	root := gjson.ParseBytes(out)

	thinking := root.Get("messages.0.content.0")
	if got := thinking.Get("type").String(); got != "thinking" {
		t.Fatalf("first content type = %q, want thinking. Output: %s", got, string(out))
	}
	if got := thinking.Get("signature").String(); got != signature {
		t.Fatalf("thinking signature = %q, want %q", got, signature)
	}
	if got := thinking.Get("thinking").String(); got != "" {
		t.Fatalf("thinking text = %q, want empty", got)
	}
	if got := root.Get("messages.1.role").String(); got != "user" {
		t.Fatalf("second message role = %q, want user. Output: %s", got, string(out))
	}
}

func TestConvertOpenAIResponsesRequestToClaude_FunctionCallOutputContent(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		want       string
		structured bool
	}{
		{
			name:   "string remains string",
			output: `"plain text"`,
			want:   "plain text",
		},
		{
			name:       "empty array remains structured",
			output:     `[]`,
			want:       `[]`,
			structured: true,
		},
		{
			name: "OpenAI content parts map to structured Claude blocks",
			output: `[
				{"type":"input_text","text":"ok"},
				{"type":"input_image","image_url":"data:image/png;base64,aGVsbG8="},
				{"type":"input_file","file_data":"data:application/pdf;base64,JVBERi0x"}
			]`,
			want:       `[{"type":"text","text":"ok"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"aGVsbG8="}},{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"JVBERi0x"}}]`,
			structured: true,
		},
		{
			name: "valid Claude blocks remain structured",
			output: `[
				{"type":"text","text":"ok"},
				{"type":"image","source":{"type":"url","url":"https://example.com/image.png"}},
				{"type":"document","source":{"type":"text","media_type":"text/plain","data":"document"}}
			]`,
			want:       `[{"type":"text","text":"ok"},{"type":"image","source":{"type":"url","url":"https://example.com/image.png"}},{"type":"document","source":{"type":"text","media_type":"text/plain","data":"document"}}]`,
			structured: true,
		},
		{
			name:       "search result with empty content remains structured",
			output:     `[{"type":"search_result","source":"https://example.com","title":"Example","content":[]}]`,
			want:       `[{"type":"search_result","source":"https://example.com","title":"Example","content":[]}]`,
			structured: true,
		},
		{
			name:       "document content with image remains structured",
			output:     `[{"type":"document","source":{"type":"content","content":[{"type":"text","text":"caption"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"aGVsbG8="}}]}}]`,
			want:       `[{"type":"document","source":{"type":"content","content":[{"type":"text","text":"caption"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"aGVsbG8="}}]}}]`,
			structured: true,
		},
		{
			name:       "document string content remains structured",
			output:     `[{"type":"document","source":{"type":"content","content":"document"}}]`,
			want:       `[{"type":"document","source":{"type":"content","content":"document"}}]`,
			structured: true,
		},
		{
			name:       "document content empty array remains structured",
			output:     `[{"type":"document","source":{"type":"content","content":[]}}]`,
			want:       `[{"type":"document","source":{"type":"content","content":[]}}]`,
			structured: true,
		},
		{
			name:       "tool reference remains structured",
			output:     `[{"type":"tool_reference","tool_name":"search"}]`,
			want:       `[{"type":"tool_reference","tool_name":"search"}]`,
			structured: true,
		},
		{
			name:       "raw PDF file with PDF filename remains structured",
			output:     `[{"type":"input_file","filename":"report.pdf","file_data":"JVBERi0x"}]`,
			want:       `[{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"JVBERi0x"}}]`,
			structured: true,
		},
		{
			name:   "unsupported input image media type uses compact JSON text",
			output: `[{"type":"input_image","image_url":"data:application/pdf;base64,JVBERi0x"}]`,
			want:   `[{"type":"input_image","image_url":"data:application/pdf;base64,JVBERi0x"}]`,
		},
		{
			name:   "malformed input image base64 uses compact JSON text",
			output: `[{"type":"input_image","image_url":"data:image/png;base64,not-base64"}]`,
			want:   `[{"type":"input_image","image_url":"data:image/png;base64,not-base64"}]`,
		},
		{
			name:   "unsupported input file media type uses compact JSON text",
			output: `[{"type":"input_file","file_data":"data:text/plain;base64,aGVsbG8="}]`,
			want:   `[{"type":"input_file","file_data":"data:text/plain;base64,aGVsbG8="}]`,
		},
		{
			name:   "raw non-base64 input file uses compact JSON text",
			output: `[{"type":"input_file","filename":"report.pdf","file_data":"not base64"}]`,
			want:   `[{"type":"input_file","filename":"report.pdf","file_data":"not base64"}]`,
		},
		{
			name:   "text document without text plain media type uses compact JSON text",
			output: `[{"type":"document","source":{"type":"text","data":"document"}}]`,
			want:   `[{"type":"document","source":{"type":"text","data":"document"}}]`,
		},
		{
			name:   "native image without source uses compact JSON text",
			output: `[{"type":"image"}]`,
			want:   `[{"type":"image"}]`,
		},
		{
			name:   "native image with unsupported media type uses compact JSON text",
			output: `[{"type":"image","source":{"type":"base64","media_type":"application/pdf","data":"JVBERi0x"}}]`,
			want:   `[{"type":"image","source":{"type":"base64","media_type":"application/pdf","data":"JVBERi0x"}}]`,
		},
		{
			name:   "native image with malformed base64 uses compact JSON text",
			output: `[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"not-base64"}}]`,
			want:   `[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"not-base64"}}]`,
		},
		{
			name:   "native document without source uses compact JSON text",
			output: `[{"type":"document"}]`,
			want:   `[{"type":"document"}]`,
		},
		{
			name:   "native document with unsupported base64 media type uses compact JSON text",
			output: `[{"type":"document","source":{"type":"base64","media_type":"text/plain","data":"aGVsbG8="}}]`,
			want:   `[{"type":"document","source":{"type":"base64","media_type":"text/plain","data":"aGVsbG8="}}]`,
		},
		{
			name:   "native document with malformed base64 uses compact JSON text",
			output: `[{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"not-base64"}}]`,
			want:   `[{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"not-base64"}}]`,
		},
		{
			name:   "search result without title uses compact JSON text",
			output: `[{"type":"search_result","source":"https://example.com","content":[]}]`,
			want:   `[{"type":"search_result","source":"https://example.com","content":[]}]`,
		},
		{
			name:   "search result with invalid content uses compact JSON text",
			output: `[{"type":"search_result","source":"https://example.com","title":"Example","content":[{"type":"image"}]}]`,
			want:   `[{"type":"search_result","source":"https://example.com","title":"Example","content":[{"type":"image"}]}]`,
		},
		{
			name:   "tool reference without tool name uses compact JSON text",
			output: `[{"type":"tool_reference","tool_name":""}]`,
			want:   `[{"type":"tool_reference","tool_name":""}]`,
		},
		{
			name:   "arbitrary object uses compact JSON text",
			output: `{ "b": 2, "a": [1, true] }`,
			want:   `{"b":2,"a":[1,true]}`,
		},
		{
			name:   "number uses JSON text",
			output: `42`,
			want:   `42`,
		},
		{
			name:   "boolean uses JSON text",
			output: `true`,
			want:   `true`,
		},
		{
			name:   "null uses JSON text",
			output: `null`,
			want:   `null`,
		},
		{
			name:   "unknown array uses compact JSON text",
			output: `[ { "type": "audio", "data": "abc" } ]`,
			want:   `[{"type":"audio","data":"abc"}]`,
		},
		{
			name:   "invalid block array uses compact JSON text",
			output: `[ { "type": "text" } ]`,
			want:   `[{"type":"text"}]`,
		},
		{
			name:   "mixed array uses compact JSON text",
			output: `[ { "type": "input_text", "text": "ok" }, { "type": "unknown" } ]`,
			want:   `[{"type":"input_text","text":"ok"},{"type":"unknown"}]`,
		},
		{
			name:   "nested array uses compact JSON text",
			output: `[[ { "type": "text", "text": "nested" } ]]`,
			want:   `[[{"type":"text","text":"nested"}]]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte(`{"model":"claude-test","input":[{"type":"function_call_output","call_id":"call_1","output":` + tt.output + `}]}`)
			out := ConvertOpenAIResponsesRequestToClaude("claude-test", raw, false)
			content := gjson.GetBytes(out, "messages.0.content.0.content")

			if tt.structured {
				if !content.IsArray() {
					t.Fatalf("content = %s, want structured array. Output: %s", content.Raw, string(out))
				}
				if content.Raw != tt.want {
					t.Fatalf("content = %s, want %s", content.Raw, tt.want)
				}
				return
			}
			if content.Type != gjson.String {
				t.Fatalf("content type = %v, want string. Output: %s", content.Type, string(out))
			}
			if content.String() != tt.want {
				t.Fatalf("content = %q, want %q", content.String(), tt.want)
			}
		})
	}
}
