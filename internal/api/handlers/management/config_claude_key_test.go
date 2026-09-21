package management

// Priority patch coverage for PATCH /v0/management/claude-api-key, ported from
// upstream CLIProxyAPI commit 93b94d22a9bd ("feat(management): support priority
// field in claude key patch"). Local symbol under test: Handler.PatchClaudeKey.
//
// Adaptation: persistence goes through the local ManagementConfigStore
// (recordingConfigStore) rather than a YAML file on disk, so the saved-YAML
// assertion inspects the captured store bytes instead of re-reading the file.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
)

func TestPatchClaudeKeyPriority(t *testing.T) {
	store := &recordingConfigStore{}
	cfg := &config.Config{
		ClaudeKey: []config.ClaudeKey{
			{APIKey: "key-0", Priority: 0},
			{APIKey: "key-1", Priority: 5},
		},
	}
	h := &Handler{cfg: cfg, configFilePath: writeTestConfigFile(t), configStore: store}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/claude-api-key",
		strings.NewReader(`{"index":1,"value":{"priority":20}}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	h.PatchClaudeKey(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := cfg.ClaudeKey[1].Priority; got != 20 {
		t.Fatalf("ClaudeKey[1].Priority = %d, want 20", got)
	}

	savedText := string(store.data)
	if !strings.Contains(savedText, "priority: 20") {
		t.Fatalf("saved YAML missing 'priority: 20':\n%s", savedText)
	}

	// Verify GET returns the updated priority with structured unmarshaling
	getRec := httptest.NewRecorder()
	getCtx, _ := gin.CreateTestContext(getRec)
	getCtx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/claude-api-key", nil)
	h.GetClaudeKeys(getCtx)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200; body=%s", getRec.Code, getRec.Body.String())
	}
	var resp struct {
		ClaudeKey []config.ClaudeKey `json:"claude-api-key"`
	}
	if errJSON := json.Unmarshal(getRec.Body.Bytes(), &resp); errJSON != nil {
		t.Fatalf("unmarshal GET response: %v", errJSON)
	}
	if len(resp.ClaudeKey) != 2 {
		t.Fatalf("GET response ClaudeKey length = %d, want 2", len(resp.ClaudeKey))
	}
	if resp.ClaudeKey[1].Priority != 20 {
		t.Fatalf("GET response ClaudeKey[1].Priority = %d, want 20", resp.ClaudeKey[1].Priority)
	}

	// Omitting priority must preserve existing non-zero value (20)
	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/claude-api-key",
		strings.NewReader(`{"index":1,"value":{"prefix":"team-test"}}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	h.PatchClaudeKey(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := cfg.ClaudeKey[1].Priority; got != 20 {
		t.Fatalf("ClaudeKey[1].Priority = %d, want preserved 20", got)
	}
	if got := cfg.ClaudeKey[1].Prefix; got != "team-test" {
		t.Fatalf("ClaudeKey[1].Prefix = %q, want %q", got, "team-test")
	}

	// Reset priority to 0 explicitly
	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/claude-api-key",
		strings.NewReader(`{"index":1,"value":{"priority":0}}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	h.PatchClaudeKey(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := cfg.ClaudeKey[1].Priority; got != 0 {
		t.Fatalf("ClaudeKey[1].Priority = %d, want 0", got)
	}

	// Invalid priority type returns 400
	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/claude-api-key",
		strings.NewReader(`{"index":1,"value":{"priority":"invalid"}}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	h.PatchClaudeKey(ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}
