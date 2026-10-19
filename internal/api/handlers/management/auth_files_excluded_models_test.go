package management

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

func TestUploadAuthFileRejectsInvalidExcludedModels(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "integer value",
			content: `{"type":"codex","email":"bad@example.com","excluded_models":123}`,
		},
		{
			name:    "boolean in array",
			content: `{"type":"codex","email":"bad@example.com","excluded_models":["valid",false]}`,
		},
		{
			name:    "object value",
			content: `{"type":"codex","email":"bad@example.com","excluded_models":{"nested":"model"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &pathlessMemoryAuthStore{}
			manager := coreauth.NewManager(store, nil, nil)
			h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)
			h.tokenStore = store

			// 1. Raw upload with query param
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v0/management/auth-files?name=bad.json", strings.NewReader(tt.content))
			c.Request.Header.Set("Content-Type", "application/json")
			h.UploadAuthFile(c)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("raw upload expected status 400, got %d: %s", rec.Code, rec.Body.String())
			}

			// 2. Multipart upload
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, errPart := writer.CreateFormFile("file", "bad.json")
			if errPart != nil {
				t.Fatalf("failed to create form file: %v", errPart)
			}
			if _, errWrite := part.Write([]byte(tt.content)); errWrite != nil {
				t.Fatalf("failed to write part: %v", errWrite)
			}
			writer.Close()

			recMulti := httptest.NewRecorder()
			cMulti, _ := gin.CreateTestContext(recMulti)
			cMulti.Request = httptest.NewRequest(http.MethodPost, "/v0/management/auth-files", body)
			cMulti.Request.Header.Set("Content-Type", writer.FormDataContentType())
			h.UploadAuthFile(cMulti)

			if recMulti.Code != http.StatusBadRequest {
				t.Fatalf("multipart upload expected status 400, got %d: %s", recMulti.Code, recMulti.Body.String())
			}
		})
	}
}

func TestPatchAuthFileFieldsExcludedModels(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	store := &memoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)
	record := &coreauth.Auth{
		ID:       "cred.json",
		FileName: "cred.json",
		Provider: "claude",
		Attributes: map[string]string{
			"auth_kind":       "oauth",
			"excluded_models": "stale-model",
		},
		Metadata: map[string]any{
			"type":            "claude",
			"excluded_models": []any{"stale-model"},
		},
	}
	if _, errRegister := manager.Register(context.Background(), record); errRegister != nil {
		t.Fatalf("failed to register auth: %v", errRegister)
	}

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)

	// Test 1: Reject nested excluded_models
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded_models.sub":"bad"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 on nested excluded_models, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	// Test 2: Reject non-string element in excluded_models
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded_models":["valid", 99]}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 on invalid element, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	// Test 3: Reject config-sourced credentials with 404
	{
		configRecord := &coreauth.Auth{
			ID:       "config-cred",
			FileName: "config-cred",
			Provider: "codex",
			Attributes: map[string]string{
				"source":    "config",
				"auth_kind": "apikey",
			},
		}
		if _, errRegister := manager.Register(context.Background(), configRecord); errRegister != nil {
			t.Fatalf("failed to register config auth: %v", errRegister)
		}

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"config-cred","excluded_models":["m1"]}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 on config-sourced cred, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	// Test 4: Successfully patch excluded_models and verify attributes updated
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded_models":["new-a", "new-b"]}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on valid patch, got %d: %s", rec.Code, rec.Body.String())
		}

		updated, ok := manager.GetByID("cred.json")
		if !ok || updated == nil {
			t.Fatal("auth not found after patch")
		}
		if got := updated.Attributes["excluded_models"]; got != "new-a,new-b" {
			t.Fatalf("expected updated excluded_models=%q, got %q", "new-a,new-b", got)
		}
		if hash := updated.Attributes["excluded_models_hash"]; hash == "" {
			t.Fatal("expected non-empty excluded_models_hash after patch")
		}
	}

	// Test 5: Comma-separated string form persists as canonical array and updates attributes
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded_models":"comma-1, comma-2"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on comma-separated string, got %d: %s", rec.Code, rec.Body.String())
		}

		updated, ok := manager.GetByID("cred.json")
		if !ok || updated == nil {
			t.Fatal("auth not found after patch")
		}
		if got := updated.Attributes["excluded_models"]; got != "comma-1,comma-2" {
			t.Fatalf("expected updated excluded_models=%q, got %q", "comma-1,comma-2", got)
		}
		// Verify metadata stores canonical array
		rawArr, okArr := updated.Metadata["excluded_models"].([]string)
		if !okArr || len(rawArr) != 2 || rawArr[0] != "comma-1" || rawArr[1] != "comma-2" {
			t.Fatalf("expected metadata excluded_models to be []string{\"comma-1\", \"comma-2\"}, got %#v", updated.Metadata["excluded_models"])
		}
	}

	// Test 6: Legacy hyphen alias "excluded-models" updates metadata and attributes
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded-models":["alias-1","alias-2"]}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on alias patch, got %d: %s", rec.Code, rec.Body.String())
		}

		updated, ok := manager.GetByID("cred.json")
		if !ok || updated == nil {
			t.Fatal("auth not found after patch")
		}
		if got := updated.Attributes["excluded_models"]; got != "alias-1,alias-2" {
			t.Fatalf("expected updated excluded_models=%q, got %q", "alias-1,alias-2", got)
		}
		if _, hasHyphen := updated.Metadata["excluded-models"]; hasHyphen {
			t.Fatal("expected legacy excluded-models key to be removed from metadata in favor of canonical excluded_models")
		}
	}

	// Test 7: Legacy hyphen alias "excluded-models" with comma-separated string
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded-models":"alias-str1, alias-str2"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on alias comma-separated string, got %d: %s", rec.Code, rec.Body.String())
		}

		updated, ok := manager.GetByID("cred.json")
		if !ok || updated == nil {
			t.Fatal("auth not found after patch")
		}
		if got := updated.Attributes["excluded_models"]; got != "alias-str1,alias-str2" {
			t.Fatalf("expected updated excluded_models=%q, got %q", "alias-str1,alias-str2", got)
		}
	}

	// Test 8: Legacy hyphen alias rejects nested and invalid types
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded-models.nested":"bad"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 on nested alias, got %d", rec.Code)
		}

		recInvalid := httptest.NewRecorder()
		cInvalid, _ := gin.CreateTestContext(recInvalid)
		cInvalid.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded-models":12345}`))
		cInvalid.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(cInvalid)
		if recInvalid.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 on non-string/non-array alias, got %d", recInvalid.Code)
		}
	}

	// Test 9: Dual spelling with canonical and legacy keys: canonical-wins deterministically regardless of JSON order
	{
		// Subtest A: canonical first, legacy second -> canonical wins
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded_models":["canon-1","canon-2"],"excluded-models":["legacy-1"]}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on dual-spelling patch, got %d: %s", rec.Code, rec.Body.String())
		}
		updated, ok := manager.GetByID("cred.json")
		if !ok || updated == nil {
			t.Fatal("auth not found after patch")
		}
		if got := updated.Attributes["excluded_models"]; got != "canon-1,canon-2" {
			t.Fatalf("expected canonical to win (canon-1,canon-2), got %q", got)
		}
		if _, hasHyphen := updated.Metadata["excluded-models"]; hasHyphen {
			t.Fatal("expected legacy excluded-models key to be dropped from metadata")
		}

		// Subtest B: legacy first, canonical second -> canonical still wins deterministically
		recB := httptest.NewRecorder()
		cB, _ := gin.CreateTestContext(recB)
		cB.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","excluded-models":["legacy-2"],"excluded_models":["canon-3"]}`))
		cB.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(cB)
		if recB.Code != http.StatusOK {
			t.Fatalf("expected 200 on dual-spelling patch B, got %d: %s", recB.Code, recB.Body.String())
		}
		updatedB, okB := manager.GetByID("cred.json")
		if !okB || updatedB == nil {
			t.Fatal("auth not found after patch")
		}
		if got := updatedB.Attributes["excluded_models"]; got != "canon-3" {
			t.Fatalf("expected canonical to win (canon-3), got %q", got)
		}
		if _, hasHyphen := updatedB.Metadata["excluded-models"]; hasHyphen {
			t.Fatal("expected legacy excluded-models key to be dropped from metadata")
		}
	}

	// Test 10: Legacy key in metadata is removed even when patching an unrelated field
	{
		// Artificially inject legacy "excluded-models" into targetAuth.Metadata
		cur, _ := manager.GetByID("cred.json")
		cloned := cur.Clone()
		cloned.Metadata["excluded-models"] = []any{"legacy-leftover"}
		_, _ = manager.Update(context.Background(), cloned)

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/auth-files/fields", strings.NewReader(`{"name":"cred.json","prefix":"new-prefix"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.PatchAuthFileFields(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on unrelated field patch, got %d: %s", rec.Code, rec.Body.String())
		}
		updated, ok := manager.GetByID("cred.json")
		if !ok || updated == nil {
			t.Fatal("auth not found after patch")
		}
		if _, hasHyphen := updated.Metadata["excluded-models"]; hasHyphen {
			t.Fatal("expected legacy excluded-models key to be dropped on every path")
		}
	}
}
