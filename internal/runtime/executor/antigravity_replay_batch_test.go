package executor

// Ported from upstream CLIProxyAPI commits:
//   - e44432ab85fd (perf(antigravity): batch replay degradation rewrites)
//   - acf919ce50fb (perf(antigravity): batch reasoning replay mutations)
//   - d8f2dceef789 (perf(antigravity): batch functionResponse name repairs)
// Local symbols under test: degradeAntigravityClaudeToolProvenanceIDs,
// applyAntigravityReasoningReplayItems, antigravityReplayBatch,
// repairAntigravityGeminiFunctionResponseNames, applyAntigravityIndexedEdits.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// TestRepairAntigravityGeminiFunctionResponseNames verifies that missing and
// placeholder functionResponse names are repaired from the matching
// functionCall while valid names and unmapped responses stay untouched.
func TestRepairAntigravityGeminiFunctionResponseNames(t *testing.T) {
	payload := []byte(`{"request":{"contents":[` +
		`{"role":"model","parts":[{"functionCall":{"id":"call_1","name":"search","args":{"q":"x"}}}]},` +
		`{"role":"user","parts":[` +
		`{"functionResponse":{"id":"call_1","name":"unknown","response":{"r":1}}},` +
		`{"functionResponse":{"id":"call_1","response":{"r":2}}},` +
		`{"functionResponse":{"id":"call_1","name":"custom","response":{"r":3}}},` +
		`{"functionResponse":{"id":"call_2","response":{"r":4}}}` +
		`]}` +
		`]}}`)

	out := repairAntigravityGeminiFunctionResponseNames(payload)
	if !gjson.ValidBytes(out) {
		t.Fatalf("repair produced invalid JSON: %s", out)
	}
	names := gjson.GetBytes(out, "request.contents.1.parts.#.functionResponse.name").Array()
	got := []string{names[0].String(), names[1].String(), names[2].String()}
	if got[0] != "search" || got[1] != "search" || got[2] != "custom" {
		t.Fatalf("unexpected repaired names: %v", got)
	}
	if id := gjson.GetBytes(out, "request.contents.1.parts.3.functionResponse.id").String(); id != "call_2" {
		t.Fatalf("unmapped response mutated: id=%q", id)
	}
	if gjson.GetBytes(out, "request.contents.1.parts.3.functionResponse.name").Exists() {
		t.Fatal("unmapped response gained a name")
	}

	// Idempotent: a second pass must be a byte-exact no-op.
	if again := repairAntigravityGeminiFunctionResponseNames(out); !bytes.Equal(again, out) {
		t.Fatalf("repair not idempotent:\nfirst:  %s\nsecond: %s", out, again)
	}
	// No call map at all: payload returned untouched.
	noCalls := []byte(`{"request":{"contents":[{"role":"user","parts":[{"functionResponse":{"id":"x","response":{}}}]}]}}`)
	if out := repairAntigravityGeminiFunctionResponseNames(noCalls); !bytes.Equal(out, noCalls) {
		t.Fatalf("no-call payload mutated: %s", out)
	}
}

