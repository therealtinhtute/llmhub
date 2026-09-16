package helps

import (
	"net/http"
	"strings"
	"testing"
)

// Focused parity tests for Claude upstream request continuity and probe/helper
// classification helpers ported from upstream CLIProxyAPI (SHAs cited per test).

// TestBeginCommitClaudeContinuity_TracksPrevReqPromptID verifies request-ID and
// prompt-ID continuity across turns (upstream 086ad91bd970).
func TestBeginCommitClaudeContinuity_TracksPrevReqPromptID(t *testing.T) {
	ResetClaudeDiagnosticsForTest()
	defer ResetClaudeDiagnosticsForTest()

	// Turn 1: new prompt turn produces a fresh prompt ID and no previous IDs.
	key, seq1, prevMsg, prevReq, prompt1 := BeginClaudeContinuity("cred-1", "sess-1", true, "")
	if prevMsg != "" || prevReq != "" || prompt1 == "" {
		t.Fatalf("turn 1 = prevMsg:%q prevReq:%q prompt:%q, want empty prevs and fresh prompt", prevMsg, prevReq, prompt1)
	}
	CommitClaudeContinuity(key, seq1, "msg_01aaa", "req_01bbb", prompt1)

	// Tool continuation turn reuses the active prompt ID and surfaces the
	// committed previous message/request IDs.
	_, seq2, prevMsg, prevReq, prompt2 := BeginClaudeContinuity("cred-1", "sess-1", false, "")
	if prevMsg != "msg_01aaa" || prevReq != "req_01bbb" {
		t.Fatalf("continuation = prevMsg:%q prevReq:%q, want msg_01aaa / req_01bbb", prevMsg, prevReq)
	}
	if prompt2 != prompt1 {
		t.Fatalf("continuation prompt = %q, want same prompt %q", prompt2, prompt1)
	}
	CommitClaudeContinuity(key, seq2, "msg_01ccc", "req_01ddd", prompt2)

	// A new prompt turn advances message/request IDs but rotates the prompt ID.
	_, _, prevMsg, prevReq, prompt3 := BeginClaudeContinuity("cred-1", "sess-1", true, "")
	if prevMsg != "msg_01ccc" || prevReq != "req_01ddd" {
		t.Fatalf("turn 2 = prevMsg:%q prevReq:%q, want msg_01ccc / req_01ddd", prevMsg, prevReq)
	}
	if prompt3 == prompt1 {
		t.Fatalf("turn 2 prompt = %q, want a fresh prompt different from %q", prompt3, prompt1)
	}
}

// TestCommitClaudeContinuity_RejectsInvalidCommit verifies that continuity only
// advances on complete, in-order responses (upstream 086ad91bd970).
func TestCommitClaudeContinuity_RejectsInvalidCommit(t *testing.T) {
	ResetClaudeDiagnosticsForTest()
	defer ResetClaudeDiagnosticsForTest()

	key, seq1, _, _, _ := BeginClaudeContinuity("cred-2", "sess-2", true, "")
	// Empty message ID (truncated stream) must not commit.
	CommitClaudeContinuity(key, seq1, "", "req_01aaa")
	_, _, prevMsg, prevReq, _ := BeginClaudeContinuity("cred-2", "sess-2", false, "")
	if prevMsg != "" || prevReq != "" {
		t.Fatalf("empty-message commit advanced continuity: %q %q", prevMsg, prevReq)
	}

	// Stale sequence from an older generation cannot overwrite newer state.
	CommitClaudeContinuity(key, seq1, "msg_01ok", "req_01ok")
	key2, seq3, _, _, _ := BeginClaudeContinuity("cred-2", "sess-2", false, "")
	_ = key2
	CommitClaudeContinuity(key, seq1, "msg_stale", "req_stale")
	CommitClaudeContinuity(key, seq3, "msg_01new", "req_01new")
	_, _, prevMsg, prevReq, _ = BeginClaudeContinuity("cred-2", "sess-2", false, "")
	if prevMsg != "msg_01new" || prevReq != "req_01new" {
		t.Fatalf("stale commit corrupted continuity: %q %q", prevMsg, prevReq)
	}
}

// TestIsValidClaudePromptID verifies strict RFC 4122 UUIDv4 semantics
// (upstream 086ad91bd970).
func TestIsValidClaudePromptID(t *testing.T) {
	valid := "3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f"
	if !IsValidClaudePromptID(valid) {
		t.Fatalf("IsValidClaudePromptID(%q) = false, want true", valid)
	}
	for _, invalid := range []string{
		"",
		"not-a-uuid",
		"3f3f3f3f3f3f43f383f33f3f3f3f3f3f",     // no dashes
		"3f3f3f3f-3f3f-33f3-83f3-3f3f3f3f3f3f", // version 3, not 4
		"3f3f3f3f-3f3f-43f3-03f3-3f3f3f3f3f3f", // variant nibble not 8-b
		"zz3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f", // non-hex
	} {
		if IsValidClaudePromptID(invalid) {
			t.Fatalf("IsValidClaudePromptID(%q) = true, want false", invalid)
		}
	}
}

