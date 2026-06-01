package kiro

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	Provider             = "kiro"
	DefaultRefreshURL    = "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken"
	DefaultAPIBaseURL    = "https://codewhisperer.us-east-1.amazonaws.com"
	DefaultRegion        = "us-east-1"
	DefaultAuthMethod    = "import"
	DefaultTokenLifetime = 3600
)

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	ExpiresAt    time.Time
}

func NormalizeImportData(ctx context.Context, data []byte, httpClient *http.Client) ([]byte, error) {
	meta, err := NormalizeImportMetadata(ctx, data, httpClient)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("kiro import: marshal normalized metadata: %w", err)
	}
	return raw, nil
}

func NormalizeImportMetadata(ctx context.Context, data []byte, httpClient *http.Client) (map[string]any, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("kiro import: empty input")
	}

	if !looksLikeJSON(trimmed) {
		return normalizeRawRefreshToken(ctx, string(trimmed), httpClient)
	}

	var raw any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("kiro import: invalid JSON: %w", err)
	}
	switch value := raw.(type) {
	case string:
		return normalizeRawRefreshToken(ctx, value, httpClient)
	case map[string]any:
		return normalizeObject(ctx, value, httpClient)
	default:
		return nil, fmt.Errorf("kiro import: unsupported JSON shape")
	}
}

