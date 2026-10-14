package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
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

func TestClaudeP4_FableFallbacksAndDisplay(t *testing.T) {
	cfg := &config.Config{}
	payload := []byte(`{"model":"claude-fable-5-1","thinking":{"type":"adaptive"},"messages":[{"role":"user","content":"hi"}]}`)
	out, _ := applyCloaking(context.Background(), cfg, nil, payload, "claude-fable-5-1", "sk-ant-oat-x", "https://api.anthropic.com")
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
