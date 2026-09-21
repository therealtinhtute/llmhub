package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestBuildOpenAIResponsesStreamErrorChunk(t *testing.T) {
	chunk := BuildOpenAIResponsesStreamErrorChunk(http.StatusInternalServerError, "unexpected EOF", 0)
	var payload map[string]any
	if err := json.Unmarshal(chunk, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["type"] != "error" {
		t.Fatalf("type = %v, want %q", payload["type"], "error")
	}
	if payload["code"] != "internal_server_error" {
		t.Fatalf("code = %v, want %q", payload["code"], "internal_server_error")
	}
	if payload["message"] != "unexpected EOF" {
		t.Fatalf("message = %v, want %q", payload["message"], "unexpected EOF")
	}
	if payload["sequence_number"] != float64(0) {
		t.Fatalf("sequence_number = %v, want %v", payload["sequence_number"], 0)
	}
}

func TestBuildOpenAIResponsesStreamErrorChunkExtractsHTTPErrorBody(t *testing.T) {
	chunk := BuildOpenAIResponsesStreamErrorChunk(
		http.StatusInternalServerError,
		`{"error":{"message":"oops","type":"server_error","code":"internal_server_error"}}`,
		0,
	)
	var payload map[string]any
	if err := json.Unmarshal(chunk, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["type"] != "error" {
		t.Fatalf("type = %v, want %q", payload["type"], "error")
	}
	if payload["code"] != "internal_server_error" {
		t.Fatalf("code = %v, want %q", payload["code"], "internal_server_error")
	}
	if payload["message"] != "oops" {
		t.Fatalf("message = %v, want %q", payload["message"], "oops")
	}
}

// TestBuildOpenAIResponsesStreamErrorChunkRequestTimeoutIsServerError ports
// upstream CLIProxyAPI commit cb62a6748b99's regression test. The fork's
// legacy error chunk is flat ({type,code,message}) so the error-type
// classification is only observable on the nested response.failed detail.
func TestBuildOpenAIResponsesStreamErrorChunkRequestTimeoutIsServerError(t *testing.T) {
	chunk := BuildOpenAIResponsesStreamErrorChunk(http.StatusRequestTimeout, "stream disconnected before completion", 0)
	var payload struct {
		Type           string         `json:"type"`
		Code           string         `json:"code"`
		Error          map[string]any `json:"error"`
		SequenceNumber int            `json:"sequence_number"`
	}
	if errUnmarshal := json.Unmarshal(chunk, &payload); errUnmarshal != nil {
		t.Fatalf("unmarshal: %v", errUnmarshal)
	}
	if payload.Type != "error" {
		t.Fatalf("type = %q, want %q", payload.Type, "error")
	}
	if payload.Code != "request_timeout" {
		t.Fatalf("code = %v, want %q", payload.Code, "request_timeout")
	}
	// When a nested error detail is present it must classify the timeout as a
	// server error so clients know the request is safe to replay.
	if payload.Error != nil {
		if got := payload.Error["type"]; got != "server_error" {
			t.Fatalf("error.type = %v, want %q", got, "server_error")
		}
	}

	failedChunk := BuildOpenAIResponsesStreamFailedChunk(http.StatusRequestTimeout, "stream disconnected before completion", 0)
	var failedPayload struct {
		Type     string `json:"type"`
		Response struct {
			Status string         `json:"status"`
			Error  map[string]any `json:"error"`
		} `json:"response"`
	}
	if errUnmarshal := json.Unmarshal(failedChunk, &failedPayload); errUnmarshal != nil {
		t.Fatalf("unmarshal: %v", errUnmarshal)
	}
	if got := failedPayload.Response.Error["code"]; got != "request_timeout" {
		t.Fatalf("failed.response.error.code = %v, want %q", got, "request_timeout")
	}
	if got := failedPayload.Response.Error["type"]; got != "server_error" {
		t.Fatalf("failed.response.error.type = %v, want %q", got, "server_error")
	}
}
