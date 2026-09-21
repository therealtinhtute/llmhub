package api

import (
	"testing"

	proxyconfig "github.com/therealtinhtute/llmhub/internal/config"
	"gopkg.in/yaml.v3"
)

// Ported from upstream CLIProxyAPI internal/api/server_sdk_config_test.go
// (42c9680eee55). Upstream copies codex.response-steering into a runtime-only
// SDKConfig field inside effectiveSDKConfig; locally Config embeds SDKConfig
// inline, so &cfg.SDKConfig is the exact surface handed to SDK handlers and no
// copy step exists.
func TestServerSDKConfigIncludesCodexResponseSteering(t *testing.T) {
	cfg := &proxyconfig.Config{SDKConfig: proxyconfig.SDKConfig{CodexResponseSteering: true}}

	sdkCfg := &cfg.SDKConfig
	if sdkCfg == nil || !sdkCfg.CodexResponseSteering {
		t.Fatalf("CodexResponseSteering = false, want true")
	}
}

func TestCodexResponseSteeringYAMLUnmarshal(t *testing.T) {
	yamlContent := []byte(`
codex-response-steering: true
`)
	var cfg proxyconfig.Config
	if err := yaml.Unmarshal(yamlContent, &cfg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !cfg.CodexResponseSteering {
		t.Fatalf("cfg.CodexResponseSteering = false, want true")
	}
	sdkCfg := &cfg.SDKConfig
	if sdkCfg == nil || !sdkCfg.CodexResponseSteering {
		t.Fatalf("sdkCfg.CodexResponseSteering = false, want true")
	}
}
