package chat_completions

import (
	"testing"

	"github.com/tidwall/gjson"
)

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
