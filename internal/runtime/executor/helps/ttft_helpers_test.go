package helps

import (
	"context"
	"testing"
	"time"
)

func TestObserveChatTokenEvent_Behavior(t *testing.T) {
	ctx := context.Background()
	reporter := NewUsageReporter(ctx, "openai", "gpt-5.4", nil)
	reporter.StartResponseTTFT()
	time.Sleep(10 * time.Millisecond)

	// 1. Initial state
	if reporter.IsTTFTSet() {
		t.Fatalf("expected IsTTFTSet() to be false initially")
	}

	// 2. Nil / empty checks
	ObserveChatTokenEvent(nil, []byte(`data: {"choices":[{"delta":{"content":"hi"}}]}`))
	ObserveChatTokenEvent(reporter, nil)
	ObserveChatTokenEvent(reporter, []byte(""))
	if reporter.IsTTFTSet() {
		t.Fatalf("nil or empty payload must not trigger IsTTFTSet()")
	}

	// 3. Metadata chunk (role announcement only) sets first packet fallback, not effective TTFT
	metadataChunk := []byte(`data: {"choices":[{"delta":{"role":"assistant"}}]}`)
	ObserveChatTokenEvent(reporter, metadataChunk)
	if reporter.IsTTFTSet() {
		t.Fatalf("role announcement must not trigger IsTTFTSet()")
	}
	if !reporter.IsFirstPacketSet() {
		t.Fatalf("expected IsFirstPacketSet() == true")
	}

	// 4. Token content chunk triggers effective TTFT
	tokenChunk := []byte(`data: {"choices":[{"delta":{"content":"Hello"}}]}`)
	ObserveChatTokenEvent(reporter, tokenChunk)
	if !reporter.IsTTFTSet() {
		t.Fatalf("content delta must trigger IsTTFTSet()")
	}
	tokenTTFT := reporter.ttftDuration()
	if tokenTTFT <= 0 {
		t.Fatalf("expected token TTFT > 0, got %v", tokenTTFT)
	}

	// 5. Subsequent calls do not overwrite TTFT
	ObserveChatTokenEvent(reporter, []byte(`data: {"choices":[{"delta":{"content":" world"}}]}`))
	if reporter.ttftDuration() != tokenTTFT {
		t.Fatalf("subsequent ObserveChatTokenEvent must not modify already set TTFT")
	}
}

func TestObserveClaudeTokenEvent_Behavior(t *testing.T) {
	ctx := context.Background()
	reporter := NewUsageReporter(ctx, "claude", "claude-3-7-sonnet-20250219", nil)
	reporter.StartResponseTTFT()
	time.Sleep(10 * time.Millisecond)

	// 1. Initial state
	if reporter.IsTTFTSet() {
		t.Fatalf("expected IsTTFTSet() to be false initially")
	}

	// 2. Nil / empty checks
	ObserveClaudeTokenEvent(nil, []byte(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"hi"}}`))
	ObserveClaudeTokenEvent(reporter, nil)
	ObserveClaudeTokenEvent(reporter, []byte(""))
	if reporter.IsTTFTSet() {
		t.Fatalf("nil or empty payload must not trigger IsTTFTSet()")
	}

	// 3. Message start metadata sets first packet fallback, not effective TTFT
	messageStart := []byte(`event: message_start` + "\n" + `data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant"}}`)
	ObserveClaudeTokenEvent(reporter, messageStart)
	if reporter.IsTTFTSet() {
		t.Fatalf("message_start must not trigger IsTTFTSet()")
	}
	if !reporter.IsFirstPacketSet() {
		t.Fatalf("expected IsFirstPacketSet() == true")
	}

	// 4. Content delta triggers effective TTFT
	contentDelta := []byte(`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`)
	ObserveClaudeTokenEvent(reporter, contentDelta)
	if !reporter.IsTTFTSet() {
		t.Fatalf("content_block_delta must trigger IsTTFTSet()")
	}
	tokenTTFT := reporter.ttftDuration()
	if tokenTTFT <= 0 {
		t.Fatalf("expected token TTFT > 0, got %v", tokenTTFT)
	}

	// 5. Subsequent calls do not overwrite TTFT
	ObserveClaudeTokenEvent(reporter, []byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`))
	if reporter.ttftDuration() != tokenTTFT {
		t.Fatalf("subsequent ObserveClaudeTokenEvent must not modify already set TTFT")
	}
}

func TestObserveGeminiTokenEvent_Behavior(t *testing.T) {
	ctx := context.Background()
	reporter := NewUsageReporter(ctx, "gemini", "gemini-2.5-flash", nil)
	reporter.StartResponseTTFT()
	time.Sleep(10 * time.Millisecond)

	// 1. Initial state
	if reporter.IsTTFTSet() {
		t.Fatalf("expected IsTTFTSet() to be false initially")
	}

	// 2. Nil / empty checks
	ObserveGeminiTokenEvent(nil, []byte(`data: {"candidates":[{"content":{"parts":[{"text":"hi"}]}}]}`))
	ObserveGeminiTokenEvent(reporter, nil)
	ObserveGeminiTokenEvent(reporter, []byte(""))
	if reporter.IsTTFTSet() {
		t.Fatalf("nil or empty payload must not trigger IsTTFTSet()")
	}

	// 3. Non-token metadata sets first packet fallback, not effective TTFT
	metadataChunk := []byte(`data: {"usageMetadata":{"promptTokenCount":10}}`)
	ObserveGeminiTokenEvent(reporter, metadataChunk)
	if reporter.IsTTFTSet() {
		t.Fatalf("usage-only frame must not trigger IsTTFTSet()")
	}
	if !reporter.IsFirstPacketSet() {
		t.Fatalf("expected IsFirstPacketSet() == true")
	}

	// 4. Content part triggers effective TTFT
	contentChunk := []byte(`data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}`)
	ObserveGeminiTokenEvent(reporter, contentChunk)
	if !reporter.IsTTFTSet() {
		t.Fatalf("content part must trigger IsTTFTSet()")
	}
	tokenTTFT := reporter.ttftDuration()
	if tokenTTFT <= 0 {
		t.Fatalf("expected token TTFT > 0, got %v", tokenTTFT)
	}

	// 5. Subsequent calls do not overwrite TTFT
	ObserveGeminiTokenEvent(reporter, []byte(`data: {"candidates":[{"content":{"parts":[{"text":" world"}]}}]}`))
	if reporter.ttftDuration() != tokenTTFT {
		t.Fatalf("subsequent ObserveGeminiTokenEvent must not modify already set TTFT")
	}
}
