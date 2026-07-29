package quotaalert

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const testSecretPurpose = "telegram-bot-token"

func TestCipherEncryptDecrypt(t *testing.T) {
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{1}, SecretKeySize))
	plaintext := []byte("123456789:telegram-token")

	encrypted, err := cipher.Encrypt(testSecretPurpose, plaintext)
	if err != nil {
		t.Fatalf("SecretCipher.Encrypt() error = %v", err)
	}
	if encrypted.Version() != SecretCipherVersion || encrypted.KeyID() != "key-1" {
		t.Fatalf("encrypted metadata = version %d, key ID %q", encrypted.Version(), encrypted.KeyID())
	}
	if len(encrypted.Nonce()) != SecretNonceSize || len(encrypted.Ciphertext()) <= len(plaintext) {
		t.Fatalf("encrypted lengths = nonce %d, ciphertext %d", len(encrypted.Nonce()), len(encrypted.Ciphertext()))
	}
	if bytes.Contains(encrypted.Ciphertext(), plaintext) {
		t.Fatal("ciphertext contains plaintext")
	}

	decrypted, err := cipher.Decrypt(testSecretPurpose, encrypted)
	if err != nil {
		t.Fatalf("SecretCipher.Decrypt() error = %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("SecretCipher.Decrypt() = %q, want original plaintext", decrypted)
	}

	nonce := encrypted.Nonce()
	nonce[0] ^= 0xff
	if bytes.Equal(nonce, encrypted.Nonce()) {
		t.Fatal("EncryptedSecret.Nonce() exposed mutable storage")
	}
	ciphertext := encrypted.Ciphertext()
	ciphertext[0] ^= 0xff
	if bytes.Equal(ciphertext, encrypted.Ciphertext()) {
		t.Fatal("EncryptedSecret.Ciphertext() exposed mutable storage")
	}
}

func TestCipherRequiresAES256KeyAndKeyID(t *testing.T) {
	for _, size := range []int{0, 16, 24, 31, 33, 64} {
		if _, err := NewSecretCipher("key-1", make([]byte, size)); err == nil {
			t.Fatalf("NewSecretCipher() error = nil for %d-byte key", size)
		}
	}
	if _, err := NewSecretCipher(" ", make([]byte, SecretKeySize)); err == nil {
		t.Fatal("NewSecretCipher() error = nil for empty key ID")
	}
	if _, err := NewSecretCipher(strings.Repeat("k", MaxSecretKeyIDLength+1), make([]byte, SecretKeySize)); err == nil {
		t.Fatal("NewSecretCipher() error = nil for oversized key ID")
	}
	cipher, err := NewSecretCipher(" key-1 ", make([]byte, SecretKeySize))
	if err != nil {
		t.Fatalf("NewSecretCipher() error = %v", err)
	}
	if cipher.KeyID() != "key-1" {
		t.Fatalf("SecretCipher.KeyID() = %q, want key-1", cipher.KeyID())
	}
}

func TestCipherBoundsSecretMaterial(t *testing.T) {
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{11}, SecretKeySize))
	oversized := strings.Repeat("s", MaxSecretValueLength+1)
	if _, err := cipher.Encrypt(testSecretPurpose, []byte(oversized)); err == nil {
		t.Fatal("SecretCipher.Encrypt() error = nil for oversized secret")
	}
	if _, err := ReplaceSecret(oversized); err == nil {
		t.Fatal("ReplaceSecret() error = nil for oversized secret")
	}
	if _, err := NewEncryptedSecret(
		SecretCipherVersion,
		"key-1",
		make([]byte, SecretNonceSize),
		make([]byte, MaxSecretValueLength+SecretCiphertextOverhead+1),
	); err == nil {
		t.Fatal("NewEncryptedSecret() error = nil for oversized payload")
	}
}

func TestCipherUsesFreshNoncePerWrite(t *testing.T) {
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{2}, SecretKeySize))
	plaintext := []byte("same-token")

	first, err := cipher.Encrypt(testSecretPurpose, plaintext)
	if err != nil {
		t.Fatalf("first SecretCipher.Encrypt() error = %v", err)
	}
	second, err := cipher.Encrypt(testSecretPurpose, plaintext)
	if err != nil {
		t.Fatalf("second SecretCipher.Encrypt() error = %v", err)
	}
	if bytes.Equal(first.Nonce(), second.Nonce()) {
		t.Fatal("SecretCipher.Encrypt() reused a nonce")
	}
	if bytes.Equal(first.Ciphertext(), second.Ciphertext()) {
		t.Fatal("SecretCipher.Encrypt() produced deterministic ciphertext")
	}
}

