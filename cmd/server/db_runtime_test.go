package main

import (
	"encoding/base64"
	"strings"
	"testing"
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

func TestLegacyRuntimeModeErrorRejectsLegacyEnv(t *testing.T) {
	t.Setenv("HOME_JWT", "legacy-home")

	err := legacyRuntimeModeError()
	if err == nil || !strings.Contains(err.Error(), "HOME_JWT") {
		t.Fatalf("legacyRuntimeModeError() = %v, want HOME_JWT rejection", err)
	}
}
