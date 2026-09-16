package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type sessionAliasCaptureDispatcher struct {
	mu       sync.Mutex
	sessions []string
}

type homeSessionAliasExecutor struct {
	schedulerTestExecutor
}

func (homeSessionAliasExecutor) Identifier() string { return "home-session-alias" }

func (*sessionAliasCaptureDispatcher) HeartbeatOK() bool { return true }

func (d *sessionAliasCaptureDispatcher) RPopAuth(_ context.Context, _ string, sessionID string, _ http.Header, _ int) ([]byte, error) {
	d.mu.Lock()
	d.sessions = append(d.sessions, sessionID)
	d.mu.Unlock()
	return json.Marshal(homeAuthDispatchResponse{Auth: Auth{
		ID:       "home-session-alias-auth",
		Provider: "home-session-alias",
		Status:   StatusActive,
	}})
}

func (*sessionAliasCaptureDispatcher) AbortAmbiguousDispatch() {}

func (d *sessionAliasCaptureDispatcher) sessionIDs() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.sessions...)
}

func TestHomeDispatchCanonicalizesPromptCacheAndConversationAliases(t *testing.T) {
	tests := []struct {
		name     string
		payloads []string
		want     string
	}{
		{
			name: "conversation then combined then prompt cache",
			payloads: []string{
				`{"conversation":{"id":"conversation-session"}}`,
				`{"conversation":{"id":"conversation-session"},"prompt_cache_key":"shared-cache-bucket"}`,
				`{"prompt_cache_key":"shared-cache-bucket"}`,
			},
			want: "conv:conversation-session",
		},
		{
			name: "prompt cache then combined then conversation",
			payloads: []string{
				`{"prompt_cache_key":"shared-cache-bucket"}`,
				`{"conversation":{"id":"conversation-session"},"prompt_cache_key":"shared-cache-bucket"}`,
				`{"conversation":{"id":"conversation-session"}}`,
			},
			want: "pck:shared-cache-bucket",
		},
		{
			name: "combined request establishes prompt cache primary",
			payloads: []string{
				`{"conversation":{"id":"conversation-session"},"prompt_cache_key":"shared-cache-bucket"}`,
				`{"conversation":{"id":"conversation-session"}}`,
				`{"prompt_cache_key":"shared-cache-bucket"}`,
			},
			want: "pck:shared-cache-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &sessionAliasCaptureDispatcher{}
			manager := newHomeSelectionTestManager(t, dispatcher)
			manager.RegisterExecutor(homeSessionAliasExecutor{})

			for _, payload := range tt.payloads {
				selection, errSelection := manager.pickHomeDispatchSelection(context.Background(), "gpt-test", cliproxyexecutor.Options{
					OriginalRequest: []byte(payload),
				})
				if errSelection != nil {
					t.Fatalf("pickHomeDispatchSelection() error = %v", errSelection)
				}
				selection.End("test_complete")
			}

			got := dispatcher.sessionIDs()
			if len(got) != len(tt.payloads) {
				t.Fatalf("Home session IDs = %#v, want %d entries", got, len(tt.payloads))
			}
			for index, sessionID := range got {
				if sessionID != tt.want {
					t.Fatalf("Home session ID[%d] = %q, want %q; all=%#v", index, sessionID, tt.want, got)
				}
			}
		})
	}
}

func TestHomeSessionAliasCachePrimaryAccessRefreshesWholeAliasGroup(t *testing.T) {
	var cache homeSessionAliasCache
	now := time.Now()
	const primary = "pck:shared-cache-bucket"
	const fallback = "conv:conversation-session"

	if got := cache.canonical(primary, fallback, time.Minute, now); got != primary {
		t.Fatalf("initial canonical = %q, want %q", got, primary)
	}
	cache.mu.Lock()
	fallbackEntry := cache.entries[fallback]
	fallbackEntry.expiresAt = now.Add(-time.Second)
	cache.entries[fallback] = fallbackEntry
	cache.mu.Unlock()

	if got := cache.canonical(primary, "", time.Minute, now.Add(10*time.Second)); got != primary {
		t.Fatalf("primary-only canonical = %q, want %q", got, primary)
	}
	if got := cache.canonical(fallback, "", time.Minute, now.Add(20*time.Second)); got != primary {
		t.Fatalf("fallback canonical after active primary traffic = %q, want %q", got, primary)
	}
}

