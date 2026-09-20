package translator

import (
	"context"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/tidwall/gjson"
)

func TestTranslateRequest_FallbackNormalizesModel(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name          string
		model         string
		payload       string
		wantModel     string
		wantUnchanged bool
	}{
		{
			name:      "prefixed model is rewritten",
			model:     "gpt-5-mini",
			payload:   `{"model":"copilot/gpt-5-mini","input":"ping"}`,
			wantModel: "gpt-5-mini",
		},
		{
			name:          "matching model is left unchanged",
			model:         "gpt-5-mini",
			payload:       `{"model":"gpt-5-mini","input":"ping"}`,
			wantModel:     "gpt-5-mini",
			wantUnchanged: true,
		},
		{
			name:          "empty model leaves payload unchanged",
			model:         "",
			payload:       `{"model":"copilot/gpt-5-mini","input":"ping"}`,
			wantModel:     "copilot/gpt-5-mini",
			wantUnchanged: true,
		},
		{
			name:      "deeply prefixed model is rewritten",
			model:     "gpt-5.3-codex",
			payload:   `{"model":"team/gpt-5.3-codex","stream":true}`,
			wantModel: "gpt-5.3-codex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.payload)
			got := r.TranslateRequest(Format("a"), Format("b"), tt.model, input, false)

			gotModel := gjson.GetBytes(got, "model").String()
			if gotModel != tt.wantModel {
				t.Errorf("model = %q, want %q", gotModel, tt.wantModel)
			}

			if tt.wantUnchanged && string(got) != tt.payload {
				t.Errorf("payload was modified when it should not have been:\ngot:  %s\nwant: %s", got, tt.payload)
			}

			// Verify other fields are preserved.
			for _, key := range []string{"input", "stream"} {
				orig := gjson.Get(tt.payload, key)
				if !orig.Exists() {
					continue
				}
				after := gjson.GetBytes(got, key)
				if orig.Raw != after.Raw {
					t.Errorf("field %q changed: got %s, want %s", key, after.Raw, orig.Raw)
				}
			}
		})
	}
}

func TestTranslateRequest_RegisteredTransformTakesPrecedence(t *testing.T) {
	r := NewRegistry()
	from := Format("openai-response")
	to := Format("openai-response")

	r.Register(from, to, func(model string, rawJSON []byte, stream bool) []byte {
		return []byte(`{"model":"from-transform"}`)
	}, ResponseTransform{})

	input := []byte(`{"model":"copilot/gpt-5-mini","input":"ping"}`)
	got := r.TranslateRequest(from, to, "gpt-5-mini", input, false)

	gotModel := gjson.GetBytes(got, "model").String()
	if gotModel != "from-transform" {
		t.Errorf("expected registered transform to take precedence, got model = %q", gotModel)
	}
}

func TestRequestEnvelopePreservesRegisteredTransformDispatch(t *testing.T) {
	r := NewRegistry()
	from := FormatOpenAIResponse
	to := FormatAntigravity
	modelInfo := &registry.ModelInfo{ID: "home-model"}

	r.RegisterRequestEnvelope(from, to, func(_ context.Context, req RequestEnvelope) RequestEnvelope {
		if req.ModelInfo == nil {
			req.Body = []byte(`{"source":"native"}`)
		} else {
			req.Body = []byte(`{"source":"model-info"}`)
		}
		return req
	})

	withoutMetadata := r.TranslateRequest(from, to, "home-model", []byte(`{"input":"hello"}`), false)
	if gjson.GetBytes(withoutMetadata, "source").String() != "native" {
		t.Fatalf("without metadata used unexpected transform: %s", withoutMetadata)
	}
	withMetadata, err := NewPipeline(r).TranslateRequest(context.Background(), from, to, RequestEnvelope{
		Format: from, Model: "home-model", Body: []byte(`{"input":"hello"}`), ModelInfo: modelInfo,
	})
	if err != nil {
		t.Fatalf("pipeline translation failed: %v", err)
	}
	if gjson.GetBytes(withMetadata.Body, "source").String() != "model-info" {
		t.Fatalf("envelope metadata was not used: %s", withMetadata.Body)
	}
	if withMetadata.ModelInfo != modelInfo {
		t.Fatal("pipeline did not preserve request-scoped model info")
	}

	// A custom registration replaces the envelope-aware native route.
	r.Register(from, to, func(string, []byte, bool) []byte {
		return []byte(`{"source":"custom"}`)
	}, ResponseTransform{})
	for _, payload := range [][]byte{
		[]byte(`{"input":"hello"}`),
		[]byte(`{"input":"weather","tools":[{"type":"web_search"}]}`),
	} {
		customWithMetadata := r.TranslateRequestEnvelope(context.Background(), from, to, RequestEnvelope{
			Format: from, Model: "home-model", Body: payload, ModelInfo: modelInfo,
		})
		if gjson.GetBytes(customWithMetadata.Body, "source").String() != "custom" {
			t.Fatalf("request metadata bypassed custom transform for %s: %s", payload, customWithMetadata.Body)
		}
	}
}
