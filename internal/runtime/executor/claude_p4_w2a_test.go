package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/runtime/executor/helps"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

func newClaudeP4HeaderTestRequest(t *testing.T, incoming http.Header) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages?beta=true", nil)
	if incoming != nil {
		for k, vs := range incoming {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
	return req
}

func TestClaudeP4_BetaEffortPruning(t *testing.T) {
	oauthAuth := &cliproxyauth.Auth{ID: "a", Attributes: map[string]string{"api_key": "k"}}
	cases := []struct {
		name    string
		body    string
		want    bool
		present string
	}{
		{"haiku prunes effort", `{"model":"claude-haiku-4-5-20251001","tools":[{"name":"t","input_schema":{"type":"object"}}]}`, false, claudeEffortBeta},
		{"disabled thinking prunes effort", `{"model":"claude-opus-5","thinking":{"type":"disabled"},"tools":[{"name":"t","input_schema":{"type":"object"}}]}`, false, claudeEffortBeta},
		{"probe prunes effort", `{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`, false, claudeEffortBeta},
		{"opus keeps effort", `{"model":"claude-opus-5","tools":[{"name":"t","input_schema":{"type":"object"}}]}`, true, claudeEffortBeta},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := newClaudeP4HeaderTestRequest(t, nil)
			applyClaudeHeaders(req, oauthAuth, "sk-ant-oat-x", false, nil, []byte(tc.body), nil)
			got := req.Header.Get("Anthropic-Beta")
			if tc.want && !strings.Contains(got, tc.present) {
				t.Fatalf("beta %q missing in %q", tc.present, got)
			}
			if !tc.want && strings.Contains(got, tc.present) {
				t.Fatalf("beta %q must be pruned in %q", tc.present, got)
			}
		})
	}
}

func TestClaudeP4_BetaThinkingDisplayAndAdvisor(t *testing.T) {
	auth := &cliproxyauth.Auth{ID: "a", Attributes: map[string]string{"api_key": "k"}}
	displayBody := []byte(`{"model":"claude-opus-5","thinking":{"type":"adaptive","display":"updates"}}`)
	req := newClaudeP4HeaderTestRequest(t, nil)
	applyClaudeHeaders(req, auth, "k", false, nil, displayBody, nil)
	if got := req.Header.Get("Anthropic-Beta"); !strings.Contains(got, claudeThinkingDisplayUpdatesBeta) {
		t.Fatalf("thinking-display-updates missing in %q", got)
	}

	advisorBody := []byte(`{"model":"claude-opus-5","tools":[{"type":"advisor_presence","name":"advisor"}]}`)
	req2 := newClaudeP4HeaderTestRequest(t, nil)
	applyClaudeHeaders(req2, auth, "k", false, nil, advisorBody, nil)
	if got := req2.Header.Get("Anthropic-Beta"); !strings.Contains(got, claudeAdvisorToolBeta) {
		t.Fatalf("advisor-tool beta missing in %q", got)
	}

	plainToolBody := []byte(`{"model":"claude-opus-5","tools":[{"name":"plain","input_schema":{"type":"object"}}]}`)
	req3 := newClaudeP4HeaderTestRequest(t, nil)
	applyClaudeHeaders(req3, auth, "k", false, nil, plainToolBody, nil)
	if got := req3.Header.Get("Anthropic-Beta"); strings.Contains(got, claudeAdvisorToolBeta) {
		t.Fatalf("advisor-tool beta must not appear for plain tools in %q", got)
	}
}

func TestClaudeP4_BetaFallbackGating(t *testing.T) {
	auth := &cliproxyauth.Auth{ID: "a", Attributes: map[string]string{"api_key": "k"}}
	withFallbacks := []byte(`{"model":"claude-opus-5","fallbacks":[{"model":"claude-opus-5"}]}`)
	req := newClaudeP4HeaderTestRequest(t, nil)
	applyClaudeHeaders(req, auth, "k", false, nil, withFallbacks, nil)
	if got := req.Header.Get("Anthropic-Beta"); !strings.Contains(got, claudeServerSideFallbackBeta) {
		t.Fatalf("server-side-fallback missing for fallbacks body in %q", got)
	}
	withoutFallbacks := []byte(`{"model":"claude-opus-5"}`)
	req2 := newClaudeP4HeaderTestRequest(t, nil)
	applyClaudeHeaders(req2, auth, "k", false, nil, withoutFallbacks, nil)
	if got := req2.Header.Get("Anthropic-Beta"); strings.Contains(got, claudeServerSideFallbackBeta) {
		t.Fatalf("server-side-fallback must not appear without fallbacks/request in %q", got)
	}
}

