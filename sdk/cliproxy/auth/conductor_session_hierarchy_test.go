package auth

import (
	"context"
	"net/http"
	"testing"

	internallogging "github.com/therealtinhtute/llmhub/internal/logging"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	coresession "github.com/therealtinhtute/llmhub/sdk/cliproxy/session"
)

// Ported from upstream CLIProxyAPI commit 580df36423e4
// (conductor_execution_quota_test.go): metadata session hierarchy is mirrored
// into ClientRequestMetadata with canonical > LCP > execution > derived
// precedence, stale-value clearing, and self-loop elimination.
func TestSyncMetadataSessionToContext(t *testing.T) {
	// 1. Canonical session from metadata overrides raw/pre-alias context session
	ctxExplicit := internallogging.WithClientRequestMetadata(context.Background(), internallogging.ClientRequestMetadata{
		SessionID: "raw-alias-sid",
	})
	res1 := syncMetadataSessionToContext(ctxExplicit, map[string]any{
		cliproxyexecutor.CanonicalSessionIDMetadataKey: "canonical-override",
	})
	meta1 := internallogging.GetClientRequestMetadata(res1)
	if meta1.SessionID != "canonical-override" {
		t.Fatalf("sessionID = %q, want canonical-override", meta1.SessionID)
	}

	// 2. Syncs canonical and parent session from metadata
	res2 := syncMetadataSessionToContext(context.Background(), map[string]any{
		cliproxyexecutor.CanonicalSessionIDMetadataKey: "lcp:branch-123",
		cliproxyexecutor.ParentSessionIDMetadataKey:    "lcp:trunk-000",
	})
	meta2 := internallogging.GetClientRequestMetadata(res2)
	if meta2.SessionID != "lcp:branch-123" || meta2.ParentSessionID != "lcp:trunk-000" {
		t.Fatalf("synced session = (%q, %q), want (lcp:branch-123, lcp:trunk-000)", meta2.SessionID, meta2.ParentSessionID)
	}

	// 3. Eliminates self-loop
	res3 := syncMetadataSessionToContext(context.Background(), map[string]any{
		cliproxyexecutor.CanonicalSessionIDMetadataKey: "same-loop",
		cliproxyexecutor.ParentSessionIDMetadataKey:    "same-loop",
	})
	meta3 := internallogging.GetClientRequestMetadata(res3)
	if meta3.SessionID != "same-loop" || meta3.ParentSessionID != "" {
		t.Fatalf("self-loop session = (%q, %q), want (same-loop, empty)", meta3.SessionID, meta3.ParentSessionID)
	}

	// 4. Clears stale parent when syncing a new root session without parent metadata
	ctxWithStaleParent := internallogging.WithClientRequestMetadata(context.Background(), internallogging.ClientRequestMetadata{
		SessionID:       "old-child",
		ParentSessionID: "old-stale-parent",
	})
	res4 := syncMetadataSessionToContext(ctxWithStaleParent, map[string]any{
		cliproxyexecutor.CanonicalSessionIDMetadataKey: "new-root-session",
	})
	meta4 := internallogging.GetClientRequestMetadata(res4)
	if meta4.SessionID != "new-root-session" || meta4.ParentSessionID != "" {
		t.Fatalf("stale parent not cleared: (%q, %q), want (new-root-session, empty)", meta4.SessionID, meta4.ParentSessionID)
	}

	// 5. Execution session key fallback with prefix
	res5 := syncMetadataSessionToContext(context.Background(), map[string]any{
		cliproxyexecutor.ExecutionSessionMetadataKey: "call-live-123",
	})
	meta5 := internallogging.GetClientRequestMetadata(res5)
	if meta5.SessionID != "execution:call-live-123" {
		t.Fatalf("execution session = %q, want execution:call-live-123", meta5.SessionID)
	}

	// 6. Derived session key fallback gets derived: prefix
	res6 := syncMetadataSessionToContext(context.Background(), map[string]any{
		cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:raw-hash",
	})
	meta6 := internallogging.GetClientRequestMetadata(res6)
	if meta6.SessionID != "derived:ctx:v1:raw-hash" {
		t.Fatalf("derived session = %q, want derived:ctx:v1:raw-hash", meta6.SessionID)
	}

	// 7. Clears hierarchy completely when metadata has no canonical session
	ctxWithExistingSession := internallogging.WithClientRequestMetadata(context.Background(), internallogging.ClientRequestMetadata{
		SessionID:       "old-stale-session",
		ParentSessionID: "old-stale-parent",
	})
	res7 := syncMetadataSessionToContext(ctxWithExistingSession, map[string]any{})
	meta7 := internallogging.GetClientRequestMetadata(res7)
	if meta7.SessionID != "" || meta7.ParentSessionID != "" {
		t.Fatalf("hierarchy not cleared when metadata empty: (%q, %q), want empty", meta7.SessionID, meta7.ParentSessionID)
	}
}

// Ported from upstream CLIProxyAPI commit 580df36423e4: an explicit root
// session must not inherit a stale parent left in metadata by a prior turn.
func TestGhostParentElimination(t *testing.T) {
	// Request has explicit root session (no parent), but options metadata carries a stale parent key
	req := cliproxyexecutor.Request{}
	opts := cliproxyexecutor.Options{
		Headers: http.Header{
			"X-Session-ID": []string{"clean-root-session"},
		},
		Metadata: map[string]any{
			cliproxyexecutor.ParentSessionIDMetadataKey: "ghost-parent-from-prior-turn",
		},
	}

	_, enrichedOpts := coresession.Enrich(req, opts)
	if _, hasGhost := enrichedOpts.Metadata[cliproxyexecutor.ParentSessionIDMetadataKey]; hasGhost {
		t.Fatalf("ghost parent was not removed by session.Enrich")
	}
	if canonical := enrichedOpts.Metadata[cliproxyexecutor.CanonicalSessionIDMetadataKey]; canonical != "header:clean-root-session" {
		t.Fatalf("canonical session = %v, want header:clean-root-session", canonical)
	}
}
