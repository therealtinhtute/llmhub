package helps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/home"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

type homeStatusErr struct {
	code int
	msg  string
}

func (e homeStatusErr) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return fmt.Sprintf("status %d", e.code)
}

func (e homeStatusErr) StatusCode() int { return e.code }

type homeErrorEnvelope struct {
	Error *homeErrorDetail `json:"error"`
}

type homeErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type homeRefreshClient interface {
	HeartbeatOK() bool
	GetRefreshAuth(ctx context.Context, authIndex string, accessTokenSHA256 string) ([]byte, error)
}

var currentHomeRefreshClient = func() homeRefreshClient {
	return home.Current()
}

// RefreshAuthViaHome replaces local refresh logic when home control plane integration is enabled.
// It returns (updatedAuth, true, nil) when home refresh succeeds; (nil, true, err) when home is
// enabled but refresh fails; and (nil, false, nil) when home is disabled.
func RefreshAuthViaHome(ctx context.Context, cfg *config.Config, auth *cliproxyauth.Auth) (*cliproxyauth.Auth, bool, error) {
	if cfg == nil || !cfg.Home.Enabled {
		return nil, false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if auth == nil {
		return nil, true, homeStatusErr{code: http.StatusInternalServerError, msg: "home refresh: auth is nil"}
	}

	client := currentHomeRefreshClient()
	if client == nil || !client.HeartbeatOK() {
		return nil, true, homeStatusErr{code: http.StatusServiceUnavailable, msg: "home control center unavailable"}
	}

	authIndex := strings.TrimSpace(auth.Index)
	if authIndex == "" {
		authIndex = strings.TrimSpace(auth.EnsureIndex())
	}
	if authIndex == "" {
		return nil, true, homeStatusErr{code: http.StatusBadGateway, msg: "home refresh: auth_index is empty"}
	}

	raw, err := client.GetRefreshAuth(ctx, authIndex, authAccessTokenSHA256(auth))
	if err != nil {
		// Preserve request-scoped context errors so cancellation and deadline
		// propagate to the caller; redact everything else so transport details or
		// provider secrets are never leaked into refresh results.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, true, err
		}
		return nil, true, homeStatusErr{code: http.StatusServiceUnavailable, msg: "home refresh temporarily unavailable"}
	}

	var env homeErrorEnvelope
	if errUnmarshal := json.Unmarshal(raw, &env); errUnmarshal == nil && env.Error != nil {
		code := strings.TrimSpace(env.Error.Type)
		if code == "" {
			code = strings.TrimSpace(env.Error.Code)
		}
		// Never echo the upstream error.message: it may carry provider secrets.
		// Map to a redacted, status-appropriate message instead.
		statusCode := statusFromHomeErrorCode(code)
		message := "credential refresh temporarily unavailable"
		switch statusCode {
		case http.StatusUnauthorized:
			message = "credential unauthorized"
		case http.StatusNotFound:
			message = "credential refresh target not found"
		}
		return nil, true, homeStatusErr{code: statusCode, msg: message}
	}

	var updated cliproxyauth.Auth
	if errUnmarshal := json.Unmarshal(raw, &updated); errUnmarshal != nil {
		return nil, true, homeStatusErr{code: http.StatusBadGateway, msg: "home returned invalid auth payload"}
	}
	if updated.Disabled || updated.Status == cliproxyauth.StatusDisabled {
		return nil, true, homeStatusErr{code: http.StatusUnauthorized, msg: "credential unauthorized"}
	}
	updated.Index = authIndex
	updated.EnsureIndex()
	return &updated, true, nil
}

func authAccessTokenSHA256(auth *cliproxyauth.Auth) string {
	return cliproxyauth.AccessTokenSHA256(auth)
}

func statusFromHomeErrorCode(code string) int {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "authentication_error", "unauthorized", "invalid_grant", "refresh_token_expired", "refresh_token_revoked", "refresh_token_reused":
		return http.StatusUnauthorized
	case "model_not_found":
		return http.StatusNotFound
	case "auth_not_found", "auth_unavailable", "refresh_temporarily_unavailable", "refresh_unsupported", "home_unavailable":
		return http.StatusServiceUnavailable
	default:
		return http.StatusServiceUnavailable
	}
}
