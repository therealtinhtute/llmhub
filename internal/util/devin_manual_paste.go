package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/therealtinhtute/llmhub/internal/misc"
)

// ErrDevinUnrecognizedPaste is returned when pasted Devin login input cannot be
// classified as an authorization code, callback URL, or session token.
// Callers that allow retrying (e.g. a keep-waiting prompt) may treat it as
// non-fatal via errors.Is.
//
// Ported from upstream CLIProxyAPI sdk/auth/devin.go errDevinUnrecognizedPaste
// (5f74accd0e83: shared manual paste parsing between headless and loopback login).
var ErrDevinUnrecognizedPaste = errors.New("unrecognized devin authorization code or token format")

// ParseDevinManualPaste classifies a pasted Devin authorization code, callback
// URL, or session token. Empty input returns empty values with a nil error;
// callers decide whether to abort or keep waiting.
//
// When the pasted callback URL carries a state parameter and expectedState is
// non-empty, a mismatch is reported as a CSRF error; a pasted code without state
// is still accepted because PKCE protects the exchange. OAuth error parameters
// in a pasted callback URL (e.g. access_denied) are surfaced as errors.
//
// Ported from upstream CLIProxyAPI sdk/auth/devin.go parseDevinManualPaste
// (fe2fdde8a8ee manual code entry; 5f74accd0e83 shared parsing and error surfacing).
func ParseDevinManualPaste(input, expectedState string) (authCode, rawToken string, err error) {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.Trim(trimmed, "\"'")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "", "", nil
	}

	// Direct manual session token paste (supports Devin --force-manual-token-flow).
	if strings.HasPrefix(trimmed, "devin-session-token$") || strings.HasPrefix(trimmed, "eyJ") {
		return "", trimmed, nil
	}

	if parsed, errParse := misc.ParseOAuthCallback(trimmed); errParse == nil && parsed != nil {
		if errMsg := strings.TrimSpace(parsed.Error); errMsg != "" {
			if desc := strings.TrimSpace(parsed.ErrorDescription); desc != "" {
				errMsg = fmt.Sprintf("%s: %s", errMsg, desc)
			}
			return "", "", fmt.Errorf("devin oauth error: %s", errMsg)
		}
		if parsed.Code != "" {
			// If state is present in the pasted URL, it must match. PKCE still protects
			// a code pasted without state.
			if expectedState != "" && parsed.State != "" && parsed.State != expectedState {
				return "", "", fmt.Errorf("devin oauth state mismatch (possible CSRF)")
			}
			return parsed.Code, "", nil
		}
	}

	if !strings.ContainsAny(trimmed, " \t\r\n/?#=") {
		return trimmed, "", nil
	}
	return "", "", ErrDevinUnrecognizedPaste
}
