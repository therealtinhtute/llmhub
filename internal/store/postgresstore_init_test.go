package store

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInitializeConfigSeedOnceIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LLMHUB_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("LLMHUB_POSTGRES_TEST_DSN is not set")
	}

	ctx := context.Background()
	schema := fmt.Sprintf("llmhub_test_init_%d", time.Now().UnixNano())
	store, err := NewPostgresStore(ctx, PostgresStoreConfig{
		DSN:    dsn,
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer store.Close()
	t.Cleanup(func() { dropTestSchema(t, dsn, schema) })

	seeded, version, err := store.InitializeConfig(ctx, []byte("port: 8317\n"))
	if err != nil {
		t.Fatalf("InitializeConfig first: %v", err)
	}
	if !seeded || version != 1 {
		t.Fatalf("first InitializeConfig = seeded:%v version:%d", seeded, version)
	}

	seeded, version, err = store.InitializeConfig(ctx, []byte("port: 9999\n"))
	if err != nil {
		t.Fatalf("InitializeConfig second: %v", err)
	}
	if seeded || version != 1 {
		t.Fatalf("second InitializeConfig = seeded:%v version:%d", seeded, version)
	}

	snapshot, err := store.LoadConfig(ctx)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got := string(snapshot.Content); got != "port: 8317\n" {
		t.Fatalf("config snapshot = %q, want first seed content", got)
	}
}
