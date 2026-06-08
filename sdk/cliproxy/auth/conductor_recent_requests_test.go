package auth

import (
	"context"
	"testing"
	"time"

	coreusage "github.com/therealtinhtute/llmhub/sdk/cliproxy/usage"
)

func TestManagerMarkResultRecordsRecentRequests(t *testing.T) {
	mgr := NewManager(nil, nil, nil)
	auth := &Auth{
		ID:       "auth-1",
		Provider: "antigravity",
		Attributes: map[string]string{
			"runtime_only": "true",
		},
		Metadata: map[string]any{
			"type": "antigravity",
		},
	}

	if _, err := mgr.Register(WithSkipPersist(context.Background()), auth); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	mgr.MarkResult(context.Background(), Result{AuthID: "auth-1", Provider: "antigravity", Model: "gpt-5", Success: true})
	mgr.MarkResult(context.Background(), Result{AuthID: "auth-1", Provider: "antigravity", Model: "gpt-5", Success: false})

	gotAuth, ok := mgr.GetByID("auth-1")
	if !ok || gotAuth == nil {
		t.Fatalf("GetByID returned ok=%v auth=%v", ok, gotAuth)
	}

	if gotAuth.Success != 1 || gotAuth.Failed != 1 {
		t.Fatalf("auth totals = success=%d failed=%d, want 1/1", gotAuth.Success, gotAuth.Failed)
	}

	snapshot := gotAuth.RecentRequestsSnapshot(time.Now())
	var successTotal int64
	var failedTotal int64
	for _, bucket := range snapshot {
		successTotal += bucket.Success
		failedTotal += bucket.Failed
	}
	if successTotal != 1 || failedTotal != 1 {
		t.Fatalf("totals = success=%d failed=%d, want 1/1", successTotal, failedTotal)
	}
}

func TestManagerUpdatePreservesRecentRequestsAndTotals(t *testing.T) {
	mgr := NewManager(nil, nil, nil)
	auth := &Auth{
		ID:       "auth-1",
		Provider: "antigravity",
		Metadata: map[string]any{
			"type": "antigravity",
		},
	}
	if _, err := mgr.Register(WithSkipPersist(context.Background()), auth); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	mgr.MarkResult(context.Background(), Result{AuthID: "auth-1", Provider: "antigravity", Model: "gpt-5", Success: true})

	updated := &Auth{
		ID:       "auth-1",
		Provider: "antigravity",
		Metadata: map[string]any{
			"type": "antigravity",
			"note": "updated",
		},
	}
	if _, err := mgr.Update(WithSkipPersist(context.Background()), updated); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	gotAuth, ok := mgr.GetByID("auth-1")
	if !ok || gotAuth == nil {
		t.Fatalf("GetByID returned ok=%v auth=%v", ok, gotAuth)
	}
	if gotAuth.Success != 1 || gotAuth.Failed != 0 {
		t.Fatalf("auth totals = success=%d failed=%d, want 1/0", gotAuth.Success, gotAuth.Failed)
	}

	snapshot := gotAuth.RecentRequestsSnapshot(time.Now())
	var successTotal int64
	var failedTotal int64
	for _, bucket := range snapshot {
		successTotal += bucket.Success
		failedTotal += bucket.Failed
	}
	if successTotal != 1 || failedTotal != 0 {
		t.Fatalf("bucket totals = success=%d failed=%d, want 1/0", successTotal, failedTotal)
	}
}

func TestManagerMarkResultRecordsKiroRuntimeUsageStats(t *testing.T) {
	mgr := NewManager(nil, nil, nil)
	auth := &Auth{
		ID:       "kiro-auth",
		Provider: "kiro",
		Metadata: map[string]any{
			"type": "kiro",
		},
	}
	if _, err := mgr.Register(WithSkipPersist(context.Background()), auth); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	mgr.MarkResult(context.Background(), Result{
		AuthID:   "kiro-auth",
		Provider: "kiro",
		Model:    "claude-sonnet-4.5",
		Success:  true,
		UsageDetail: coreusage.Detail{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
	})
	mgr.MarkResult(context.Background(), Result{
		AuthID:         "kiro-auth",
		Provider:       "kiro",
		Model:          "claude-sonnet-4.5",
		Success:        true,
		UsageEstimated: true,
		UsageDetail: coreusage.Detail{
			InputTokens:  20,
			OutputTokens: 8,
			TotalTokens:  28,
		},
	})

	gotAuth, ok := mgr.GetByID("kiro-auth")
	if !ok || gotAuth == nil {
		t.Fatalf("GetByID returned ok=%v auth=%v", ok, gotAuth)
	}
	stats, ok := gotAuth.Metadata["kiro_usage_stats"].(map[string]any)
	if !ok {
		t.Fatalf("kiro_usage_stats missing: %#v", gotAuth.Metadata)
	}
	if got := int64FromMetadata(stats["requests"]); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
	if got := int64FromMetadata(stats["prompt_tokens"]); got != 30 {
		t.Fatalf("prompt_tokens = %d, want 30", got)
	}
	if got := int64FromMetadata(stats["completion_tokens"]); got != 13 {
		t.Fatalf("completion_tokens = %d, want 13", got)
	}
	if got := int64FromMetadata(stats["total_tokens"]); got != 43 {
		t.Fatalf("total_tokens = %d, want 43", got)
	}
	if got := int64FromMetadata(stats["estimated_tokens"]); got != 28 {
		t.Fatalf("estimated_tokens = %d, want 28", got)
	}
	if got := stats["last_model"]; got != "claude-sonnet-4.5" {
		t.Fatalf("last_model = %#v, want claude-sonnet-4.5", got)
	}
}
