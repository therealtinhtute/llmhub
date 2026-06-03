package management

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
)

type recordingConfigStore struct {
	data      []byte
	saveErr   error
	saveCount int
}

func (s *recordingConfigStore) LoadConfigBytes(context.Context) ([]byte, error) {
	return append([]byte(nil), s.data...), nil
}

func (s *recordingConfigStore) SaveConfig(_ context.Context, data []byte) (int64, error) {
	if s.saveErr != nil {
		return 0, s.saveErr
	}
	s.saveCount++
	s.data = append([]byte(nil), data...)
	return int64(s.saveCount), nil
}

func TestPutCodexKeys_UsesConfigStore(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &recordingConfigStore{}
	h := NewHandlerWithoutConfigFilePath(&config.Config{}, nil)
	h.SetConfigStore(store)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(
		http.MethodPut,
		"/v0/management/codex-api-key",
		bytes.NewBufferString(`[{"api-key":"codex-key","base-url":"https://codex.example.com"}]`),
	)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PutCodexKeys(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if store.saveCount != 1 {
		t.Fatalf("expected config store save count 1, got %d", store.saveCount)
	}
	if !bytes.Contains(store.data, []byte("codex-api-key")) {
		t.Fatalf("expected saved config to contain codex-api-key, got %s", string(store.data))
	}
}

func TestPutCodexKeys_ConfigStoreSaveErrorReturnsFailure(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	h := NewHandlerWithoutConfigFilePath(&config.Config{}, nil)
	h.SetConfigStore(&recordingConfigStore{saveErr: errors.New("database unavailable")})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(
		http.MethodPut,
		"/v0/management/codex-api-key",
		bytes.NewBufferString(`[{"api-key":"codex-key","base-url":"https://codex.example.com"}]`),
	)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	h.PutCodexKeys(ctx)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}