func TestCipherRejectsTampering(t *testing.T) {
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{3}, SecretKeySize))
	encrypted := encryptTestSecret(t, cipher, testSecretPurpose, "sensitive-token")

	for _, test := range []struct {
		name   string
		mutate func(*EncryptedSecret)
	}{
		{
			name: "ciphertext",
			mutate: func(secret *EncryptedSecret) {
				secret.ciphertext[len(secret.ciphertext)-1] ^= 0xff
			},
		},
		{
			name: "nonce",
			mutate: func(secret *EncryptedSecret) {
				secret.nonce[0] ^= 0xff
			},
		},
		{
			name: "version",
			mutate: func(secret *EncryptedSecret) {
				secret.version++
			},
		},
		{
			name: "key ID",
			mutate: func(secret *EncryptedSecret) {
				secret.keyID = "key-2"
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tampered := cloneEncryptedSecret(&encrypted)
			test.mutate(tampered)
			if _, err := cipher.Decrypt(testSecretPurpose, *tampered); err == nil {
				t.Fatal("SecretCipher.Decrypt() error = nil for tampered secret")
			}
		})
	}
}

func TestCipherRejectsWrongKey(t *testing.T) {
	first := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{4}, SecretKeySize))
	second := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{5}, SecretKeySize))
	encrypted := encryptTestSecret(t, first, testSecretPurpose, "sensitive-token")

	if _, err := second.Decrypt(testSecretPurpose, encrypted); err == nil {
		t.Fatal("SecretCipher.Decrypt() error = nil for wrong key")
	}
}

func TestCipherBindsAuthenticatedPurpose(t *testing.T) {
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{6}, SecretKeySize))
	encrypted := encryptTestSecret(t, cipher, testSecretPurpose, "sensitive-token")

	if _, err := cipher.Decrypt("different-purpose", encrypted); err == nil {
		t.Fatal("SecretCipher.Decrypt() error = nil for wrong purpose")
	}
	if _, err := cipher.Decrypt("", encrypted); err == nil {
		t.Fatal("SecretCipher.Decrypt() error = nil for empty purpose")
	}
}

func TestCipherStorageConstructorValidation(t *testing.T) {
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{7}, SecretKeySize))
	encrypted := encryptTestSecret(t, cipher, testSecretPurpose, "sensitive-token")

	restored, err := NewEncryptedSecret(
		encrypted.Version(),
		encrypted.KeyID(),
		encrypted.Nonce(),
		encrypted.Ciphertext(),
	)
	if err != nil {
		t.Fatalf("NewEncryptedSecret() error = %v", err)
	}
	if _, err := cipher.Decrypt(testSecretPurpose, restored); err != nil {
		t.Fatalf("SecretCipher.Decrypt(restored) error = %v", err)
	}

	for _, test := range []struct {
		name       string
		version    uint8
		keyID      string
		nonce      []byte
		ciphertext []byte
	}{
		{name: "unsupported version", version: SecretCipherVersion + 1, keyID: "key-1", nonce: encrypted.Nonce(), ciphertext: encrypted.Ciphertext()},
		{name: "empty key ID", version: SecretCipherVersion, nonce: encrypted.Nonce(), ciphertext: encrypted.Ciphertext()},
		{name: "invalid nonce", version: SecretCipherVersion, keyID: "key-1", nonce: make([]byte, SecretNonceSize-1), ciphertext: encrypted.Ciphertext()},
		{name: "invalid ciphertext", version: SecretCipherVersion, keyID: "key-1", nonce: encrypted.Nonce(), ciphertext: make([]byte, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewEncryptedSecret(test.version, test.keyID, test.nonce, test.ciphertext); err == nil {
				t.Fatal("NewEncryptedSecret() error = nil")
			}
		})
	}
}

func TestCipherErrorsDoNotExposeTokenMaterial(t *testing.T) {
	const token = "do-not-leak-this-token"
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{8}, SecretKeySize))
	encrypted := encryptTestSecret(t, cipher, testSecretPurpose, token)
	encrypted.ciphertext[0] ^= 0xff

	_, err := cipher.Decrypt(testSecretPurpose, encrypted)
	if err == nil {
		t.Fatal("SecretCipher.Decrypt() error = nil")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("SecretCipher.Decrypt() error exposed token material: %v", err)
	}

	if _, err := ReplaceSecret(" "); err == nil {
		t.Fatal("ReplaceSecret() error = nil for empty replacement")
	} else if strings.Contains(err.Error(), token) {
		t.Fatalf("ReplaceSecret() error exposed token material: %v", err)
	}
}