func TestHomeSessionAliasCacheClearsWhenConfiguredTTLChanges(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{
		Home:    internalconfig.HomeConfig{Enabled: true},
		Routing: internalconfig.RoutingConfig{SessionAffinityTTL: "1h"},
	})
	combined := cliproxyexecutor.Options{OriginalRequest: []byte(
		`{"conversation":{"id":"ttl-conversation"},"prompt_cache_key":"ttl-prompt"}`,
	)}
	conversationOnly := cliproxyexecutor.Options{OriginalRequest: []byte(
		`{"conversation":{"id":"ttl-conversation"}}`,
	)}
	if got := manager.homeDispatchSessionID(combined); got != "pck:ttl-prompt" {
		t.Fatalf("combined canonical = %q, want pck:ttl-prompt", got)
	}
	if got := manager.homeDispatchSessionID(conversationOnly); got != "pck:ttl-prompt" {
		t.Fatalf("conversation canonical before reload = %q, want existing prompt canonical", got)
	}

	manager.SetConfig(&internalconfig.Config{
		Home:    internalconfig.HomeConfig{Enabled: true},
		Routing: internalconfig.RoutingConfig{SessionAffinityTTL: "1m"},
	})
	if got := manager.homeDispatchSessionID(conversationOnly); got != "conv:ttl-conversation" {
		t.Fatalf("conversation canonical after TTL change = %q, want cleared alias cache", got)
	}
}

// TestHomeDispatchSessionIDsExtractsParentSessionID ports upstream
// e899f0e53985/390589159eff end-state: parent lineage resolved from the
// extraction fallback lands in the second return of homeDispatchSessionIDs.
func TestHomeDispatchSessionIDsExtractsParentSessionID(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{
		Home:    internalconfig.HomeConfig{Enabled: true},
		Routing: internalconfig.RoutingConfig{SessionAffinityTTL: "1h"},
	})

	// 1. Root Claude session
	rootOpts := cliproxyexecutor.Options{
		Headers: http.Header{"X-Claude-Code-Session-Id": []string{"claude-root-1"}},
	}
	sessionID, parentID := manager.homeDispatchSessionIDs(rootOpts)
	if sessionID != "claude:claude-root-1" || parentID != "" {
		t.Fatalf("root session = (%q, %q), want (claude:claude-root-1, \"\")", sessionID, parentID)
	}

	// 2. Subagent Claude session
	subOpts := cliproxyexecutor.Options{
		Headers: http.Header{
			"X-Claude-Code-Session-Id": []string{"claude-root-1"},
			"X-Claude-Code-Agent-Id":   []string{"sub-checker"},
		},
	}
	subSessionID, subParentID := manager.homeDispatchSessionIDs(subOpts)
	if subSessionID != "claude:claude-root-1:agent:sub-checker" || subParentID != "claude:claude-root-1" {
		t.Fatalf("subagent session = (%q, %q), want (claude:claude-root-1:agent:sub-checker, claude:claude-root-1)", subSessionID, subParentID)
	}

	// 3. Pi slot session with parent
	piSubOpts := cliproxyexecutor.Options{
		Headers: http.Header{
			"X-Slot-Session-Id":   []string{"pi-slot-worker-1"},
			"X-Parent-Session-ID": []string{"pi-slot-main-0"},
		},
	}
	piSessionID, piParentID := manager.homeDispatchSessionIDs(piSubOpts)
	if piSessionID != "slot:pi-slot-worker-1" || piParentID != "slot:pi-slot-main-0" {
		t.Fatalf("pi subagent session = (%q, %q), want (slot:pi-slot-worker-1, slot:pi-slot-main-0)", piSessionID, piParentID)
	}

	// 4. LCP-derived session in metadata
	lcpOpts := cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.CanonicalSessionIDMetadataKey: "lcp:v1:abc12345",
		},
	}
	lcpSessionID, lcpParentID := manager.homeDispatchSessionIDs(lcpOpts)
	if lcpSessionID != "lcp:v1:abc12345" || lcpParentID != "" {
		t.Fatalf("lcp session = (%q, %q), want (lcp:v1:abc12345, \"\")", lcpSessionID, lcpParentID)
	}
}

