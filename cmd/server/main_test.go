package main

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
)

func TestResolveManagementBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		flagURL string
		cfg     *config.Config
		want    string
	}{
		{
			name:    "flag takes highest precedence",
			flagURL: "https://flag.example.com",
			cfg: &config.Config{
				RemoteManagement: config.RemoteManagement{
					BaseURL: "https://cfg.example.com",
				},
				Port: 9000,
			},
			want: "https://flag.example.com",
		},
		{
			name:    "config base url used when flag is empty",
			flagURL: "",
			cfg: &config.Config{
				RemoteManagement: config.RemoteManagement{
					BaseURL: "https://cfg.example.com",
				},
				Port: 9000,
			},
			want: "https://cfg.example.com",
		},
		{
			name:    "config port used when neither flag nor base url provided",
			flagURL: "",
			cfg: &config.Config{
				Port: 9090,
			},
			want: "http://127.0.0.1:9090",
		},
		{
			name:    "nil config falls back to default port 8317",
			flagURL: "",
			cfg:     nil,
			want:    "http://127.0.0.1:8317",
		},
		{
			name:    "zero port falls back to default port 8317",
			flagURL: "",
			cfg: &config.Config{
				Port: 0,
			},
			want: "http://127.0.0.1:8317",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveManagementBaseURL(tt.flagURL, tt.cfg)
			if got != tt.want {
				t.Fatalf("resolveManagementBaseURL(%q, %v) = %q, want %q", tt.flagURL, tt.cfg, got, tt.want)
			}
		})
	}
}
