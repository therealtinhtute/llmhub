package synthesizer

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
)

func TestExtractExcludedModelsFromMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		want     []string
		wantErr  bool
	}{
		{
			name:     "nil metadata",
			metadata: nil,
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "empty metadata",
			metadata: map[string]any{},
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "canonical string slice",
			metadata: map[string]any{"excluded_models": []string{"model-a", "model-b"}},
			want:     []string{"model-a", "model-b"},
			wantErr:  false,
		},
		{
			name:     "legacy hyphen key",
			metadata: map[string]any{"excluded-models": []string{"model-x"}},
			want:     []string{"model-x"},
			wantErr:  false,
		},
		{
			name:     "comma separated string",
			metadata: map[string]any{"excluded_models": "model-1, model-2 ,model-3"},
			want:     []string{"model-1", "model-2", "model-3"},
			wantErr:  false,
		},
		{
			name:     "mixed interface slice with valid and invalid items",
			metadata: map[string]any{"excluded_models": []any{"valid-1", 123, "valid-2", false}},
			want:     []string{"valid-1", "valid-2"},
			wantErr:  true,
		},
		{
			name:     "invalid non-array non-string type",
			metadata: map[string]any{"excluded_models": 12345},
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractExcludedModelsFromMetadata(tc.metadata)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ExtractExcludedModelsFromMetadata() err = %v, wantErr %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ExtractExcludedModelsFromMetadata() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSynthesizeAuthFileKeepsCredentialWithInvalidExcludedModels(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "auth-bad.json")
	content := []byte(`{"type":"codex","email":"test@example.com","excluded_models":["valid-model", 42]}`)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to write auth file: %v", err)
	}

	ctx := &SynthesisContext{
		Config:      &config.Config{},
		AuthDir:     tempDir,
		Now:         time.Now(),
		IDGenerator: NewStableIDGenerator(),
	}

	auths, errSynthesize := SynthesizeAuthFile(ctx, filePath, content)
	if errSynthesize != nil {
		t.Fatalf("unexpected error from SynthesizeAuthFile: %v", errSynthesize)
	}
	if len(auths) != 1 {
		t.Fatalf("expected 1 auth even with invalid excluded_models elements, got %d", len(auths))
	}
	if got := auths[0].Attributes["excluded_models"]; got != "valid-model" {
		t.Fatalf("expected valid excluded_models to be retained (%q), got %q", "valid-model", got)
	}
}
