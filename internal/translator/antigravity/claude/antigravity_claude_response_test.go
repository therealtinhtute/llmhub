package claude

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/cache"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ============================================================================
// Signature Caching Tests
// ============================================================================

func TestConvertAntigravityResponseToClaude_ParamsInitialized(t *testing.T) {
	cache.ClearSignatureCache("")

	// Request with user message - should initialize params
	requestJSON := []byte(`{
		"messages": [
			{"role": "user", "content": [{"type": "text", "text": "Hello world"}]}
		]
	}`)

	// First response chunk with thinking
	responseJSON := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "Let me think...", "thought": true}]
				}
			}]
		}
	}`)

	var param any
	ctx := context.Background()
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, responseJSON, &param)

	params := param.(*Params)
	if !params.HasFirstResponse {
		t.Error("HasFirstResponse should be set after first chunk")
	}
	if params.CurrentThinkingText.Len() == 0 {
		t.Error("Thinking text should be accumulated")
	}
}

func TestConvertAntigravityResponseToClaude_ThinkingTextAccumulated(t *testing.T) {
	cache.ClearSignatureCache("")

	requestJSON := []byte(`{
		"messages": [{"role": "user", "content": [{"type": "text", "text": "Test"}]}]
	}`)

	// First thinking chunk
	chunk1 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "First part of thinking...", "thought": true}]
				}
			}]
		}
	}`)

	// Second thinking chunk (continuation)
	chunk2 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": " Second part of thinking...", "thought": true}]
				}
			}]
		}
	}`)

	var param any
	ctx := context.Background()

	// Process first chunk - starts new thinking block
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, chunk1, &param)
	params := param.(*Params)

	if params.CurrentThinkingText.Len() == 0 {
		t.Error("Thinking text should be accumulated after first chunk")
	}

	// Process second chunk - continues thinking block
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, chunk2, &param)

	text := params.CurrentThinkingText.String()
	if !strings.Contains(text, "First part") || !strings.Contains(text, "Second part") {
		t.Errorf("Thinking text should accumulate both parts, got: %s", text)
	}
}

func TestConvertAntigravityResponseToClaude_SignatureCached(t *testing.T) {
	cache.ClearSignatureCache("")

	requestJSON := []byte(`{
		"model": "claude-sonnet-4-5-thinking",
		"messages": [{"role": "user", "content": [{"type": "text", "text": "Cache test"}]}]
	}`)

	// Thinking chunk
	thinkingChunk := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "My thinking process here", "thought": true}]
				}
			}]
		}
	}`)

	// Signature chunk
	validSignature := "abc123validSignature1234567890123456789012345678901234567890"
	signatureChunk := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "", "thought": true, "thoughtSignature": "` + validSignature + `"}]
				}
			}]
		}
	}`)

	var param any
	ctx := context.Background()

	// Process thinking chunk
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, thinkingChunk, &param)
	params := param.(*Params)
	thinkingText := params.CurrentThinkingText.String()

	if thinkingText == "" {
		t.Fatal("Thinking text should be accumulated")
	}

	// Process signature chunk - should cache the signature
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, signatureChunk, &param)

	// Verify signature was cached
	cachedSig := cache.GetCachedSignature("claude-sonnet-4-5-thinking", thinkingText)
	if cachedSig != validSignature {
		t.Errorf("Expected cached signature '%s', got '%s'", validSignature, cachedSig)
	}

	// Verify thinking text was reset after caching
	if params.CurrentThinkingText.Len() != 0 {
		t.Error("Thinking text should be reset after signature is cached")
	}
}

func TestConvertAntigravityResponseToClaude_MultipleThinkingBlocks(t *testing.T) {
	cache.ClearSignatureCache("")

	requestJSON := []byte(`{
		"model": "claude-sonnet-4-5-thinking",
		"messages": [{"role": "user", "content": [{"type": "text", "text": "Multi block test"}]}]
	}`)

	validSig1 := "signature1_12345678901234567890123456789012345678901234567"
	validSig2 := "signature2_12345678901234567890123456789012345678901234567"

	// First thinking block with signature
	block1Thinking := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "First thinking block", "thought": true}]
				}
			}]
		}
	}`)
	block1Sig := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "", "thought": true, "thoughtSignature": "` + validSig1 + `"}]
				}
			}]
		}
	}`)

	// Text content (breaks thinking)
	textBlock := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "Regular text output"}]
				}
			}]
		}
	}`)

	// Second thinking block with signature
	block2Thinking := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "Second thinking block", "thought": true}]
				}
			}]
		}
	}`)
	block2Sig := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "", "thought": true, "thoughtSignature": "` + validSig2 + `"}]
				}
			}]
		}
	}`)

	var param any
	ctx := context.Background()

	// Process first thinking block
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, block1Thinking, &param)
	params := param.(*Params)
	firstThinkingText := params.CurrentThinkingText.String()

	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, block1Sig, &param)

	// Verify first signature cached
	if cache.GetCachedSignature("claude-sonnet-4-5-thinking", firstThinkingText) != validSig1 {
		t.Error("First thinking block signature should be cached")
	}

	// Process text (transitions out of thinking)
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, textBlock, &param)

	// Process second thinking block
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, block2Thinking, &param)
	secondThinkingText := params.CurrentThinkingText.String()

	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, block2Sig, &param)

	// Verify second signature cached
	if cache.GetCachedSignature("claude-sonnet-4-5-thinking", secondThinkingText) != validSig2 {
		t.Error("Second thinking block signature should be cached")
	}
}

