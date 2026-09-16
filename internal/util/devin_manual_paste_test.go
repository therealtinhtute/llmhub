package util

import (
	"errors"
	"strings"
	"testing"
)

// TestParseDevinManualPaste locks the shared paste classification used by the
// headless and loopback Devin login paths.
// Ported from upstream CLIProxyAPI sdk/auth/devin_test.go TestParseDevinManualPaste
// (fe2fdde8a8ee manual paste; 5f74accd0e83 shared parsing + error surfacing).
func TestParseDevinManualPaste(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedState string
		wantCode      string
		wantToken     string
		wantErr       string
		wantSentinel  error
	}{
		{name: "empty", input: "  "},
		{name: "quoted empty", input: `"   "`},
		{name: "session token", input: "devin-session-token$eyJabc", wantToken: "devin-session-token$eyJabc"},
		{name: "jwt", input: "eyJabc.def", wantToken: "eyJabc.def"},
		{name: "quoted jwt", input: `"eyJabc"`, wantToken: "eyJabc"},
		{name: "raw code", input: "devin-cli-auth-code-123", wantCode: "devin-cli-auth-code-123"},
		{name: "quoted code", input: `"quoted-code-xyz"`, wantCode: "quoted-code-xyz"},
		{name: "callback without state", input: "http://127.0.0.1:9/callback?code=c1", wantCode: "c1"},
		{name: "callback matching state", input: "http://127.0.0.1:9/callback?code=c1&state=abc", expectedState: "abc", wantCode: "c1"},
		{name: "callback mismatched state", input: "http://127.0.0.1:9/callback?code=stolen-code&state=other", expectedState: "abc", wantErr: "state mismatch"},
		{name: "oauth error", input: "http://127.0.0.1:9/callback?error=access_denied&state=abc", wantErr: "oauth error"},
		{name: "oauth error with description", input: "http://127.0.0.1:9/callback?error=access_denied&error_description=user%20denied", wantErr: "user denied"},
		{name: "unrecognized", input: "not a code, has spaces", wantErr: "unrecognized", wantSentinel: ErrDevinUnrecognizedPaste},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, token, errPaste := ParseDevinManualPaste(tt.input, tt.expectedState)
			if tt.wantErr != "" {
				if errPaste == nil || !strings.Contains(errPaste.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", errPaste, tt.wantErr)
				}
				if tt.wantSentinel != nil && !errors.Is(errPaste, tt.wantSentinel) {
					t.Fatalf("error %v is not %v", errPaste, tt.wantSentinel)
				}
				return
			}
			if errPaste != nil {
				t.Fatalf("unexpected error: %v", errPaste)
			}
			if code != tt.wantCode || token != tt.wantToken {
				t.Fatalf("code=%q token=%q, want code=%q token=%q", code, token, tt.wantCode, tt.wantToken)
			}
		})
	}
}
