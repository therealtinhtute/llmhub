package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/runtime/executor/helps"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	"github.com/tidwall/gjson"
)

// Focused parity tests for the Claude Code 2.1.280 fingerprint chain ported
// from upstream CLIProxyAPI. Each test cites the upstream SHA it verifies and
// names the local symbols it exercises.

// TestClaudeFingerprintBaseline_2_1_258 verifies the default software
// fingerprint tuple emitted by applyClaudeHeaders when the caller supplies no
// Claude Code headers (upstream df7e04ea2850).
func TestClaudeFingerprintBaseline_2_1_280(t *testing.T) {
	resetClaudeDeviceProfileCache()
	req := newClaudeHeaderTestRequest(t, http.Header{})
	auth := &cliproxyauth.Auth{ID: "auth-fp", Attributes: map[string]string{"api_key": "key-1"}}
	body := []byte(`{"model":"claude-opus-5"}`)
	applyClaudeHeaders(req, auth, "key-1", false, nil, body, nil)

	if got := req.Header.Get("User-Agent"); got != "claude-cli/2.1.280 (external, cli)" {
		t.Fatalf("User-Agent = %q, want claude-cli/2.1.280 (external, cli)", got)
	}
	if got := req.Header.Get("X-Stainless-Package-Version"); got != "0.112.1" {
		t.Fatalf("X-Stainless-Package-Version = %q, want 0.112.1", got)
	}
	if got := req.Header.Get("X-Stainless-Runtime-Version"); got != "v26.3.0" {
		t.Fatalf("X-Stainless-Runtime-Version = %q, want v26.3.0", got)
	}
}

