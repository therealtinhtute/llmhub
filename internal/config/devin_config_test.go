package config

import (
	"reflect"
	"testing"
)

// TestParseConfigBytesDevinSensitiveWords mirrors the upstream Devin
// sensitive-words surface (CLIProxyAPI 5b8e3821b1fe, d115fe2c450f,
// c0b76c2d0991): the word list is strictly external in config.yaml under
// devin.sensitive-words, parsed into Config.Devin.SensitiveWords.
func TestParseConfigBytesDevinSensitiveWords(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`
devin:
  sensitive-words:
    - sample-word-1
    - sample-word-2
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	want := []string{"sample-word-1", "sample-word-2"}
	if !reflect.DeepEqual(cfg.Devin.SensitiveWords, want) {
		t.Fatalf("Devin.SensitiveWords = %#v, want %#v", cfg.Devin.SensitiveWords, want)
	}
}

// TestParseConfigBytesDevinSensitiveWordsAbsent confirms an absent devin block
// leaves the word list empty so the executor cloak stays dormant.
func TestParseConfigBytesDevinSensitiveWordsAbsent(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte("port: 8317\n"))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if len(cfg.Devin.SensitiveWords) != 0 {
		t.Fatalf("Devin.SensitiveWords = %#v, want empty", cfg.Devin.SensitiveWords)
	}
}