func RefreshAccessToken(ctx context.Context, refreshToken string, providerSpecificData map[string]any, httpClient *http.Client) (*RefreshResult, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("kiro refresh: refresh token is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	clientID := metadataString(providerSpecificData, "clientId", "client_id")
	clientSecret := metadataString(providerSpecificData, "clientSecret", "client_secret")
	region := metadataString(providerSpecificData, "region")
	if region == "" {
		region = DefaultRegion
	}

	body := map[string]any{"refreshToken": refreshToken}
	url := DefaultRefreshURL
	if clientID != "" && clientSecret != "" {
		url = fmt.Sprintf("https://oidc.%s.amazonaws.com/token", region)
		body["clientId"] = clientID
		body["clientSecret"] = clientSecret
		body["grantType"] = "refresh_token"
	}
	if override := metadataString(providerSpecificData, "refreshUrl", "refresh_url"); override != "" {
		url = override
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("kiro refresh: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("kiro refresh: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "kiro-cli/1.0.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kiro refresh: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("kiro refresh: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kiro refresh: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	result, err := parseRefreshResponse(respBody, refreshToken)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func CredentialFileName(email, refreshToken string) string {
	segment := sanitizeFileSegment(email)
	if segment != "" {
		return "kiro-" + segment + ".json"
	}
	token := strings.TrimSpace(refreshToken)
	if token == "" {
		return fmt.Sprintf("kiro-%d.json", time.Now().UnixMilli())
	}
	sum := sha256.Sum256([]byte(token))
	return "kiro-" + hex.EncodeToString(sum[:6]) + ".json"
}

func normalizeRawRefreshToken(ctx context.Context, value string, httpClient *http.Client) (map[string]any, error) {
	refreshToken := strings.TrimSpace(value)
	if refreshToken == "" {
		return nil, fmt.Errorf("kiro import: refresh token is required")
	}
	refreshed, err := RefreshAccessToken(ctx, refreshToken, nil, httpClient)
	if err != nil {
		return nil, err
	}
	meta := baseMetadata()
	meta["refresh_token"] = refreshed.RefreshToken
	meta["access_token"] = refreshed.AccessToken
	meta["expired"] = refreshed.ExpiresAt.Format(time.RFC3339)
	meta["auth_method"] = DefaultAuthMethod
	return meta, nil
}

func normalizeObject(ctx context.Context, obj map[string]any, httpClient *http.Client) (map[string]any, error) {
	if objectType := strings.ToLower(strings.TrimSpace(stringValue(obj["type"]))); objectType == Provider {
		return normalizeLLMHubKiroObject(obj), nil
	}
	if provider := strings.ToLower(strings.TrimSpace(stringValue(obj["provider"]))); provider != Provider {
		return nil, fmt.Errorf("kiro import: provider must be %q", Provider)
	}
	if authType := strings.ToLower(strings.TrimSpace(stringValue(obj["authType"]))); authType != "" && authType != "oauth" {
		return nil, fmt.Errorf("kiro import: authType must be oauth")
	}

	refreshToken := strings.TrimSpace(stringValue(obj["refreshToken"]))
	if refreshToken == "" {
		return nil, fmt.Errorf("kiro import: refreshToken is required")
	}
	psd, _ := obj["providerSpecificData"].(map[string]any)
	meta := baseMetadata()
	meta["refresh_token"] = refreshToken
	if accessToken := strings.TrimSpace(stringValue(obj["accessToken"])); accessToken != "" {
		meta["access_token"] = accessToken
	} else {
		refreshed, err := RefreshAccessToken(ctx, refreshToken, psd, httpClient)
		if err != nil {
			return nil, err
		}
		meta["access_token"] = refreshed.AccessToken
		meta["refresh_token"] = refreshed.RefreshToken
		meta["expired"] = refreshed.ExpiresAt.Format(time.RFC3339)
	}
	if expiresAt := strings.TrimSpace(stringValue(obj["expiresAt"])); expiresAt != "" {
		meta["expired"] = expiresAt
	}
	if email := strings.TrimSpace(stringValue(obj["email"])); email != "" {
		meta["email"] = email
	}
	if profileARN := metadataString(psd, "profileArn", "profile_arn"); profileARN != "" {
		meta["profile_arn"] = profileARN
	}
	if authMethod := metadataString(psd, "authMethod", "auth_method"); authMethod != "" {
		meta["auth_method"] = authMethod
	} else {
		meta["auth_method"] = "oauth"
	}
	if region := metadataString(psd, "region"); region != "" {
		meta["region"] = region
	}
	if clientID := metadataString(psd, "clientId", "client_id"); clientID != "" {
		meta["client_id"] = clientID
	}
	if clientSecret := metadataString(psd, "clientSecret", "client_secret"); clientSecret != "" {
		meta["client_secret"] = clientSecret
	}
	if startURL := metadataString(psd, "startUrl", "start_url"); startURL != "" {
		meta["start_url"] = startURL
	}
	if obj["isActive"] != nil {
		active, ok := boolValue(obj["isActive"])
		if ok && !active {
			meta["disabled"] = true
		}
	}
	return meta, nil
}

func normalizeLLMHubKiroObject(obj map[string]any) map[string]any {
	meta := baseMetadata()
	for k, v := range obj {
		meta[k] = v
	}
	meta["type"] = Provider
	if disabled, ok := boolValue(meta["disabled"]); ok {
		meta["disabled"] = disabled
	}
	return meta
}

func parseRefreshResponse(body []byte, fallbackRefreshToken string) (*RefreshResult, error) {
	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("kiro refresh: decode response: %w", err)
	}
	accessToken := firstString(payload, "accessToken", "access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("kiro refresh: response missing access token")
	}
	refreshToken := firstString(payload, "refreshToken", "refresh_token")
	if refreshToken == "" {
		refreshToken = fallbackRefreshToken
	}
	expiresIn := intValue(firstPresent(payload, "expiresIn", "expires_in"))
	if expiresIn <= 0 {
		expiresIn = DefaultTokenLifetime
	}
	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		ExpiresAt:    time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

func baseMetadata() map[string]any {
	return map[string]any{
		"type":        Provider,
		"disabled":    false,
		"auth_method": DefaultAuthMethod,
	}
}

func looksLikeJSON(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	return data[0] == '{' || data[0] == '[' || data[0] == '"'
}

func firstString(m map[string]any, keys ...string) string {
	return strings.TrimSpace(stringValue(firstPresent(m, keys...)))
}

func firstPresent(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}

func metadataString(m map[string]any, keys ...string) string {
	if len(m) == 0 {
		return ""
	}
	return firstString(m, keys...)
}

func stringValue(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func intValue(v any) int {
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

func boolValue(v any) (bool, bool) {
	switch typed := v.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	}
	return false, false
}

func sanitizeFileSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '@' || r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