func TestSecretUpdateSemantics(t *testing.T) {
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{9}, SecretKeySize))
	current := encryptTestSecret(t, cipher, testSecretPurpose, "current-token")

	preserved, err := (SecretUpdate{}).Apply(&current, nil, "")
	if err != nil {
		t.Fatalf("zero-value SecretUpdate.Apply() error = %v", err)
	}
	if preserved == nil || preserved == &current {
		t.Fatal("preserve did not return an independent stored value")
	}
	if !bytes.Equal(preserved.Ciphertext(), current.Ciphertext()) {
		t.Fatal("preserve changed the stored secret")
	}
	preserved.ciphertext[0] ^= 0xff
	if bytes.Equal(preserved.Ciphertext(), current.Ciphertext()) {
		t.Fatal("preserve reused mutable encrypted storage")
	}
	if PreserveSecret().Mode() != SecretPreserve {
		t.Fatal("PreserveSecret().Mode() is not SecretPreserve")
	}

	replace, err := ReplaceSecret("replacement-token")
	if err != nil {
		t.Fatalf("ReplaceSecret() error = %v", err)
	}
	if replace.Mode() != SecretReplace {
		t.Fatal("ReplaceSecret().Mode() is not SecretReplace")
	}
	replaced, err := replace.Apply(&current, cipher, testSecretPurpose)
	if err != nil {
		t.Fatalf("replace SecretUpdate.Apply() error = %v", err)
	}
	plaintext, err := cipher.Decrypt(testSecretPurpose, *replaced)
	if err != nil {
		t.Fatalf("SecretCipher.Decrypt(replaced) error = %v", err)
	}
	if string(plaintext) != "replacement-token" {
		t.Fatalf("replacement plaintext = %q", plaintext)
	}
	if bytes.Equal(replaced.Ciphertext(), current.Ciphertext()) {
		t.Fatal("replace retained the previous ciphertext")
	}

	if ClearSecret().Mode() != SecretClear {
		t.Fatal("ClearSecret().Mode() is not SecretClear")
	}
	cleared, err := ClearSecret().Apply(&current, nil, "")
	if err != nil {
		t.Fatalf("clear SecretUpdate.Apply() error = %v", err)
	}
	if cleared != nil {
		t.Fatal("clear retained a stored secret")
	}

	invalid := SecretUpdate{mode: SecretUpdateMode(255)}
	if _, err := invalid.Apply(&current, cipher, testSecretPurpose); err == nil {
		t.Fatal("invalid SecretUpdate.Apply() error = nil")
	}
}

func TestSecretReadIsRedacted(t *testing.T) {
	cipher := newTestSecretCipher(t, "key-1", bytes.Repeat([]byte{10}, SecretKeySize))
	encrypted := encryptTestSecret(t, cipher, testSecretPurpose, "plaintext-token")

	readJSON, err := json.Marshal(RedactSecret(&encrypted))
	if err != nil {
		t.Fatalf("json.Marshal(RedactSecret()) error = %v", err)
	}
	if string(readJSON) != `{"configured":true}` {
		t.Fatalf("redacted JSON = %s", readJSON)
	}
	for _, forbidden := range [][]byte{
		[]byte("plaintext-token"),
		encrypted.Nonce(),
		encrypted.Ciphertext(),
		[]byte(encrypted.KeyID()),
	} {
		if len(forbidden) > 0 && bytes.Contains(readJSON, forbidden) {
			t.Fatalf("redacted JSON exposed secret storage material: %s", readJSON)
		}
	}

	emptyJSON, err := json.Marshal(RedactSecret(nil))
	if err != nil {
		t.Fatalf("json.Marshal(RedactSecret(nil)) error = %v", err)
	}
	if string(emptyJSON) != `{"configured":false}` {
		t.Fatalf("empty redacted JSON = %s", emptyJSON)
	}

	encryptedJSON, err := json.Marshal(encrypted)
	if err != nil {
		t.Fatalf("json.Marshal(EncryptedSecret) error = %v", err)
	}
	if string(encryptedJSON) != `{}` {
		t.Fatalf("EncryptedSecret serialized storage material: %s", encryptedJSON)
	}
	update, err := ReplaceSecret("plaintext-token")
	if err != nil {
		t.Fatalf("ReplaceSecret() error = %v", err)
	}
	updateJSON, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("json.Marshal(SecretUpdate) error = %v", err)
	}
	if string(updateJSON) != `{}` {
		t.Fatalf("SecretUpdate serialized replacement material: %s", updateJSON)
	}
}

func newTestSecretCipher(t *testing.T, keyID string, key []byte) *SecretCipher {
	t.Helper()
	cipher, err := NewSecretCipher(keyID, key)
	if err != nil {
		t.Fatalf("NewSecretCipher() error = %v", err)
	}
	return cipher
}

func encryptTestSecret(t *testing.T, cipher *SecretCipher, purpose, plaintext string) EncryptedSecret {
	t.Helper()
	encrypted, err := cipher.Encrypt(purpose, []byte(plaintext))
	if err != nil {
		t.Fatalf("SecretCipher.Encrypt() error = %v", err)
	}
	return encrypted
}
