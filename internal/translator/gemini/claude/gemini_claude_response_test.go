package claude

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertGeminiResponseToClaudeNonStream_PreservesThoughtSignature(t *testing.T) {
	requestJSON := []byte(`{"model":"gemini-2.5-pro","messages":[{"role":"user","content":"hi"}]}`)
	geminiResponse := []byte(`{
		"candidates": [{
			"content": {
				"parts": [
					{"text": "thinking step 1\n", "thought": true},
					{"text": "thinking step 2", "thought": true, "thoughtSignature": "sig-xyz-123"},
					{"text": "visible answer"}
				]
			},
			"finishReason": "STOP"
		}],
		"usageMetadata": {
			"promptTokenCount": 10,
			"candidatesTokenCount": 5
		},
		"modelVersion": "gemini-2.5-pro",
		"responseId": "resp-non-stream"
	}`)

	ctx := context.Background()
	output := ConvertGeminiResponseToClaudeNonStream(ctx, "gemini-2.5-pro", requestJSON, requestJSON, geminiResponse, nil)
	outputJSON := gjson.ParseBytes(output)

	blocks := outputJSON.Get("content").Array()
	if len(blocks) != 2 {
		t.Fatalf("expected 2 content blocks (thinking + text), got %d: %s", len(blocks), string(output))
	}

	thinkingBlock := blocks[0]
	if thinkingBlock.Get("type").String() != "thinking" {
		t.Fatalf("expected first block to be thinking, got %s", thinkingBlock.Get("type").String())
	}
	if thinkingBlock.Get("thinking").String() != "thinking step 1\nthinking step 2" {
		t.Fatalf("unexpected thinking content: %s", thinkingBlock.Get("thinking").String())
	}
	if thinkingBlock.Get("signature").String() != "sig-xyz-123" {
		t.Fatalf("expected signature 'sig-xyz-123', got %q. Output: %s", thinkingBlock.Get("signature").String(), string(output))
	}

	textBlock := blocks[1]
	if textBlock.Get("type").String() != "text" || textBlock.Get("text").String() != "visible answer" {
		t.Fatalf("unexpected text block: %s", textBlock.Raw)
	}
}

func TestConvertGeminiResponseToClaudeNonStream_PartWithThoughtSignatureWithoutThoughtBool(t *testing.T) {
	requestJSON := []byte(`{"model":"gemini-2.5-pro","messages":[{"role":"user","content":"hi"}]}`)
	geminiResponse := []byte(`{
		"candidates": [{
			"content": {
				"parts": [
					{"text": "inferred reasoning", "thought_signature": "sig-snake-case"},
					{"text": "final answer"}
				]
			},
			"finishReason": "STOP"
		}],
		"usageMetadata": {
			"promptTokenCount": 10,
			"candidatesTokenCount": 5
		},
		"modelVersion": "gemini-2.5-pro",
		"responseId": "resp-non-stream-2"
	}`)

	ctx := context.Background()
	output := ConvertGeminiResponseToClaudeNonStream(ctx, "gemini-2.5-pro", requestJSON, requestJSON, geminiResponse, nil)
	outputJSON := gjson.ParseBytes(output)

	blocks := outputJSON.Get("content").Array()
	if len(blocks) != 2 {
		t.Fatalf("expected 2 content blocks (thinking + text), got %d: %s", len(blocks), string(output))
	}

	thinkingBlock := blocks[0]
	if thinkingBlock.Get("type").String() != "thinking" {
		t.Fatalf("expected first block to be thinking, got %s", thinkingBlock.Get("type").String())
	}
	if thinkingBlock.Get("thinking").String() != "inferred reasoning" {
		t.Fatalf("unexpected thinking content: %s", thinkingBlock.Get("thinking").String())
	}
	if thinkingBlock.Get("signature").String() != "sig-snake-case" {
		t.Fatalf("expected signature 'sig-snake-case', got %q. Output: %s", thinkingBlock.Get("signature").String(), string(output))
	}

	textBlock := blocks[1]
	if textBlock.Get("type").String() != "text" || textBlock.Get("text").String() != "final answer" {
		t.Fatalf("unexpected text block: %s", textBlock.Raw)
	}
}