// TestClaudeDeterministicPromptID_StableAndValid verifies the seeded prompt ID
// fallback is deterministic and a valid UUIDv4 (upstream 086ad91bd970).
func TestClaudeDeterministicPromptID_StableAndValid(t *testing.T) {
	a := ClaudeDeterministicPromptID("cpa:prompt:seed-1")
	b := ClaudeDeterministicPromptID("cpa:prompt:seed-1")
	c := ClaudeDeterministicPromptID("cpa:prompt:seed-2")
	if a != b {
		t.Fatalf("deterministic prompt ID not stable: %q vs %q", a, b)
	}
	if a == c {
		t.Fatalf("distinct seeds produced identical prompt ID: %q", a)
	}
	if !IsValidClaudePromptID(a) {
		t.Fatalf("deterministic prompt ID %q is not a valid RFC 4122 UUIDv4", a)
	}
}

// TestClaudePayloadHas1hTTL verifies 1h cache TTL detection across tools,
// system, and message blocks (upstream d7052c96af78).
func TestClaudePayloadHas1hTTL(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    bool
	}{
		{"tool 1h", `{"tools":[{"name":"t","cache_control":{"type":"ephemeral","ttl":"1h"}}]}`, true},
		{"system 1h", `{"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral","ttl":"1h"}}]}`, true},
		{"message 1h", `{"messages":[{"role":"user","content":[{"type":"text","text":"m","cache_control":{"type":"ephemeral","ttl":"1h"}}]}]}`, true},
		{"5m only", `{"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral","ttl":"5m"}}]}`, false},
		{"no ttl", `{"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral"}}]}`, false},
		{"empty", `{}`, false},
	}
	for _, tc := range cases {
		if got := ClaudePayloadHas1hTTL([]byte(tc.payload)); got != tc.want {
			t.Fatalf("ClaudePayloadHas1hTTL(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestClaudeSubagentRequests1h verifies the subagent 1h opt-in check
// (upstream d7052c96af78).
func TestClaudeSubagentRequests1h(t *testing.T) {
	withTTL := []byte(`{"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral","ttl":"1h"}}]}`)
	if !ClaudeSubagentRequests1h(nil, withTTL) {
		t.Fatal("body with 1h TTL must count as requesting 1h")
	}
	headers := http.Header{"Anthropic-Beta": []string{"extended-cache-ttl-2025-04-11"}}
	if !ClaudeSubagentRequests1h(headers, nil) {
		t.Fatal("explicit extended-cache-ttl beta must count as requesting 1h")
	}
	if ClaudeSubagentRequests1h(nil, []byte(`{"system":[{"type":"text","text":"s"}]}`)) {
		t.Fatal("plain body must not count as requesting 1h")
	}
}

// TestExtractInjectStripBillingTags verifies billing tag helpers
// (upstream 4a5ab534f827).
func TestExtractInjectStripBillingTags(t *testing.T) {
	body := []byte(`{"system":[{"type":"text","text":"x-anthropic-billing-header: cc_version=2.1.258.abc; cc_entrypoint=cli; cch=00000; cc_prev_req=req_01x; cc_prompt_id=3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f;"}]}`)

	prevReq, promptID := ExtractClaudeBillingTags(body)
	if prevReq != "req_01x" || promptID != "3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f" {
		t.Fatalf("ExtractClaudeBillingTags = (%q, %q)", prevReq, promptID)
	}
	stripped := StripClaudeBillingTags(body)
	if strings.Contains(string(stripped), "cc_prev_req") || strings.Contains(string(stripped), "cc_prompt_id") {
		t.Fatalf("StripClaudeBillingTags left tags: %s", stripped)
	}
	reinjected := InjectClaudeBillingTags(stripped, "req_02y", "4f4f4f4f-4f4f-44f4-84f4-4f4f4f4f4f4f")
	prevReq, promptID = ExtractClaudeBillingTags(reinjected)
	if prevReq != "req_02y" || promptID != "4f4f4f4f-4f4f-44f4-84f4-4f4f4f4f4f4f" {
		t.Fatalf("InjectClaudeBillingTags round trip = (%q, %q)", prevReq, promptID)
	}
}
