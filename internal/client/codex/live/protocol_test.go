package live

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestPrepareCallRequestConvertsMultipartSDP(t *testing.T) {
	const boundary = "codex-realtime-call-boundary"
	body := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"sdp\"\r\n" +
		"Content-Type: application/sdp\r\n\r\n" +
		"v=0\r\na=setup:actpass" + "\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"session\"\r\n" +
		"Content-Type: application/json\r\n\r\n" +
		`{"model":"gpt-live-1-codex-custom"}` + "\r\n" +
		"--" + boundary + "--\r\n"

	payload, contentType, model, err := PrepareCallRequest([]byte(body), "multipart/form-data; boundary="+boundary)
	if err != nil {
		t.Fatalf("PrepareCallRequest() error = %v", err)
	}
	if contentType != "application/json" || model != "gpt-live-1-codex-custom" {
		t.Fatalf("contentType/model = %q/%q", contentType, model)
	}
	var got struct {
		SDP     string          `json:"sdp"`
		Session json.RawMessage `json:"session"`
	}
	if err = json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v; body=%s", err, payload)
	}
	if got.SDP != "v=0\r\na=setup:actpass" || !json.Valid(got.Session) {
		t.Fatalf("payload = %#v", got)
	}
}

func TestPrepareCallRequestDefaultsJSONModel(t *testing.T) {
	payload, contentType, model, err := PrepareCallRequest([]byte(`{"session":{"model":""}}`), "")
	if err != nil {
		t.Fatalf("PrepareCallRequest() error = %v", err)
	}
	if string(payload) != `{"session":{"model":""}}` || contentType != "application/json" || model != DefaultLiveModel {
		t.Fatalf("payload/contentType/model = %s/%q/%q", payload, contentType, model)
	}
}

func TestPrepareCallRequestRejectsBadMultipart(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		match       string
	}{
		{name: "missing boundary", contentType: "multipart/form-data", match: "boundary"},
		{name: "missing sdp", body: "--b--\r\n", contentType: "multipart/form-data; boundary=b", match: "sdp"},
		{name: "bad session json", body: "--b\r\nContent-Disposition: form-data; name=\"sdp\"\r\n\r\nv=0\r\n--b\r\nContent-Disposition: form-data; name=\"session\"\r\n\r\nnot-json\r\n--b--\r\n", contentType: "multipart/form-data; boundary=b", match: "valid JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := PrepareCallRequest([]byte(tt.body), tt.contentType)
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("PrepareCallRequest() error = %v, want %q", err, tt.match)
			}
		})
	}
}

func TestProtocolHeadersAndResponseHeadersAreAllowlisted(t *testing.T) {
	source := http.Header{}
	source.Set("Authorization", "Bearer downstream")
	source.Set("OpenAI-Alpha", "quicksilver=v2")
	source.Set("X-Oai-Attestation", "attestation-token")
	source.Set("Content-Type", "application/sdp")
	source.Set("Location", "/v1/live/call-123")
	source.Set("Set-Cookie", "secret")

	protocol := ProtocolHeaders(source)
	if protocol.Get("OpenAI-Alpha") != "quicksilver=v2" || protocol.Get("Authorization") != "" {
		t.Fatalf("protocol headers = %#v", protocol)
	}
	redacted := RedactedHeadersForLogging(protocol)
	if got := redacted.Get("X-Oai-Attestation"); got != "[REDACTED]" {
		t.Fatalf("redacted attestation = %q", got)
	}
	response := CallResponseHeaders(source)
	if response.Get("Content-Type") != "application/sdp" || response.Get("Location") != "/v1/live/call-123" || response.Get("Set-Cookie") != "" {
		t.Fatalf("response headers = %#v", response)
	}
}

func TestCallIDFromLocation(t *testing.T) {
	for _, location := range []string{"call-123", "/v1/live/call-123", "/v1/realtime/calls/call-123", "https://example.test/realtime?call_id=call-123"} {
		if got := CallIDFromLocation(location); got != "call-123" {
			t.Fatalf("CallIDFromLocation(%q) = %q", location, got)
		}
	}
	for _, location := range []string{"", "/v1/other/call-123", "../bad"} {
		if got := CallIDFromLocation(location); got != "" {
			t.Fatalf("CallIDFromLocation(%q) = %q, want empty", location, got)
		}
	}
}

func TestReadLimitedBodyRejectsLargePayload(t *testing.T) {
	_, err := ReadLimitedBody(strings.NewReader(strings.Repeat("x", MaxBodySize+1)))
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("ReadLimitedBody() error = %v, want ErrBodyTooLarge", err)
	}
}
