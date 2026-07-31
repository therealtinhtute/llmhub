package management

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
)

type runtimeControlsMemoryStore struct {
	settings             runtimecontrol.Settings
	loadErr              error
	saveErr              error
	saveCount            int
	lastExpectedRevision int64
	lastSavedSettings    runtimecontrol.Settings
}

func newRuntimeControlsMemoryStore() *runtimeControlsMemoryStore {
	settings := runtimecontrol.DefaultSettings()
	settings.Revision = 3
	return &runtimeControlsMemoryStore{settings: settings}
}

func (s *runtimeControlsMemoryStore) LoadRuntimeSettings(context.Context) (runtimecontrol.Settings, error) {
	if s.loadErr != nil {
		return runtimecontrol.Settings{}, s.loadErr
	}
	return s.settings, nil
}

func (s *runtimeControlsMemoryStore) SaveRuntimeSettings(_ context.Context, expectedRevision int64, settings runtimecontrol.Settings) (runtimecontrol.Settings, error) {
	if s.saveErr != nil {
		return runtimecontrol.Settings{}, s.saveErr
	}
	s.saveCount++
	s.lastExpectedRevision = expectedRevision
	s.lastSavedSettings = settings
	settings.Revision = expectedRevision + 1
	s.settings = settings
	return settings, nil
}

func TestRuntimeControlsGetReturnsNormalizedSettings(t *testing.T) {
	store := newRuntimeControlsMemoryStore()
	store.settings.CodexLive.MaxSessions = 0
	h := &Handler{runtimeSettingsStore: store}

	rec := performRuntimeControlsRequest(t, h.GetRuntimeControls, http.MethodGet, "/v0/management/runtime-controls", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"max_sessions":32`) {
		t.Fatalf("response did not include normalized max_sessions: %s", rec.Body.String())
	}
}

func TestRuntimeControlsPutSavesWithRevision(t *testing.T) {
	store := newRuntimeControlsMemoryStore()
	h := &Handler{runtimeSettingsStore: store}
	body := `{"revision":3,"credential_routing":{"strategy":"weighted-round-robin","weights":[]},"codex_live":{"enabled":true,"max_sessions":8},"cloaking":{"disable_codex":true},"home":{"enabled":true},"cooldown_persistence_enabled":true}`

	rec := performRuntimeControlsRequest(t, h.PutRuntimeControls, http.MethodPut, "/v0/management/runtime-controls", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if store.saveCount != 1 || store.lastExpectedRevision != 3 {
		t.Fatalf("save count/revision = %d/%d", store.saveCount, store.lastExpectedRevision)
	}
	if store.lastSavedSettings.CredentialRouting.Strategy != runtimecontrol.RoutingWeightedRoundRobin {
		t.Fatalf("strategy = %q", store.lastSavedSettings.CredentialRouting.Strategy)
	}
	if store.lastSavedSettings.CodexLive.MaxSessions != 8 || !store.lastSavedSettings.CodexLive.Enabled {
		t.Fatalf("codex live settings = %#v", store.lastSavedSettings.CodexLive)
	}
}

func TestRuntimeControlsPutRejectsInvalidSettings(t *testing.T) {
	store := newRuntimeControlsMemoryStore()
	h := &Handler{runtimeSettingsStore: store}
	body := `{"revision":3,"credential_routing":{"strategy":"unknown"},"codex_live":{"max_sessions":8}}`

	rec := performRuntimeControlsRequest(t, h.PutRuntimeControls, http.MethodPut, "/v0/management/runtime-controls", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if store.saveCount != 0 {
		t.Fatalf("invalid settings should not be saved, save count=%d", store.saveCount)
	}
}

func TestRuntimeControlsPutMapsRevisionConflict(t *testing.T) {
	store := newRuntimeControlsMemoryStore()
	store.saveErr = runtimecontrol.ErrRevisionConflict
	h := &Handler{runtimeSettingsStore: store}
	body := `{"revision":3,"credential_routing":{"strategy":"round-robin"},"codex_live":{"max_sessions":8}}`

	rec := performRuntimeControlsRequest(t, h.PutRuntimeControls, http.MethodPut, "/v0/management/runtime-controls", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestRuntimeControlsRequireConfiguredStore(t *testing.T) {
	h := &Handler{}
	rec := performRuntimeControlsRequest(t, h.GetRuntimeControls, http.MethodGet, "/v0/management/runtime-controls", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestRuntimeControlsGetMapsLoadError(t *testing.T) {
	h := &Handler{runtimeSettingsStore: &runtimeControlsMemoryStore{loadErr: errors.New("database unavailable")}}
	rec := performRuntimeControlsRequest(t, h.GetRuntimeControls, http.MethodGet, "/v0/management/runtime-controls", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func performRuntimeControlsRequest(t *testing.T, handler gin.HandlerFunc, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	reader := strings.NewReader(body)
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	ctx.Request = req
	handler(ctx)
	return rec
}