func TestClaudeP4_BetaExtendedCacheTTLSuppression(t *testing.T) {
	oauthAuth := &cliproxyauth.Auth{ID: "a", Attributes: map[string]string{"api_key": "k"}}
	mainBody := []byte(`{"model":"claude-opus-5"}`)
	req := newClaudeP4HeaderTestRequest(t, nil)
	applyClaudeHeaders(req, oauthAuth, "sk-ant-oat-x", false, nil, mainBody, nil)
	if got := req.Header.Get("Anthropic-Beta"); !strings.HasSuffix(got, claudeExtendedCacheTTLBeta) {
		t.Fatalf("main OAuth must end with extended-cache-ttl, got %q", got)
	}

	subBody := []byte(`{"model":"claude-opus-5"}`)
	subHeaders := http.Header{"X-Claude-Code-Agent-Id": []string{"agent-1"}}
	req2 := newClaudeP4HeaderTestRequest(t, nil)
	applyClaudeHeaders(req2, oauthAuth, "sk-ant-oat-x", false, nil, subBody, nil, subHeaders)
	if got := req2.Header.Get("Anthropic-Beta"); strings.Contains(got, claudeExtendedCacheTTLBeta) {
		t.Fatalf("subagent must suppress extended-cache-ttl, got %q", got)
	}

	probeBody := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	req3 := newClaudeP4HeaderTestRequest(t, nil)
	applyClaudeHeaders(req3, oauthAuth, "sk-ant-oat-x", false, nil, probeBody, nil)
	if got := req3.Header.Get("Anthropic-Beta"); strings.Contains(got, claudeExtendedCacheTTLBeta) {
		t.Fatalf("probe must suppress extended-cache-ttl, got %q", got)
	}
}

func TestClaudeP4_HelperRequestIDFollowsUpstreamBase(t *testing.T) {
	const callerID = "66666666-7777-4888-8999-aaaaaaaaaaaa"
	auth := &cliproxyauth.Auth{Attributes: map[string]string{"api_key": "k"}}
	body := []byte(`{"model":"claude-haiku-4-5-20251001"}`)

	firstParty := httptest.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages?beta=true", nil)
	applyClaudeHeaders(firstParty, auth, "k", false, nil, body, nil)
	if got := firstParty.Header.Get("X-Client-Request-Id"); got == "" {
		t.Fatal("first-party base must generate x-client-request-id")
	} else if _, err := uuid.Parse(got); err != nil {
		t.Fatalf("first-party request id %q is not a UUID", got)
	}

	customBare := httptest.NewRequest(http.MethodPost, "https://gateway.example.com/v1/messages", nil)
	applyClaudeHeaders(customBare, auth, "k", false, nil, body, nil)
	if got := customBare.Header.Get("X-Client-Request-Id"); got != "" {
		t.Fatalf("custom base without caller id must stay absent, got %q", got)
	}

	customCaller := httptest.NewRequest(http.MethodPost, "https://gateway.example.com/v1/messages", nil)
	customCaller.Header.Set("X-Client-Request-Id", callerID)
	applyClaudeHeaders(customCaller, auth, "k", false, nil, body, nil, customCaller.Header)
	if got := customCaller.Header.Get("X-Client-Request-Id"); got != callerID {
		t.Fatalf("custom base must preserve caller id, got %q", got)
	}
}

func TestClaudeP4_StripCacheControlTTL(t *testing.T) {
	body := []byte(`{"tools":[{"name":"a","cache_control":{"type":"ephemeral","ttl":"1h"}}],"system":[{"type":"text","text":"s","cache_control":{"type":"ephemeral","ttl":"1h"}}],"messages":[{"role":"user","content":[{"type":"text","text":"hi","cache_control":{"type":"ephemeral","ttl":"1h"}}]}]}`)
	stripped := stripClaudeCacheControlTTL(body)
	for _, path := range []string{"tools.0.cache_control.ttl", "system.0.cache_control.ttl", "messages.0.content.0.cache_control.ttl"} {
		if gjson.GetBytes(stripped, path).Exists() {
			t.Fatalf("ttl at %s not stripped: %s", path, stripped)
		}
	}
	if gjson.GetBytes(stripped, "tools.0.cache_control.type").String() != "ephemeral" {
		t.Fatalf("cache_control type lost: %s", stripped)
	}
}

