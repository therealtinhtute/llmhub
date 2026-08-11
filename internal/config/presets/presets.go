// Package presets ships a curated, embedded catalog of known OpenAI-compatible providers
// (base URL, headers, signup info) that the management panel and API can offer as starting
// points for a new openai-compatibility config entry.
package presets

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed providers.json
var providersJSON []byte

// Preset describes one known OpenAI-compatible provider.
type Preset struct {
	ID            string            `json:"id"`
	DisplayName   string            `json:"display_name"`
	BaseURL       string            `json:"base_url"`
	Headers       map[string]string `json:"headers,omitempty"`
	ModelsURL     string            `json:"models_url,omitempty"`
	SignupURL     string            `json:"signup_url,omitempty"`
	FreeTierNote  string            `json:"free_tier_note,omitempty"`
	Passthrough   bool              `json:"passthrough"`
	DefaultAPIKey string            `json:"default_api_key,omitempty"`
	Verified      bool              `json:"verified"`
	VerifiedAt    string            `json:"verified_at,omitempty"`
	Category      string            `json:"category"`
}

var (
	loadOnce sync.Once
	loaded   []Preset
)

// All returns the embedded preset catalog. The catalog is parsed once and cached.
func All() []Preset {
	loadOnce.Do(func() {
		var presets []Preset
		if err := json.Unmarshal(providersJSON, &presets); err != nil {
			panic("presets: invalid embedded providers.json: " + err.Error())
		}
		loaded = presets
	})
	return loaded
}
