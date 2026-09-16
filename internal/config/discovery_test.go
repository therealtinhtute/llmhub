package config

import "testing"

// TestParseConfigBytesDiscoveryDefaults mirrors the upstream discovery defaults
// normalization (CLIProxyAPI 3428110d49be): an absent discovery block keeps
// advertising disabled but still yields the default service type and subtype
// list so advertisers never start with empty DNS-SD parameters.
func TestParseConfigBytesDiscoveryDefaults(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte("port: 8317\n"))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if cfg.Discovery.Enabled {
		t.Fatal("discovery should default to disabled")
	}
	if cfg.Discovery.ServiceType != DefaultDiscoveryServiceType {
		t.Fatalf("discovery service type = %q, want %q", cfg.Discovery.ServiceType, DefaultDiscoveryServiceType)
	}
	if len(cfg.Discovery.Subtypes) != 5 {
		t.Fatalf("discovery subtypes = %v, want 5 default protocol subtypes", cfg.Discovery.Subtypes)
	}
}

// TestParseConfigBytesDiscoveryIncludeOverridesDefaultExcludes mirrors upstream
// c1b7c91f2f8a: an explicit include list must not inherit default excludes so
// operators can scan non-default adapters such as docker0.
func TestParseConfigBytesDiscoveryIncludeOverridesDefaultExcludes(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte(`
discovery:
  enabled: true
  interfaces:
    include:
      - docker0
`))
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}
	if len(cfg.Discovery.Interfaces.Exclude) != 0 {
		t.Fatalf("default discovery excludes = %v, want none when include is explicit", cfg.Discovery.Interfaces.Exclude)
	}
	if len(cfg.Discovery.Interfaces.Include) != 1 || cfg.Discovery.Interfaces.Include[0] != "docker0" {
		t.Fatalf("discovery includes = %v", cfg.Discovery.Interfaces.Include)
	}
}
