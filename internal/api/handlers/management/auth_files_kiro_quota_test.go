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
	runtimeexecutor "github.com/therealtinhtute/llmhub/internal/runtime/executor"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestRefreshKiroQuota_UpdatesAuthRuntimeState(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getUsageLimits" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q, want access-token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"usageBreakdownList":[{"currentUsage":100,"usageLimit":100}],
			"nextDateReset":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `",
			"subscriptionInfo":{"subscriptionType":"paid","subscriptionTitle":"Kiro Pro"}
		}`))
	}))
	defer upstream.Close()

	store := &memoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)
	record := &coreauth.Auth{
		ID:       "kiro-auth.json",
		FileName: "kiro-auth.json",
		Provider: "kiro",
		Status:   coreauth.StatusActive,
		Metadata: map[string]any{
			"type":         "kiro",
			"access_token": "access-token",
			"quota_url":    strings.TrimRight(upstream.URL, "/") + "/getUsageLimits",
		},
	}
	if _, err := manager.Register(context.Background(), record); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/auth-files/kiro/quota", strings.NewReader(`{"name":"kiro-auth.json"}`))

	h.RefreshKiroQuota(ginCtx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	quota, ok := payload["quota"].(map[string]any)
	if !ok {
		t.Fatalf("quota payload missing: %#v", payload)
	}
	if quota["current"].(float64) != 100 || quota["limit"].(float64) != 100 {
		t.Fatalf("quota = %#v, want exhausted 100/100", quota)
	}

	updated, ok := manager.GetByID("kiro-auth.json")
	if !ok {
		t.Fatal("updated auth not found")
	}
	if !updated.Quota.Exceeded || updated.Quota.Reason != "provider_quota" {
		t.Fatalf("auth quota = %#v, want provider quota exceeded", updated.Quota)
	}
	if updated.Metadata["kiro_quota"] == nil {
		t.Fatalf("metadata missing kiro_quota: %#v", updated.Metadata)
	}
	if store.SaveCount() < 2 {
		t.Fatalf("store SaveCount = %d, want register + update saves", store.SaveCount())
	}
}

func TestRefreshKiroQuota_QuotaRefreshErrorDoesNotMarkAuthError(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"quota refresh throttled"}`))
	}))
	defer upstream.Close()

	store := &memoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)
	record := &coreauth.Auth{
		ID:       "kiro-auth.json",
		FileName: "kiro-auth.json",
		Provider: "kiro",
		Status:   coreauth.StatusActive,
		Metadata: map[string]any{
			"type":         "kiro",
			"access_token": "access-token",
			"quota_url":    strings.TrimRight(upstream.URL, "/") + "/getUsageLimits",
		},
	}
	if _, err := manager.Register(context.Background(), record); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/auth-files/kiro/quota", strings.NewReader(`{"name":"kiro-auth.json"}`))

	h.RefreshKiroQuota(ginCtx)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	updated, ok := manager.GetByID("kiro-auth.json")
	if !ok {
		t.Fatal("updated auth not found")
	}
	if updated.Status == coreauth.StatusError {
		t.Fatalf("auth status = %s, want not error", updated.Status)
	}
	if updated.LastError != nil {
		t.Fatalf("LastError = %#v, want nil", updated.LastError)
	}
	if updated.Quota.Exceeded {
		t.Fatalf("auth quota exceeded = true, want false")
	}
	quota, ok := updated.Metadata["kiro_quota"].(runtimeexecutor.KiroQuotaState)
	if !ok {
		t.Fatalf("kiro_quota = %#v, want KiroQuotaState", updated.Metadata["kiro_quota"])
	}
	if quota.ProviderQuotaAvailable {
		t.Fatalf("ProviderQuotaAvailable = true, want false")
	}
	if !strings.Contains(quota.Message, "429") {
		t.Fatalf("quota message = %q, want status detail", quota.Message)
	}
}
