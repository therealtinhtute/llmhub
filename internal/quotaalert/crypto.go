package quotaalert

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	SecretCipherVersion      uint8 = 1
	SecretKeySize                  = 32
	SecretNonceSize                = 12
	SecretCiphertextOverhead       = 16
	MaxSecretKeyIDLength           = 256
	MaxSecretValueLength           = 256
)

var errSecretDecryption = errors.New("quota alert secret could not be decrypted")

// SecretCipher encrypts database-backed secrets with one explicit key identity.
type SecretCipher struct {
	keyID string
	aead  cipher.AEAD
}

// NewSecretCipher constructs an AES-256-GCM secret cipher.
func NewSecretCipher(keyID string, key []byte) (*SecretCipher, error) {
	if len(keyID) > MaxSecretKeyIDLength {
		return nil, fmt.Errorf("quota alert secret key ID must not exceed %d bytes", MaxSecretKeyIDLength)
	}
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return nil, fmt.Errorf("quota alert secret key ID is required")
	}
	if len(key) != SecretKeySize {
		return nil, fmt.Errorf("quota alert secret key must be exactly %d bytes", SecretKeySize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize quota alert secret cipher")
	}
	aead, err := cipher.NewGCMWithNonceSize(block, SecretNonceSize)
	if err != nil {
		return nil, fmt.Errorf("initialize quota alert secret cipher")
	}
	return &SecretCipher{keyID: keyID, aead: aead}, nil
}

// KeyID returns the non-secret identity of the active encryption key.
func (c *SecretCipher) KeyID() string {
	if c == nil {
		return ""
	}
	return c.keyID
}

// EncryptedSecret is an immutable storage representation of an encrypted secret.
type EncryptedSecret struct {
	version    uint8
	keyID      string
	nonce      []byte
	ciphertext []byte
}

// NewEncryptedSecret validates encrypted fields loaded from durable storage.
func NewEncryptedSecret(version uint8, keyID string, nonce, ciphertext []byte) (EncryptedSecret, error) {
	if len(keyID) > MaxSecretKeyIDLength {
		return EncryptedSecret{}, fmt.Errorf("encrypted quota alert secret key ID must not exceed %d bytes", MaxSecretKeyIDLength)
	}
	keyID = strings.TrimSpace(keyID)
	if version != SecretCipherVersion {
		return EncryptedSecret{}, fmt.Errorf("unsupported quota alert secret cipher version")
	}
	if keyID == "" {
		return EncryptedSecret{}, fmt.Errorf("encrypted quota alert secret key ID is required")
	}
	if len(nonce) != SecretNonceSize {
		return EncryptedSecret{}, fmt.Errorf("encrypted quota alert secret nonce is invalid")
	}
	if len(ciphertext) < SecretCiphertextOverhead || len(ciphertext) > MaxSecretValueLength+SecretCiphertextOverhead {
		return EncryptedSecret{}, fmt.Errorf("encrypted quota alert secret payload is invalid")
	}
	return EncryptedSecret{
		version:    version,
		keyID:      keyID,
		nonce:      slices.Clone(nonce),
		ciphertext: slices.Clone(ciphertext),
	}, nil
}

func (s EncryptedSecret) Version() uint8     { return s.version }
func (s EncryptedSecret) KeyID() string      { return s.keyID }
func (s EncryptedSecret) Nonce() []byte      { return slices.Clone(s.nonce) }
func (s EncryptedSecret) Ciphertext() []byte { return slices.Clone(s.ciphertext) }

// Encrypt encrypts non-empty secret material with purpose-bound authenticated data.
func (c *SecretCipher) Encrypt(purpose string, plaintext []byte) (EncryptedSecret, error) {
	if c == nil || c.aead == nil {
		return EncryptedSecret{}, fmt.Errorf("quota alert secret cipher is not initialized")
	}
	if strings.TrimSpace(purpose) == "" {
		return EncryptedSecret{}, fmt.Errorf("quota alert secret purpose is required")
	}
	if len(plaintext) == 0 {
		return EncryptedSecret{}, fmt.Errorf("quota alert secret value is required")
	}
	if len(plaintext) > MaxSecretValueLength {
		return EncryptedSecret{}, fmt.Errorf("quota alert secret value must not exceed %d bytes", MaxSecretValueLength)
	}

	nonce := make([]byte, SecretNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return EncryptedSecret{}, fmt.Errorf("generate quota alert secret nonce")
	}
	ciphertext := c.aead.Seal(nil, nonce, plaintext, secretAAD(SecretCipherVersion, c.keyID, purpose))
	return NewEncryptedSecret(SecretCipherVersion, c.keyID, nonce, ciphertext)
}

