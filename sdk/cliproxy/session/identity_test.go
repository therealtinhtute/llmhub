package session

import (
	"net/http"
	"strings"
	"testing"

	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
)

func TestDeriveIDStableAcrossConversationGrowth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format sdktranslator.Format
		first  string
		later  string
	}{
		{
			name:   "openai chat",
			format: sdktranslator.FormatOpenAI,
			first:  `{"messages":[{"role":"system","content":"system prompt"},{"role":"developer","content":"developer prompt"},{"role":"user","content":"complete first user prompt"}]}`,
			later:  `{"messages":[{"role":"system","content":"system prompt"},{"role":"developer","content":"developer prompt"},{"role":"user","content":"complete first user prompt"},{"role":"assistant","content":"answer"},{"role":"developer","content":"later instruction"},{"role":"user","content":"next"}]}`,
		},
		{
			name:   "claude messages",
			format: sdktranslator.FormatClaude,
			first:  `{"system":[{"type":"text","text":"system prompt"}],"messages":[{"role":"user","content":[{"type":"text","text":"complete first user prompt"}]}]}`,
			later:  `{"system":[{"type":"text","text":"system prompt"}],"messages":[{"role":"user","content":[{"type":"text","text":"complete first user prompt"}]},{"role":"assistant","content":"answer"},{"role":"user","content":"next"}]}`,
		},
		{
			name:   "openai responses",
			format: sdktranslator.FormatOpenAIResponse,
			first:  `{"instructions":"system prompt","input":[{"type":"message","role":"developer","content":[{"type":"input_text","text":"developer prompt"}]},{"type":"message","role":"user","content":[{"type":"input_text","text":"complete first user prompt"}]}]}`,
			later:  `{"instructions":"system prompt","input":[{"type":"message","role":"developer","content":[{"type":"input_text","text":"developer prompt"}]},{"type":"message","role":"user","content":[{"type":"input_text","text":"complete first user prompt"}]},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"answer"}]},{"type":"message","role":"user","content":[{"type":"input_text","text":"next"}]}]}`,
		},
		{
			name:   "gemini",
			format: sdktranslator.FormatGemini,
			first:  `{"systemInstruction":{"parts":[{"text":"system prompt"}]},"contents":[{"role":"user","parts":[{"text":"complete first user prompt"}]}]}`,
			later:  `{"systemInstruction":{"parts":[{"text":"system prompt"}]},"contents":[{"role":"user","parts":[{"text":"complete first user prompt"}]},{"role":"model","parts":[{"text":"answer"}]},{"role":"user","parts":[{"text":"next"}]}]}`,
		},
		{
			name:   "interactions",
			format: sdktranslator.FormatInteractions,
			first:  `{"system_instruction":"system prompt","input":[{"type":"developer_instruction","text":"developer prompt"},{"type":"user_input","content":[{"type":"text","text":"complete first user prompt"}]}]}`,
			later:  `{"system_instruction":"system prompt","input":[{"type":"developer_instruction","text":"developer prompt"},{"type":"user_input","content":[{"type":"text","text":"complete first user prompt"}]},{"type":"model_output","content":[{"type":"text","text":"answer"}]},{"type":"user_input","content":[{"type":"text","text":"next"}]}]}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			firstID := DeriveID(test.format, []byte(test.first), "caller-a")
			laterID := DeriveID(test.format, []byte(test.later), "caller-a")
			if firstID == "" {
				t.Fatal("DeriveID() returned empty")
			}
			if firstID != laterID {
				t.Fatalf("conversation growth changed identity: first=%q later=%q", firstID, laterID)
			}
		})
	}
}

func TestDeriveIDInstructionPrefixAndFullUser(t *testing.T) {
	t.Parallel()

	prefix := strings.Repeat("界", 50)
	first := []byte(`{"messages":[{"role":"system","content":"` + prefix + `timestamp-a"},{"role":"user","content":"` + strings.Repeat("u", 120) + `a"}]}`)
	sameRoot := []byte(`{"messages":[{"role":"system","content":"` + prefix + `timestamp-b"},{"role":"user","content":"` + strings.Repeat("u", 120) + `a"}]}`)
	differentUser := []byte(`{"messages":[{"role":"system","content":"` + prefix + `timestamp-b"},{"role":"user","content":"` + strings.Repeat("u", 120) + `b"}]}`)

	firstID := DeriveID(sdktranslator.FormatOpenAI, first, "caller-a")
	if firstID == "" {
		t.Fatal("DeriveID() returned empty")
	}
	if got := DeriveID(sdktranslator.FormatOpenAI, sameRoot, "caller-a"); got != firstID {
		t.Fatalf("content after 50 Unicode characters changed identity: got=%q want=%q", got, firstID)
	}
	if got := DeriveID(sdktranslator.FormatOpenAI, differentUser, "caller-a"); got == firstID {
		t.Fatal("different full first user prompt produced the same identity")
	}
}

func TestDeriveIDCallerIsolationAndGeminiCachedContent(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"messages":[{"role":"user","content":"same prompt"}]}`)
	callerA := DeriveID(sdktranslator.FormatOpenAI, payload, CallerScope("api-key-a"))
	callerB := DeriveID(sdktranslator.FormatOpenAI, payload, CallerScope("api-key-b"))
	if callerA == "" || callerB == "" || callerA == callerB {
		t.Fatalf("caller isolation failed: callerA=%q callerB=%q", callerA, callerB)
	}

	firstCached := []byte(`{"cachedContent":"cachedContents/abc","contents":[{"role":"user","parts":[{"text":"first"}]}]}`)
	grownCached := []byte(`{"cachedContent":"cachedContents/abc","contents":[{"role":"user","parts":[{"text":"first"}]},{"role":"model","parts":[{"text":"answer"}]},{"role":"user","parts":[{"text":"next"}]}]}`)
	differentCached := []byte(`{"cachedContent":"cachedContents/abc","contents":[{"role":"user","parts":[{"text":"different"}]}]}`)
	firstID := DeriveID(sdktranslator.FormatGemini, firstCached, "caller-a")
	grownID := DeriveID(sdktranslator.FormatGemini, grownCached, "caller-a")
	differentID := DeriveID(sdktranslator.FormatGemini, differentCached, "caller-a")
	if firstID == "" || firstID != grownID {
		t.Fatalf("cachedContent conversation growth changed identity: first=%q grown=%q", firstID, grownID)
	}
	if differentID == firstID {
		t.Fatalf("different first user prompts sharing cachedContent produced the same identity: %q", firstID)
	}
}

