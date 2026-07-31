package live

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

const (
	UpstreamCallURL  = "https://chatgpt.com/backend-api/codex/realtime/calls?intent=quicksilver&architecture=avas"
	DefaultLiveModel = "gpt-live-1-codex"
	MaxBodySize      = 16 << 20
)

var ErrBodyTooLarge = errors.New("Codex live request body too large")

var ProtocolHeaderNames = []string{
	"OpenAI-Alpha",
	"X-Session-Id",
	"Session-Id",
	"Thread-Id",
	"Originator",
	"X-Oai-Attestation",
}

type SidebandStyle int

const (
	SidebandFrameless SidebandStyle = iota
	SidebandRealtimeCalls
	SidebandRealtimeQuery
)

func ReadBody(body io.Reader) ([]byte, error) {
	payload, err := ReadLimitedBody(body)
	if err != nil {
		if errors.Is(err, ErrBodyTooLarge) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read Codex live request: %w", err)
	}
	return payload, nil
}

func ReadLimitedBody(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	payload, err := io.ReadAll(io.LimitReader(body, MaxBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > MaxBodySize {
		return nil, ErrBodyTooLarge
	}
	return payload, nil
}

func PrepareCallRequest(body []byte, contentType string) ([]byte, string, string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		return multipartCallRequest(body, strings.TrimSpace(params["boundary"]))
	}
	model := ModelFromJSON(body)
	if model == "" {
		model = DefaultLiveModel
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}
	return body, contentType, model, nil
}

func multipartCallRequest(body []byte, boundary string) ([]byte, string, string, error) {
	if boundary == "" {
		return nil, "", "", errors.New("Codex live multipart boundary is missing")
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var sdp *string
	var session json.RawMessage
	model := ""
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to parse Codex live multipart body: %w", err)
		}
		partBody, readErr := io.ReadAll(part)
		closeErr := part.Close()
		if readErr != nil {
			return nil, "", "", fmt.Errorf("failed to read Codex live multipart field: %w", readErr)
		}
		if closeErr != nil {
			return nil, "", "", fmt.Errorf("failed to close Codex live multipart field: %w", closeErr)
		}
		switch part.FormName() {
		case "sdp":
			value := string(partBody)
			sdp = &value
		case "session":
			if !json.Valid(partBody) {
				return nil, "", "", errors.New("Codex live session field must contain valid JSON")
			}
			session = append(json.RawMessage(nil), partBody...)
			model = ModelFromJSON(partBody)
		}
	}
	if sdp == nil {
		return nil, "", "", errors.New("Codex live multipart body requires an sdp field")
	}
	if model == "" {
		model = DefaultLiveModel
	}
	encoded, err := EncodeCallRequest(*sdp, session)
	if err != nil {
		return nil, "", "", err
	}
	return encoded, "application/json", model, nil
}

func EncodeCallRequest(sdp string, session json.RawMessage) ([]byte, error) {
	payload := struct {
		SDP     string          `json:"sdp"`
		Session json.RawMessage `json:"session,omitempty"`
	}{
		SDP:     sdp,
		Session: session,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Codex live request: %w", err)
	}
	return encoded, nil
}

func ModelFromJSON(body []byte) string {
	var payload struct {
		Model   string `json:"model"`
		Session struct {
			Model string `json:"model"`
		} `json:"session"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if model := strings.TrimSpace(payload.Session.Model); model != "" {
		return model
	}
	return strings.TrimSpace(payload.Model)
}

func ProtocolHeaders(source http.Header) http.Header {
	headers := make(http.Header)
	for _, name := range ProtocolHeaderNames {
		for _, value := range source.Values(name) {
			headers.Add(name, value)
		}
	}
	return headers
}

func RedactedHeadersForLogging(source http.Header) http.Header {
	headers := source.Clone()
	if headers.Get("X-Oai-Attestation") != "" {
		headers.Set("X-Oai-Attestation", "[REDACTED]")
	}
	return headers
}

func CallResponseHeaders(source http.Header) http.Header {
	headers := make(http.Header)
	for _, name := range []string{"Content-Type", "Location"} {
		for _, value := range source.Values(name) {
			headers.Add(name, value)
		}
	}
	return headers
}

func WriteResponseHeaders(destination, source http.Header) {
	for name, values := range source {
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}

func SidebandTarget(path string, params map[string]string, query url.Values) (SidebandStyle, string, bool) {
	if params != nil {
		if callID := strings.TrimSpace(params["call_id"]); callID != "" {
			style := SidebandFrameless
			if strings.Contains(path, "/realtime/calls/") {
				style = SidebandRealtimeCalls
			}
			return style, callID, ValidCallID(callID)
		}
	}
	callID := strings.TrimSpace(query.Get("call_id"))
	return SidebandRealtimeQuery, callID, ValidCallID(callID)
}

func BuildSidebandURL(baseURL string, style SidebandStyle, callID string) string {
	root := strings.TrimRight(baseURL, "/")
	switch style {
	case SidebandRealtimeCalls:
		return root + "/realtime/calls/" + callID
	case SidebandRealtimeQuery:
		return root + "/realtime?intent=quicksilver&call_id=" + url.QueryEscape(callID)
	default:
		return root + "/live/" + callID
	}
}

func WebsocketHTTPURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	switch strings.ToLower(parsed.Scheme) {
	case "ws":
		parsed.Scheme = "http"
	case "wss":
		parsed.Scheme = "https"
	}
	return parsed.String()
}

func CallIDFromLocation(location string) string {
	location = strings.TrimSpace(location)
	if ValidCallID(location) {
		return location
	}
	parsed, err := url.Parse(location)
	if err != nil {
		return ""
	}
	if callID := strings.TrimSpace(parsed.Query().Get("call_id")); ValidCallID(callID) {
		return callID
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	callID := parts[len(parts)-1]
	previous := parts[len(parts)-2]
	if !ValidCallID(callID) || (previous != "live" && previous != "calls") {
		return ""
	}
	return callID
}