func TestConvertAntigravityResponseToClaude_TextAndSignatureInSameChunk(t *testing.T) {
	cache.ClearSignatureCache("")

	requestJSON := []byte(`{
		"model": "claude-sonnet-4-5-thinking",
		"messages": [{"role": "user", "content": [{"type": "text", "text": "Test"}]}]
	}`)

	validSignature := "RtestSig1234567890123456789012345678901234567890123456789"

	// Chunk 1: thinking text only (no signature)
	chunk1 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "First part.", "thought": true}]
				}
			}]
		}
	}`)

	// Chunk 2: thinking text AND signature in the same part
	chunk2 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": " Second part.", "thought": true, "thoughtSignature": "` + validSignature + `"}]
				}
			}]
		}
	}`)

	var param any
	ctx := context.Background()

	result1 := ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, chunk1, &param)
	result2 := ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, chunk2, &param)

	allOutput := string(bytes.Join(result1, nil)) + string(bytes.Join(result2, nil))

	// The text " Second part." must appear as a thinking_delta, not be silently dropped
	if !strings.Contains(allOutput, "Second part.") {
		t.Error("Text co-located with signature must be emitted as thinking_delta before the signature")
	}

	// The signature must also be emitted
	if !strings.Contains(allOutput, "signature_delta") {
		t.Error("Signature delta must still be emitted")
	}

	// Verify the cached signature covers the FULL text (both parts)
	fullText := "First part. Second part."
	cachedSig := cache.GetCachedSignature("claude-sonnet-4-5-thinking", fullText)
	if cachedSig != validSignature {
		t.Errorf("Cached signature should cover full text %q, got sig=%q", fullText, cachedSig)
	}
}

func TestConvertAntigravityResponseToClaude_SignatureOnlyChunk(t *testing.T) {
	cache.ClearSignatureCache("")

	requestJSON := []byte(`{
		"model": "claude-sonnet-4-5-thinking",
		"messages": [{"role": "user", "content": [{"type": "text", "text": "Test"}]}]
	}`)

	validSignature := "RtestSig1234567890123456789012345678901234567890123456789"

	// Chunk 1: thinking text
	chunk1 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "Full thinking text.", "thought": true}]
				}
			}]
		}
	}`)

	// Chunk 2: signature only (empty text) — the normal case
	chunk2 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{"text": "", "thought": true, "thoughtSignature": "` + validSignature + `"}]
				}
			}]
		}
	}`)

	var param any
	ctx := context.Background()

	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, chunk1, &param)
	ConvertAntigravityResponseToClaude(ctx, "claude-sonnet-4-5-thinking", requestJSON, requestJSON, chunk2, &param)

	cachedSig := cache.GetCachedSignature("claude-sonnet-4-5-thinking", "Full thinking text.")
	if cachedSig != validSignature {
		t.Errorf("Signature-only chunk should still cache correctly, got %q", cachedSig)
	}
}