func TestDeriveIDRequiresFirstUser(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"messages":[{"role":"system","content":"shared system"}]}`)
	if got := DeriveID(sdktranslator.FormatOpenAI, payload, "caller-a"); got != "" {
		t.Fatalf("DeriveID() = %q, want empty without first user", got)
	}
}

func TestEnrichSkipsDerivationForExplicitSessions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		payload         []byte
		headers         http.Header
		requestMetadata map[string]any
		optionMetadata  map[string]any
	}{
		{
			name:    "session header avoids malformed body parsing",
			payload: []byte(`not-json`),
			headers: http.Header{"X-Session-ID": []string{"header-session"}},
		},
		{
			name:    "Claude Code session header",
			payload: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			headers: http.Header{"X-Claude-Code-Session-Id": []string{"claude-session"}},
		},
		{
			name:    "later valid multi-value session header",
			payload: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			headers: http.Header{"X-Session-Affinity": []string{"", "later-valid-session"}},
		},
		{
			name:    "OpenCode affinity header",
			payload: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			headers: http.Header{"X-Session-Affinity": []string{"opencode-session"}},
		},
		{
			name:    "Responses conversation object",
			payload: []byte(`{"conversation":{"id":"conversation-session"},"messages":[{"role":"user","content":"hello"}]}`),
		},
		{
			name:    "Responses conversation string",
			payload: []byte(`{"conversation":"conversation-session","messages":[{"role":"user","content":"hello"}]}`),
		},
		{
			name:    "metadata user id",
			payload: []byte(`{"metadata":{"user_id":"explicit-user"},"messages":[{"role":"user","content":"hello"}]}`),
		},
		{
			name: "long legacy Claude metadata session",
			payload: []byte(`{"metadata":{"user_id":"` + strings.Repeat("x", 300) +
				`_session_ac980658-63bd-4fb3-97ba-8da64cb1e344"},"messages":[{"role":"user","content":"hello"}]}`),
		},
		{
			name:    "JSON metadata user id without nested session",
			payload: []byte(`{"metadata":{"user_id":"{\"device_id\":\"abc123\"}"},"messages":[{"role":"user","content":"hello"}]}`),
		},
		{
			name:    "body session id",
			payload: []byte(`{"session_id":"body-session","messages":[{"role":"user","content":"hello"}]}`),
		},
		{
			name:    "prompt cache key",
			payload: []byte(`{"prompt_cache_key":"cache-session","input":"hello"}`),
		},
		{
			name:           "execution session option metadata",
			payload:        []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			optionMetadata: map[string]any{cliproxyexecutor.ExecutionSessionMetadataKey: "execution-session"},
		},
		{
			name:            "execution session request metadata",
			payload:         []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			requestMetadata: map[string]any{cliproxyexecutor.ExecutionSessionMetadataKey: "execution-session"},
		},
		{
			name:    "explicit header removes stale derived identity",
			payload: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			headers: http.Header{"x-session-id": []string{"header-session"}},
			optionMetadata: map[string]any{
				cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:stale",
			},
		},
		{
			name:    "nested request sessionId",
			payload: []byte(`{"request":{"sessionId":"nested-session"},"messages":[{"role":"user","content":"hello"}]}`),
		},
		{
			name:    "nested request subagent",
			payload: []byte(`{"request":{"sessionId":"nested-session","metadata":{"agent_id":"worker"}},"messages":[{"role":"user","content":"hello"}]}`),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			req := cliproxyexecutor.Request{Payload: test.payload, Metadata: test.requestMetadata}
			opts := cliproxyexecutor.Options{
				OriginalRequest: test.payload,
				SourceFormat:    sdktranslator.FormatOpenAI,
				Headers:         test.headers,
				Metadata:        test.optionMetadata,
			}
			enrichedReq, enrichedOpts := Enrich(req, opts)
			if got := DerivedID(enrichedReq.Metadata); got != "" {
				t.Fatalf("request DerivedSessionID = %q, want empty", got)
			}
			if got := DerivedID(enrichedOpts.Metadata); got != "" {
				t.Fatalf("options DerivedSessionID = %q, want empty", got)
			}
			if test.name == "execution session option metadata" || test.name == "execution session request metadata" {
				if got := metadataString(enrichedReq.Metadata, cliproxyexecutor.ExecutionSessionMetadataKey); got != "execution-session" {
					t.Fatalf("request execution session = %q, want execution-session", got)
				}
				if got := metadataString(enrichedOpts.Metadata, cliproxyexecutor.ExecutionSessionMetadataKey); got != "execution-session" {
					t.Fatalf("options execution session = %q, want execution-session", got)
				}
			}
		})
	}
}

