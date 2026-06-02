package logging

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/runtimepolicy"
)

func TestConfigureLogOutputWithPolicy_PostgresDurableModeSkipsLogDirectory(t *testing.T) {
	t.Setenv("WRITABLE_PATH", "")
	t.Setenv("writable_path", "")

	tmpDir := t.TempDir()
	originalWD, errGetwd := os.Getwd()
	if errGetwd != nil {
		t.Fatalf("Getwd: %v", errGetwd)
	}
	if errChdir := os.Chdir(tmpDir); errChdir != nil {
		t.Fatalf("Chdir: %v", errChdir)
	}
	defer func() {
		if errChdirBack := os.Chdir(originalWD); errChdirBack != nil {
			t.Fatalf("restore cwd: %v", errChdirBack)
		}
	}()

	cfg := &config.Config{LoggingToFile: true}
	policy := runtimepolicy.RuntimeStorage{PostgresDurable: true}
	if err := ConfigureLogOutputWithPolicy(cfg, policy); err != nil {
		t.Fatalf("ConfigureLogOutputWithPolicy: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "logs")); !os.IsNotExist(err) {
		t.Fatalf("expected logs directory to stay absent, stat err=%v", err)
	}
}

func TestDisabledRequestLoggerNeverWritesFiles(t *testing.T) {
	logsDir := t.TempDir()
	logger := NewDisabledRequestLogger()

	if err := logger.LogRequest(
		"/v1/chat/completions",
		http.MethodPost,
		map[string][]string{"Content-Type": {"application/json"}},
		[]byte(`{"input":"hello"}`),
		http.StatusBadGateway,
		map[string][]string{"Content-Type": {"application/json"}},
		[]byte(`{"error":"upstream failure"}`),
		nil,
		nil,
		nil,
		nil,
		nil,
		"req-disabled",
		time.Now(),
		time.Now(),
	); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}

	writer, err := logger.LogStreamingRequest(
		"/v1/responses",
		http.MethodPost,
		map[string][]string{"Content-Type": {"application/json"}},
		[]byte(`{"input":"hello"}`),
		"stream-disabled",
	)
	if err != nil {
		t.Fatalf("LogStreamingRequest: %v", err)
	}
	writer.WriteChunkAsync([]byte("data: ok\n\n"))
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, errRead := os.ReadDir(logsDir)
	if errRead != nil {
		t.Fatalf("ReadDir: %v", errRead)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no local files, got %+v", entries)
	}
}
