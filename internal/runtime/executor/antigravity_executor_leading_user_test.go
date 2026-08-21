package executor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestEnsureAntigravityGeminiLeadingUserContentSkipsClaude(t *testing.T) {
	payload := []byte(`{"request":{"contents":[{"role":"model","parts":[{"text":"prior output"}]},{"role":"user","parts":[{"text":"continue"}]}]}}`)

	gemini := ensureAntigravityGeminiLeadingUserContent("gemini-3.7-flash", payload)
	contents := gjson.GetBytes(gemini, "request.contents").Array()
	if len(contents) != 3 || contents[0].Get("role").String() != "user" || contents[1].Get("role").String() != "model" {
		t.Fatalf("Gemini leading contents malformed: %s", gemini)
	}

	claude := ensureAntigravityGeminiLeadingUserContent("claude-sonnet-4-6", payload)
	if &claude[0] != &payload[0] {
		t.Fatal("Claude payload should remain unchanged")
	}
	claudeContents := gjson.GetBytes(claude, "request.contents").Array()
	if len(claudeContents) != 2 || claudeContents[0].Get("role").String() != "model" {
		t.Fatalf("Claude leading contents changed: %s", claude)
	}
}

func TestAntigravityCountTokensPrependsLeadingUser(t *testing.T) {
	captured := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			t.Errorf("read request body: %v", errRead)
			return
		}
		captured <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalTokens":42}`))
	}))
	defer server.Close()

	auth := testAntigravityAuth(server.URL)
	auth.Metadata["project_id"] = "project-1"
	executor := NewAntigravityExecutor(&config.Config{RequestRetry: 1})
	_, errCount := executor.CountTokens(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "gemini-3.7-flash",
		Payload: []byte(`{"contents":[{"role":"model","parts":[{"text":"prior output"}]},{"role":"user","parts":[{"text":"continue"}]}]}`),
	}, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatGemini})
	if errCount != nil {
		t.Fatalf("CountTokens() error = %v", errCount)
	}

	contents := gjson.GetBytes(<-captured, "request.contents").Array()
	if len(contents) != 3 || contents[0].Get("role").String() != "user" || contents[1].Get("role").String() != "model" {
		t.Fatalf("countTokens leading contents malformed: %s", contents)
	}
	if text := contents[0].Get("parts.0.text"); !text.Exists() || text.String() != "" {
		t.Fatalf("countTokens synthetic user missing: %s", contents[0].Raw)
	}
}