func TestEnrichDerivesAfterInvalidSessionIdentity(t *testing.T) {
	t.Parallel()

	baseMessages := `"input":"hello"`
	tests := []struct {
		name            string
		payload         []byte
		headers         http.Header
		requestMetadata map[string]any
		optionMetadata  map[string]any
	}{
		{
			name:    "oversized prompt cache key",
			payload: []byte(`{"prompt_cache_key":"` + strings.Repeat("x", 257) + `",` + baseMessages + `}`),
		},
		{
			name:    "trailing control character prompt cache key",
			payload: []byte(`{"prompt_cache_key":"tenant\n",` + baseMessages + `}`),
		},
		{
			name:    "leading control character prompt cache key",
			payload: []byte(`{"prompt_cache_key":"\ttenant",` + baseMessages + `}`),
		},
		{
			name:    "control character session header",
			payload: []byte(`{` + baseMessages + `}`),
			headers: http.Header{"X-Session-Affinity": []string{"bad\nsession"}},
		},
		{
			name:           "oversized execution session option metadata",
			payload:        []byte(`{"input":"hello"}`),
			optionMetadata: map[string]any{cliproxyexecutor.ExecutionSessionMetadataKey: strings.Repeat("x", 257)},
		},
		{
			name:            "control character execution session request metadata",
			payload:         []byte(`{"input":"hello"}`),
			requestMetadata: map[string]any{cliproxyexecutor.ExecutionSessionMetadataKey: "bad\nsession"},
		},
		{
			name:           "oversized retained derived session option metadata",
			payload:        []byte(`{"input":"hello"}`),
			optionMetadata: map[string]any{cliproxyexecutor.DerivedSessionIDMetadataKey: strings.Repeat("x", 257)},
		},
		{
			name:            "control character retained derived session request metadata",
			payload:         []byte(`{"input":"hello"}`),
			requestMetadata: map[string]any{cliproxyexecutor.DerivedSessionIDMetadataKey: "bad\nsession"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			req := cliproxyexecutor.Request{Payload: test.payload, Metadata: test.requestMetadata}
			opts := cliproxyexecutor.Options{
				OriginalRequest: test.payload,
				SourceFormat:    sdktranslator.FormatOpenAIResponse,
				Headers:         test.headers,
				Metadata:        test.optionMetadata,
			}
			enrichedReq, enrichedOpts := Enrich(req, opts)
			requestID := DerivedID(enrichedReq.Metadata)
			optionsID := DerivedID(enrichedOpts.Metadata)
			wantID := DeriveID(sdktranslator.FormatOpenAIResponse, test.payload, "")
			if requestID != wantID || optionsID != wantID {
				t.Fatalf("derived identities = request:%q options:%q, want %q", requestID, optionsID, wantID)
			}
			if got := metadataString(enrichedReq.Metadata, cliproxyexecutor.ExecutionSessionMetadataKey); got != "" {
				t.Fatalf("request execution session = %q, want invalid value removed", got)
			}
			if got := metadataString(enrichedOpts.Metadata, cliproxyexecutor.ExecutionSessionMetadataKey); got != "" {
				t.Fatalf("options execution session = %q, want invalid value removed", got)
			}
		})
	}
}