func TestClaudeP4_BillingChainSubagentAndPromptID(t *testing.T) {
	main := generateBillingHeaderWithChain(nil, false, "2.1.258", "hello world this is a test message", "cli", "", false, "prompt-1")
	if !strings.Contains(main, "cc_version=2.1.258.") {
		t.Fatalf("billing missing versioned fingerprint: %q", main)
	}
	if !strings.Contains(main, "cc_prompt_id=prompt-1;") {
		t.Fatalf("billing missing prompt id: %q", main)
	}
	if strings.Contains(main, "cc_is_subagent=true;") {
		t.Fatalf("main billing must not mark subagent: %q", main)
	}
	sub := generateBillingHeaderWithChain(nil, false, "2.1.258", "hello world this is a test message", "cli", "", true, "prompt-2")
	if !strings.Contains(sub, "cc_is_subagent=true;") {
		t.Fatalf("subagent billing missing marker: %q", sub)
	}

	payload := []byte(`{"model":"claude-opus-5","system":[{"type":"text","text":"hi"}],"messages":[{"role":"user","content":"hi"}]}`)
	out := checkSystemInstructionsWithChain(payload, false, false, true, "2.1.258", "cli", "", true, "prompt-3")
	if got := gjson.GetBytes(out, "system.0.text").String(); !strings.Contains(got, "cc_is_subagent=true;") || !strings.Contains(got, "cc_prompt_id=prompt-3;") {
		t.Fatalf("system billing chain incomplete: %q", got)
	}
}

func TestClaudeP4_FableFallbacksAndDisplay(t *testing.T) {
	cfg := &config.Config{}
	payload := []byte(`{"model":"claude-fable-5-1","thinking":{"type":"adaptive"},"messages":[{"role":"user","content":"hi"}]}`)
	out := applyCloaking(context.Background(), cfg, nil, payload, "claude-fable-5-1", "sk-ant-oat-x", "https://api.anthropic.com")
	if !gjson.GetBytes(out, "fallbacks").Exists() {
		t.Fatalf("fable fallbacks missing: %s", out)
	}
	if got := gjson.GetBytes(out, "thinking.display").String(); got != "updates" {
		t.Fatalf("fable thinking.display = %q, want updates: %s", got, out)
	}
	if !isClaudeFable51Model("claude-fable-5-1") || !isClaudeFable51Model("mythos-5.1-pro") {
		t.Fatal("fable family detection failed")
	}
	if isClaudeFable51Model("claude-opus-5") {
		t.Fatal("opus-5 misdetected as fable")
	}
}

func TestClaudeP4_MCPFullNameRecovery(t *testing.T) {
	wrapped := "mcp__virtualserver__mcp__docs-search__search-pages"
	restored, ok := restorePassthroughWrappedMCPName(wrapped)
	if !ok || restored != "mcp__docs-search__search-pages" {
		t.Fatalf("wrapped recovery = %q,%v, want full mcp__ name", restored, ok)
	}
	if _, ok := restorePassthroughWrappedMCPName("mcp__server__tool"); ok {
		t.Fatal("plain mcp name must not trigger wrapped recovery")
	}
	if _, ok := restorePassthroughWrappedMCPName("Bash"); ok {
		t.Fatal("non-mcp name must not trigger wrapped recovery")
	}

	resp := []byte(`{"content":[{"type":"tool_use","id":"toolu_1","name":"` + wrapped + `","input":{}}]}`)
	got := reverseRemapOAuthToolNames(resp, map[string]string{})
	if name := gjson.GetBytes(got, "content.0.name").String(); name != "mcp__docs-search__search-pages" {
		t.Fatalf("response wrapped restore = %q", name)
	}
	line := []byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"` + wrapped + `","input":{}}}`)
	restoredLine := reverseRemapOAuthToolNamesFromStreamLine(line, map[string]string{})
	if name := gjson.GetBytes(helps.JSONPayload(restoredLine), "content_block.name").String(); name != "mcp__docs-search__search-pages" {
		t.Fatalf("stream wrapped restore = %q", name)
	}
}

func TestClaudeP4_StripTranslatedDefaultMaxTokens(t *testing.T) {
	translated := []byte(`{"model":"claude-opus-5-5","max_tokens":128000,"messages":[]}`)
	omitted := []byte(`{"model":"claude-opus-5-5","messages":[{"role":"user","content":"hi"}]}`)
	stripped := stripTranslatedDefaultMaxTokens(sdktranslator.FormatOpenAI, omitted, translated)
	if gjson.GetBytes(stripped, "max_tokens").Exists() {
		t.Fatalf("omitted limit must strip translated default: %s", stripped)
	}
	explicit := []byte(`{"model":"claude-opus-5-5","max_tokens":4096,"messages":[{"role":"user","content":"hi"}]}`)
	kept := stripTranslatedDefaultMaxTokens(sdktranslator.FormatOpenAI, explicit, translated)
	if gjson.GetBytes(kept, "max_tokens").Int() != 128000 {
		t.Fatalf("explicit limit must keep translated value: %s", kept)
	}
	claudeSrc := stripTranslatedDefaultMaxTokens(sdktranslator.FormatClaude, omitted, translated)
	if gjson.GetBytes(claudeSrc, "max_tokens").Int() != 128000 {
		t.Fatalf("claude source must pass through: %s", claudeSrc)
	}
}
