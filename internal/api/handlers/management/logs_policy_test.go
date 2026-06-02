package management

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/runtimepolicy"
)

func TestRuntimeStoragePolicyDisablesLogSurfaces(t *testing.T) {
	h := &Handler{
		cfg: &config.Config{
			LoggingToFile: true,
			SDKConfig: config.SDKConfig{
				RequestLog: true,
			},
		},
		runtimeStoragePolicy: runtimepolicy.RuntimeStorage{PostgresDurable: true},
	}

	if h.fileLoggingEnabled() {
		t.Fatal("expected file logging to be disabled in postgres durable mode")
	}
	if h.requestLogArchiveEnabled() {
		t.Fatal("expected request log archives to be disabled in postgres durable mode")
	}
	if h.requestErrorLogEnabled() {
		t.Fatal("expected request error log archives to be disabled in postgres durable mode")
	}
	if dir := h.logDirectory(); dir != "" {
		t.Fatalf("expected empty log directory in postgres durable mode, got %q", dir)
	}
}
