package auth

import (
	"reflect"
	"sync"
	"testing"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
)

func testComboConfigs() []internalconfig.ComboConfig {
	return []internalconfig.ComboConfig{
		{
			Name:     "daily",
			Strategy: "fallback",
			Models:   []string{"claude/claude-opus-4-7", "openrouter/deepseek-v4:free", "opencode/grok-code"},
		},
		{
			Name:        "rr",
			Strategy:    "round-robin",
			StickyLimit: 3,
			Models:      []string{"claude/claude-opus-4-7", "openrouter/deepseek-v4:free"},
		},
	}
}

func TestComboResolverResolveFallbackOrder(t *testing.T) {
	r := NewComboResolver()
	r.SetCombos(testComboConfigs())

	candidates, ok := r.Resolve("daily")
	if !ok {
		t.Fatal("expected daily to resolve")
	}
	want := []ComboCandidate{
		{Provider: "claude", Model: "claude-opus-4-7"},
		{Provider: "openrouter", Model: "deepseek-v4:free"},
		{Provider: "opencode", Model: "grok-code"},
	}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("Resolve(daily) = %+v, want %+v", candidates, want)
	}
	if _, ok := r.Resolve("not-a-combo"); ok {
		t.Fatal("expected unknown combo to not resolve")
	}
}

func TestComboResolverFallbackIgnoresRotation(t *testing.T) {
	r := NewComboResolver()
	r.SetCombos(testComboConfigs())

	for i := 0; i < 5; i++ {
		rotated := r.Rotate("daily", "fallback", 1)
		if rotated[0].Provider != "claude" {
			t.Fatalf("call %d: fallback must always start at index 0, got %+v", i, rotated)
		}
	}
}

func TestComboResolverRoundRobinStickyLimit(t *testing.T) {
	r := NewComboResolver()
	r.SetCombos(testComboConfigs())

	// sticky-limit: 3 → requests 1..3 serve candidate 0, request 4 serves candidate 1.
	for i := 1; i <= 3; i++ {
		rotated := r.Rotate("rr", "round-robin", 3)
		if rotated[0].Provider != "claude" {
			t.Fatalf("request %d: expected candidate 0 first, got %+v", i, rotated)
		}
		if len(rotated) != 2 {
			t.Fatalf("request %d: expected 2 candidates, got %d", i, len(rotated))
		}
	}
	rotated := r.Rotate("rr", "round-robin", 3)
	if rotated[0].Provider != "openrouter" {
		t.Fatalf("request 4: expected candidate 1 first, got %+v", rotated)
	}
	// requests 5-6 stay on openrouter; request 7 wraps back to candidate 0
	for i := 5; i <= 6; i++ {
		rotated := r.Rotate("rr", "round-robin", 3)
		if rotated[0].Provider != "openrouter" {
			t.Fatalf("request %d: expected candidate 1 first, got %+v", i, rotated)
		}
	}
	rotated = r.Rotate("rr", "round-robin", 3)
	if rotated[0].Provider != "claude" {
		t.Fatalf("request 7: expected wrap to candidate 0, got %+v", rotated)
	}
}

func TestComboResolverStickyLimitOneAdvancesImmediately(t *testing.T) {
	r := NewComboResolver()
	r.SetCombos(testComboConfigs())

	if got := r.Rotate("rr", "round-robin", 1); got[0].Provider != "claude" {
		t.Fatalf("request 1: got %+v", got)
	}
	if got := r.Rotate("rr", "round-robin", 1); got[0].Provider != "openrouter" {
		t.Fatalf("request 2: got %+v", got)
	}
}

func TestComboResolverSetCombosClearsCursor(t *testing.T) {
	r := NewComboResolver()
	r.SetCombos(testComboConfigs())

	r.Rotate("rr", "round-robin", 3)
	r.Rotate("rr", "round-robin", 3)
	r.Rotate("rr", "round-robin", 3)
	// cursor should now be on candidate 1; reload clears it back to 0.
	r.SetCombos(testComboConfigs())
	if got := r.Rotate("rr", "round-robin", 3); got[0].Provider != "claude" {
		t.Fatalf("after reload, rotation must restart at candidate 0, got %+v", got)
	}
}

func TestComboResolverConcurrentRotate(t *testing.T) {
	r := NewComboResolver()
	r.SetCombos(testComboConfigs())

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if _, ok := r.Resolve("daily"); !ok {
					t.Error("Resolve(daily) failed under concurrency")
				}
				if got := r.Rotate("rr", "round-robin", 2); len(got) != 2 {
					t.Errorf("Rotate returned %d candidates", len(got))
				}
			}
		}()
	}
	wg.Wait()
}

func TestComboResolverOrderUsesConfigStrategy(t *testing.T) {
	r := NewComboResolver()
	r.SetCombos(testComboConfigs())

	if got, ok := r.Order("daily"); !ok || got[0].Provider != "claude" {
		t.Fatalf("Order(daily) = %+v, %v; want config order starting claude", got, ok)
	}
	for i := 0; i < 3; i++ {
		if got, ok := r.Order("rr"); !ok || got[0].Provider != "claude" {
			t.Fatalf("Order(rr) round %d = %+v, %v; want sticky block on claude", i, got, ok)
		}
	}
	if got, ok := r.Order("rr"); !ok || got[0].Provider != "openrouter" {
		t.Fatalf("Order(rr) after sticky block = %+v, %v; want openrouter", got, ok)
	}
	if _, ok := r.Order("not-a-combo"); ok {
		t.Fatal("expected unknown combo to not order")
	}
}

func TestComboResolverManagerWiring(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	cfg := &internalconfig.Config{Combos: testComboConfigs()}
	manager.SetConfig(cfg)

	if manager.ComboResolver() == nil {
		t.Fatal("expected a combo resolver on the manager")
	}
	if _, ok := manager.ComboResolver().Resolve("daily"); !ok {
		t.Fatal("manager resolver should resolve daily after SetConfig")
	}
}
