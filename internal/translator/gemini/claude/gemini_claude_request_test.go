package claude

import (
	"testing"

	signature "github.com/therealtinhtute/llmhub/internal/signature"
	"github.com/tidwall/gjson"
)

func TestConvertClaudeRequestToGemini_ToolChoice_SpecificTool(t *testing.T) {
	inputJSON := []byte(`{
		"model": "gemini-3-flash-preview",
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "hi"}
				]
			}
		],
		"tools": [
			{
				"name": "json",
				"description": "A JSON tool",
				"input_schema": {
					"type": "object",
					"properties": {}
				}
			}
		],
		"tool_choice": {"type": "tool", "name": "json"}
	}`)

	output := ConvertClaudeRequestToGemini("gemini-3-flash-preview", inputJSON, false)

	if got := gjson.GetBytes(output, "toolConfig.functionCallingConfig.mode").String(); got != "ANY" {
		t.Fatalf("Expected toolConfig.functionCallingConfig.mode 'ANY', got '%s'", got)
	}
	allowed := gjson.GetBytes(output, "toolConfig.functionCallingConfig.allowedFunctionNames").Array()
	if len(allowed) != 1 || allowed[0].String() != "json" {
		t.Fatalf("Expected allowedFunctionNames ['json'], got %s", gjson.GetBytes(output, "toolConfig.functionCallingConfig.allowedFunctionNames").Raw)
	}
}

func TestConvertClaudeRequestToGemini_ImageContent(t *testing.T) {
	inputJSON := []byte(`{
		"model": "gemini-3-flash-preview",
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "describe this image"},
					{
						"type": "image",
						"source": {
							"type": "base64",
							"media_type": "image/png",
							"data": "aGVsbG8="
						}
					}
				]
			}
		]
	}`)

	output := ConvertClaudeRequestToGemini("gemini-3-flash-preview", inputJSON, false)

	parts := gjson.GetBytes(output, "contents.0.parts").Array()
	if len(parts) != 2 {
		t.Fatalf("Expected 2 parts, got %d", len(parts))
	}
	if got := parts[0].Get("text").String(); got != "describe this image" {
		t.Fatalf("Expected first part text 'describe this image', got '%s'", got)
	}
	if got := parts[1].Get("inline_data.mime_type").String(); got != "image/png" {
		t.Fatalf("Expected image mime type 'image/png', got '%s'", got)
	}
	if got := parts[1].Get("inline_data.data").String(); got != "aGVsbG8=" {
		t.Fatalf("Expected image data 'aGVsbG8=', got '%s'", got)
	}
}

func TestConvertClaudeRequestToGemini_StripsClaudeCodeAttribution(t *testing.T) {
	inputJSON := []byte(`{
		"model": "claude-sonnet-4-5",
		"system": [
			{"type": "text", "text": "x-anthropic-billing-header: cc_version=2.1.63.abc; cc_entrypoint=cli; cch=12345;"},
			{"type": "text", "text": "You are a Claude agent, built on Anthropic's Claude Agent SDK."},
			{"type": "text", "text": "User system prompt"}
		],
		"messages": [{"role": "user", "content": [{"type": "text", "text": "hi"}]}]
	}`)

	output := ConvertClaudeRequestToGemini("gemini-3-flash-preview", inputJSON, false)

	parts := gjson.GetBytes(output, "system_instruction.parts").Array()
	if len(parts) != 2 {
		t.Fatalf("Expected 2 system parts after attribution strip, got %d: %s", len(parts), gjson.GetBytes(output, "system_instruction.parts").Raw)
	}
	if got := parts[0].Get("text").String(); got != "You are a Claude agent, built on Anthropic's Claude Agent SDK." {
		t.Fatalf("Unexpected first system part: %q", got)
	}
	if got := parts[1].Get("text").String(); got != "User system prompt" {
		t.Fatalf("Unexpected second system part: %q", got)
	}
	if gjson.GetBytes(output, `system_instruction.parts.#(text%"x-anthropic-billing-header:*")`).Exists() {
		t.Fatalf("Claude Code attribution block was forwarded: %s", gjson.GetBytes(output, "system_instruction.parts").Raw)
	}
}

func TestConvertClaudeRequestToGemini_SkipsEmptyTextParts(t *testing.T) {
	inputJSON := []byte(`{
		"model": "claude-3-5-sonnet",
		"messages": [
			{
				"role": "assistant",
				"content": [
					{"type": "text", "text": ""},
					{"type": "text", "text": "hello"},
					{"type": "text", "text": ""}
				]
			}
		]
	}`)

	output := ConvertClaudeRequestToGemini("gemini-3-flash-preview", inputJSON, false)

	parts := gjson.GetBytes(output, "contents.0.parts").Array()
	if len(parts) != 1 {
		t.Fatalf("Expected 1 part after skipping empty text, got %d: %s", len(parts), output)
	}
	if got := parts[0].Get("text").String(); got != "hello" {
		t.Fatalf("Expected part text 'hello', got '%s'", got)
	}
}

const capturedGeminiThinkingSignature = "EjQKMgEMOdbHO0Gd+c9Mxk4ELwPGbpCEcp2mFfYYLix2UVtBH3fL8GECc4+JITVnHF4qZDsA"

func TestConvertClaudeRequestToGemini_SignatureCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		signature     string
		wantSignature string
	}{
		{
			name:          "preserves valid gemini signature",
			signature:     "gemini#" + capturedGeminiThinkingSignature,
			wantSignature: capturedGeminiThinkingSignature,
		},
		{
			name:          "foreign claude signature maps to bypass sentinel",
			signature:     "claude#opaque-signature-12345",
			wantSignature: signature.GeminiSkipThoughtSignatureValidator,
		},
		{
			name:          "empty signature maps to bypass sentinel",
			signature:     "",
			wantSignature: signature.GeminiSkipThoughtSignatureValidator,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := []byte(`{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"reason","signature":"` + tt.signature + `"}]}]}`)
			out := ConvertClaudeRequestToGemini("deepseek-v4", payload, false)
			part := gjson.GetBytes(out, "contents.0.parts.0")
			if !part.Get("thought").Bool() || part.Get("text").String() != "reason" {
				t.Fatalf("translation missing thought part: %s", out)
			}
			if got := part.Get("thoughtSignature").String(); got != tt.wantSignature {
				t.Fatalf("thoughtSignature = %q, want %q; output: %s", got, tt.wantSignature, out)
			}
		})
	}
}