func TestConvertGeminiResponseToClaudeNonStream_TrailingSignatureOnlyPart(t *testing.T) {
	requestJSON := []byte(`{"model":"gemini-2.5-pro","messages":[{"role":"user","content":"hi"}]}`)
	geminiResponse := []byte(`{
		"candidates": [{
			"content": {
				"parts": [
					{"text": "thinking step 1\n", "thought": true},
					{"text": "", "thoughtSignature": "sig-trailing"},
					{"text": "visible answer"}
				]
			},
			"finishReason": "STOP"
		}],
		"usageMetadata": {
			"promptTokenCount": 10,
			"candidatesTokenCount": 5
		},
		"modelVersion": "gemini-2.5-pro",
		"responseId": "resp-non-stream-trailing"
	}`)

	ctx := context.Background()
	output := ConvertGeminiResponseToClaudeNonStream(ctx, "gemini-2.5-pro", requestJSON, requestJSON, geminiResponse, nil)
	outputJSON := gjson.ParseBytes(output)

	blocks := outputJSON.Get("content").Array()
	if len(blocks) != 2 {
		t.Fatalf("expected 2 content blocks (thinking + text), got %d: %s", len(blocks), string(output))
	}

	thinkingBlock := blocks[0]
	if thinkingBlock.Get("type").String() != "thinking" {
		t.Fatalf("expected first block to be thinking, got %s", thinkingBlock.Get("type").String())
	}
	if thinkingBlock.Get("thinking").String() != "thinking step 1\n" {
		t.Fatalf("unexpected thinking content: %s", thinkingBlock.Get("thinking").String())
	}
	if thinkingBlock.Get("signature").String() != "sig-trailing" {
		t.Fatalf("expected signature 'sig-trailing', got %q. Output: %s", thinkingBlock.Get("signature").String(), string(output))
	}

	textBlock := blocks[1]
	if textBlock.Get("type").String() != "text" || textBlock.Get("text").String() != "visible answer" {
		t.Fatalf("unexpected text block: %s", textBlock.Raw)
	}
}

func geminiClaudeInputJSONDeltas(t *testing.T, streamText string) string {
	t.Helper()
	var builder strings.Builder
	for _, line := range strings.Split(streamText, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if gjson.Get(payload, "delta.type").String() == "input_json_delta" {
			builder.WriteString(gjson.Get(payload, "delta.partial_json").String())
		}
	}
	return builder.String()
}

func TestConvertGeminiResponseToClaude_SanitizesParameterlessToolArguments(t *testing.T) {
	origRequest := []byte(`{"model":"gemini-3-flash","tools":[{"name":"EnterPlanMode","input_schema":{"type":"object","properties":{}}}]}`)
	rawResponse := []byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"EnterPlanMode","args":{"reason":"Plan mode switch"}}}],"role":"model"},"finishReason":"STOP"}],"modelVersion":"gemini-3-flash","responseId":"resp-zero-arg"}`)

	ctx := context.Background()

	nonStream := ConvertGeminiResponseToClaudeNonStream(ctx, "gemini-3-flash", origRequest, origRequest, rawResponse, nil)
	if got := gjson.GetBytes(nonStream, "content.0.input").Raw; got != "{}" {
		t.Fatalf("expected non-streaming input {}, got %s", got)
	}

	var param any
	streamBytes := bytes.Join(ConvertGeminiResponseToClaude(ctx, "gemini-3-flash", origRequest, origRequest, rawResponse, &param), nil)
	if got := geminiClaudeInputJSONDeltas(t, string(streamBytes)); got != "{}" {
		t.Fatalf("expected streaming partial_json {}, got %q (stream: %s)", got, string(streamBytes))
	}
}

func TestConvertGeminiResponseToClaude_CanonicalNameCollisionKeepsExactToolArguments(t *testing.T) {
	origRequest := []byte(`{"model":"gemini-3-flash","tools":[{"name":"Read","input_schema":{"type":"object","properties":{}}},{"name":"read","input_schema":{"type":"object","properties":{"path":{"type":"string"}}}}]}`)
	readWithArgs := []byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"read","args":{"path":"/tmp/a"}}}],"role":"model"},"finishReason":"STOP"}],"modelVersion":"gemini-3-flash","responseId":"resp-collision"}`)
	readParameterless := []byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"Read","args":{"reason":"x"}}}],"role":"model"},"finishReason":"STOP"}],"modelVersion":"gemini-3-flash","responseId":"resp-collision-2"}`)

	ctx := context.Background()

	nonStream := ConvertGeminiResponseToClaudeNonStream(ctx, "gemini-3-flash", origRequest, origRequest, readWithArgs, nil)
	if got := gjson.GetBytes(nonStream, "content.0.input").Raw; got != `{"path":"/tmp/a"}` {
		t.Fatalf("expected exact read tool args preserved, got %s (%s)", got, string(nonStream))
	}
	if got := gjson.GetBytes(nonStream, "content.0.name").String(); got != "read" {
		t.Fatalf("expected exact tool name read, got %q (%s)", got, string(nonStream))
	}

	var param any
	streamBytes := bytes.Join(ConvertGeminiResponseToClaude(ctx, "gemini-3-flash", origRequest, origRequest, readWithArgs, &param), nil)
	if got := geminiClaudeInputJSONDeltas(t, string(streamBytes)); got != `{"path":"/tmp/a"}` {
		t.Fatalf("expected streaming exact read args preserved, got %q (stream: %s)", got, string(streamBytes))
	}

	nonStreamParameterless := ConvertGeminiResponseToClaudeNonStream(ctx, "gemini-3-flash", origRequest, origRequest, readParameterless, nil)
	if got := gjson.GetBytes(nonStreamParameterless, "content.0.input").Raw; got != "{}" {
		t.Fatalf("expected exact Read tool normalized to {}, got %s", got)
	}
}

func TestConvertGeminiResponseToClaude_AmbiguousCanonicalNameKeepsUpstreamNameAndArguments(t *testing.T) {
	origRequest := []byte(`{"model":"gemini-3-flash","tools":[{"name":"Read","input_schema":{"type":"object","properties":{}}},{"name":"read","input_schema":{"type":"object","properties":{"path":{"type":"string"}}}}]}`)
	ambiguousResponse := []byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"READ","args":{"path":"/tmp/a"}}}],"role":"model"},"finishReason":"STOP"}],"modelVersion":"gemini-3-flash","responseId":"resp-ambiguous"}`)

	nonStream := ConvertGeminiResponseToClaudeNonStream(context.Background(), "gemini-3-flash", origRequest, origRequest, ambiguousResponse, nil)
	if got := gjson.GetBytes(nonStream, "content.0.name").String(); got != "READ" {
		t.Fatalf("expected ambiguous upstream name READ preserved, got %q (%s)", got, string(nonStream))
	}
	if got := gjson.GetBytes(nonStream, "content.0.input").Raw; got != `{"path":"/tmp/a"}` {
		t.Fatalf("expected ambiguous tool args preserved, got %s", got)
	}
}

