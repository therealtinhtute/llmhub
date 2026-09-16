package devin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const (
	// DefaultAppBaseURL is the user-facing web app for Devin OAuth.
	DefaultAppBaseURL = "https://app.devin.ai"
	// DefaultAPIBaseURL is the API backend for token exchange and user status.
	DefaultAPIBaseURL = "https://api.devin.ai"
	// DefaultServerURL is the upstream Codeium/Devin reasoning backend.
	DefaultServerURL = "https://server.codeium.com"

	devinTokenPrefix = "devin-session-token$"
)

// DevinTokenBundle holds the completed authentication result.
// Ported from upstream CLIProxyAPI internal/auth/devin/devin_auth.go (f94752762bb9).
type DevinTokenBundle struct {
	SessionToken string `json:"session_token"`
	RawToken     string `json:"raw_token"`
	UserName     string `json:"user_name,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	OrgID        string `json:"org_id,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
}

// DevinAuthService coordinates Devin PKCE authorization and token exchange.
type DevinAuthService struct {
	client        *http.Client
	appBaseURL    string
	apiBaseURL    string
	serverBaseURL string
}

// NewDevinAuthService creates a new Devin authentication service instance.
func NewDevinAuthService(client *http.Client) *DevinAuthService {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &DevinAuthService{
		client:        client,
		appBaseURL:    DefaultAppBaseURL,
		apiBaseURL:    DefaultAPIBaseURL,
		serverBaseURL: DefaultServerURL,
	}
}

// SetServerBaseURL overrides the upstream reasoning/seat management base URL (used in tests).
func (s *DevinAuthService) SetServerBaseURL(url string) {
	if s != nil && strings.TrimSpace(url) != "" {
		s.serverBaseURL = strings.TrimRight(strings.TrimSpace(url), "/")
	}
}

// SetAPIBaseURL overrides the token exchange and profile API base URL (used in tests).
func (s *DevinAuthService) SetAPIBaseURL(url string) {
	if s != nil && strings.TrimSpace(url) != "" {
		s.apiBaseURL = strings.TrimRight(strings.TrimSpace(url), "/")
	}
}

// SetAppBaseURL overrides the user-facing OAuth authorization base URL (used in tests).
func (s *DevinAuthService) SetAppBaseURL(url string) {
	if s != nil && strings.TrimSpace(url) != "" {
		s.appBaseURL = strings.TrimRight(strings.TrimSpace(url), "/")
	}
}

// BuildAuthorizationURL constructs the PKCE login URL. When redirectURI is empty,
// it generates a headless manual code URL (with cli_pkce_marker=1) matching the exact
// query parameter ordering of the official Devin CLI binary.
// Ported from upstream CLIProxyAPI (09807c57ea8e, fe2fdde8a8ee).
func (s *DevinAuthService) BuildAuthorizationURL(redirectURI, codeChallenge, state string) string {
	trimmedRedirect := strings.TrimSpace(redirectURI)
	var queryParts []string
	if trimmedRedirect != "" {
		queryParts = append(queryParts, "redirect_uri="+url.QueryEscape(trimmedRedirect))
	}
	if state != "" {
		queryParts = append(queryParts, "state="+url.QueryEscape(state))
	}
	queryParts = append(queryParts,
		"prompt=select_account",
		"code_challenge="+url.QueryEscape(codeChallenge),
		"code_challenge_method=S256",
	)
	if trimmedRedirect == "" {
		queryParts = append(queryParts, "cli_pkce_marker=1")
	}
	return fmt.Sprintf("%s/auth/cli/continue?%s", strings.TrimRight(s.appBaseURL, "/"), strings.Join(queryParts, "&"))
}

// ExchangeCodeForToken exchanges the authorization code for a session token.
func (s *DevinAuthService) ExchangeCodeForToken(ctx context.Context, code, codeVerifier string) (string, error) {
	reqBody := map[string]string{
		"code":          strings.TrimSpace(code),
		"code_verifier": strings.TrimSpace(codeVerifier),
	}
	jsonBody, errMarshal := json.Marshal(reqBody)
	if errMarshal != nil {
		return "", errMarshal
	}

	endpoint := fmt.Sprintf("%s/auth/cli/token", strings.TrimRight(s.apiBaseURL, "/"))
	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if errReq != nil {
		return "", errReq
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, errDo := s.client.Do(req)
	if errDo != nil {
		return "", fmt.Errorf("devin token exchange failed: %w", errDo)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBytes, errRead := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if errRead != nil {
		return "", fmt.Errorf("read token exchange response: %w", errRead)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	token := strings.TrimSpace(gjson.GetBytes(respBytes, "token").String())
	if token == "" {
		return "", fmt.Errorf("response did not contain a valid token: %s", string(respBytes))
	}

	return token, nil
}

// FetchSelfProfile retrieves the authenticated user's profile from api.devin.ai/v3/self.
func (s *DevinAuthService) FetchSelfProfile(ctx context.Context, sessionToken string) (userName, userID, orgID string, err error) {
	endpoint := fmt.Sprintf("%s/v3/self", strings.TrimRight(s.apiBaseURL, "/"))
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if errReq != nil {
		return "", "", "", errReq
	}
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	req.Header.Set("Accept", "application/json")

	resp, errDo := s.client.Do(req)
	if errDo != nil {
		return "", "", "", errDo
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBytes, errRead := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if errRead != nil {
		return "", "", "", errRead
	}

	if resp.StatusCode == http.StatusOK {
		root := gjson.ParseBytes(respBytes)
		userName = root.Get("user_name").String()
		userID = root.Get("user_id").String()
		orgID = root.Get("org_id").String()
	}

	return userName, userID, orgID, nil
}

// FormatSessionToken ensures the token carries the mandatory devin-session-token$ prefix.
func FormatSessionToken(rawToken string) string {
	t := strings.TrimSpace(rawToken)
	if strings.HasPrefix(t, devinTokenPrefix) {
		return t
	}
	if strings.HasPrefix(t, "eyJ") {
		return devinTokenPrefix + t
	}
	return t
}
