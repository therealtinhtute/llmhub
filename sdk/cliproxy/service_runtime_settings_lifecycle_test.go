package cliproxy

import (
	"context"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
	"github.com/therealtinhtute/llmhub/sdk/config"
)

type dummyRuntimeSettingsStore struct{}

func (d *dummyRuntimeSettingsStore) LoadRuntimeSettings(context.Context) (runtimecontrol.Settings, error) {
	return runtimecontrol.Settings{}, nil
}

func (d *dummyRuntimeSettingsStore) SaveRuntimeSettings(context.Context, int64, runtimecontrol.Settings) (runtimecontrol.Settings, error) {
	return runtimecontrol.Settings{}, nil
}

func TestServiceRuntimeSettingsStorePropagation(t *testing.T) {
	store := &dummyRuntimeSettingsStore{}
	service, err := NewBuilder().
		WithConfig(&config.Config{}).
		WithConfigPath("config.yaml").
		WithRuntimeSettingsStore(store).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if service.runtimeSettingsStore != store {
		t.Fatalf("service.runtimeSettingsStore = %v, want %v", service.runtimeSettingsStore, store)
	}
}
