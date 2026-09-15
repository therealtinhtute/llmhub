package chat_completions

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIRequestToGeminiMapsJSONSchemaResponseFormat(t *testing.T) {
	input := []byte(`{
		"generationConfig": {
			"temperature": 0.2,
			"responseSchema": {"type":"string"},
			"responseJsonSchema": {"type":"number"}
		},
		"messages": [{"role":"user","content":"Return JSON"}],
		"response_format": {
			"type": "json_schema",
			"json_schema": {
				"name": "answer",
				"schema": {"type":"object","properties":{"ok":{"type":"boolean"}},"additionalProperties":false}
			}
		}
	}`)

	out := ConvertOpenAIRequestToGemini("gemini-test", input, false)
	if got := gjson.GetBytes(out, "generationConfig.responseMimeType").String(); got != "application/json" {
		t.Fatalf("responseMimeType = %q, want application/json; out=%s", got, out)
	}
	if got := gjson.GetBytes(out, "generationConfig.responseJsonSchema.properties.ok.type").String(); got != "boolean" {
		t.Fatalf("responseJsonSchema ok.type = %q, want boolean; out=%s", got, out)
	}
	if gjson.GetBytes(out, "generationConfig.responseSchema").Exists() {
		t.Fatalf("stale responseSchema survived: %s", out)
	}
	if got := gjson.GetBytes(out, "generationConfig.responseJsonSchema.additionalProperties"); !got.Exists() || got.Bool() {
		t.Fatalf("responseJsonSchema.additionalProperties = %s, want false; out=%s", got.Raw, out)
	}
	if got := gjson.GetBytes(out, "generationConfig.temperature").Float(); got != 0.2 {
		t.Fatalf("temperature = %v, want 0.2; out=%s", got, out)
	}
}

func TestConvertOpenAIRequestToGeminiMapsJSONObjectResponseFormat(t *testing.T) {
	input := []byte(`{"generationConfig":{"responseJsonSchema":{"type":"string"}},"messages":[{"role":"user","content":"Return JSON"}],"response_format":{"type":"json_object"}}`)

	out := ConvertOpenAIRequestToGemini("gemini-test", input, false)
	if got := gjson.GetBytes(out, "generationConfig.responseMimeType").String(); got != "application/json" {
		t.Fatalf("responseMimeType = %q, want application/json; out=%s", got, out)
	}
	if gjson.GetBytes(out, "generationConfig.responseJsonSchema").Exists() {
		t.Fatalf("responseJsonSchema should not be set for json_object; out=%s", out)
	}
}

func TestConvertOpenAIRequestToGeminiJSONSchemaWithoutSchemaDoesNotSetStaleSchema(t *testing.T) {
	input := []byte(`{"generationConfig":{"responseJsonSchema":{"type":"string"}},"messages":[{"role":"user","content":"Return JSON"}],"response_format":{"type":"json_schema","json_schema":{"name":"answer"}}}`)

	out := ConvertOpenAIRequestToGemini("gemini-test", input, false)
	if got := gjson.GetBytes(out, "generationConfig.responseMimeType").String(); got != "application/json" {
		t.Fatalf("responseMimeType = %q, want application/json; out=%s", got, out)
	}
	if gjson.GetBytes(out, "generationConfig.responseJsonSchema").Exists() {
		t.Fatalf("responseJsonSchema should not be set without schema; out=%s", out)
	}
}

