package cliproxy

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/runtimepolicy"
)

func TestRequiresLocalAuthDirDisabledInPostgresDurableMode(t *testing.T) {
	service := &Service{
		runtimeStoragePolicy: runtimepolicy.RuntimeStorage{PostgresDurable: true},
	}
	if service.requiresLocalAuthDir() {
		t.Fatal("expected postgres durable mode to skip local auth directory requirement")
	}
}
