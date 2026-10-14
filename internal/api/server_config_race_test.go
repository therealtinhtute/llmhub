package api

import (
	"sync"
	"testing"

	proxyconfig "github.com/therealtinhtute/llmhub/internal/config"
)

// TestUpdateClientsConcurrentReadsAndWrites hammers UpdateClients while
// readers snapshot the config, exercising the cfgMu-guarded swap and YAML
// snapshot handling under -race.
func TestUpdateClientsConcurrentReadsAndWrites(t *testing.T) {
	s := newTestServer(t)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					if cfg := s.getConfig(); cfg == nil {
						t.Error("getConfig() = nil")
						return
					}
				}
			}
		}()
	}

	authDir := s.getConfig().AuthDir
	for i := 0; i < 50; i++ {
		cfg := &proxyconfig.Config{
			Host:    "127.0.0.1",
			Port:    10000 + i,
			AuthDir: authDir,
			Debug:   i%2 == 0,
		}
		cfg.SDKConfig.APIKeys = []string{"test-key"}
		s.UpdateClients(cfg)
	}
	close(stop)
	wg.Wait()

	if got := s.getConfig(); got == nil || got.Port != 10049 {
		t.Fatalf("getConfig().Port = %v, want 10049", got)
	}
}

// TestUpdateClientsClonesInput ensures later caller-side mutations cannot
// race with the serving snapshot.
func TestUpdateClientsClonesInput(t *testing.T) {
	s := newTestServer(t)
	authDir := s.getConfig().AuthDir

	input := &proxyconfig.Config{Host: "127.0.0.1", Port: 18080, AuthDir: authDir}
	input.SDKConfig.APIKeys = []string{"original"}
	s.UpdateClients(input)

	input.Port = 9999
	input.SDKConfig.APIKeys[0] = "mutated"

	got := s.getConfig()
	if got == nil {
		t.Fatal("getConfig() = nil")
	}
	if got.Port != 18080 {
		t.Fatalf("Port = %d, want 18080 (input mutated after UpdateClients)", got.Port)
	}
	if len(got.APIKeys) != 1 || got.APIKeys[0] != "original" {
		t.Fatalf("APIKeys = %v, want [original]", got.APIKeys)
	}
}

// TestUpdateClientsNilIsNoOp guards management/hook paths that may fire
// without a config.
func TestUpdateClientsNilIsNoOp(t *testing.T) {
	s := newTestServer(t)
	before := s.getConfig()
	s.UpdateClients(nil)
	if after := s.getConfig(); after != before {
		t.Fatal("UpdateClients(nil) replaced the serving config")
	}
}