// TestHomeDispatchSessionIDsExtractsParentFromHeaderPlusBody ports upstream
// e899f0e53985 including its 3b/3c additions: LCP metadata parent propagation
// and self-referential parent suppression.
func TestHomeDispatchSessionIDsExtractsParentFromHeaderPlusBody(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{
		Home:    internalconfig.HomeConfig{Enabled: true},
		Routing: internalconfig.RoutingConfig{SessionAffinityTTL: "1h"},
	})

	// 1. Header session + Body parent_session_id
	opts := cliproxyexecutor.Options{
		Headers: http.Header{
			"X-Claude-Code-Session-Id": []string{"child-session-001"},
		},
		OriginalRequest: []byte(`{"parent_session_id":"parent-session-999"}`),
	}
	sessionID, parentID := manager.homeDispatchSessionIDs(opts)
	if sessionID != "claude:child-session-001" {
		t.Fatalf("sessionID = %q, want claude:child-session-001", sessionID)
	}
	if parentID != "claude:parent-session-999" {
		t.Fatalf("parentID = %q, want claude:parent-session-999", parentID)
	}

	// 2. Nested Claude metadata.user_id with parent_session_id
	nestedOpts := cliproxyexecutor.Options{
		OriginalRequest: []byte(`{
			"metadata": {
				"user_id": "{\"session_id\":\"child-session-002\",\"parent_session_id\":\"parent-session-888\",\"agent_id\":\"sub-agent-1\"}"
			}
		}`),
	}
	nestedSessionID, nestedParentID := manager.homeDispatchSessionIDs(nestedOpts)
	if nestedSessionID != "claude:child-session-002:agent:sub-agent-1" {
		t.Fatalf("nestedSessionID = %q, want claude:child-session-002:agent:sub-agent-1", nestedSessionID)
	}
	if nestedParentID != "claude:parent-session-888" {
		t.Fatalf("nestedParentID = %q, want claude:parent-session-888", nestedParentID)
	}

	// 3. LCP metadata prioritization over msg-hash fallback
	lcpOpts := cliproxyexecutor.Options{
		OriginalRequest: []byte(`{"messages":[{"role":"user","content":"hello world"}]}`),
		Metadata: map[string]any{
			cliproxyexecutor.CanonicalSessionIDMetadataKey: "lcp:v1:canonical-hash-xyz",
		},
	}
	lcpSessionID, lcpParentID := manager.homeDispatchSessionIDs(lcpOpts)
	if lcpSessionID != "lcp:v1:canonical-hash-xyz" {
		t.Fatalf("lcpSessionID = %q, want lcp:v1:canonical-hash-xyz (not msg-hash)", lcpSessionID)
	}
	if lcpParentID != "" {
		t.Fatalf("lcpParentID = %q, want empty", lcpParentID)
	}

	// 3b. LCP with metadata parent
	lcpForkOpts := cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.CanonicalSessionIDMetadataKey: "lcp:v1:fork-child-xyz",
			cliproxyexecutor.ParentSessionIDMetadataKey:    "lcp:v1:parent-root-abc",
		},
	}
	lcpForkSessionID, lcpForkParentID := manager.homeDispatchSessionIDs(lcpForkOpts)
	if lcpForkSessionID != "lcp:v1:fork-child-xyz" || lcpForkParentID != "lcp:v1:parent-root-abc" {
		t.Fatalf("lcpFork = (%q, %q), want (lcp:v1:fork-child-xyz, lcp:v1:parent-root-abc)", lcpForkSessionID, lcpForkParentID)
	}

	// 3c. Self-referential parent is suppressed
	lcpSelfParentOpts := cliproxyexecutor.Options{
		Metadata: map[string]any{
			cliproxyexecutor.CanonicalSessionIDMetadataKey: "lcp:v1:same-session",
			cliproxyexecutor.ParentSessionIDMetadataKey:    "lcp:v1:same-session",
		},
	}
	selfSessionID, selfParentID := manager.homeDispatchSessionIDs(lcpSelfParentOpts)
	if selfSessionID != "lcp:v1:same-session" || selfParentID != "" {
		t.Fatalf("self-referential parent = (%q, %q), want (%q, empty)", selfSessionID, selfParentID, "lcp:v1:same-session")
	}

	// 4. Antigravity hierarchy
	agyOpts := cliproxyexecutor.Options{
		Headers: http.Header{
			"X-Http-Session-Id":   []string{"agy-child-101"},
			"X-Parent-Session-ID": []string{"agy-parent-100"},
		},
	}
	agySessionID, agyParentID := manager.homeDispatchSessionIDs(agyOpts)
	if agySessionID != "agy:agy-child-101" {
		t.Fatalf("agySessionID = %q, want agy:agy-child-101", agySessionID)
	}
	if agyParentID != "agy:agy-parent-100" {
		t.Fatalf("agyParentID = %q, want agy:agy-parent-100", agyParentID)
	}

	// 5. Gemini cachedContent hierarchy
	geminiOpts := cliproxyexecutor.Options{
		OriginalRequest: []byte(`{"cachedContent":"cache-child-201","parent_session_id":"cache-parent-200"}`),
	}
	geminiSessionID, geminiParentID := manager.homeDispatchSessionIDs(geminiOpts)
	if geminiSessionID != "geminicache:cache-child-201" {
		t.Fatalf("geminiSessionID = %q, want geminicache:cache-child-201", geminiSessionID)
	}
	if geminiParentID != "geminicache:cache-parent-200" {
		t.Fatalf("geminiParentID = %q, want geminicache:cache-parent-200", geminiParentID)
	}
}