// TestDegradeAntigravityClaudeToolProvenanceIDsBatch verifies that opaque
// Claude provenance IDs on both functionCall and functionResponse parts are
// rewritten to synthetic IDs with one splice, while non-opaque parts and
// thought signatures are preserved.
func TestDegradeAntigravityClaudeToolProvenanceIDsBatch(t *testing.T) {
	const calls = 8
	var modelParts, userParts strings.Builder
	for i := 0; i < calls; i++ {
		if i > 0 {
			modelParts.WriteByte(',')
			userParts.WriteByte(',')
		}
		fmt.Fprintf(&modelParts, `{"functionCall":{"id":"cpa_claude_tool_%d","name":"tool%d","args":{"i":%d}},"thoughtSignature":"sig-%d"}`, i, i, i, i)
		fmt.Fprintf(&userParts, `{"functionResponse":{"id":"cpa_claude_tool_%d","name":"tool%d","response":{"ok":true}}}`, i, i)
	}
	payload := []byte(fmt.Sprintf(`{"request":{"contents":[`+
		`{"role":"model","parts":[%s,{"functionCall":{"id":"plain_id","name":"keep","args":{}}}]},`+
		`{"role":"user","parts":[%s,{"functionResponse":{"id":"plain_id","name":"keep","response":{}}}]}]}}`,
		modelParts.String(), userParts.String()))

	out, degraded := degradeAntigravityClaudeToolProvenanceIDs(payload)
	if degraded != calls*2 {
		t.Fatalf("degraded count = %d, want %d", degraded, calls*2)
	}
	if !gjson.ValidBytes(out) {
		t.Fatalf("degrade produced invalid JSON: %s", out)
	}
	for i := 0; i < calls; i++ {
		wantID := antigravitySyntheticToolCallID(fmt.Sprintf("cpa_claude_tool_%d", i))
		if got := gjson.GetBytes(out, fmt.Sprintf("request.contents.0.parts.%d.functionCall.id", i)).String(); got != wantID {
			t.Fatalf("call %d id = %q, want %q", i, got, wantID)
		}
		if got := gjson.GetBytes(out, fmt.Sprintf("request.contents.1.parts.%d.functionResponse.id", i)).String(); got != wantID {
			t.Fatalf("response %d id = %q, want %q", i, got, wantID)
		}
		if got := gjson.GetBytes(out, fmt.Sprintf("request.contents.0.parts.%d.thoughtSignature", i)).String(); got != fmt.Sprintf("sig-%d", i) {
			t.Fatalf("call %d lost thoughtSignature: %q", i, got)
		}
	}
	if got := gjson.GetBytes(out, "request.contents.0.parts.8.functionCall.id").String(); got != "plain_id" {
		t.Fatalf("non-opaque call id mutated: %q", got)
	}
	if got := gjson.GetBytes(out, "request.contents.1.parts.8.functionResponse.id").String(); got != "plain_id" {
		t.Fatalf("non-opaque response id mutated: %q", got)
	}
	// Second pass finds nothing left to degrade.
	if again, n := degradeAntigravityClaudeToolProvenanceIDs(out); n != 0 || !bytes.Equal(again, out) {
		t.Fatalf("second degrade pass changed payload (n=%d)", n)
	}
}

// t3SignedReplayPayload builds a conversation whose model turn carries
// Gemini thought signatures. Items extracted from it replay into the same
// payload with signatures stripped.
func t3SignedReplayPayload() []byte {
	return []byte(`{"request":{"contents":[` +
		`{"role":"user","parts":[{"text":"call the tool"}]},` +
		`{"role":"model","parts":[` +
		`{"text":"thinking ahead","thoughtSignature":"sig-text-1"},` +
		`{"functionCall":{"id":"call_a","name":"search","args":{"q":"x"}},"thoughtSignature":"sig-fc-1"}` +
		`]},` +
		`{"role":"user","parts":[{"functionResponse":{"id":"call_a","name":"search","response":{"ok":true}}}]}` +
		`]}}`)
}

func t3StripThoughtSignatures(t *testing.T, payload []byte) []byte {
	t.Helper()
	out := payload
	contents := gjson.GetBytes(payload, "request.contents")
	if !contents.IsArray() {
		t.Fatal("fixture missing request.contents")
	}
	for ci, content := range contents.Array() {
		parts := content.Get("parts")
		if !parts.IsArray() {
			continue
		}
		for pi, part := range parts.Array() {
			if !part.Get("thoughtSignature").Exists() {
				continue
			}
			var err error
			out, err = sjson.DeleteBytes(out, fmt.Sprintf("request.contents.%d.parts.%d.thoughtSignature", ci, pi))
			if err != nil {
				t.Fatalf("strip signature: %v", err)
			}
		}
	}
	return out
}