func TestEnrichCopiesDerivedIdentityToRequestAndOptions(t *testing.T) {
	t.Parallel()

	req := cliproxyexecutor.Request{Payload: []byte(`{"messages":[{"role":"user","content":"hello"}]}`)}
	opts := cliproxyexecutor.Options{
		OriginalRequest: req.Payload,
		SourceFormat:    sdktranslator.FormatOpenAI,
		Metadata:        map[string]any{cliproxyexecutor.CallerScopeMetadataKey: "caller-a"},
	}

	enrichedReq, enrichedOpts := Enrich(req, opts)
	reqID := DerivedID(enrichedReq.Metadata)
	optsID := DerivedID(enrichedOpts.Metadata)
	if reqID == "" || reqID != optsID {
		t.Fatalf("derived metadata mismatch: request=%q options=%q", reqID, optsID)
	}
	if _, exists := req.Metadata[cliproxyexecutor.DerivedSessionIDMetadataKey]; exists {
		t.Fatal("Enrich() mutated original request metadata")
	}
}

func TestDeriveIDAntigravityNestedRequestAndEmptyFirstUser(t *testing.T) {
	t.Parallel()

	nestedAntigravity := []byte(`{
		"project_id": "test-project",
		"request": {
			"systemInstruction": {"parts":[{"text":"system prompt"}]},
			"contents": [
				{"role":"user","parts":[{"text":""}]},
				{"role":"user","parts":[{"text":"actual user prompt"}]}
			]
		}
	}`)
	id := DeriveID(sdktranslator.FormatAntigravity, nestedAntigravity, "caller-a")
	if id == "" {
		t.Fatal("DeriveID returned empty for nested Antigravity request with empty first turn")
	}

	directAntigravity := []byte(`{
		"systemInstruction": {"parts":[{"text":"system prompt"}]},
		"contents": [
			{"role":"user","parts":[{"text":"actual user prompt"}]}
		]
	}`)
	directID := DeriveID(sdktranslator.FormatAntigravity, directAntigravity, "caller-a")
	if id != directID {
		t.Fatalf("DeriveID mismatch: nested=%s, direct=%s", id, directID)
	}
}

