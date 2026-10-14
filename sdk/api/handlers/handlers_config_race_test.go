package handlers

import (
	"sync"
	"testing"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
)

// TestBaseAPIHandlerConcurrentConfigSwap exercises the runtimeMu-guarded
// Cfg swap against concurrent snapshot readers under -race.
func TestBaseAPIHandlerConcurrentConfigSwap(t *testing.T) {
	h := NewBaseAPIHandlers(&internalconfig.SDKConfig{RequestLog: true}, nil)

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
					if cfg := h.CurrentConfig(); cfg == nil {
						t.Error("CurrentConfig() = nil")
						return
					}
					_ = PassthroughHeadersEnabled(h.CurrentConfig())
					_ = StreamingBootstrapRetries(h.CurrentConfig())
				}
			}
		}()
	}

	for i := 0; i < 100; i++ {
		h.UpdateClients(&internalconfig.SDKConfig{RequestLog: i%2 == 0})
	}
	close(stop)
	wg.Wait()

	if h.CurrentConfig() == nil {
		t.Fatal("CurrentConfig() = nil after swaps")
	}
}