// TestApplyAntigravityReasoningReplayItemsBatchMatchesSequential replays items
// extracted from a signed payload into its unsigned twin and requires the
// batched splice to be byte-identical to the sequential implementation.
func TestApplyAntigravityReasoningReplayItemsBatchMatchesSequential(t *testing.T) {
	signed := t3SignedReplayPayload()
	items := antigravityReasoningReplayItemsFromRequest(signed)
	if len(items) < 2 {
		t.Fatalf("expected at least 2 replay items, got %d", len(items))
	}
	degraded := t3StripThoughtSignatures(t, signed)
	if bytes.Contains(degraded, []byte("sig-text-1")) || bytes.Contains(degraded, []byte("sig-fc-1")) {
		t.Fatal("fixture still contains signatures")
	}

	batched, batchChanged := applyAntigravityReasoningReplayItems(degraded, items, nil)
	sequential, seqChanged := applyAntigravityReasoningReplayItemsSequential(degraded, items, nil)

	if !batchChanged || !seqChanged {
		t.Fatalf("replay did not apply (batch=%t seq=%t)", batchChanged, seqChanged)
	}
	if !bytes.Equal(batched, sequential) {
		t.Fatalf("batch output diverged from sequential:\nbatch: %s\nseq:   %s", batched, sequential)
	}
	if got := gjson.GetBytes(batched, "request.contents.1.parts.0.thoughtSignature").String(); got != "sig-text-1" {
		t.Fatalf("text signature not replayed: %q", got)
	}
	if got := gjson.GetBytes(batched, "request.contents.1.parts.1.thoughtSignature").String(); got != "sig-fc-1" {
		t.Fatalf("function call signature not replayed: %q", got)
	}
	// Input payload must not be mutated in place.
	if !gjson.ValidBytes(degraded) || bytes.Contains(degraded, []byte("sig-fc-1")) {
		t.Fatal("replay mutated the source payload")
	}
}

// TestApplyAntigravityReasoningReplayItemsFallbackInterleaved forces a
// non-batchable item between two batchable ones; the interleaved sequential
// fallback must still produce the same result as the pure sequential path.
func TestApplyAntigravityReasoningReplayItemsFallbackInterleaved(t *testing.T) {
	signed := t3SignedReplayPayload()
	items := antigravityReasoningReplayItemsFromRequest(signed)
	if len(items) < 2 {
		t.Fatalf("expected at least 2 replay items, got %d", len(items))
	}
	degraded := t3StripThoughtSignatures(t, signed)

	// A function_call_part item whose call ID does not exist cannot be staged
	// in the batch and must route through the sequential fallback.
	bogus := []byte(`{"type":"function_call_part","contentIndex":1,"partIndex":1,` +
		`"targetOccurrence":0,"name":"missing_tool","call_id":"call_missing",` +
		`"args":{"q":"none"},"thoughtSignature":"sig-bogus"}`)
	interleaved := [][]byte{items[0], bogus, items[1]}

	batched, _ := applyAntigravityReasoningReplayItems(degraded, interleaved, nil)
	sequential, _ := applyAntigravityReasoningReplayItemsSequential(degraded, interleaved, nil)
	if !bytes.Equal(batched, sequential) {
		t.Fatalf("interleaved batch output diverged:\nbatch: %s\nseq:   %s", batched, sequential)
	}
	if bytes.Contains(batched, []byte("sig-bogus")) {
		t.Fatal("unlocatable item was applied")
	}
	if got := gjson.GetBytes(batched, "request.contents.1.parts.0.thoughtSignature").String(); got != "sig-text-1" {
		t.Fatalf("text signature not replayed after fallback: %q", got)
	}
	if got := gjson.GetBytes(batched, "request.contents.1.parts.1.thoughtSignature").String(); got != "sig-fc-1" {
		t.Fatalf("function call signature not replayed after fallback: %q", got)
	}
}