// TestClaudeCodeCLIBetas_DynamicGating verifies per-request beta assembly and
// the model/turn gating rules (upstream d7052c96af78).
func TestClaudeCodeCLIBetas_DynamicGating(t *testing.T) {
	requested := map[string]bool{}

	// Baseline OAuth Sonnet request: claude-code first, oauth second, effort +
	// fallback-credit + extended-cache-ttl present.
	base := []byte(`{"model":"claude-sonnet-5","messages":[{"role":"user","content":"hi"}],"thinking":{"type":"enabled"}}`)
	betas := claudeCodeCLIBetas(base, requested, true)
	parts := strings.Split(betas, ",")
	if parts[0] != "claude-code-20250219" || parts[1] != "oauth-2025-04-20" {
		t.Fatalf("beta head = %q, want claude-code then oauth", betas)
	}
	for _, want := range []string{"effort-2025-11-24", "fallback-credit-2026-06-01", "extended-cache-ttl-2025-04-11", "redact-thinking-2026-02-12"} {
		if !strings.Contains(betas, want) {
			t.Fatalf("baseline betas %q missing %q", betas, want)
		}
	}
	if strings.Contains(betas, "server-side-fallback-2026-06-01") {
		t.Fatalf("baseline without fallbacks must not emit server-side-fallback, got %q", betas)
	}
	if strings.Contains(betas, "advanced-tool-use-2025-11-20") {
		t.Fatalf("plain request without advanced tools must not emit advanced-tool-use, got %q", betas)
	}

	// Haiku model: effort is pruned (upstream d7052c96af78).
	haiku := []byte(`{"model":"claude-haiku-4-5","messages":[{"role":"user","content":"hi"}]}`)
	if got := claudeCodeCLIBetas(haiku, requested, true); strings.Contains(got, "effort-2025-11-24") {
		t.Fatalf("haiku betas %q must not contain effort-2025-11-24", got)
	}

	// Probe request (max_tokens:1, single "quota" message): effort, fallback,
	// thinking-display-updates and extended-cache-ttl are all pruned.
	probe := []byte(`{"model":"claude-sonnet-5","max_tokens":1,"messages":[{"role":"user","content":"quota"}]}`)
	got := claudeCodeCLIBetas(probe, map[string]bool{claudeServerSideFallbackBeta: true, claudeExtendedCacheTTLBeta: true}, true)
	for _, absent := range []string{"effort-2025-11-24", "server-side-fallback-2026-06-01", "thinking-display-updates-2026-08-18", "extended-cache-ttl-2025-04-11"} {
		if strings.Contains(got, absent) {
			t.Fatalf("probe betas %q must not contain %q", got, absent)
		}
	}

	// Disabled thinking: effort and thinking-display-updates are pruned.
	disabled := []byte(`{"model":"claude-sonnet-5","messages":[{"role":"user","content":"hi"}],"thinking":{"type":"disabled","display":"updates"}}`)
	got = claudeCodeCLIBetas(disabled, requested, true)
	for _, absent := range []string{"effort-2025-11-24", "thinking-display-updates-2026-08-18"} {
		if strings.Contains(got, absent) {
			t.Fatalf("disabled-thinking betas %q must not contain %q", got, absent)
		}
	}

	// Adaptive thinking display: thinking-display-updates emitted and
	// redact-thinking suppressed when thinking.display is set.
	adaptive := []byte(`{"model":"claude-fable-5-1","messages":[{"role":"user","content":"hi"}],"thinking":{"type":"adaptive","display":"updates"},"fallbacks":[{"model":"claude-opus-5"}]}`)
	got = claudeCodeCLIBetas(adaptive, requested, true)
	for _, want := range []string{"thinking-display-updates-2026-08-18", "server-side-fallback-2026-06-01"} {
		if !strings.Contains(got, want) {
			t.Fatalf("adaptive betas %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "redact-thinking-2026-02-12") {
		t.Fatalf("thinking.display request must not emit redact-thinking, got %q", got)
	}

	// Advanced tool use requires a real feature marker; advisor tools emit the
	// advisor beta; afk-mode is forwarded only when requested.
	adv := []byte(`{"model":"claude-sonnet-5","tools":[{"name":"t","defer_loading":true}]}`)
	if got = claudeCodeCLIBetas(adv, requested, false); !strings.Contains(got, "advanced-tool-use-2025-11-20") {
		t.Fatalf("defer_loading tool must emit advanced-tool-use, got %q", got)
	}
	advisor := []byte(`{"model":"claude-sonnet-5","tools":[{"name":"a","type":"advisor_tool"}]}`)
	if got = claudeCodeCLIBetas(advisor, requested, false); !strings.Contains(got, "advisor-tool-2026-03-01") {
		t.Fatalf("advisor tool must emit advisor-tool-2026-03-01, got %q", got)
	}
	if got = claudeCodeCLIBetas(base, map[string]bool{"afk-mode-2026-01-31": true}, false); !strings.Contains(got, "afk-mode-2026-01-31") {
		t.Fatalf("requested afk-mode must be forwarded, got %q", got)
	}

	// Diagnostics object pairs with cache-diagnosis-2026-04-07.
	diag := []byte(`{"model":"claude-sonnet-5","diagnostics":{"previous_message_id":null}}`)
	if got = claudeCodeCLIBetas(diag, requested, false); !strings.Contains(got, "cache-diagnosis-2026-04-07") {
		t.Fatalf("diagnostics request must emit cache-diagnosis-2026-04-07, got %q", got)
	}

	// A body carrying 1h cache TTL pairs with extended-cache-ttl even on API
	// keys (upstream d7052c96af78 pairing rule).
	ttlBody := []byte(`{"model":"claude-sonnet-5","system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral","ttl":"1h"}}]}`)
	if got = claudeCodeCLIBetas(ttlBody, requested, false); !strings.Contains(got, "extended-cache-ttl-2025-04-11") {
		t.Fatalf("1h cache body must pair extended-cache-ttl, got %q", got)
	}
}

// TestGenerateBillingHeader_ContinuityChain verifies the billing header tag
// order and conditional tags (upstream 086ad91bd970).
func TestGenerateBillingHeader_ContinuityChain(t *testing.T) {
	payload := []byte(`{"messages":[{"role":"user","content":"hello world this is a long message"}]}`)
	header := generateBillingHeader(payload, true, "2.1.280", "fingerprint-source-text", "cli", "main", true, "req_01abc", "3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f")

	wantOrder := []string{"cc_version=2.1.280.", "cc_entrypoint=cli;", "cch=00000;", "cc_workload=main;", "cc_is_subagent=true;", "cc_prev_req=req_01abc;", "cc_prompt_id=3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f;"}
	if !strings.HasPrefix(header, "x-anthropic-billing-header: ") {
		t.Fatalf("billing header %q missing prefix", header)
	}
	last := -1
	for _, tag := range wantOrder {
		idx := strings.Index(header, tag)
		if idx < 0 {
			t.Fatalf("billing header %q missing tag %q", header, tag)
		}
		if idx <= last {
			t.Fatalf("billing header %q tag %q out of order", header, tag)
		}
		last = idx
	}

	// Unsigned path emits no cch and no continuity tags; non-subagent emits no
	// cc_is_subagent (upstream 086ad91bd970).
	unsigned := generateBillingHeader(payload, false, "2.1.280", "text", "cli", "main", false, "req_01abc", "3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f")
	for _, absent := range []string{"cch=", "cc_is_subagent", "cc_prev_req", "cc_prompt_id"} {
		if strings.Contains(unsigned, absent) {
			t.Fatalf("unsigned billing header %q must not contain %q", unsigned, absent)
		}
	}
	if !strings.Contains(unsigned, "cc_workload=main;") {
		t.Fatalf("unsigned billing header %q missing cc_workload", unsigned)
	}
}

// TestClaudeBillingTags_StripInjectExtract exercises the billing tag helpers
// used for post-payload probe reclassification (upstream 4a5ab534f827).
func TestClaudeBillingTags_StripInjectExtract(t *testing.T) {
	body := []byte(`{"system":[{"type":"text","text":"x-anthropic-billing-header: cc_version=2.1.280.abc; cc_entrypoint=cli; cch=00000; cc_prev_req=req_01x; cc_prompt_id=3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f;"}]}`)

	prevReq, promptID := helps.ExtractClaudeBillingTags(body)
	if prevReq != "req_01x" || promptID != "3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f" {
		t.Fatalf("ExtractClaudeBillingTags = (%q, %q)", prevReq, promptID)
	}
	stripped := helps.StripClaudeBillingTags(body)
	text := gjson.GetBytes(stripped, "system.0.text").String()
	if strings.Contains(text, "cc_prev_req") || strings.Contains(text, "cc_prompt_id") {
		t.Fatalf("StripClaudeBillingTags left continuity tags: %q", text)
	}
	reinjected := helps.InjectClaudeBillingTags(stripped, "req_01y", "3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f")
	prevReq, promptID = helps.ExtractClaudeBillingTags(reinjected)
	if prevReq != "req_01y" || promptID != "3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f" {
		t.Fatalf("InjectClaudeBillingTags round trip = (%q, %q)", prevReq, promptID)
	}
}

// TestInjectClaudeDiagnostics_FieldOrder verifies diagnostics field placement
// after context_management and continuity carry-over (upstream 086ad91bd970).
func TestInjectClaudeDiagnostics_FieldOrder(t *testing.T) {
	helps.ResetClaudeDiagnosticsForTest()
	defer helps.ResetClaudeDiagnosticsForTest()

	body := []byte(`{"context_management":{"edits":[{"type":"clear_thinking_20251015","keep":"all"}]},"max_tokens":1,"messages":[]}`)
	auth := &cliproxyauth.Auth{ID: "credential-diagnostics-order-test"}
	session := "session-diagnostics-order-test"

	first, state := injectClaudeDiagnostics(body, auth, session)
	wantOrder := `"context_management":{"edits":[{"type":"clear_thinking_20251015","keep":"all"}]},"diagnostics":{"previous_message_id":null},"max_tokens"`
	if !strings.Contains(string(first), wantOrder) {
		t.Fatalf("diagnostics field order differs from native: %s", first)
	}

	commitClaudeDiagnostics(state, "msg_01ABCDEF0123456789ABCDEFG")
	second, _ := injectClaudeDiagnostics(body, auth, session)
	if got := gjson.GetBytes(second, "diagnostics.previous_message_id").String(); got != "msg_01ABCDEF0123456789ABCDEFG" {
		t.Fatalf("second previous_message_id = %q, want committed upstream ID", got)
	}
}

// TestClaudeMessageIDFromSSE_CompletedOnly verifies continuity commits only
// complete upstream streams (upstream 086ad91bd970).
func TestClaudeMessageIDFromSSE_CompletedOnly(t *testing.T) {
	complete := []byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_complete\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	if got := claudeMessageIDFromSSE(complete); got != "msg_complete" {
		t.Fatalf("completed SSE message ID = %q, want msg_complete", got)
	}
	incomplete := []byte(strings.Replace(string(complete), "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n", "", 1))
	if got := claudeMessageIDFromSSE(incomplete); got != "" {
		t.Fatalf("incomplete SSE message ID = %q, want empty", got)
	}
}

// TestClaudeProbeHelperClassification verifies probe and title-helper request
// detection (upstream 4a5ab534f827).
func TestClaudeProbeHelperClassification(t *testing.T) {
	probe := []byte(`{"model":"claude-sonnet-5","max_tokens":1,"messages":[{"role":"user","content":"quota"}]}`)
	if !helps.IsClaudeProbeOrHelperRequest(probe) {
		t.Fatal("expected probe request to classify as probe/helper")
	}
	titleHelper := []byte(`{"model":"claude-sonnet-5","output_config":{"format":{"schema":{"properties":{"title":{"type":"string"}}}}},"system":"Return a short title for naming a coding session","messages":[{"role":"user","content":"x"}]}`)
	if !helps.IsClaudeProbeOrHelperRequest(titleHelper) {
		t.Fatal("expected title helper request to classify as probe/helper")
	}
	normal := []byte(`{"model":"claude-sonnet-5","max_tokens":1024,"messages":[{"role":"user","content":"refactor this function"}]}`)
	if helps.IsClaudeProbeOrHelperRequest(normal) {
		t.Fatal("normal request must not classify as probe/helper")
	}
	multiBlock := []byte(`{"model":"claude-sonnet-5","max_tokens":1,"messages":[{"role":"user","content":[{"type":"text","text":"quota"},{"type":"text","text":"extra"}]}]}`)
	if helps.IsClaudeProbeOrHelperRequest(multiBlock) {
		t.Fatal("multi-block content must not classify as probe")
	}
}

// TestClaudeSubagentDetection verifies subagent classification from headers and
// payload metadata (upstream 4a5ab534f827).
func TestClaudeSubagentDetection(t *testing.T) {
	headers := http.Header{"X-Claude-Code-Agent-Id": []string{"agent-7"}}
	if !helps.IsClaudeSubagentRequest(headers, nil) {
		t.Fatal("X-Claude-Code-Agent-Id header must classify as subagent")
	}
	parented := []byte(`{"metadata":{"user_id":"{\"parent_session_id\":\"sess-1\"}"}}`)
	if !helps.IsClaudeSubagentRequest(nil, parented) {
		t.Fatal("parent_session_id metadata must classify as subagent")
	}
	normal := []byte(`{"metadata":{"user_id":"user_abc_session_3f3f3f3f-3f3f-43f3-83f3-3f3f3f3f3f3f"}}`)
	if helps.IsClaudeSubagentRequest(nil, normal) {
		t.Fatal("normal request must not classify as subagent")
	}
}

// TestClaudeCacheControlTTL_UpgradeStrip verifies the 1h cache TTL pairing
// helpers (upstream d7052c96af78).
func TestClaudeCacheControlTTL_UpgradeStrip(t *testing.T) {
	payload := []byte(`{"tools":[{"name":"t","cache_control":{"type":"ephemeral"}}],"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral","ttl":"5m"}}],"messages":[{"role":"user","content":[{"type":"text","text":"m","cache_control":{"type":"ephemeral"}}]}]}`)

	upgraded := upgradeClaudeCacheControlTTL(payload, claudeCacheControlTTL1h)
	if got := gjson.GetBytes(upgraded, "tools.0.cache_control.ttl").String(); got != "1h" {
		t.Fatalf("tool ttl = %q, want 1h", got)
	}
	if got := gjson.GetBytes(upgraded, "messages.0.content.0.cache_control.ttl").String(); got != "1h" {
		t.Fatalf("message ttl = %q, want 1h", got)
	}
	// Caller-explicit ttl survives the upgrade.
	if got := gjson.GetBytes(upgraded, "system.0.cache_control.ttl").String(); got != "5m" {
		t.Fatalf("explicit ttl = %q, want 5m preserved", got)
	}

	stripped := stripClaudeCacheControlTTL(upgraded)
	if gjson.GetBytes(stripped, "tools.0.cache_control.ttl").Exists() {
		t.Fatal("strip must remove tool ttl")
	}
	if gjson.GetBytes(stripped, "system.0.cache_control.ttl").Exists() {
		t.Fatal("strip must remove caller ttl on probes")
	}
	if got := gjson.GetBytes(stripped, "system.0.cache_control.type").String(); got != "ephemeral" {
		t.Fatalf("strip must keep cache_control object, got %q", got)
	}
}

// TestIsClaudeFable51Model_Boundaries verifies boundary-aware Fable/Mythos 5.1
// detection (upstream de4aa600280e).
func TestIsClaudeFable51Model_Boundaries(t *testing.T) {
	for _, model := range []string{"claude-fable-5-1", "claude-fable-5-1-20260101", "FABLE-5.1", "claude-mythos-5-1", "mythos-5.1-latest"} {
		if !isClaudeFable51Model(model) {
			t.Fatalf("isClaudeFable51Model(%q) = false, want true", model)
		}
	}
	for _, model := range []string{"claude-fable-5-10", "claude-fable-5-12-20260101", "claude-sonnet-5", "fable-4-1"} {
		if isClaudeFable51Model(model) {
			t.Fatalf("isClaudeFable51Model(%q) = true, want false", model)
		}
	}
}

// TestCheckSystemInstructions_FableReportingBlock verifies the reporting
// outcomes block placement for Fable models (upstream de4aa600280e).
func TestCheckSystemInstructions_FableReportingBlock(t *testing.T) {
	fable := []byte(`{"model":"claude-fable-5-1","messages":[{"role":"user","content":"hi"}]}`)
	out := checkSystemInstructionsWithMode(fable, false)
	system := gjson.GetBytes(out, "system")
	if !system.IsArray() || len(system.Array()) < 4 {
		t.Fatalf("fable system blocks = %s, want billing+agent+reporting+static", system.Raw)
	}
	if got := system.Array()[2].Get("text").String(); !strings.HasPrefix(got, "# Reporting outcomes") {
		t.Fatalf("system[2] = %q, want reporting outcomes block", got)
	}

	sonnet := []byte(`{"model":"claude-sonnet-5","messages":[{"role":"user","content":"hi"}]}`)
	out = checkSystemInstructionsWithMode(sonnet, false)
	system = gjson.GetBytes(out, "system")
	for _, blk := range system.Array() {
		if strings.Contains(blk.Get("text").String(), "# Reporting outcomes") {
			t.Fatalf("non-fable request must not carry reporting block: %s", system.Raw)
		}
	}
}

// fableReportingSystemJSON returns a system array carrying the reporting
// outcomes block with correct JSON escaping (the constant contains newlines).
func fableReportingSystemJSON(t *testing.T, extraBilling bool) string {
	t.Helper()
	esc, err := json.Marshal(claudeCodeFableReportingOutcomes)
	if err != nil {
		t.Fatalf("marshal reporting constant: %v", err)
	}
	var blocks []string
	if extraBilling {
		blocks = append(blocks, `{"type":"text","text":"billing"}`)
	}
	blocks = append(blocks, `{"type":"text","text":`+string(esc)+`}`)
	return `[` + strings.Join(blocks, ",") + `]`
}

// TestReconcileFableModelAfterPayload verifies post-payload reconciliation of
// Fable additions when payload rules rewrite the model or related fields
// (upstream de4aa600280e).
func TestReconcileFableModelAfterPayload(t *testing.T) {
	// Fable rewritten to non-Fable: CPA-injected additions are removed.
	body := []byte(`{"model":"claude-sonnet-5","fallbacks":[{"model":"claude-opus-5"}],"thinking":{"type":"adaptive","display":"updates"},"system":` + fableReportingSystemJSON(t, true) + `,"messages":[{"role":"user","content":"hi"}]}`)
	if !hasFableReportingBlock(body) {
		t.Fatal("precondition: input must carry a detectable reporting block")
	}
	state := claudeCodeFableState{injectedFallbacks: true, injectedDisplay: true, injectedReporting: true}
	out := reconcileClaudeCodeFableModelAfterPayload(body, state, false, false, true, false)
	if gjson.GetBytes(out, "fallbacks").Exists() {
		t.Fatalf("fable→non-fable must drop injected fallbacks: %s", out)
	}
	if gjson.GetBytes(out, "thinking.display").Exists() {
		t.Fatalf("fable→non-fable must drop injected thinking.display: %s", out)
	}
	if hasFableReportingBlock(out) {
		t.Fatalf("fable→non-fable must drop reporting block: %s", out)
	}

	// Non-Fable rewritten to Fable: additions are attached.
	body = []byte(`{"model":"claude-fable-5-1","thinking":{"type":"adaptive"},"system":[{"type":"text","text":"billing"}],"messages":[{"role":"user","content":"hi"}]}`)
	out = reconcileClaudeCodeFableModelAfterPayload(body, claudeCodeFableState{}, false, false, true, false)
	if !gjson.GetBytes(out, "fallbacks").Exists() {
		t.Fatalf("non-fable→fable must attach fallbacks: %s", out)
	}
	if got := gjson.GetBytes(out, "thinking.display").String(); got != "updates" {
		t.Fatalf("non-fable→fable must attach thinking.display=updates, got %q", got)
	}
	if !hasFableReportingBlock(out) {
		t.Fatalf("non-fable→fable must attach reporting block: %s", out)
	}

	// Payload rules owning the field are respected.
	body = []byte(`{"model":"claude-sonnet-5","fallbacks":[{"model":"claude-opus-5"}],"thinking":{"type":"adaptive","display":"updates"},"messages":[{"role":"user","content":"hi"}]}`)
	out = reconcileClaudeCodeFableModelAfterPayload(body, state, true, true, true, false)
	if !gjson.GetBytes(out, "fallbacks").Exists() || !gjson.GetBytes(out, "thinking.display").Exists() {
		t.Fatalf("payload-touched fields must survive reconcile: %s", out)
	}

	// Probes never carry Fable additions.
	body = []byte(`{"model":"claude-fable-5-1","max_tokens":1,"fallbacks":[{"model":"claude-opus-5"}],"thinking":{"type":"adaptive","display":"updates"},"system":` + fableReportingSystemJSON(t, false) + `,"messages":[{"role":"user","content":"quota"}]}`)
	if !hasFableReportingBlock(body) {
		t.Fatal("precondition: probe input must carry a detectable reporting block")
	}
	out = reconcileClaudeCodeFableModelAfterPayload(body, state, false, false, true, true)
	if gjson.GetBytes(out, "fallbacks").Exists() || gjson.GetBytes(out, "thinking.display").Exists() || hasFableReportingBlock(out) {
		t.Fatalf("probe must not carry Fable additions: %s", out)
	}

	// Adaptive → disabled thinking drops the injected display field.
	body = []byte(`{"model":"claude-fable-5-1","thinking":{"type":"disabled","display":"updates"},"system":` + fableReportingSystemJSON(t, false) + `,"messages":[{"role":"user","content":"hi"}]}`)
	out = reconcileClaudeCodeFableModelAfterPayload(body, state, false, false, true, false)
	if gjson.GetBytes(out, "thinking.display").Exists() {
		t.Fatalf("adaptive→disabled must drop injected thinking.display: %s", out)
	}
}

// TestApplyPayloadConfigWithTrackedPaths_AncestorMatch verifies tracked-path
// reporting and ancestor/descendant matching (upstream de4aa600280e).
func TestApplyPayloadConfigWithTrackedPaths_AncestorMatch(t *testing.T) {
	cfg := &config.Config{
		Payload: config.PayloadConfig{
			Override: []config.PayloadRule{
				{
					Models: []config.PayloadModelRule{{Name: "claude-*"}},
					Params: map[string]any{"thinking": map[string]any{"type": "disabled"}},
				},
			},
		},
	}
	payload := []byte(`{"model":"claude-fable-5-1","thinking":{"type":"adaptive","display":"updates"}}`)
	out, touched := helps.ApplyPayloadConfigWithTrackedPaths(cfg, "claude-fable-5-1", "claude", "claude", "", payload, nil, "", "", nil, "fallbacks", "thinking.display", "diagnostics")
	if !touched["thinking.display"] {
		t.Fatalf("rule on ancestor path 'thinking' must mark 'thinking.display' touched, got %v", touched)
	}
	if touched["fallbacks"] || touched["diagnostics"] {
		t.Fatalf("untouched paths reported touched: %v", touched)
	}
	if got := gjson.GetBytes(out, "thinking.type").String(); got != "disabled" {
		t.Fatalf("payload rule not applied, thinking.type=%q", got)
	}
}

// TestApplyCloaking_BillingContinuityTags verifies a cloaked OAuth request
// emits the billing chain with continuity tags and that a subagent request
// carries cc_is_subagent (upstream 086ad91bd970, 4a5ab534f827).
func TestApplyCloaking_BillingContinuityTags(t *testing.T) {
	helps.ResetClaudeDiagnosticsForTest()
	defer helps.ResetClaudeDiagnosticsForTest()

	auth := &cliproxyauth.Auth{ID: "cloak-cont-1", Attributes: map[string]string{"api_key": "sk-ant-oat-test", "cloak_mode": "always"}}
	payload := []byte(`{"model":"claude-sonnet-5","max_tokens":64,"messages":[{"role":"user","content":"refactor the parser"}]}`)

	continuityCtx := &helps.ClaudeContinuityContext{}
	ctx := helps.WithClaudeContinuityContext(context.Background(), continuityCtx)
	out, cloaked := applyCloaking(ctx, nil, auth, payload, "claude-sonnet-5", "sk-ant-oat-test", "https://api.anthropic.com")
	if !cloaked {
		t.Fatal("expected cloaking to run for cloak_mode=always")
	}
	billing := gjson.GetBytes(out, "system.0.text").String()
	if !strings.HasPrefix(billing, "x-anthropic-billing-header: cc_version=2.1.280.") {
		t.Fatalf("billing header = %q, want 2.1.280 chain", billing)
	}
	if !strings.Contains(billing, "cch=00000;") {
		t.Fatalf("oauth cloaked billing must carry cch placeholder: %q", billing)
	}
	if !strings.Contains(billing, "cc_prompt_id=") {
		t.Fatalf("cloaked billing must carry cc_prompt_id: %q", billing)
	}
	if !continuityCtx.Initialized {
		t.Fatal("continuity context was not initialized by applyCloaking")
	}

	// Subagent request via incoming agent header carries cc_is_subagent.
	continuityCtx2 := &helps.ClaudeContinuityContext{}
	ctx = helps.WithClaudeContinuityContext(context.Background(), continuityCtx2)
	ctx = helps.WithIncomingHeaders(ctx, http.Header{"X-Claude-Code-Agent-Id": []string{"agent-9"}})
	out, _ = applyCloaking(ctx, nil, auth, payload, "claude-sonnet-5", "sk-ant-oat-test", "https://api.anthropic.com")
	billing = gjson.GetBytes(out, "system.0.text").String()
	if !strings.Contains(billing, "cc_is_subagent=true;") {
		t.Fatalf("subagent billing must carry cc_is_subagent=true: %q", billing)
	}

	// Probe request: no continuity is initialized and billing carries no
	// continuity tags (upstream 4a5ab534f827).
	probePayload := []byte(`{"model":"claude-sonnet-5","max_tokens":1,"messages":[{"role":"user","content":"quota"}]}`)
	continuityCtx3 := &helps.ClaudeContinuityContext{}
	ctx = helps.WithClaudeContinuityContext(context.Background(), continuityCtx3)
	out, _ = applyCloaking(ctx, nil, auth, probePayload, "claude-sonnet-5", "sk-ant-oat-test", "https://api.anthropic.com")
	billing = gjson.GetBytes(out, "system.0.text").String()
	if strings.Contains(billing, "cc_prev_req") || strings.Contains(billing, "cc_prompt_id") {
		t.Fatalf("probe billing must not carry continuity tags: %q", billing)
	}
	if continuityCtx3.Initialized {
		t.Fatal("probe request must not initialize continuity")
	}
}

// TestClaudeCodeCLIBetas_21280GatedBetas verifies the Claude Code 2.1.280
// feature-gated betas and their wire positions (upstream bd584a752329 and
// 779bf317e030).
func TestClaudeCodeCLIBetas_21280GatedBetas(t *testing.T) {
	constants := "claude-code-20250219,interleaved-thinking-2025-05-14," +
		"redact-thinking-2026-02-12,thinking-token-count-2026-05-13," +
		"context-management-2025-06-27,prompt-caching-scope-2026-01-05"

	cases := []struct {
		name      string
		body      string
		requested map[string]bool
		want      string
	}{
		{
			name: "plain non-legacy model keeps mid-conversation pair",
			body: `{"model":"claude-fable-5"}`,
			want: constants + ",mid-conversation-system-2026-04-07,mid-conversation-tool-changes-2026-07-01",
		},
		{
			name: "opus-5-5 carries per-turn-control between mid-conversation betas",
			body: `{"model":"claude-opus-5-5"}`,
			want: constants + ",mid-conversation-system-2026-04-07,per-turn-control-2026-07-01,mid-conversation-tool-changes-2026-07-01",
		},
		{
			name: "fable-5-1 carries per-turn-control but not timing unless the body asks",
			body: `{"model":"claude-fable-5-1"}`,
			want: constants + ",mid-conversation-system-2026-04-07,per-turn-control-2026-07-01,mid-conversation-tool-changes-2026-07-01",
		},
		{
			name: "opus-5-5 timing and the other 2.1.280 gated betas keep wire order",
			body: `{"model":"claude-opus-5-5","safeguards":[{}],"thinking":{"type":"adaptive","block_binding":{"prefix_mismatch_behavior":"omit"}},"messages":[{"role":"system","clear_at":"next_user_message","content":[{"type":"tool_addition","tool":{"definition":{"name":"bash"}}}]},{"role":"user","content":"x","output_config":{"timing":{"now":"2026-09-23T00:00:00Z"}}}],"cache_control":{"type":"ephemeral","evict_on_complete":true}}`,
			requested: map[string]bool{
				claudeThinkingResumptionBeta: true,
			},
			want: constants + ",mid-conversation-system-2026-04-07,per-turn-control-2026-07-01,timing-2026-09-09," +
				"mid-conversation-tool-changes-2026-07-01,inline-tools-2026-09-15," +
				"mid-conversation-system-clear-at-2026-08-21,dangerous-tool-use-2026-09-03," +
				"thinking-binding-controls-2026-08-01,thinking-resumption-2026-07-17,prompt-caching-evict-2026-05-12",
		},
		{
			name: "legacy model still honors requested per-turn betas ahead of effort position",
			body: `{"model":"claude-opus-4-7"}`,
			requested: map[string]bool{
				claudePerTurnControlBeta: true,
				claudePerTurnTimingBeta:  true,
			},
			want: constants + ",per-turn-control-2026-07-01,timing-2026-09-09",
		},
		{
			name: "legacy model does not emit mid-conversation betas",
			body: `{"model":"claude-opus-4-7"}`,
			want: constants,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := claudeCodeCLIBetas([]byte(tc.body), tc.requested, false)
			for _, want := range strings.Split(tc.want, ",") {
				if !strings.Contains(got, want) {
					t.Fatalf("betas %q missing %q", got, want)
				}
			}
			// Exact-order assertion: the emitted list must match the measured
			// relative order in tc.want (effort/fallback betas may trail).
			wantParts := strings.Split(tc.want, ",")
			gotParts := strings.Split(got, ",")
			idx := 0
			for _, wp := range wantParts {
				found := false
				for ; idx < len(gotParts); idx++ {
					if gotParts[idx] == wp {
						found = true
						idx++
						break
					}
				}
				if !found {
					t.Fatalf("betas %q missing %q in wire order", got, wp)
				}
			}
			// New betas must not appear unless the case asks for them.
			unwanted := []string{
				claudePerTurnControlBeta, claudePerTurnTimingBeta, claudeInlineToolsBeta,
				claudeMidConvSystemClearAtBeta, claudeDangerousToolUseBeta,
				claudeThinkingBindingBeta, claudeThinkingResumptionBeta, claudePromptCachingEvictBeta,
			}
			for _, u := range unwanted {
				if !strings.Contains(tc.want, u) && strings.Contains(got, u) {
					t.Fatalf("betas %q unexpectedly contains %q", got, u)
				}
			}
		})
	}
}

// TestApplyClaudeHeaders_ForwardsUnmanagedCallerBetas verifies that caller
// betas the proxy does not manage survive on direct Anthropic while managed
// caller betas are emitted at their assembled positions (upstream #5738,
// bd584a752329).
func TestApplyClaudeHeaders_ForwardsUnmanagedCallerBetas(t *testing.T) {
	resetClaudeDeviceProfileCache()
	req := newClaudeHeaderTestRequest(t, http.Header{})
	auth := &cliproxyauth.Auth{ID: "auth-beta", Attributes: map[string]string{"api_key": "key-1"}}
	incoming := http.Header{}
	incoming.Set("Anthropic-Beta", "message-threads-2026-08-12,per-turn-control-2026-07-01")
	body := []byte(`{"model":"claude-fable-5-1"}`)
	applyClaudeHeaders(req, auth, "key-1", false, nil, body, nil, incoming)

	betas := req.Header.Get("Anthropic-Beta")
	if !strings.Contains(betas, "message-threads-2026-08-12") {
		t.Fatalf("Anthropic-Beta = %q, want unmanaged caller beta forwarded", betas)
	}
	if !strings.Contains(betas, "mid-conversation-system-2026-04-07,per-turn-control-2026-07-01,mid-conversation-tool-changes-2026-07-01") {
		t.Fatalf("Anthropic-Beta = %q, want managed per-turn-control at its assembled position", betas)
	}
	if strings.Count(betas, "per-turn-control-2026-07-01") != 1 {
		t.Fatalf("Anthropic-Beta = %q, managed caller beta must not duplicate", betas)
	}
}
