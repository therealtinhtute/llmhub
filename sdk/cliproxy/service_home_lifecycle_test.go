package cliproxy

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	"github.com/therealtinhtute/llmhub/sdk/cliproxy/executionregistry"
	"github.com/therealtinhtute/llmhub/sdk/config"
)

type serviceHomeDispatcher struct{}

func (serviceHomeDispatcher) HeartbeatOK() bool { return true }
func (serviceHomeDispatcher) RPopAuth(context.Context, string, string, http.Header, int) ([]byte, error) {
	return nil, nil
}
func (serviceHomeDispatcher) AbortAmbiguousDispatch() {}

type serviceHomeReleaseFlusher struct {
	flushed atomic.Bool
}

func (f *serviceHomeReleaseFlusher) Flush(context.Context) error {
	f.flushed.Store(true)
	return nil
}

func TestApplyConfigUpdateStopsHomeLifetimeWhenDisabled(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	registry := executionregistry.New()
	bundle := manager.PublishHomeDispatch(serviceHomeDispatcher{}, registry, 1)
	service := &Service{
		cfg:                &config.Config{Home: config.HomeConfig{Enabled: true}},
		coreManager:        manager,
		homeRegistry:       registry,
		homeDispatchBundle: bundle,
	}

	service.applyConfigUpdate(&config.Config{})

	if manager.HomeDispatchBundle() != nil {
		t.Fatal("Home dispatch bundle remained published after Home was disabled")
	}
	if service.homeRegistry != nil || service.homeDispatchBundle != nil {
		t.Fatal("Home lifetime state remained attached after Home was disabled")
	}
}

func TestStopHomeLifetimeDetachesDrainsAndFlushes(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	registry := executionregistry.New()
	pending, errBegin := registry.BeginDispatch()
	if errBegin != nil {
		t.Fatal(errBegin)
	}
	scope, errInstall := registry.Install(pending, executionregistry.ScopeSpec{
		RequestID: "request-1", CredentialID: "credential-1", Model: "model-1", Kind: "http", StartedAt: time.Now(),
	})
	if errInstall != nil {
		t.Fatal(errInstall)
	}
	var closed atomic.Bool
	closeRequested := make(chan struct{})
	if errBind := scope.Bind(func() error {
		closed.Store(true)
		close(closeRequested)
		return nil
	}); errBind != nil {
		t.Fatal(errBind)
	}
	go func() {
		<-closeRequested
		scope.End("drained")
	}()

	flusher := &serviceHomeReleaseFlusher{}
	bundle := manager.PublishHomeDispatch(serviceHomeDispatcher{}, registry, 1)
	service := &Service{
		cfg:                &config.Config{},
		coreManager:        manager,
		homeRegistry:       registry,
		homeDispatchBundle: bundle,
		homeReleaseFlusher: flusher,
	}
	service.stopHomeLifetime(context.Background())

	if manager.HomeDispatchBundle() != nil {
		t.Fatal("Home dispatch bundle remained published")
	}
	if !closed.Load() {
		t.Fatal("active execution resource was not drained")
	}
	if !flusher.flushed.Load() {
		t.Fatal("credential releases were not flushed")
	}
}
