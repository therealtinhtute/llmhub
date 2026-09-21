package management

// Pagination coverage for GET /v0/management/auth-files, ported from upstream
// CLIProxyAPI commit 76ac75e68ae6 ("feat(management): paginate auth file
// listings"). Local symbols under test: Handler.ListAuthFiles,
// Handler.listAuthFilesFromDisk, parseAuthFilesPagination,
// authFilesPagination.bounds, authFilesListResponse, compareAuthFileListOrder,
// isAuthFileListable.
//
// Adaptations for the diverged local handler:
//   - Local ListAuthFiles returns 503 when the auth manager is unavailable
//     instead of the upstream disk fallback, so the disk-pagination case calls
//     listAuthFilesFromDisk directly (mirroring auth_files_cooldown_test.go).
//   - Local EnsureIndex assigns a stable hash (upstream honors a preset Index),
//     so no assertion depends on auth_index values.
//   - The disk listing is ordered by os.ReadDir (lexical filename order), which
//     matches the upstream expectations for page contents.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

type authFilesPaginationPayload struct {
	Files    []map[string]any `json:"files"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	HasMore  bool             `json:"has_more"`
}

func TestListAuthFilesPaginationFromManager(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")

	authDir := t.TempDir()
	manager := coreauth.NewManager(nil, nil, nil)
	registerPaginatedAuthFiles(t, authDir, manager)
	handler := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, manager)

	payload := requestAuthFilesPage(t, handler, "/v0/management/auth-files?page=2&page_size=2")
	if payload.Total != 5 || payload.Page != 2 || payload.PageSize != 2 || !payload.HasMore {
		t.Fatalf("pagination metadata = %#v", payload)
	}
	if got := authFileNames(payload.Files); !equalStrings(got, []string{"Charlie.json", "delta.json"}) {
		t.Fatalf("page names = %#v, want Charlie.json and delta.json", got)
	}

	payload = requestAuthFilesPage(t, handler, "/v0/management/auth-files?page=99&page_size=2")
	if payload.Total != 5 || payload.Page != 99 || payload.PageSize != 2 || payload.HasMore || len(payload.Files) != 0 {
		t.Fatalf("out-of-range page = %#v", payload)
	}
}

func TestListAuthFilesPaginationFromDisk(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")

	authDir := t.TempDir()
	registerPaginatedAuthFiles(t, authDir, nil)
	handler := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, nil)

	// Local divergence: ListAuthFiles returns 503 without an auth manager, so the
	// disk fallback is exercised directly with the parsed pagination.
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/auth-files?page_size=2", nil)
	pagination, errPagination := parseAuthFilesPagination(ctx)
	if errPagination != nil {
		t.Fatalf("parseAuthFilesPagination: %v", errPagination)
	}
	handler.listAuthFilesFromDisk(ctx, pagination)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload authFilesPaginationPayload
	if errDecode := json.Unmarshal(recorder.Body.Bytes(), &payload); errDecode != nil {
		t.Fatalf("decode response: %v", errDecode)
	}
	if payload.Total != 5 || payload.Page != 1 || payload.PageSize != 2 || !payload.HasMore {
		t.Fatalf("pagination metadata = %#v", payload)
	}
	if got := authFileNames(payload.Files); !equalStrings(got, []string{"Alpha.json", "Charlie.json"}) {
		t.Fatalf("page names = %#v, want Alpha.json and Charlie.json", got)
	}
}

func TestListAuthFilesPaginationAppliesLookupFiltersBeforePaging(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")

	authDir := t.TempDir()
	fileName := "shared.json"
	filePath := filepath.Join(authDir, fileName)
	if errWrite := os.WriteFile(filePath, []byte(`{"type":"codex"}`), 0o600); errWrite != nil {
		t.Fatalf("write shared auth file: %v", errWrite)
	}
	manager := coreauth.NewManager(nil, nil, nil)
	for _, id := range []string{"auth-b", "auth-a"} {
		auth := &coreauth.Auth{
			ID:       id,
			FileName: fileName,
			Provider: "codex",
			Status:   coreauth.StatusActive,
			Attributes: map[string]string{
				"path": filePath,
			},
		}
		if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
			t.Fatalf("register %s: %v", id, errRegister)
		}
	}
	handler := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, manager)

	payload := requestAuthFilesPage(t, handler, "/v0/management/auth-files?name=shared.json&page=2&page_size=1")
	if payload.Total != 2 || payload.Page != 2 || payload.PageSize != 1 || payload.HasMore || len(payload.Files) != 1 {
		t.Fatalf("filtered pagination = %#v", payload)
	}
	if got := payload.Files[0]["id"]; got != "auth-b" {
		t.Fatalf("filtered page id = %#v, want auth-b", got)
	}
}

func TestListAuthFilesPaginationDefaultsAndCompatibility(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")

	authDir := t.TempDir()
	manager := coreauth.NewManager(nil, nil, nil)
	registerPaginatedAuthFiles(t, authDir, manager)
	handler := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, manager)

	payload := requestAuthFilesPage(t, handler, "/v0/management/auth-files?page=1")
	if payload.Total != 5 || payload.Page != 1 || payload.PageSize != defaultAuthFilesPageSize || payload.HasMore || len(payload.Files) != 5 {
		t.Fatalf("default pagination = %#v", payload)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/auth-files", nil)
	handler.ListAuthFiles(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var unpaginated map[string]json.RawMessage
	if errDecode := json.Unmarshal(recorder.Body.Bytes(), &unpaginated); errDecode != nil {
		t.Fatalf("decode unpaginated response: %v", errDecode)
	}
	for _, field := range []string{"total", "page", "page_size", "has_more"} {
		if _, exists := unpaginated[field]; exists {
			t.Fatalf("unpaginated response unexpectedly contains %q: %s", field, recorder.Body.String())
		}
	}
}

func TestListAuthFilesPaginationRejectsInvalidValues(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	handler := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)

	for _, requestPath := range []string{
		"/v0/management/auth-files?page=0&page_size=10",
		"/v0/management/auth-files?page=invalid&page_size=10",
		"/v0/management/auth-files?page=1&page_size=0",
		"/v0/management/auth-files?page=1&page_size=invalid",
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, requestPath, nil)

		handler.ListAuthFiles(ctx)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s status = %d, want %d body=%s", requestPath, recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	}
}

func registerPaginatedAuthFiles(t *testing.T, authDir string, manager *coreauth.Manager) {
	t.Helper()
	for index, name := range []string{"delta.json", "Alpha.json", "echo.json", "Charlie.json", "bravo.json"} {
		path := filepath.Join(authDir, name)
		if errWrite := os.WriteFile(path, []byte(`{"type":"codex","email":"user@example.com"}`), 0o600); errWrite != nil {
			t.Fatalf("write auth file %s: %v", name, errWrite)
		}
		if manager == nil {
			continue
		}
		auth := &coreauth.Auth{
			ID:       name,
			FileName: name,
			Provider: "codex",
			Status:   coreauth.StatusActive,
			Attributes: map[string]string{
				"path": path,
			},
			Metadata: map[string]any{"ordinal": index},
		}
		if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
			t.Fatalf("register auth %s: %v", name, errRegister)
		}
	}
}

func requestAuthFilesPage(t *testing.T, handler *Handler, requestPath string) authFilesPaginationPayload {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, requestPath, nil)

	handler.ListAuthFiles(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want %d body=%s", requestPath, recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload authFilesPaginationPayload
	if errDecode := json.Unmarshal(recorder.Body.Bytes(), &payload); errDecode != nil {
		t.Fatalf("decode %s response: %v", requestPath, errDecode)
	}
	return payload
}

func authFileNames(files []map[string]any) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		name, _ := file["name"].(string)
		names = append(names, name)
	}
	return names
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
