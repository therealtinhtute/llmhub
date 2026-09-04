package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/quotaalert"
)

func TestLoadInitConfigBytesFromEnvYAML(t *testing.T) {
	t.Setenv("LLMHUB_INIT_CONFIG_B64", "")
	t.Setenv("LLMHUB_INIT_CONFIG_YAML", "host: 0.0.0.0\nport: 9090\n")

	data, err := loadInitConfigBytesFromEnv()
	if err != nil {
		t.Fatalf("loadInitConfigBytesFromEnv() error = %v", err)
	}
	if got := string(data); got != "host: 0.0.0.0\nport: 9090" {
		t.Fatalf("config yaml = %q", got)
	}
}

func TestLoadInitConfigBytesFromEnvBase64TakesPriority(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("host: 127.0.0.1\nport: 8317\n"))
	t.Setenv("LLMHUB_INIT_CONFIG_B64", encoded)
	t.Setenv("LLMHUB_INIT_CONFIG_YAML", "host: 0.0.0.0\nport: 9090\n")

	data, err := loadInitConfigBytesFromEnv()
	if err != nil {
		t.Fatalf("loadInitConfigBytesFromEnv() error = %v", err)
	}
	if got := string(data); got != "host: 127.0.0.1\nport: 8317\n" {
		t.Fatalf("config yaml = %q", got)
	}
}

func TestLoadInitConfigBytesForVersionSkipsWhenAlreadySeeded(t *testing.T) {
	data, err := loadInitConfigBytesForVersion(3)
	if err != nil {
		t.Fatalf("loadInitConfigBytesForVersion() error = %v", err)
	}
	if data != nil {
		t.Fatalf("loadInitConfigBytesForVersion() data = %q, want nil", string(data))
	}
}

func TestLoadInitConfigBytesForVersionRequiresEnvWhenEmpty(t *testing.T) {
	t.Setenv("LLMHUB_INIT_CONFIG_B64", "")
	t.Setenv("LLMHUB_INIT_CONFIG_YAML", "")

	_, err := loadInitConfigBytesForVersion(0)
	if err == nil || !strings.Contains(err.Error(), "missing LLMHUB_INIT_CONFIG") {
		t.Fatalf("loadInitConfigBytesForVersion() error = %v, want missing init config", err)
	}
}

func TestLegacyRuntimeModeErrorRejectsLegacyEnv(t *testing.T) {
	t.Setenv("HOME_JWT", "legacy-home")

	err := legacyRuntimeModeError()
	if err == nil || !strings.Contains(err.Error(), "HOME_JWT") {
		t.Fatalf("legacyRuntimeModeError() = %v, want HOME_JWT rejection", err)
	}
}

func TestQuotaSecretKeyFromEnv(t *testing.T) {
	t.Setenv("LLMHUB_QUOTA_SECRET_KEY_B64", base64.StdEncoding.EncodeToString(make([]byte, quotaalert.SecretKeySize)))
	t.Setenv("LLMHUB_QUOTA_SECRET_KEY_ID", " runtime-key ")
	cipher, err := loadQuotaSecretCipherFromEnv()
	if err != nil {
		t.Fatalf("loadQuotaSecretCipherFromEnv() error = %v", err)
	}
	if cipher == nil || cipher.KeyID() != "runtime-key" {
		t.Fatalf("cipher key ID = %q", cipher.KeyID())
	}
}

func TestQuotaSecretKeyFromEnvAllowsMissingKey(t *testing.T) {
	cipher, err := loadQuotaSecretCipherFromEnv()
	if err != nil {
		t.Fatalf("loadQuotaSecretCipherFromEnv() error = %v", err)
	}
	if cipher != nil {
		t.Fatalf("cipher = %#v, want nil", cipher)
	}
}

func TestQuotaSecretKeyFromEnvRejectsInvalidKey(t *testing.T) {
	for _, value := range []string{"not-base64", base64.StdEncoding.EncodeToString(make([]byte, quotaalert.SecretKeySize-1))} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("LLMHUB_QUOTA_SECRET_KEY_B64", value)
			_, err := loadQuotaSecretCipherFromEnv()
			if err == nil {
				t.Fatal("loadQuotaSecretCipherFromEnv() error = nil")
			}
		})
	}
}

func TestQuotaSecretKeyFromEnvDefaultKeyID(t *testing.T) {
	t.Setenv("LLMHUB_QUOTA_SECRET_KEY_B64", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", quotaalert.SecretKeySize))))
	cipher, err := loadQuotaSecretCipherFromEnv()
	if err != nil {
		t.Fatalf("loadQuotaSecretCipherFromEnv() error = %v", err)
	}
	if cipher.KeyID() != defaultQuotaSecretKeyID {
		t.Fatalf("key ID = %q, want %q", cipher.KeyID(), defaultQuotaSecretKeyID)
	}
}

func TestQuotaSecretKeyFromEnvWithWrappingQuotes(t *testing.T) {
	rawKey := base64.StdEncoding.EncodeToString(make([]byte, quotaalert.SecretKeySize))
	for _, tc := range []struct {
		name  string
		key   string
		keyID string
		want  string
	}{
		{
			name:  "double quotes",
			key:   `"` + rawKey + `"`,
			keyID: `"custom-id"`,
			want:  "custom-id",
		},
		{
			name:  "single quotes",
			key:   `'` + rawKey + `'`,
			keyID: `'single-id'`,
			want:  "single-id",
		},
		{
			name:  "whitespace around quotes",
			key:   ` "` + rawKey + `" `,
			keyID: ` "space-id" `,
			want:  "space-id",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LLMHUB_QUOTA_SECRET_KEY_B64", tc.key)
			t.Setenv("LLMHUB_QUOTA_SECRET_KEY_ID", tc.keyID)
			cipher, err := loadQuotaSecretCipherFromEnv()
			if err != nil {
				t.Fatalf("loadQuotaSecretCipherFromEnv() error = %v", err)
			}
			if cipher == nil || cipher.KeyID() != tc.want {
				t.Fatalf("cipher = %#v, want key ID %q", cipher, tc.want)
			}
		})
	}
}

func TestAutoLoadDotEnv(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte("TEST_AUTOLOAD_VAR=loaded_value\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Chdir(tempDir)

	t.Run("skip", func(t *testing.T) {
		t.Setenv("LLMHUB_SKIP_DOTENV", "1")
		t.Setenv("TEST_AUTOLOAD_VAR", "")
		autoLoadDotEnv()
		if val := os.Getenv("TEST_AUTOLOAD_VAR"); val != "" {
			t.Fatalf("TEST_AUTOLOAD_VAR = %q, want empty (skipped)", val)
		}
	})
	t.Run("autoloaded", func(t *testing.T) {
		t.Setenv("LLMHUB_SKIP_DOTENV", "")
		_ = os.Unsetenv("TEST_AUTOLOAD_VAR")
		autoLoadDotEnv()
		if val := os.Getenv("TEST_AUTOLOAD_VAR"); val != "loaded_value" {
			t.Fatalf("TEST_AUTOLOAD_VAR = %q, want loaded_value", val)
		}
	})
}
