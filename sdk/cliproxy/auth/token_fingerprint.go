package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// AccessTokenSHA256 returns the normalized OAuth access-token fingerprint used
// to fence asynchronous Home execution results and refresh requests without
// exposing the token itself.
func AccessTokenSHA256(auth *Auth) string {
	accessToken := accessTokenForFingerprint(auth)
	if accessToken == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(accessToken))
	return hex.EncodeToString(digest[:])
}

// accessTokenForFingerprint extracts the access token from the common metadata
// shapes: flat access_token/accessToken keys, or a nested token object.
func accessTokenForFingerprint(auth *Auth) string {
	if auth == nil || auth.Metadata == nil {
		return ""
	}
	for _, key := range []string{"access_token", "accessToken"} {
		if value, ok := auth.Metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	for _, key := range []string{"token", "Token"} {
		switch token := auth.Metadata[key].(type) {
		case map[string]any:
			for _, tokenKey := range []string{"access_token", "accessToken"} {
				if value, ok := token[tokenKey].(string); ok && strings.TrimSpace(value) != "" {
					return strings.TrimSpace(value)
				}
			}
		case map[string]string:
			for _, tokenKey := range []string{"access_token", "accessToken"} {
				if value := strings.TrimSpace(token[tokenKey]); value != "" {
					return value
				}
			}
		}
	}
	return ""
}