func TestConvertOpenAIRequestToGeminiNormalizesFileData(t *testing.T) {
	input := []byte(`{
		"messages": [
			{
				"role": "user",
				"content": [
					{"type":"image_url","image_url":{"url":"data:image/png;base64,image-user"}},
					{"type":"image_url","image_url":{"url":"data:image/png,image-invalid"}},
					{"type":"file","file":{"filename":"report.PDF","file_data":"file-raw"}},
					{"type":"file","file":{"filename":"wrong.txt","file_data":"data:image/jpeg;base64,file-url"}},
					{"type":"file","file":{"filename":"guess.pdf","file_data":"data:;base64,file-invalid"}}
				]
			},
			{
				"role": "assistant",
				"content": [
					{"type":"image_url","image_url":{"url":"data:image/webp;base64,image-assistant"}},
					{"type":"image_url","image_url":{"url":"data:image/webp;base64,"}}
				]
			}
		]
	}`)

	out := ConvertOpenAIRequestToGemini("gemini-test", input, false)
	userParts := gjson.GetBytes(out, "contents.0.parts").Array()
	if len(userParts) != 3 {
		t.Fatalf("expected 3 normalized user parts, got %d: %s", len(userParts), gjson.GetBytes(out, "contents.0.parts").Raw)
	}
	assertGeminiInlineData(t, userParts[0], "image/png", "image-user", geminiFunctionThoughtSignature)
	assertGeminiInlineData(t, userParts[1], "application/pdf", "file-raw", "")
	assertGeminiInlineData(t, userParts[2], "image/jpeg", "file-url", "")

	assistantParts := gjson.GetBytes(out, "contents.1.parts").Array()
	if len(assistantParts) != 1 {
		t.Fatalf("expected 1 normalized assistant part, got %d: %s", len(assistantParts), gjson.GetBytes(out, "contents.1.parts").Raw)
	}
	assertGeminiInlineData(t, assistantParts[0], "image/webp", "image-assistant", geminiFunctionThoughtSignature)
}

func assertGeminiInlineData(t *testing.T, part gjson.Result, wantMIMEType, wantData, wantThoughtSignature string) {
	t.Helper()
	if got := part.Get("inlineData.mime_type").String(); got != wantMIMEType {
		t.Errorf("inlineData.mime_type = %q, want %q", got, wantMIMEType)
	}
	if got := part.Get("inlineData.data").String(); got != wantData {
		t.Errorf("inlineData.data = %q, want %q", got, wantData)
	}
	if got := part.Get("thoughtSignature").String(); got != wantThoughtSignature {
		t.Errorf("thoughtSignature = %q, want %q", got, wantThoughtSignature)
	}
}

// Ported from upstream CLIProxyAPI commit 0fe19ede90a4 ("preserve Gemini prompt
// cache by demoting mid-session developer messages"): a mid-session developer
// message demotes to a user turn instead of mutating systemInstruction.
func TestConvertOpenAIRequestToGemini_MidSessionDeveloperMessageDoesNotMutateSystemInstruction(t *testing.T) {
	inputJSON := `{
		"model": "gemini-3-flash",
		"messages": [
			{"role": "system", "content": "You are a helpful assistant"},
			{"role": "user", "content": "Turn 1 user"},
			{"role": "assistant", "content": "Turn 1 assistant"},
			{"role": "developer", "content": "<image_resize_notice>Image 1 was resized to 800x600</image_resize_notice>"},
			{"role": "user", "content": "Turn 2 user"}
		]
	}`

	result := ConvertOpenAIRequestToGemini("gemini-3-flash", []byte(inputJSON), false)
	output := gjson.ParseBytes(result)

	// systemInstruction must contain only original system prompt
	sysParts := output.Get("systemInstruction.parts").Array()
	if len(sysParts) != 1 {
		t.Fatalf("systemInstruction parts = %d, want 1. Output: %s", len(sysParts), result)
	}
	if got := sysParts[0].Get("text").String(); got != "You are a helpful assistant" {
		t.Fatalf("systemInstruction text = %q, want %q", got, "You are a helpful assistant")
	}

	// contents must contain user, model, user (demoted dev message), user
	contents := output.Get("contents").Array()
	if len(contents) != 4 {
		t.Fatalf("contents length = %d, want 4. Output: %s", len(contents), result)
	}
	if contents[0].Get("role").String() != "user" || contents[0].Get("parts.0.text").String() != "Turn 1 user" {
		t.Fatalf("turn 0 mismatch: %s", contents[0].Raw)
	}
	if contents[1].Get("role").String() != "model" || contents[1].Get("parts.0.text").String() != "Turn 1 assistant" {
		t.Fatalf("turn 1 mismatch: %s", contents[1].Raw)
	}
	if contents[2].Get("role").String() != "user" || contents[2].Get("parts.0.text").String() != "<image_resize_notice>Image 1 was resized to 800x600</image_resize_notice>" {
		t.Fatalf("turn 2 mismatch: %s", contents[2].Raw)
	}
	if contents[3].Get("role").String() != "user" || contents[3].Get("parts.0.text").String() != "Turn 2 user" {
		t.Fatalf("turn 3 mismatch: %s", contents[3].Raw)
	}
}
