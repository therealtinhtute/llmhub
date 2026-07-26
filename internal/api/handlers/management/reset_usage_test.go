package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func newResetUsageHandler(t *testing.T, authID string) (*Handler, string) {
	t.Helper()
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	manager := coreauth.NewManager(nil, nil, nil)
	registered, errRegister := manager.Register(context.Background(), &coreauth.Auth{
		ID:       authID,
		Provider: "codex",
		Attributes: map[string]string{
			"api_key":  "codex-key",
			"base_url": "https://codex.example.com",
		},
	})
	if errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}
	registered.EnsureIndex()
	if registered.Index == "" {
		t.Fatal("registered auth has an empty index")
	}

	return NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager), registered.Index
}

func callResetUsage(t *testing.T, h *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v0/management/reset-usage", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ginCtx.Request = req
	h.ResetUsage(ginCtx)
	return rec
}

func usageTotalsFor(t *testing.T, h *Handler, authID string) (int64, int64, int64, int64) {
	t.Helper()
	for _, auth := range h.authManager.List() {
		if auth == nil || auth.ID != authID {
			continue
		}
		recentSuccess, recentFailed := sumRecentRequestBuckets(auth.RecentRequestsSnapshot(time.Now()))
		return auth.Success, auth.Failed, recentSuccess, recentFailed
	}
	t.Fatalf("auth %q not found in manager", authID)
	return 0, 0, 0, 0
}

func TestResetUsageClearsCountersAndRecentBuckets(t *testing.T) {
	h, authIndex := newResetUsageHandler(t, "codex-auth")

	h.authManager.MarkResult(context.Background(), coreauth.Result{AuthID: "codex-auth", Provider: "codex", Model: "gpt-5", Success: true})
	h.authManager.MarkResult(context.Background(), coreauth.Result{AuthID: "codex-auth", Provider: "codex", Model: "gpt-5", Success: false})

	success, failed, recentSuccess, recentFailed := usageTotalsFor(t, h, "codex-auth")
	if success != 1 || failed != 1 || recentSuccess != 1 || recentFailed != 1 {
		t.Fatalf("pre-reset totals = %d/%d counters, %d/%d buckets; want 1/1 and 1/1", success, failed, recentSuccess, recentFailed)
	}

	rec := callResetUsage(t, h, `{"auth_index":"`+authIndex+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload map[string]any
	if errUnmarshal := json.Unmarshal(rec.Body.Bytes(), &payload); errUnmarshal != nil {
		t.Fatalf("decode payload: %v", errUnmarshal)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status field = %#v, want \"ok\"", payload["status"])
	}
	if payload["auth_index"] != authIndex {
		t.Fatalf("auth_index = %#v, want %q", payload["auth_index"], authIndex)
	}

	success, failed, recentSuccess, recentFailed = usageTotalsFor(t, h, "codex-auth")
	if success != 0 || failed != 0 || recentSuccess != 0 || recentFailed != 0 {
		t.Fatalf("post-reset totals = %d/%d counters, %d/%d buckets; want all zero", success, failed, recentSuccess, recentFailed)
	}
}

func TestResetUsageIsIdempotent(t *testing.T) {
	h, authIndex := newResetUsageHandler(t, "codex-auth")
	h.authManager.MarkResult(context.Background(), coreauth.Result{AuthID: "codex-auth", Provider: "codex", Model: "gpt-5", Success: true})

	first := callResetUsage(t, h, `{"auth_index":"`+authIndex+`"}`)
	second := callResetUsage(t, h, `{"auth_index":"`+authIndex+`"}`)

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("statuses = %d and %d, want both %d", first.Code, second.Code, http.StatusOK)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("second call body = %s, want identical to first %s", second.Body.String(), first.Body.String())
	}

	success, failed, recentSuccess, recentFailed := usageTotalsFor(t, h, "codex-auth")
	if success != 0 || failed != 0 || recentSuccess != 0 || recentFailed != 0 {
		t.Fatalf("totals after two resets = %d/%d counters, %d/%d buckets; want all zero", success, failed, recentSuccess, recentFailed)
	}
}

func TestResetUsageLeavesQuotaAndRoutingStateAlone(t *testing.T) {
	h, authIndex := newResetUsageHandler(t, "codex-auth")

	h.authManager.MarkResult(context.Background(), coreauth.Result{
		AuthID:   "codex-auth",
		Provider: "codex",
		Model:    "gpt-5",
		Success:  false,
		Error:    &coreauth.Error{Message: "quota exceeded", HTTPStatus: http.StatusTooManyRequests},
	})

	before, ok := h.authManager.GetByID("codex-auth")
	if !ok || before == nil {
		t.Fatal("auth not found before reset")
	}
	beforeState, hasState := before.ModelStates["gpt-5"]
	if !hasState || beforeState == nil {
		t.Fatalf("expected a model state for gpt-5, got %#v", before.ModelStates)
	}
	wantUnavailable := beforeState.Unavailable
	wantQuotaExceeded := beforeState.Quota.Exceeded

	if rec := callResetUsage(t, h, `{"auth_index":"`+authIndex+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	after, ok := h.authManager.GetByID("codex-auth")
	if !ok || after == nil {
		t.Fatal("auth not found after reset")
	}
	afterState, hasState := after.ModelStates["gpt-5"]
	if !hasState || afterState == nil {
		t.Fatalf("model state for gpt-5 disappeared after reset: %#v", after.ModelStates)
	}
	if afterState.Unavailable != wantUnavailable {
		t.Fatalf("model unavailable = %v, want %v — reset-usage must not touch routing state", afterState.Unavailable, wantUnavailable)
	}
	if afterState.Quota.Exceeded != wantQuotaExceeded {
		t.Fatalf("model quota exceeded = %v, want %v — reset-usage must not touch quota state", afterState.Quota.Exceeded, wantQuotaExceeded)
	}
}

func TestResetUsageRejectsBadRequests(t *testing.T) {
	h, _ := newResetUsageHandler(t, "codex-auth")

	cases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{name: "invalid json", body: `{`, wantCode: http.StatusBadRequest},
		{name: "empty auth index", body: `{"auth_index":"  "}`, wantCode: http.StatusBadRequest},
		{name: "unknown auth index", body: `{"auth_index":"does-not-exist"}`, wantCode: http.StatusNotFound},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := callResetUsage(t, h, testCase.body)
			if rec.Code != testCase.wantCode {
				t.Fatalf("status = %d, want %d body=%s", rec.Code, testCase.wantCode, rec.Body.String())
			}
		})
	}
}

func TestResetUsageWithoutAuthManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}
	if rec := callResetUsage(t, h, `{"auth_index":"anything"}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}
