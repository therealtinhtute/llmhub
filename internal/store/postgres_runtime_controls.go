package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
)

const (
	runtimeControlSettingsTable = "runtime_control_settings"
	runtimeControlSettingsID    = 1
)

var _ runtimecontrol.SettingsStore = (*PostgresStore)(nil)

func (s *PostgresStore) ensureRuntimeControlSchema(ctx context.Context) error {
	table := s.fullTableName(runtimeControlSettingsTable)
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SMALLINT PRIMARY KEY CHECK (id = 1),
			settings JSONB NOT NULL CHECK (jsonb_typeof(settings) = 'object'),
			revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`, table)); err != nil {
		return fmt.Errorf("postgres store: create runtime control settings table: %w", err)
	}

	defaults := runtimecontrol.DefaultSettings()
	defaults.Revision = 1
	payload, err := json.Marshal(defaults)
	if err != nil {
		return fmt.Errorf("postgres store: encode default runtime controls: %w", err)
	}
	if _, err = s.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, settings, revision)
		VALUES ($1, $2, 1)
		ON CONFLICT (id) DO NOTHING
	`, table), runtimeControlSettingsID, json.RawMessage(payload)); err != nil {
		return fmt.Errorf("postgres store: seed runtime control settings: %w", err)
	}
	return nil
}

// LoadRuntimeSettings loads the database-authoritative runtime controls.
func (s *PostgresStore) LoadRuntimeSettings(ctx context.Context) (runtimecontrol.Settings, error) {
	if s == nil || s.db == nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: not initialized")
	}
	query := fmt.Sprintf("SELECT settings, revision FROM %s WHERE id = $1", s.fullTableName(runtimeControlSettingsTable))
	var payload []byte
	var revision int64
	if err := s.db.QueryRowContext(ctx, query, runtimeControlSettingsID).Scan(&payload, &revision); err != nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: load runtime control settings: %w", err)
	}
	var settings runtimecontrol.Settings
	if err := json.Unmarshal(payload, &settings); err != nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: decode runtime control settings: %w", err)
	}
	if settings.Revision != 0 && settings.Revision != revision {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: runtime control settings revision mismatch")
	}
	settings.Revision = revision
	normalized, err := settings.Normalize()
	if err != nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: invalid stored runtime control settings: %w", err)
	}
	return normalized, nil
}

// SaveRuntimeSettings atomically replaces runtime controls when expectedRevision matches.
func (s *PostgresStore) SaveRuntimeSettings(ctx context.Context, expectedRevision int64, settings runtimecontrol.Settings) (runtimecontrol.Settings, error) {
	if s == nil || s.db == nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: not initialized")
	}
	if expectedRevision <= 0 {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: expected runtime control settings revision must be positive")
	}
	if expectedRevision == math.MaxInt64 {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: runtime control settings revision exhausted")
	}
	settings.Revision = expectedRevision + 1
	normalized, err := settings.Normalize()
	if err != nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: validate runtime control settings: %w", err)
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: encode runtime control settings: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: begin runtime control settings update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	currentRevision, err := s.lockRuntimeControlSettings(ctx, tx)
	if err != nil {
		return runtimecontrol.Settings{}, err
	}
	if currentRevision != expectedRevision {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: %w", runtimecontrol.ErrRevisionConflict)
	}
	query := fmt.Sprintf(`
		UPDATE %s
		SET settings = $1, revision = $2, updated_at = NOW()
		WHERE id = $3
	`, s.fullTableName(runtimeControlSettingsTable))
	if _, err = tx.ExecContext(ctx, query, json.RawMessage(payload), normalized.Revision, runtimeControlSettingsID); err != nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: update runtime control settings: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return runtimecontrol.Settings{}, fmt.Errorf("postgres store: commit runtime control settings update: %w", err)
	}
	return normalized.Clone(), nil
}

func (s *PostgresStore) lockRuntimeControlSettings(ctx context.Context, tx *sql.Tx) (int64, error) {
	query := fmt.Sprintf("SELECT revision FROM %s WHERE id = $1 FOR UPDATE", s.fullTableName(runtimeControlSettingsTable))
	var revision int64
	if err := tx.QueryRowContext(ctx, query, runtimeControlSettingsID).Scan(&revision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("postgres store: runtime control settings row is missing")
		}
		return 0, fmt.Errorf("postgres store: lock runtime control settings: %w", err)
	}
	return revision, nil
}