// TestHomeDispatchSessionIDsNestedRequestSubagent ports upstream
// 390589159eff: nested request envelopes resolve session + agent hierarchy.
func TestHomeDispatchSessionIDsNestedRequestSubagent(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{
		Home:    internalconfig.HomeConfig{Enabled: true},
		Routing: internalconfig.RoutingConfig{SessionAffinityTTL: "1h"},
	})

	// 1. Nested request with sessionId and agent_id
	nestedAgentOpts := cliproxyexecutor.Options{
		OriginalRequest: []byte(`{
			"request": {
				"sessionId": "root",
				"metadata": {
					"agent_id": "worker"
				}
			}
		}`),
	}
	sessionID, parentID := manager.homeDispatchSessionIDs(nestedAgentOpts)
	if sessionID != "session:root:agent:worker" {
		t.Fatalf("sessionID = %q, want session:root:agent:worker", sessionID)
	}
	if parentID != "session:root" {
		t.Fatalf("parentID = %q, want session:root", parentID)
	}

	// 2. Nested request with sessionId and subagent_id
	nestedSubagentOpts := cliproxyexecutor.Options{
		OriginalRequest: []byte(`{
			"request": {
				"sessionId": "root",
				"metadata": {
					"subagent_id": "worker-sub"
				}
			}
		}`),
	}
	subSessionID, subParentID := manager.homeDispatchSessionIDs(nestedSubagentOpts)
	if subSessionID != "session:root:agent:worker-sub" {
		t.Fatalf("subSessionID = %q, want session:root:agent:worker-sub", subSessionID)
	}
	if subParentID != "session:root" {
		t.Fatalf("subParentID = %q, want session:root", subParentID)
	}
}
