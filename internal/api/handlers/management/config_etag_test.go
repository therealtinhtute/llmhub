package management

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	"gopkg.in/yaml.v3"
)

// TestConfigETagIfMatch verifies optimistic concurrency control via ETag and If-Match
// headers adapted to the Postgres-authoritative config store.
func TestConfigETagIfMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initialYAML := "port: 8317\naccess:\n  api-keys:\n    - revoked-key\n"
	store := &recordingConfigStore{
		data:      []byte(initialYAML),
		saveCount: 1,
		version:   1,
	}

	cfg := &config.Config{
		Port:          8317,
		ConfigVersion: 1,
	}
	cfg.APIKeys = []string{"revoked-key"}
	h := NewHandlerWithoutConfigFilePath(cfg, nil)
	h.SetConfigStore(store)

	router := gin.New()
	router.GET("/v0/management/config", h.GetConfig)
	router.GET("/v0/management/config.yaml", h.GetConfigYAML)
	router.PUT("/v0/management/config.yaml", h.PutConfigYAML)
	router.GET("/v0/management/api-keys", h.GetAPIKeys)
	router.PUT("/v0/management/api-keys", h.PutAPIKeys)
	router.PATCH("/v0/management/api-keys", h.PatchAPIKeys)
	router.DELETE("/v0/management/api-keys", h.DeleteAPIKeys)

	do := func(method, url, body, ifMatch string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, url, strings.NewReader(body))
		if ifMatch != "" {
			req.Header.Set("If-Match", ifMatch)
		}
		router.ServeHTTP(recorder, req)
		return recorder
	}
	wantStatus := func(recorder *httptest.ResponseRecorder, status int) {
		t.Helper()
		if recorder.Code != status {
			t.Fatalf("status=%d want %d body=%s", recorder.Code, status, recorder.Body.String())
		}
	}

	// 1. GET endpoints advertise the store version as a strong ETag.
	getYAML := do(http.MethodGet, "/v0/management/config.yaml", "", "")
	wantStatus(getYAML, http.StatusOK)
	etag := getYAML.Header().Get("ETag")
	if etag != `"1"` {
		t.Fatalf("GET config.yaml ETag=%q want %q", etag, `"1"`)
	}

	getObj := do(http.MethodGet, "/v0/management/config", "", "")
	wantStatus(getObj, http.StatusOK)
	if got := getObj.Header().Get("ETag"); got != etag {
		t.Fatalf("GET config ETag=%q want %q", got, etag)
	}

	// 2. PUT with a stale ETag fails with 412 Precondition Failed.
	stalePut := do(http.MethodPut, "/v0/management/config.yaml", "port: 8319\n", `"stale"`)
	wantStatus(stalePut, http.StatusPreconditionFailed)
	if got := stalePut.Header().Get("ETag"); got != `"1"` {
		t.Fatalf("412 response ETag=%q want current %q", got, `"1"`)
	}
	if string(store.data) != initialYAML {
		t.Fatal("stale PUT changed the store data")
	}

	// 3. PUT with matching If-Match succeeds and advances the ETag.
	matchingPut := do(http.MethodPut, "/v0/management/config.yaml", "port: 8319\n", `"1"`)
	wantStatus(matchingPut, http.StatusOK)
	newETag := matchingPut.Header().Get("ETag")
	if newETag != `"2"` {
		t.Fatalf("PUT response ETag=%q want %q", newETag, `"2"`)
	}
	if h.cfg.Port != 8319 {
		t.Fatalf("h.cfg.Port = %d, want 8319", h.cfg.Port)
	}

	// 4. Stale whole-document PUT based on read (ETag "1") must be rejected.
	staleSecond := do(http.MethodPut, "/v0/management/config.yaml", initialYAML, etag)
	wantStatus(staleSecond, http.StatusPreconditionFailed)
	if got := staleSecond.Header().Get("ETag"); got != `"2"` {
		t.Fatalf("412 response ETag=%q want current %q", got, `"2"`)
	}

	// 5. Unconditional PUT succeeds.
	unconditional := do(http.MethodPut, "/v0/management/config.yaml", "port: 8320\n", "")
	wantStatus(unconditional, http.StatusOK)
	if got := unconditional.Header().Get("ETag"); got != `"3"` {
		t.Fatalf("unconditional PUT ETag=%q want %q", got, `"3"`)
	}

	// 6. Star If-Match matches any existing version.
	star := do(http.MethodPut, "/v0/management/config.yaml", "port: 8321\n", "*")
	wantStatus(star, http.StatusOK)
	if got := star.Header().Get("ETag"); got != `"4"` {
		t.Fatalf("star PUT ETag=%q want %q", got, `"4"`)
	}

	// 7. Weak and malformed validators never match under strong comparison.
	currentETag := `"4"`
	for _, badIfMatch := range []string{
		"W/" + currentETag,
		"not-an-etag",
		`"unterminated`,
		`"a"b"`,
		`"stale,*,tag"`,
		`W/"stale,*,tag"`,
		`"stale", *`,
	} {
		rec := do(http.MethodPut, "/v0/management/config.yaml", "port: 8322\n", badIfMatch)
		wantStatus(rec, http.StatusPreconditionFailed)
	}

	// 8. Blank or whitespace-only If-Match fails.
	for _, blank := range []string{"", "  \t  "} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/v0/management/config.yaml", strings.NewReader("port: 8322\n"))
		req.Header.Add("If-Match", blank)
		router.ServeHTTP(recorder, req)
		wantStatus(recorder, http.StatusPreconditionFailed)
	}

	// 9. Split header lines (repeated headers) match if one is current.
	splitRec := httptest.NewRecorder()
	splitReq := httptest.NewRequest(http.MethodPut, "/v0/management/config.yaml", strings.NewReader("port: 8323\n"))
	splitReq.Header.Add("If-Match", `"stale"`)
	splitReq.Header.Add("If-Match", currentETag)
	router.ServeHTTP(splitRec, splitReq)
	wantStatus(splitRec, http.StatusOK)
	if got := splitRec.Header().Get("ETag"); got != `"5"` {
		t.Fatalf("split header PUT ETag=%q want %q", got, `"5"`)
	}

	// 10. List endpoints (PUT/PATCH/DELETE APIKeys) enforce If-Match and emit ETag.
	curListETag := `"5"`
	staleListPut := do(http.MethodPut, "/v0/management/api-keys", `["new-key"]`, `"stale"`)
	wantStatus(staleListPut, http.StatusPreconditionFailed)
	if got := staleListPut.Header().Get("ETag"); got != curListETag {
		t.Fatalf("stale APIKeys PUT ETag=%q want %q", got, curListETag)
	}

	// Matching If-Match on list mutation succeeds and increments ETag.
	okListPut := do(http.MethodPut, "/v0/management/api-keys", `["new-key"]`, curListETag)
	wantStatus(okListPut, http.StatusOK)
	afterListETag := okListPut.Header().Get("ETag")
	if afterListETag != `"6"` {
		t.Fatalf("ok APIKeys PUT ETag=%q want %q", afterListETag, `"6"`)
	}

	// PATCH APIKeys with stale ETag is rejected.
	stalePatch := do(http.MethodPatch, "/v0/management/api-keys", `{"old":"new-key","new":"updated-key"}`, `"stale"`)
	wantStatus(stalePatch, http.StatusPreconditionFailed)

	// PATCH APIKeys with matching ETag succeeds.
	okPatch := do(http.MethodPatch, "/v0/management/api-keys", `{"old":"new-key","new":"updated-key"}`, afterListETag)
	wantStatus(okPatch, http.StatusOK)
	afterPatchETag := okPatch.Header().Get("ETag")
	if afterPatchETag != `"7"` {
		t.Fatalf("ok APIKeys PATCH ETag=%q want %q", afterPatchETag, `"7"`)
	}

	// DELETE APIKeys with stale ETag is rejected.
	staleDelete := do(http.MethodDelete, "/v0/management/api-keys?value=updated-key", "", `"stale"`)
	wantStatus(staleDelete, http.StatusPreconditionFailed)

	// DELETE APIKeys with matching ETag succeeds.
	okDelete := do(http.MethodDelete, "/v0/management/api-keys?value=updated-key", "", afterPatchETag)
	wantStatus(okDelete, http.StatusOK)
	afterDeleteETag := okDelete.Header().Get("ETag")
	if afterDeleteETag != `"8"` {
		t.Fatalf("ok APIKeys DELETE ETag=%q want %q", afterDeleteETag, `"8"`)
	}

	// Verify the final state in store.
	var saved config.Config
	if err := yaml.Unmarshal(store.data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.APIKeys) != 0 {
		t.Fatalf("expected API keys to be empty after delete, got %v", saved.APIKeys)
	}
}