// Ported from upstream CLIProxyAPI commits 6b187e778ceb ("normalize reported session
// hierarchy to canonical UUIDv8", which introduced NormalizeToCanonicalUUID) and
// 1119ef142466 ("harden canonical UUIDv8 normalization for empty prefixes and context
// roots") against the local NormalizeToCanonicalUUID symbol.
func TestNormalizeToCanonicalUUID(t *testing.T) {
	t.Parallel()

	// 1. Empty, whitespace, and bare/empty prefixes
	if got := NormalizeToCanonicalUUID(""); got != "" {
		t.Fatalf("NormalizeToCanonicalUUID(\"\") = %q, want empty", got)
	}
	if got := NormalizeToCanonicalUUID("   "); got != "" {
		t.Fatalf("NormalizeToCanonicalUUID(\"   \") = %q, want empty", got)
	}
	emptyPrefixCases := []string{
		"lcp:v1:", "lcp:",
		"ctx:v1:", "ctx:",
		"codex:", "claude:", "header:", "session:",
		"affinity:", "slot:", "task:", "conv:",
		"thread:", "clientreq:", "geminicache:",
		"pck:", "user:", "execution:", "agy:", "derived:",
		"slot:   ",
		"task:   ",
		"derived:ctx:v1:",
		"derived:slot:   ",
	}
	for _, input := range emptyPrefixCases {
		if got := NormalizeToCanonicalUUID(input); got != "" {
			t.Fatalf("NormalizeToCanonicalUUID(%q) = %q, want empty", input, got)
		}
	}

	// 2. Native UUIDs (v4 and v7, various casings)
	rawUUIDv4 := "b2839f64-668d-4dc3-a42a-64da829d1e33"
	if got := NormalizeToCanonicalUUID(rawUUIDv4); got != rawUUIDv4 {
		t.Fatalf("NormalizeToCanonicalUUID(rawUUIDv4) = %q, want %q", got, rawUUIDv4)
	}
	upperUUID := "B2839F64-668D-4DC3-A42A-64DA829D1E33"
	if got := NormalizeToCanonicalUUID(upperUUID); got != rawUUIDv4 {
		t.Fatalf("NormalizeToCanonicalUUID(upperUUID) = %q, want %q", got, rawUUIDv4)
	}
	rawUUIDv7 := "01a07e72-c84d-7fd3-8207-d217b41cc649"
	if got := NormalizeToCanonicalUUID(rawUUIDv7); got != rawUUIDv7 {
		t.Fatalf("NormalizeToCanonicalUUID(rawUUIDv7) = %q, want %q", got, rawUUIDv7)
	}

	// 3. Known prefixes with UUIDs
	prefixedCases := map[string]string{
		"codex:01a07e72-c84d-7fd3-8207-d217b41cc649":         "01a07e72-c84d-7fd3-8207-d217b41cc649",
		"claude:b2839f64-668d-4dc3-a42a-64da829d1e33":        "b2839f64-668d-4dc3-a42a-64da829d1e33",
		"header:7a8b9c0d-1111-2222-3333-444455556666":        "7a8b9c0d-1111-2222-3333-444455556666",
		"session:b2839f64-668d-4dc3-a42a-64da829d1e33":       "b2839f64-668d-4dc3-a42a-64da829d1e33",
		"thread:01a07e72-c84d-7fd3-8207-d217b41cc649":        "01a07e72-c84d-7fd3-8207-d217b41cc649",
		"custom-prefix:01a07e72-c84d-7fd3-8207-d217b41cc649": "01a07e72-c84d-7fd3-8207-d217b41cc649",
	}
	for input, want := range prefixedCases {
		got := NormalizeToCanonicalUUID(input)
		if got != want {
			t.Errorf("NormalizeToCanonicalUUID(%q) = %q, want %q", input, got, want)
		}
	}

	// 4. LCP 64-hex and non-UUID inputs projected to RFC 9562 UUIDv8
	nonUUIDCases := []string{
		"lcp:v1:c28621bab78eacdb3ae128c0f6aaa0147842f063fda10ae9dc5473cc81d58985",
		"lcp:c28621bab78eacdb3ae128c0f6aaa0147842f063fda10ae9dc5473cc81d58985",
		"c28621bab78eacdb3ae128c0f6aaa0147842f063fda10ae9dc5473cc81d58985",
		"claude:b2839f64-668d-4dc3-a42a-64da829d1e33:agent:worker-reviewer",
		"task:task-abc-1",
		"slot:pi-slot-789",
		"ses_f8189891effeCLIq0MasUgMQsC",
		"my-custom-test-task",
	}

	for _, input := range nonUUIDCases {
		got := NormalizeToCanonicalUUID(input)
		if len(got) != 36 {
			t.Errorf("NormalizeToCanonicalUUID(%q) length = %d, want 36", input, len(got))
		}
		if !canonicalUUIDPattern.MatchString(got) {
			t.Errorf("NormalizeToCanonicalUUID(%q) = %q, does not match UUID pattern", input, got)
		}
		// Verify RFC 9562 Version 8 (13th char is '8', index 14)
		if got[14] != '8' {
			t.Errorf("NormalizeToCanonicalUUID(%q) = %q, version char is %c, want '8'", input, got, got[14])
		}
		// Verify RFC 4122 Variant (17th char is '8', '9', 'a', or 'b', index 19)
		variantChar := got[19]
		if variantChar != '8' && variantChar != '9' && variantChar != 'a' && variantChar != 'b' {
			t.Errorf("NormalizeToCanonicalUUID(%q) = %q, variant char is %c, want 8/9/a/b", input, got, variantChar)
		}
		// Verify Idempotency
		if reGot := NormalizeToCanonicalUUID(got); reGot != got {
			t.Errorf("NormalizeToCanonicalUUID is not idempotent: first=%q, second=%q", got, reGot)
		}
	}

	// 5. Uniqueness and determinism
	id1 := NormalizeToCanonicalUUID("lcp:v1:hash-A")
	id1Again := NormalizeToCanonicalUUID("lcp:v1:hash-A")
	id2 := NormalizeToCanonicalUUID("lcp:v1:hash-B")
	if id1 != id1Again {
		t.Fatalf("NormalizeToCanonicalUUID is non-deterministic: %q != %q", id1, id1Again)
	}
	if id1 == id2 {
		t.Fatalf("NormalizeToCanonicalUUID collided for different inputs: %q == %q", id1, id2)
	}

	// 6. Same content with or without "lcp:v1:" prefix produces identical UUID
	lcpPrefixed := "lcp:v1:c28621bab78eacdb3ae128c0f6aaa0147842f063fda10ae9dc5473cc81d58985"
	lcpBare := "c28621bab78eacdb3ae128c0f6aaa0147842f063fda10ae9dc5473cc81d58985"
	if NormalizeToCanonicalUUID(lcpPrefixed) != NormalizeToCanonicalUUID(lcpBare) {
		t.Fatalf("lcp prefixed (%q) and bare (%q) produced different UUIDs",
			NormalizeToCanonicalUUID(lcpPrefixed), NormalizeToCanonicalUUID(lcpBare))
	}

	// 7. Same content with or without "ctx:v1:" / "ctx:" prefix produces identical UUID
	ctxV1Prefixed := "ctx:v1:c28621bab78eacdb3ae128c0f6aaa0147842f063fda10ae9dc5473cc81d58985"
	ctxShortPrefixed := "ctx:c28621bab78eacdb3ae128c0f6aaa0147842f063fda10ae9dc5473cc81d58985"
	if NormalizeToCanonicalUUID(ctxV1Prefixed) != NormalizeToCanonicalUUID(lcpBare) {
		t.Fatalf("ctx:v1: prefixed (%q) and bare (%q) produced different UUIDs: %q vs %q",
			ctxV1Prefixed, lcpBare, NormalizeToCanonicalUUID(ctxV1Prefixed), NormalizeToCanonicalUUID(lcpBare))
	}
	if NormalizeToCanonicalUUID(ctxShortPrefixed) != NormalizeToCanonicalUUID(lcpBare) {
		t.Fatalf("ctx: prefixed (%q) and bare (%q) produced different UUIDs: %q vs %q",
			ctxShortPrefixed, lcpBare, NormalizeToCanonicalUUID(ctxShortPrefixed), NormalizeToCanonicalUUID(lcpBare))
	}
	// Fixed Golden UUIDv8 check
	const wantGoldenUUID = "2ad1939c-98ca-81da-8b69-3d084d5614c4"
	if got := NormalizeToCanonicalUUID(lcpBare); got != wantGoldenUUID {
		t.Fatalf("NormalizeToCanonicalUUID(lcpBare) = %q, want golden %q", got, wantGoldenUUID)
	}

	// 8. Chained prefixes (e.g. "derived:ctx:v1:") are stripped iteratively
	derivedCtxPrefixed := "derived:ctx:v1:c28621bab78eacdb3ae128c0f6aaa0147842f063fda10ae9dc5473cc81d58985"
	if got := NormalizeToCanonicalUUID(derivedCtxPrefixed); got != NormalizeToCanonicalUUID(lcpBare) {
		t.Fatalf("derived:ctx:v1: prefixed (%q) produced %q, want %q",
			derivedCtxPrefixed, got, NormalizeToCanonicalUUID(lcpBare))
	}

	// 9. Chained prefixes with standard UUID
	derivedUUID := "derived:ctx:v1:01a07e72-c84d-7fd3-8207-d217b41cc649"
	if got := NormalizeToCanonicalUUID(derivedUUID); got != "01a07e72-c84d-7fd3-8207-d217b41cc649" {
		t.Fatalf("NormalizeToCanonicalUUID(%q) = %q, want 01a07e72-c84d-7fd3-8207-d217b41cc649", derivedUUID, got)
	}
}

