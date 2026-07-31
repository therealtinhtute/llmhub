package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

const cooldownStateTable = "cooldown_store"

var _ cliproxyauth.CooldownStateStoreProvider = (*PostgresStore)(nil)
var _ cliproxyauth.CooldownStateStore = (*postgresCooldownStateStore)(nil)

type postgresCooldownStateKey struct {
	authID string
	model  string
}

type postgresCooldownStateRecord struct {
	key       postgresCooldownStateKey
	content   []byte
	updatedAt time.Time
}

type postgresCooldownStateVersion struct {
	updatedAt time.Time
}

type postgresCooldownStateStore struct {
	store    *PostgresStore
	mu       sync.Mutex
	previous map[postgresCooldownStateKey]postgresCooldownStateVersion
}

func (s *PostgresStore) ensureCooldownSchema(ctx context.Context) error {
	table := s.fullTableName(cooldownStateTable)
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			auth_id TEXT NOT NULL CHECK (octet_length(auth_id) BETWEEN 1 AND 256),
			model TEXT NOT NULL DEFAULT '' CHECK (octet_length(model) <= 256),
			content JSONB NOT NULL CHECK (jsonb_typeof(content) = 'object'),
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (auth_id, model)
		)
	`, table)); err != nil {
		return fmt.Errorf("postgres store: create cooldown state table: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s ON %s (deleted, updated_at)",
		s.indexName("cooldown_store_active_idx"),
		table,
	)); err != nil {
		return fmt.Errorf("postgres store: create cooldown state index: %w", err)
	}
	return nil
}

// CooldownStateStore returns the PostgreSQL-backed runtime cooldown store.
func (s *PostgresStore) CooldownStateStore() cliproxyauth.CooldownStateStore {
	if s == nil {
		return nil
	}
	return s.cooldownStore
}

func (s *postgresCooldownStateStore) Load(ctx context.Context) (records []cliproxyauth.CooldownStateRecord, err error) {
	if s == nil || s.store == nil || s.store.db == nil {
		return nil, fmt.Errorf("postgres cooldown store: not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := fmt.Sprintf("SELECT content, updated_at FROM %s WHERE deleted = FALSE", s.store.fullTableName(cooldownStateTable))
	rows, err := s.store.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres cooldown store: load state: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("postgres cooldown store: close state rows: %w", closeErr))
		}
	}()

	records = make([]cliproxyauth.CooldownStateRecord, 0)
	previous := make(map[postgresCooldownStateKey]postgresCooldownStateVersion)
	for rows.Next() {
		var content []byte
		var updatedAt time.Time
		if err = rows.Scan(&content, &updatedAt); err != nil {
			return nil, fmt.Errorf("postgres cooldown store: scan state: %w", err)
		}
		var record cliproxyauth.CooldownStateRecord
		if err = json.Unmarshal(content, &record); err != nil {
			return nil, fmt.Errorf("postgres cooldown store: decode state: %w", err)
		}
		record, err = record.Normalize(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("postgres cooldown store: invalid stored state: %w", err)
		}
		key := cooldownStateKey(record)
		records = append(records, record)
		previous[key] = postgresCooldownStateVersion{updatedAt: normalizePostgresCooldownTime(updatedAt, record.UpdatedAt)}
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres cooldown store: iterate state: %w", err)
	}
	s.previous = previous
	return records, nil
}

func (s *postgresCooldownStateStore) Save(ctx context.Context, records []cliproxyauth.CooldownStateRecord) error {
	if s == nil || s.store == nil || s.store.db == nil {
		return fmt.Errorf("postgres cooldown store: not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := normalizePostgresCooldownTime(time.Now(), time.Time{})
	current := make(map[postgresCooldownStateKey]postgresCooldownStateVersion, len(records))
	encoded := make([]postgresCooldownStateRecord, 0, len(records))
	for i := range records {
		record, err := records[i].Normalize(now)
		if err != nil {
			return fmt.Errorf("postgres cooldown store: validate state: %w", err)
		}
		key := cooldownStateKey(record)
		if _, exists := current[key]; exists {
			return fmt.Errorf("postgres cooldown store: duplicate state for auth %q model %q", key.authID, key.model)
		}
		content, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("postgres cooldown store: encode state for %q: %w", key.authID, err)
		}
		current[key] = postgresCooldownStateVersion{updatedAt: record.UpdatedAt}
		encoded = append(encoded, postgresCooldownStateRecord{key: key, content: content, updatedAt: record.UpdatedAt})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres cooldown store: begin save: %w", err)
	}
	upsertQuery := fmt.Sprintf(`
		INSERT INTO %s AS target (auth_id, model, content, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, FALSE, NOW(), $4)
		ON CONFLICT (auth_id, model) DO UPDATE SET
			content = EXCLUDED.content,
			deleted = FALSE,
			updated_at = EXCLUDED.updated_at
		WHERE target.updated_at <= EXCLUDED.updated_at
	`, s.store.fullTableName(cooldownStateTable))
	for _, record := range encoded {
		if _, err = tx.ExecContext(ctx, upsertQuery, record.key.authID, record.key.model, record.content, record.updatedAt); err != nil {
			return rollbackPostgresCooldownTransaction(tx, fmt.Errorf("postgres cooldown store: save state for %q: %w", record.key.authID, err))
		}
	}
	deleteQuery := fmt.Sprintf(`
		INSERT INTO %s AS target (auth_id, model, content, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, TRUE, NOW(), $4)
		ON CONFLICT (auth_id, model) DO UPDATE SET
			content = EXCLUDED.content,
			deleted = TRUE,
			updated_at = EXCLUDED.updated_at
		WHERE NOT target.deleted AND target.updated_at <= $5
	`, s.store.fullTableName(cooldownStateTable))
	for key, previous := range s.previous {
		if _, exists := current[key]; exists {
			continue
		}
		deletedAt := now
		if !deletedAt.After(previous.updatedAt) {
			deletedAt = previous.updatedAt.Add(time.Microsecond)
		}
		if _, err = tx.ExecContext(ctx, deleteQuery, key.authID, key.model, []byte(`{}`), deletedAt, previous.updatedAt); err != nil {
			return rollbackPostgresCooldownTransaction(tx, fmt.Errorf("postgres cooldown store: clear state for %q: %w", key.authID, err))
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("postgres cooldown store: commit save: %w", err)
	}
	s.previous = current
	return nil
}

func cooldownStateKey(record cliproxyauth.CooldownStateRecord) postgresCooldownStateKey {
	return postgresCooldownStateKey{
		authID: strings.TrimSpace(record.AuthID),
		model:  strings.TrimSpace(record.Model),
	}
}

func normalizePostgresCooldownTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		value = fallback
	}
	return value.UTC().Truncate(time.Microsecond)
}

func rollbackPostgresCooldownTransaction(tx *sql.Tx, operationErr error) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
		return errors.Join(operationErr, fmt.Errorf("postgres cooldown store: rollback save: %w", rollbackErr))
	}
	return operationErr
}