// Ported from upstream CLIProxyAPI commit 15231e9fdc93 ("support native thinking
// signatures without prefixes in claude translator"): formatClaudeSignatureValue
// emits provider-native signatures without "<group>#" prefixes, and the
// request-side round-trip accepts them in cache mode. The local translator does
// not emit signatures on tool_use content blocks, so only thinking signatures
// are covered here.
func TestConvertAntigravityResponseToClaude_EmitsNativeSignaturesWithoutProviderPrefixes(t *testing.T) {
	cache.ClearSignatureCache("")
	previousCache := cache.SignatureCacheEnabled()
	cache.SetSignatureCacheEnabled(true)
	t.Cleanup(func() {
		cache.SetSignatureCacheEnabled(previousCache)
		cache.ClearSignatureCache("")
	})

	nativeSig := testAnthropicNativeSignature(t)
	upstreamSig := base64.StdEncoding.EncodeToString([]byte(nativeSig))
	requestJSON := []byte(`{"model":"claude-sonnet-4-6"}`)
	responseJSON := []byte(`{"response":{"candidates":[{"content":{"parts":[{"text":"thought content","thought":true,"thoughtSignature":"` + upstreamSig + `"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1,"thoughtsTokenCount":1,"totalTokenCount":3},"modelVersion":"claude-sonnet-4-6-thinking","responseId":"resp-claude-native-sigs"}}`)

	nonStream := ConvertAntigravityResponseToClaudeNonStream(context.Background(), "claude-sonnet-4-6", requestJSON, requestJSON, responseJSON, nil)
	thinkingSig := gjson.GetBytes(nonStream, "content.0.signature").String()
	if strings.Contains(thinkingSig, "#") {
		t.Fatalf("thinking signature contains prefix: %q, want provider-native %q", thinkingSig, nativeSig)
	}
	if thinkingSig != nativeSig {
		t.Fatalf("thinking signature = %q, want provider-native %q", thinkingSig, nativeSig)
	}

	var param any
	stream := bytes.Join(ConvertAntigravityResponseToClaude(context.Background(), "claude-sonnet-4-6", requestJSON, requestJSON, responseJSON, &param), nil)
	streamText := string(stream)
	if strings.Contains(streamText, `"signature":"claude#`) {
		t.Fatalf("streaming response contains claude# prefix: %s", streamText)
	}
	if !strings.Contains(streamText, `"signature":"`+nativeSig+`"`) {
		t.Fatalf("streaming thinking signature missing or mismatched: %s", streamText)
	}

	// Verify multi-turn replay with native signatures in cache mode.
	replayRequest := []byte(`{"model":"claude-sonnet-4-6","messages":[{"role":"assistant","content":[]},{"role":"user","content":[{"type":"text","text":"continue"}]}]}`)
	replayRequest, _ = sjson.SetRawBytes(replayRequest, "messages.0.content", []byte(gjson.GetBytes(nonStream, "content").Raw))
	replayRequest = StripEmptySignatureThinkingBlocks(replayRequest)
	translated := ConvertClaudeRequestToAntigravity("claude-sonnet-4-6", replayRequest, false)
	parts := gjson.GetBytes(translated, "request.contents.0.parts").Array()
	if len(parts) != 1 || parts[0].Get("thoughtSignature").String() != upstreamSig {
		t.Fatalf("Claude native thought signature did not round-trip in cache mode: %s", translated)
	}
}
