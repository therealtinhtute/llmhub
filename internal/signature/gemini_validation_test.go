package signature

import (
	"strings"
	"testing"
)

func TestValidateGeminiFunctionCallPairingValidParallelGroup(t *testing.T) {
	input := []byte(`{
		"contents": [
			{
				"role": "model",
				"parts": [
					{"functionCall": {"id": "call-1", "name": "weather", "args": {"city": "Paris"}}},
					{"functionCall": {"id": "call-2", "name": "weather", "args": {"city": "London"}}}
				]
			},
			{
				"role": "user",
				"parts": [
					{"functionResponse": {"id": "call-1", "name": "weather", "response": {"temp": "15C"}}},
					{"functionResponse": {"id": "call-2", "name": "weather", "response": {"temp": "12C"}}}
				]
			}
		]
	}`)

	if err := ValidateGeminiFunctionCallPairing(input); err != nil {
		t.Fatalf("valid pairing failed: %v", err)
	}
}

func TestValidateGeminiFunctionCallPairingRejectsUserBoundaryBeforeResponse(t *testing.T) {
	payload := []byte(`{"request":{"contents":[{"role":"model","parts":[{"functionCall":{"id":"call-1","name":"run","args":{}}}]},{"role":"user","parts":[{"text":"boundary"}]},{"role":"function","parts":[{"functionResponse":{"id":"call-1","name":"run","response":{"result":"ok"}}}]}]}}`)

	err := ValidateGeminiFunctionCallPairing(payload)
	if err == nil {
		t.Fatal("user boundary before function response was accepted")
	}
	if !strings.Contains(err.Error(), "content appears before") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGeminiFunctionCallPairingRejectsIDMismatch(t *testing.T) {
	input := []byte(`{
		"contents": [
			{"role": "model", "parts": [{"functionCall": {"id": "call-1", "name": "weather", "args": {}}}]},
			{"role": "user", "parts": [{"functionResponse": {"id": "call-other", "name": "weather", "response": {}}}]}
		]
	}`)

	err := ValidateGeminiFunctionCallPairing(input)
	if err == nil {
		t.Fatal("id mismatch should fail")
	}
	if !strings.Contains(err.Error(), "does not match functionCall.id") {
		t.Fatalf("unexpected error: %v", err)
	}
}
