package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/therealtinhtute/llmhub/internal/nativeproviders"
)

const nativeProviderTable = "native_provider_resources"

func (s *PostgresStore) ensureNativeProviderSchema(ctx context.Context) error {
	table := s.fullTableName(nativeProviderTable)
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			provider TEXT NOT NULL,
			id TEXT NOT NULL,
			content JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (provider, id)
		)
	`, table)); err != nil {
		return fmt.Errorf("postgres store: create native provider table: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (provider, updated_at)", s.indexName("native_provider_resources_updated_idx"), table)); err != nil {
		return fmt.Errorf("postgres store: create native provider index: %w", err)
	}
	return nil
}

// ListNativeProviderResources lists structured provider-owned records without decoding their payload.
func (s *PostgresStore) ListNativeProviderResources(ctx context.Context, provider string) ([]nativeproviders.ResourceRecord, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return nil, fmt.Errorf("postgres store: native provider is empty")
	}
	query := fmt.Sprintf("SELECT provider, id, content, created_at, updated_at FROM %s WHERE provider = $1 ORDER BY id", s.fullTableName(nativeProviderTable))
	rows, err := s.db.QueryContext(ctx, query, provider)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list native provider resources: %w", err)
	}
	defer rows.Close()

	resources := make([]nativeproviders.ResourceRecord, 0, 8)
	for rows.Next() {
		var record nativeproviders.ResourceRecord
		if err := rows.Scan(&record.Provider, &record.ID, &record.Payload, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres store: scan native provider resource: %w", err)
		}
		resources = append(resources, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate native provider resources: %w", err)
	}
	return resources, nil
}

// SaveNativeProviderResource upserts one structured provider-owned record.
func (s *PostgresStore) SaveNativeProviderResource(ctx context.Context, provider, id string, payload []byte) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	id = strings.TrimSpace(id)
	if provider == "" || id == "" {
		return fmt.Errorf("postgres store: native provider and id are required")
	}
	if !json.Valid(payload) {
		return fmt.Errorf("postgres store: native provider payload is not valid json")
	}
	query := fmt.Sprintf(`
		INSERT INTO %s (provider, id, content, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (provider, id)
		DO UPDATE SET content = EXCLUDED.content, updated_at = NOW()
	`, s.fullTableName(nativeProviderTable))
	if _, err := s.db.ExecContext(ctx, query, provider, id, json.RawMessage(payload)); err != nil {
		return fmt.Errorf("postgres store: save native provider resource: %w", err)
	}
	return nil
}

// DeleteNativeProviderResource removes one provider-owned record. Missing records are harmless.
func (s *PostgresStore) DeleteNativeProviderResource(ctx context.Context, provider, id string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	id = strings.TrimSpace(id)
	if provider == "" || id == "" {
		return fmt.Errorf("postgres store: native provider and id are required")
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE provider = $1 AND id = $2", s.fullTableName(nativeProviderTable))
	if _, err := s.db.ExecContext(ctx, query, provider, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("postgres store: delete native provider resource: %w", err)
	}
	return nil
}

var _ nativeproviders.Store = (*PostgresStore)(nil)
