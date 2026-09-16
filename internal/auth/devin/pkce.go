// Package devin provides authentication functionality for Devin / Cognition.
// It implements the PKCE (Proof Key for Code Exchange) authorization flow,
// loopback and manual-paste login, and session token management used by the
// Devin executor and login paths.
package devin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCECodes contains the PKCE code verifier and code challenge pair.
// Ported from upstream CLIProxyAPI internal/auth/devin/pkce.go (f94752762bb9).
type PKCECodes struct {
	CodeVerifier  string
	CodeChallenge string
}

// GeneratePKCECodes generates a random code verifier and its S256 challenge.
func GeneratePKCECodes() (*PKCECodes, error) {
	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	verifier := base64.RawURLEncoding.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCECodes{
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
	}, nil
}
