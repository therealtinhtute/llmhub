package config

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCodexModelSupportConfigurationUpdate(t *testing.T) {
	const yamlConfig = `codex-api-key:
  - models:
      - name: custom-one
        support-configuration-update: true
      - name: custom-two
`
	const jsonConfig = `{"codex-api-key":[{"models":[{"name":"custom-one","support-configuration-update":true},{"name":"custom-two"}]}]}`

	for _, testCase := range []struct {
		name   string
		decode func(*Config) error
	}{
		{
			name: "YAML",
			decode: func(cfg *Config) error {
				return yaml.Unmarshal([]byte(yamlConfig), cfg)
			},
		},
		{
			name: "JSON",
			decode: func(cfg *Config) error {
				return json.Unmarshal([]byte(jsonConfig), cfg)
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var cfg Config
			if errDecode := testCase.decode(&cfg); errDecode != nil {
				t.Fatalf("decode error: %v", errDecode)
			}
			if len(cfg.CodexKey) != 1 || len(cfg.CodexKey[0].Models) != 2 {
				t.Fatalf("unexpected codex-api-key models: %+v", cfg.CodexKey)
			}
			if !cfg.CodexKey[0].Models[0].SupportConfigurationUpdate || cfg.CodexKey[0].Models[1].SupportConfigurationUpdate {
				t.Fatalf("unexpected configuration update flags: %+v", cfg.CodexKey[0].Models)
			}
		})
	}
}
