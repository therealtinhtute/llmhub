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