// Decrypt authenticates and decrypts a secret for the exact purpose and key ID.
func (c *SecretCipher) Decrypt(purpose string, encrypted EncryptedSecret) ([]byte, error) {
	if c == nil || c.aead == nil || strings.TrimSpace(purpose) == "" {
		return nil, errSecretDecryption
	}
	if encrypted.version != SecretCipherVersion || encrypted.keyID != c.keyID {
		return nil, errSecretDecryption
	}
	if len(encrypted.nonce) != c.aead.NonceSize() || len(encrypted.ciphertext) < c.aead.Overhead() {
		return nil, errSecretDecryption
	}
	plaintext, err := c.aead.Open(nil, encrypted.nonce, encrypted.ciphertext, secretAAD(encrypted.version, encrypted.keyID, purpose))
	if err != nil {
		return nil, errSecretDecryption
	}
	return plaintext, nil
}

func secretAAD(version uint8, keyID, purpose string) []byte {
	aad := []byte("llmhub/quotaalert/secret")
	aad = append(aad, version)
	for _, part := range []string{keyID, purpose} {
		aad = binary.BigEndian.AppendUint64(aad, uint64(len(part)))
		aad = append(aad, part...)
	}
	return aad
}

// SecretUpdateMode defines explicit write semantics for a stored secret.
type SecretUpdateMode uint8

const (
	SecretPreserve SecretUpdateMode = iota
	SecretReplace
	SecretClear
)

// SecretUpdate carries write-only secret update intent.
type SecretUpdate struct {
	mode        SecretUpdateMode
	replacement []byte
}

// PreserveSecret leaves the stored secret unchanged.
func PreserveSecret() SecretUpdate {
	return SecretUpdate{mode: SecretPreserve}
}

// ReplaceSecret requests encryption and replacement of the stored secret.
func ReplaceSecret(value string) (SecretUpdate, error) {
	if strings.TrimSpace(value) == "" {
		return SecretUpdate{}, fmt.Errorf("replacement quota alert secret is required")
	}
	if len(value) > MaxSecretValueLength {
		return SecretUpdate{}, fmt.Errorf("replacement quota alert secret must not exceed %d bytes", MaxSecretValueLength)
	}
	return SecretUpdate{mode: SecretReplace, replacement: []byte(value)}, nil
}

// ClearSecret removes the stored secret.
func ClearSecret() SecretUpdate {
	return SecretUpdate{mode: SecretClear}
}

// Mode returns the update operation without exposing replacement material.
func (u SecretUpdate) Mode() SecretUpdateMode {
	return u.mode
}

// Apply executes preserve, replace, or clear semantics against the current value.
func (u SecretUpdate) Apply(current *EncryptedSecret, cipher *SecretCipher, purpose string) (*EncryptedSecret, error) {
	switch u.mode {
	case SecretPreserve:
		return cloneEncryptedSecret(current), nil
	case SecretReplace:
		if len(u.replacement) == 0 {
			return nil, fmt.Errorf("replacement quota alert secret is required")
		}
		encrypted, err := cipher.Encrypt(purpose, u.replacement)
		if err != nil {
			return nil, err
		}
		return &encrypted, nil
	case SecretClear:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid quota alert secret update mode")
	}
}

func cloneEncryptedSecret(secret *EncryptedSecret) *EncryptedSecret {
	if secret == nil {
		return nil
	}
	cloned := EncryptedSecret{
		version:    secret.version,
		keyID:      secret.keyID,
		nonce:      slices.Clone(secret.nonce),
		ciphertext: slices.Clone(secret.ciphertext),
	}
	return &cloned
}

// SecretRead is the only secret shape intended for management read responses.
type SecretRead struct {
	Configured bool `json:"configured"`
}

// RedactSecret reports whether a secret exists without exposing encrypted or plaintext material.
func RedactSecret(secret *EncryptedSecret) SecretRead {
	return SecretRead{Configured: secret != nil}
}