// Ported from upstream CLIProxyAPI commit 6b187e778ceb: ClaudeMetadataIdentities
// derives parent and subagent identities from alternate harness hierarchy keys
// (parent_agent_id, parent_id, subagent_id) in both JSON and legacy user_id forms.
func TestClaudeMetadataIdentitiesHarnessHierarchyFallbacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		payload    string
		wantID     string
		wantParent string
		wantAgent  string
	}{
		{
			name:       "json parent_agent_id and subagent_id",
			payload:    `{"metadata":{"user_id":"{\"session_id\":\"sess-1\",\"parent_agent_id\":\"agent-parent-9\",\"subagent_id\":\"sub-7\"}"}}`,
			wantID:     "sess-1",
			wantParent: "agent-parent-9",
			wantAgent:  "sub-7",
		},
		{
			name:       "json parent_id fallback",
			payload:    `{"metadata":{"user_id":"{\"session_id\":\"sess-2\",\"parent_id\":\"parent-3\"}"}}`,
			wantID:     "sess-2",
			wantParent: "parent-3",
			wantAgent:  "",
		},
		{
			name:       "legacy suffix with top-level parent and agent metadata",
			payload:    `{"metadata":{"user_id":"acct_session_ac980658-63bd-4fb3-97ba-8da64cb1e344","parent_agent_id":"p-1","subagent_id":"s-1"}}`,
			wantID:     "ac980658-63bd-4fb3-97ba-8da64cb1e344",
			wantParent: "p-1",
			wantAgent:  "s-1",
		},
		{
			name:       "legacy suffix with parent_session_id and agent_id metadata",
			payload:    `{"metadata":{"user_id":"acct_session_ac980658-63bd-4fb3-97ba-8da64cb1e344","parent_session_id":"p-2","agent_id":"a-2"}}`,
			wantID:     "ac980658-63bd-4fb3-97ba-8da64cb1e344",
			wantParent: "p-2",
			wantAgent:  "a-2",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sessionID, parentSessionID, agentID := ClaudeMetadataIdentities([]byte(test.payload))
			if sessionID != test.wantID {
				t.Fatalf("sessionID = %q, want %q", sessionID, test.wantID)
			}
			if parentSessionID != test.wantParent {
				t.Fatalf("parentSessionID = %q, want %q", parentSessionID, test.wantParent)
			}
			if agentID != test.wantAgent {
				t.Fatalf("agentID = %q, want %q", agentID, test.wantAgent)
			}
		})
	}
}

