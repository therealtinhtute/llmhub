package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

// TestManagementResetUsageRouteEndToEnd drives POST /v0/management/reset-usage through
// the real router, so it covers route registration and the management auth middleware
// that handler-level tests bypass.
func TestManagementResetUsageRouteEndToEnd(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "test-management-key")

	authManager := auth.NewManager(nil, nil, nil)
	server := newTestServerWithAuthManager(t, authManager)

	registered, errRegister := authManager.Register(context.Background(), &auth.Auth{
		ID:       "codex-auth",
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
	authIndex := registered.Index
	if authIndex == "" {
		t.Fatal("registered auth has an empty index")
	}

	authManager.MarkResult(context.Background(), auth.Result{AuthID: "codex-auth", Provider: "codex", Model: "gpt-5", Success: true})
	authManager.MarkResult(context.Background(), auth.Result{AuthID: "codex-auth", Provider: "codex", Model: "gpt-5", Success: false})

	body := func() *bytes.Reader {
		return bytes.NewReader([]byte(`{"auth_index":"` + authIndex + `"}`))
	}

	unauthorized := httptest.NewRequest(http.MethodPost, "/v0/management/reset-usage", body())
	unauthorized.Header.Set("Content-Type", "application/json")
	unauthorizedRR := httptest.NewRecorder()
	server.engine.ServeHTTP(unauthorizedRR, unauthorized)
	if unauthorizedRR.Code != http.StatusUnauthorized {
		t.Fatalf("missing key status = %d, want %d body=%s", unauthorizedRR.Code, http.StatusUnauthorized, unauthorizedRR.Body.String())
	}

	before, ok := authManager.GetByID("codex-auth")
	if !ok || before == nil {
		t.Fatal("auth not found before reset")
	}
	if before.Success != 1 || before.Failed != 1 {
		t.Fatalf("pre-reset counters = %d/%d, want 1/1 — the unauthorized call must not have reset anything", before.Success, before.Failed)
	}

	authorized := httptest.NewRequest(http.MethodPost, "/v0/management/reset-usage", body())
	authorized.Header.Set("Content-Type", "application/json")
	authorized.Header.Set("Authorization", "Bearer test-management-key")
	authorizedRR := httptest.NewRecorder()
	server.engine.ServeHTTP(authorizedRR, authorized)
	if authorizedRR.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d, want %d body=%s", authorizedRR.Code, http.StatusOK, authorizedRR.Body.String())
	}

	var payload struct {
		Status    string `json:"status"`
		AuthIndex string `json:"auth_index"`
	}
	if errUnmarshal := json.Unmarshal(authorizedRR.Body.Bytes(), &payload); errUnmarshal != nil {
		t.Fatalf("unmarshal response: %v body=%s", errUnmarshal, authorizedRR.Body.String())
	}
	if payload.Status != "ok" {
		t.Fatalf("status = %q, want %q", payload.Status, "ok")
	}
	if payload.AuthIndex != authIndex {
		t.Fatalf("auth_index = %q, want %q", payload.AuthIndex, authIndex)
	}

	after, ok := authManager.GetByID("codex-auth")
	if !ok || after == nil {
		t.Fatal("auth not found after reset")
	}
	if after.Success != 0 || after.Failed != 0 {
		t.Fatalf("post-reset counters = %d/%d, want 0/0", after.Success, after.Failed)
	}
}
