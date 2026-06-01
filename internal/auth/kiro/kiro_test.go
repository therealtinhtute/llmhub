package kiro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeImportMetadata_9RouterJSON(t *testing.T) {
	input := `{
		"provider":"kiro",
		"authType":"oauth",
		"accessToken":"access-1",
		"refreshToken":"refresh-1",
		"expiresAt":"2026-06-01T10:00:00Z",
		"email":"user@example.com",
		"isActive":false,
		"providerSpecificData":{
			"profileArn":"arn:aws:codewhisperer:us-east-1:123456789012:profile/ABC",
			"authMethod":"builder-id",
			"clientId":"client-1",
			"clientSecret":"secret-1",
			"region":"us-east-1"
		}
	}`

	meta, err := NormalizeImportMetadata(context.Background(), []byte(input), nil)
	if err != nil {
		t.Fatalf("NormalizeImportMetadata() error = %v", err)
	}
	assertMetaString(t, meta, "type", "kiro")
	assertMetaString(t, meta, "access_token", "access-1")
	assertMetaString(t, meta, "refresh_token", "refresh-1")
	assertMetaString(t, meta, "expired", "2026-06-01T10:00:00Z")
	assertMetaString(t, meta, "email", "user@example.com")
	assertMetaString(t, meta, "profile_arn", "arn:aws:codewhisperer:us-east-1:123456789012:profile/ABC")
	assertMetaString(t, meta, "auth_method", "builder-id")
	if disabled, _ := meta["disabled"].(bool); !disabled {
		t.Fatalf("disabled = %#v, want true", meta["disabled"])
	}
}

func TestNormalizeImportMetadata_RawRefreshTokenRefreshes(t *testing.T) {
	var sawBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"accessToken":"access-2","refreshToken":"refresh-2","expiresIn":1800}`))
	}))
	defer server.Close()

	meta, err := NormalizeImportMetadata(context.Background(), []byte(" refresh-raw "), &http.Client{
		Transport: rewriteHostTransport{target: server.URL},
	})
	if err != nil {
		t.Fatalf("NormalizeImportMetadata() error = %v", err)
	}
	if got := sawBody["refreshToken"]; got != "refresh-raw" {
		t.Fatalf("refreshToken body = %#v, want refresh-raw", got)
	}
	assertMetaString(t, meta, "access_token", "access-2")
	assertMetaString(t, meta, "refresh_token", "refresh-2")
	if got := strings.TrimSpace(meta["expired"].(string)); got == "" {
		t.Fatal("expired is empty")
	}
}

func TestRefreshAccessToken_AWSOIDCUsesClientCredentials(t *testing.T) {
	var sawBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"accessToken":"aws-access","expiresIn":3600}`))
	}))
	defer server.Close()

	result, err := RefreshAccessToken(context.Background(), "refresh-aws", map[string]any{
		"clientId":     "client",
		"clientSecret": "secret",
		"refreshUrl":   server.URL,
	}, server.Client())
	if err != nil {
		t.Fatalf("RefreshAccessToken() error = %v", err)
	}
	if result.AccessToken != "aws-access" {
		t.Fatalf("AccessToken = %q, want aws-access", result.AccessToken)
	}
	if sawBody["clientId"] != "client" || sawBody["clientSecret"] != "secret" || sawBody["grantType"] != "refresh_token" {
		t.Fatalf("unexpected request body: %#v", sawBody)
	}
}

type rewriteHostTransport struct {
	target string
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	targetReq := req.Clone(req.Context())
	targetReq.URL.Scheme = "http"
	targetReq.URL.Host = strings.TrimPrefix(t.target, "http://")
	return http.DefaultTransport.RoundTrip(targetReq)
}

func assertMetaString(t *testing.T, meta map[string]any, key, want string) {
	t.Helper()
	got, _ := meta[key].(string)
	if got != want {
		t.Fatalf("meta[%s] = %q, want %q", key, got, want)
	}
}