// Ported from upstream CLIProxyAPI commit 6b187e778ceb: hasExplicitSession treats
// harness hierarchy headers (Codex turn metadata, subagent, task, parent IDs) and
// payload fields (task_id, parent_id, action_id, fork sources) as explicit session
// signals, so Enrich must not derive a fallback identity for them.
func TestEnrichSkipsDerivationForHarnessHierarchySignals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload []byte
		headers http.Header
	}{
		{
			name:    "codex turn metadata header",
			payload: []byte(`{"input":"hello"}`),
			headers: http.Header{"X-Codex-Turn-Metadata": []string{`{"session_id":"codex-turn-1","thread_id":"thread-1"}`}},
		},
		{
			name:    "openai subagent header",
			payload: []byte(`{"input":"hello"}`),
			headers: http.Header{"X-Openai-Subagent": []string{"collab_spawn"}},
		},
		{
			name:    "task id header",
			payload: []byte(`{"input":"hello"}`),
			headers: http.Header{"X-Task-ID": []string{"task-1"}},
		},
		{
			name:    "generic parent id header",
			payload: []byte(`{"input":"hello"}`),
			headers: http.Header{"X-Parent-ID": []string{"parent-1"}},
		},
		{
			name:    "parent task id header",
			payload: []byte(`{"input":"hello"}`),
			headers: http.Header{"X-Parent-Task-Id": []string{"task-parent-1"}},
		},
		{
			name:    "parent conversation header",
			payload: []byte(`{"input":"hello"}`),
			headers: http.Header{"X-Parent-Conversation-Id": []string{"conv-parent-1"}},
		},
		{
			name:    "body task id",
			payload: []byte(`{"task_id":"task-body-1","input":"hello"}`),
		},
		{
			name:    "body parent id",
			payload: []byte(`{"parent_id":"parent-body-1","input":"hello"}`),
		},
		{
			name:    "body action id",
			payload: []byte(`{"action_id":"action-1","input":"hello"}`),
		},
		{
			name:    "body child session id",
			payload: []byte(`{"child_session_id":"child-1","input":"hello"}`),
		},
		{
			name:    "metadata parent agent id",
			payload: []byte(`{"metadata":{"parent_agent_id":"pa-1"},"input":"hello"}`),
		},
		{
			name:    "forkSource session id",
			payload: []byte(`{"forkSource":{"sessionId":"fork-src-1"},"input":"hello"}`),
		},
		{
			name:    "previousSessionId",
			payload: []byte(`{"previousSessionId":"prev-1","input":"hello"}`),
		},
		{
			name:    "nested request task id",
			payload: []byte(`{"request":{"task_id":"nested-task-1"},"input":"hello"}`),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			req := cliproxyexecutor.Request{Payload: test.payload}
			opts := cliproxyexecutor.Options{
				OriginalRequest: test.payload,
				SourceFormat:    sdktranslator.FormatOpenAIResponse,
				Headers:         test.headers,
			}
			enrichedReq, enrichedOpts := Enrich(req, opts)
			if got := DerivedID(enrichedReq.Metadata); got != "" {
				t.Fatalf("request DerivedSessionID = %q, want empty", got)
			}
			if got := DerivedID(enrichedOpts.Metadata); got != "" {
				t.Fatalf("options DerivedSessionID = %q, want empty", got)
			}
		})
	}
}

func TestClaudeMetadataIdentitiesNormalizesAgentID(t *testing.T) {
	t.Parallel()

	validPayload := []byte(`{
		"metadata": {
			"user_id": "{\"session_id\":\"sess-123\",\"parent_session_id\":\"parent-456\",\"agent_id\":\"  subagent-alpha  \"}"
		}
	}`)
	sessionID, parentSessionID, agentID := ClaudeMetadataIdentities(validPayload)
	if sessionID != "sess-123" {
		t.Fatalf("sessionID = %q, want sess-123", sessionID)
	}
	if parentSessionID != "parent-456" {
		t.Fatalf("parentSessionID = %q, want parent-456", parentSessionID)
	}
	if agentID != "subagent-alpha" {
		t.Fatalf("agentID = %q, want subagent-alpha", agentID)
	}

	invalidPayload := []byte(`{
		"metadata": {
			"user_id": "{\"session_id\":\"sess-123\",\"agent_id\":\"bad\nagent\"}"
		}
	}`)
	_, _, badAgentID := ClaudeMetadataIdentities(invalidPayload)
	if badAgentID != "" {
		t.Fatalf("expected badAgentID to be empty for control character, got %q", badAgentID)
	}
}
