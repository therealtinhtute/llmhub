package executor

import (
	"context"
	"testing"

	"github.com/tidwall/gjson"
)

// The shared OpenAI-Responses reasoning sanitizer tests live in
// meta_executor_signature_test.go (ported from upstream
// openai_responses_signature_test.go before this file existed). This file
// carries the compat-mode coverage added by upstream commit 81d6ba774621.

func TestSanitizeOpenAIResponsesReasoningEncryptedContentWithCompat_PreservesReasoningContentAndID(t *testing.T) {
	body := []byte(`{"store":false,"input":[` +
		`{"id":"rs_compat","type":"reasoning","summary":[],"content":[{"type":"reasoning_text","text":"keep cleartext thinking"}],"encrypted_content":null},` +
		`{"id":"msg_1","type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}` +
		`]}`)

	got := sanitizeOpenAIResponsesReasoningEncryptedContentWithCompat(context.Background(), "test", body, true)

	reasoningItem := gjson.GetBytes(got, "input.0")
	if gotID := reasoningItem.Get("id").String(); gotID != "rs_compat" {
		t.Fatalf("reasoning id = %q, want rs_compat; body=%s", gotID, got)
	}
	content := reasoningItem.Get("content")
	if !content.Exists() || !content.IsArray() || len(content.Array()) != 1 {
		t.Fatalf("content should have 1 item, got %s body=%s", content.Raw, got)
	}
	if gotType := reasoningItem.Get("content.0.type").String(); gotType != "reasoning_text" {
		t.Fatalf("content.0.type = %q, want reasoning_text; body=%s", gotType, got)
	}
	if gotText := reasoningItem.Get("content.0.text").String(); gotText != "keep cleartext thinking" {
		t.Fatalf("content.0.text = %q, want keep cleartext thinking; body=%s", gotText, got)
	}
	if gotLen := len(reasoningItem.Get("summary").Array()); gotLen != 0 {
		t.Fatalf("summary should remain empty for compat, got %d items; body=%s", gotLen, got)
	}
	if reasoningItem.Get("encrypted_content").Exists() {
		t.Fatalf("null encrypted_content should still be stripped: %s", got)
	}
}