func TestConvertGeminiResponseToClaude_KeepsReasonWhenTranslatedSchemaHasNoPlaceholder(t *testing.T) {
	origRequest := []byte(`{"model":"gemini-3-flash","tools":[{"name":"Bash","input_schema":{"type":"object","properties":{"command":{"type":"string"}},"required":["command"]}}]}`)
	ctx := context.Background()
	translatedRequest := ConvertClaudeRequestToGemini("gemini-3-flash", origRequest, false)
	rawResponse := []byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"Bash","args":{"command":"ls","reason":"why"}}}],"role":"model"},"finishReason":"STOP"}],"modelVersion":"gemini-3-flash","responseId":"resp-bash-reason"}`)

	nonStream := ConvertGeminiResponseToClaudeNonStream(ctx, "gemini-3-flash", origRequest, translatedRequest, rawResponse, nil)
	if got := gjson.GetBytes(nonStream, "content.0.input").Raw; got != `{"command":"ls","reason":"why"}` {
		t.Fatalf("expected non-streaming input preserved, got %s (translated: %s)", got, string(translatedRequest))
	}
}

func TestConvertGeminiResponseToClaude_ParameterlessToolContinuationDoesNotDuplicateEmptyObject(t *testing.T) {
	origRequest := []byte(`{"model":"gemini-3-flash","tools":[{"name":"EnterPlanMode","input_schema":{"type":"object","properties":{}}}]}`)
	firstChunk := []byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"EnterPlanMode","args":{"reason":"Plan mode switch"}}}],"role":"model"}}],"modelVersion":"gemini-3-flash","responseId":"resp-zero-arg"}`)
	continuationChunk := []byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"args":{"reason":"more"}}}]},"finishReason":"STOP"}],"modelVersion":"gemini-3-flash","responseId":"resp-zero-arg"}`)

	ctx := context.Background()
	var param any
	streamBytes := bytes.Join(ConvertGeminiResponseToClaude(ctx, "gemini-3-flash", origRequest, origRequest, firstChunk, &param), nil)
	streamBytes = append(streamBytes, bytes.Join(ConvertGeminiResponseToClaude(ctx, "gemini-3-flash", origRequest, origRequest, continuationChunk, &param), nil)...)
	if got := geminiClaudeInputJSONDeltas(t, string(streamBytes)); got != "{}" {
		t.Fatalf("expected concatenated partial_json {}, got %q (stream: %s)", got, string(streamBytes))
	}
}
