package management

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
)

func newLogsTestHandler(dir string, loggingToFile bool) *Handler {
	h := NewHandlerWithoutConfigFilePath(&config.Config{LoggingToFile: loggingToFile, SDKConfig: config.SDKConfig{RequestLog: loggingToFile}}, nil)
	h.SetLogDirectory(dir)
	return h
}

func TestGetRequestLogByID_SelectsLatestOnReusedID(t *testing.T) {
	dir := t.TempDir()
	h := newLogsTestHandler(dir, true)

	oldPath := filepath.Join(dir, "v1_chat_completions-2026-09-23T100000-00000000.log")
	newPath := filepath.Join(dir, "v1_chat_completions-2026-09-23T110000-00000000.log")

	if errWrite := os.WriteFile(oldPath, []byte("old log content"), 0o644); errWrite != nil {
		t.Fatalf("write old log: %v", errWrite)
	}
	oldTime := time.Now().Add(-time.Hour)
	if errChtimes := os.Chtimes(oldPath, oldTime, oldTime); errChtimes != nil {
		t.Fatalf("chtimes old log: %v", errChtimes)
	}

	if errWrite := os.WriteFile(newPath, []byte("new log content"), 0o644); errWrite != nil {
		t.Fatalf("write new log: %v", errWrite)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: "00000000"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/logs/request/00000000", nil)

	h.GetRequestLogByID(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "new log content" {
		t.Fatalf("body = %q, want %q", body, "new log content")
	}
}

func TestGetRequestLogByID_SelectsLatestOnSameModTimeTieBreak(t *testing.T) {
	dir := t.TempDir()
	h := newLogsTestHandler(dir, true)

	file1 := filepath.Join(dir, "v1_chat_completions-2026-09-23T100000-00000000.log")
	file2 := filepath.Join(dir, "v1_chat_completions-2026-09-23T100000_1-00000000.log")

	if errWrite := os.WriteFile(file1, []byte("file1 content"), 0o644); errWrite != nil {
		t.Fatalf("write file1: %v", errWrite)
	}
	if errWrite := os.WriteFile(file2, []byte("file2 content"), 0o644); errWrite != nil {
		t.Fatalf("write file2: %v", errWrite)
	}
	sameTime := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	if errChtimes := os.Chtimes(file1, sameTime, sameTime); errChtimes != nil {
		t.Fatalf("chtimes file1: %v", errChtimes)
	}
	if errChtimes := os.Chtimes(file2, sameTime, sameTime); errChtimes != nil {
		t.Fatalf("chtimes file2: %v", errChtimes)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: "00000000"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/logs/request/00000000", nil)

	h.GetRequestLogByID(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "file2 content" {
		t.Fatalf("body = %q, want %q", body, "file2 content")
	}
}

func TestGetRequestLogByID_SelectsLatestOnSameModTimeSequence9Vs10(t *testing.T) {
	dir := t.TempDir()
	h := newLogsTestHandler(dir, true)

	file9 := filepath.Join(dir, "v1_chat_completions-2026-09-23T100000_9-00000000.log")
	file10 := filepath.Join(dir, "v1_chat_completions-2026-09-23T100000_10-00000000.log")

	if errWrite := os.WriteFile(file9, []byte("file9 content"), 0o644); errWrite != nil {
		t.Fatalf("write file9: %v", errWrite)
	}
	if errWrite := os.WriteFile(file10, []byte("file10 content"), 0o644); errWrite != nil {
		t.Fatalf("write file10: %v", errWrite)
	}
	sameTime := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	if errChtimes := os.Chtimes(file9, sameTime, sameTime); errChtimes != nil {
		t.Fatalf("chtimes file9: %v", errChtimes)
	}
	if errChtimes := os.Chtimes(file10, sameTime, sameTime); errChtimes != nil {
		t.Fatalf("chtimes file10: %v", errChtimes)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: "00000000"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/logs/request/00000000", nil)

	h.GetRequestLogByID(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "file10 content" {
		t.Fatalf("body = %q, want %q (numerical sequence 10 must beat 9)", body, "file10 content")
	}
}

func TestGetRequestLogByID_SelectsLatestAcrossTimestampsWithSequenceOnModTimeTie(t *testing.T) {
	dir := t.TempDir()
	h := newLogsTestHandler(dir, true)

	oldFileWithSeq := filepath.Join(dir, "v1_chat_completions-2026-09-23T100000_9-00000000.log")
	newFileWithoutSeq := filepath.Join(dir, "v1_chat_completions-2026-09-23T110000-00000000.log")

	if errWrite := os.WriteFile(oldFileWithSeq, []byte("old with seq content"), 0o644); errWrite != nil {
		t.Fatalf("write oldFileWithSeq: %v", errWrite)
	}
	if errWrite := os.WriteFile(newFileWithoutSeq, []byte("new without seq content"), 0o644); errWrite != nil {
		t.Fatalf("write newFileWithoutSeq: %v", errWrite)
	}
	sameTime := time.Date(2026, 9, 23, 11, 0, 0, 0, time.UTC)
	if errChtimes := os.Chtimes(oldFileWithSeq, sameTime, sameTime); errChtimes != nil {
		t.Fatalf("chtimes oldFileWithSeq: %v", errChtimes)
	}
	if errChtimes := os.Chtimes(newFileWithoutSeq, sameTime, sameTime); errChtimes != nil {
		t.Fatalf("chtimes newFileWithoutSeq: %v", errChtimes)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: "00000000"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/logs/request/00000000", nil)

	h.GetRequestLogByID(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "new without seq content" {
		t.Fatalf("body = %q, want %q (timestamp T110000 must beat T100000 even if older had higher sequence)", body, "new without seq content")
	}
}

func TestGetRequestLogByID_SelectsLatestCrossYearBoundaryOnModTimeTie(t *testing.T) {
	dir := t.TempDir()
	h := newLogsTestHandler(dir, true)

	oldYearFile := filepath.Join(dir, "v1_chat_completions-2026-12-31T235959-00000000.log")
	newYearFile := filepath.Join(dir, "v1_chat_completions-2027-01-01T000000-00000000.log")

	if errWrite := os.WriteFile(oldYearFile, []byte("2026 content"), 0o644); errWrite != nil {
		t.Fatalf("write oldYearFile: %v", errWrite)
	}
	if errWrite := os.WriteFile(newYearFile, []byte("2027 content"), 0o644); errWrite != nil {
		t.Fatalf("write newYearFile: %v", errWrite)
	}
	sameTime := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if errChtimes := os.Chtimes(oldYearFile, sameTime, sameTime); errChtimes != nil {
		t.Fatalf("chtimes oldYearFile: %v", errChtimes)
	}
	if errChtimes := os.Chtimes(newYearFile, sameTime, sameTime); errChtimes != nil {
		t.Fatalf("chtimes newYearFile: %v", errChtimes)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: "00000000"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/logs/request/00000000", nil)

	h.GetRequestLogByID(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	if body := rec.Body.String(); body != "2027 content" {
		t.Fatalf("body = %q, want %q (2027-01-01 must beat 2026-12-31 on mod time tie)", body, "2027 content")
	}
}

func TestGetRequestLogByID_MatchesFullUUIDAndShortID(t *testing.T) {
	dir := t.TempDir()
	h := newLogsTestHandler(dir, true)

	logPath := filepath.Join(dir, "v1_chat_completions-2026-09-28T100000-1234abcd.log")
	if errWrite := os.WriteFile(logPath, []byte("uuid v7 request log content"), 0o644); errWrite != nil {
		t.Fatalf("write log: %v", errWrite)
	}

	// 1. Fetch via short 8-char ID
	recShort := httptest.NewRecorder()
	cShort, _ := gin.CreateTestContext(recShort)
	cShort.Params = gin.Params{{Key: "id", Value: "1234abcd"}}
	cShort.Request = httptest.NewRequest(http.MethodGet, "/v0/management/logs/request/1234abcd", nil)
	h.GetRequestLogByID(cShort)

	if recShort.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", recShort.Code, recShort.Body.String())
	}
	if body := recShort.Body.String(); body != "uuid v7 request log content" {
		t.Fatalf("body = %q, want %q", body, "uuid v7 request log content")
	}

	// 2. Fetch via full 36-char UUID string ending with same 8 chars
	recFull := httptest.NewRecorder()
	cFull, _ := gin.CreateTestContext(recFull)
	cFull.Params = gin.Params{{Key: "id", Value: "018f3a5b-1234-7abc-def0-00001234abcd"}}
	cFull.Request = httptest.NewRequest(http.MethodGet, "/v0/management/logs/request/018f3a5b-1234-7abc-def0-00001234abcd", nil)
	h.GetRequestLogByID(cFull)

	if recFull.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", recFull.Code, recFull.Body.String())
	}
	if body := recFull.Body.String(); body != "uuid v7 request log content" {
		t.Fatalf("body = %q, want %q", body, "uuid v7 request log content")
	}
}
