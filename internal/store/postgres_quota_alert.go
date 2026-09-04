package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/therealtinhtute/llmhub/internal/quotaalert"
)

const (
	quotaAlertSettingsTable           = "quota_alert_settings"
	quotaAlertProviderSettingsTable   = "quota_alert_provider_settings"
	quotaAlertStateTable              = "quota_alert_state"
	quotaAlertEventsTable             = "quota_alert_events"
	quotaNotificationBatchesTable     = "quota_notification_batches"
	quotaNotificationBatchEventsTable = "quota_notification_batch_events"
	quotaAlertSettingsID              = 1
	defaultNotificationClaimLimit     = 10
	maxCollectionCommitItems          = 1000
	maxLoadStateIdentities            = 16_384
	maxLoadStatesBatchSize            = 1000
	maxNotificationBatchPayloadBytes  = 256 << 10
	maxQuotaAlertCursorLength         = 2048
	quotaAlertLockCleanupTimeout      = 5 * time.Second
)

var _ quotaalert.Store = (*PostgresStore)(nil)

func (s *PostgresStore) ensureQuotaAlertSchema(ctx context.Context) error {
	providers := "'claude', 'codex', 'gemini-cli', 'antigravity', 'kimi', 'xai', 'kiro'"
	statements := []struct {
		name  string
		query string
	}{
		{
			name: "settings table",
			query: fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					id SMALLINT PRIMARY KEY CHECK (id = 1),
					enabled BOOLEAN NOT NULL DEFAULT FALSE,
					poll_interval_seconds BIGINT NOT NULL DEFAULT 300 CHECK (poll_interval_seconds BETWEEN 60 AND 86400),
					warning_threshold DOUBLE PRECISION NOT NULL DEFAULT 10 CHECK (warning_threshold >= 0 AND warning_threshold <= 100),
					notify_recovery BOOLEAN NOT NULL DEFAULT FALSE,
					reminder_interval_seconds BIGINT NOT NULL DEFAULT 0 CHECK (reminder_interval_seconds >= 0 AND (reminder_interval_seconds = 0 OR reminder_interval_seconds >= poll_interval_seconds)),
					telegram_enabled BOOLEAN NOT NULL DEFAULT FALSE,
					telegram_chat_id TEXT NOT NULL DEFAULT '' CHECK (octet_length(telegram_chat_id) <= %d),
					telegram_secret_version SMALLINT CHECK (telegram_secret_version IS NULL OR telegram_secret_version = %d),
					telegram_secret_key_id TEXT CHECK (telegram_secret_key_id IS NULL OR octet_length(telegram_secret_key_id) <= %d),
					telegram_secret_nonce BYTEA CHECK (telegram_secret_nonce IS NULL OR octet_length(telegram_secret_nonce) = %d),
					telegram_secret_ciphertext BYTEA CHECK (telegram_secret_ciphertext IS NULL OR octet_length(telegram_secret_ciphertext) BETWEEN %d AND %d),
					revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					CHECK (
						(telegram_secret_version IS NULL AND telegram_secret_key_id IS NULL AND telegram_secret_nonce IS NULL AND telegram_secret_ciphertext IS NULL)
						OR
						(telegram_secret_version IS NOT NULL AND telegram_secret_key_id IS NOT NULL AND telegram_secret_nonce IS NOT NULL AND telegram_secret_ciphertext IS NOT NULL)
					),
					CHECK (NOT telegram_enabled OR (length(btrim(telegram_chat_id)) > 0 AND telegram_secret_ciphertext IS NOT NULL))
				)
			`,
				s.fullTableName(quotaAlertSettingsTable),
				quotaalert.MaxTelegramChatIDLength,
				quotaalert.SecretCipherVersion,
				quotaalert.MaxSecretKeyIDLength,
				quotaalert.SecretNonceSize,
				quotaalert.SecretCiphertextOverhead,
				quotaalert.MaxSecretValueLength+quotaalert.SecretCiphertextOverhead,
			),
		},
		{
			name: "provider settings table",
			query: fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					provider TEXT PRIMARY KEY CHECK (provider IN (%s)),
					enabled BOOLEAN NOT NULL,
					warning_threshold DOUBLE PRECISION CHECK (warning_threshold >= 0 AND warning_threshold <= 100),
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				)
			`, s.fullTableName(quotaAlertProviderSettingsTable), providers),
		},
		{
			name: "state table",
			query: fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					auth_id TEXT NOT NULL CHECK (octet_length(auth_id) BETWEEN 1 AND %d),
					provider TEXT NOT NULL CHECK (provider IN (%s)),
					resource TEXT NOT NULL CHECK (octet_length(resource) BETWEEN 1 AND %d),
					window_key TEXT NOT NULL CHECK (octet_length(window_key) BETWEEN 1 AND %d),
					auth_label TEXT NOT NULL CHECK (octet_length(auth_label) BETWEEN 1 AND %d),
					alert_state TEXT NOT NULL CHECK (alert_state IN ('healthy', 'warning', 'exhausted', 'unknown')),
					collection_health TEXT NOT NULL CHECK (collection_health IN ('reliable', 'unknown')),
					remaining DOUBLE PRECISION CHECK (remaining >= 0 AND remaining <= 100),
					reset_at TIMESTAMPTZ,
					observed_at TIMESTAMPTZ NOT NULL,
					transitioned_at TIMESTAMPTZ NOT NULL,
					updated_at TIMESTAMPTZ NOT NULL,
					revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
					PRIMARY KEY (auth_id, provider, resource, window_key)
				)
			`,
				s.fullTableName(quotaAlertStateTable),
				quotaalert.MaxIdentityFieldLength,
				providers,
				quotaalert.MaxIdentityFieldLength,
				quotaalert.MaxIdentityFieldLength,
				quotaalert.MaxAuthLabelLength,
			),
		},
		{
			name: "events table",
			query: fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					id TEXT PRIMARY KEY CHECK (octet_length(id) BETWEEN 1 AND %d),
					auth_id TEXT NOT NULL CHECK (octet_length(auth_id) BETWEEN 1 AND %d),
					provider TEXT NOT NULL CHECK (provider IN (%s)),
					resource TEXT NOT NULL CHECK (octet_length(resource) BETWEEN 1 AND %d),
					window_key TEXT NOT NULL CHECK (octet_length(window_key) BETWEEN 1 AND %d),
					auth_label TEXT NOT NULL CHECK (octet_length(auth_label) BETWEEN 1 AND %d),
					kind TEXT NOT NULL CHECK (kind IN ('warning', 'exhausted', 'recovery', 'reminder')),
					from_state TEXT NOT NULL CHECK (from_state IN ('healthy', 'warning', 'exhausted', 'unknown')),
					to_state TEXT NOT NULL CHECK (to_state IN ('healthy', 'warning', 'exhausted', 'unknown')),
					remaining DOUBLE PRECISION CHECK (remaining >= 0 AND remaining <= 100),
					reset_at TIMESTAMPTZ,
					occurred_at TIMESTAMPTZ NOT NULL,
					acknowledged_at TIMESTAMPTZ,
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				)
			`,
				s.fullTableName(quotaAlertEventsTable),
				quotaalert.MaxTransitionEventIDLength,
				quotaalert.MaxIdentityFieldLength,
				providers,
				quotaalert.MaxIdentityFieldLength,
				quotaalert.MaxIdentityFieldLength,
				quotaalert.MaxAuthLabelLength,
			),
		},
		{
			name: "notification batches table",
			query: fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					id TEXT PRIMARY KEY CHECK (octet_length(id) BETWEEN 1 AND %d),
					provider TEXT NOT NULL CHECK (provider IN (%s)),
					events JSONB NOT NULL CHECK (octet_length(events::text) <= %d),
					status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
					attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
					available_at TIMESTAMPTZ NOT NULL,
					claimable_at TIMESTAMPTZ NOT NULL,
					lease_id TEXT,
					lease_until TIMESTAMPTZ,
					sent_at TIMESTAMPTZ,
					failure_code TEXT,
					created_at TIMESTAMPTZ NOT NULL,
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					CHECK ((lease_id IS NULL AND lease_until IS NULL) OR (lease_id IS NOT NULL AND lease_until IS NOT NULL))
				)
			`,
				s.fullTableName(quotaNotificationBatchesTable),
				quotaalert.MaxTransitionEventIDLength,
				providers,
				maxNotificationBatchPayloadBytes,
			),
		},
		{
			name: "notification claim schedule column",
			query: fmt.Sprintf(`
				ALTER TABLE %s
				ADD COLUMN IF NOT EXISTS claimable_at TIMESTAMPTZ
			`, s.fullTableName(quotaNotificationBatchesTable)),
		},
		{
			name: "notification claim schedule backfill",
			query: fmt.Sprintf(`
				UPDATE %s
				SET claimable_at = COALESCE(lease_until, available_at)
				WHERE claimable_at IS NULL
			`, s.fullTableName(quotaNotificationBatchesTable)),
		},
		{
			name: "notification claim schedule requirement",
			query: fmt.Sprintf(`
				ALTER TABLE %s
				ALTER COLUMN claimable_at SET NOT NULL
			`, s.fullTableName(quotaNotificationBatchesTable)),
		},
		{
			name: "notification batch events table",
			query: fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					batch_id TEXT NOT NULL,
					event_id TEXT PRIMARY KEY,
					position SMALLINT NOT NULL CHECK (position >= 0 AND position < %d),
					UNIQUE (batch_id, position)
				)
			`,
				s.fullTableName(quotaNotificationBatchEventsTable),
				quotaalert.MaxNotificationBatchEvents,
			),
		},
		{
			name: "notification batch event tombstone constraints",
			query: fmt.Sprintf(`
				ALTER TABLE %s
					DROP CONSTRAINT IF EXISTS %s,
					DROP CONSTRAINT IF EXISTS %s
			`,
				s.fullTableName(quotaNotificationBatchEventsTable),
				quoteIdentifier(quotaNotificationBatchEventsTable+"_batch_id_fkey"),
				quoteIdentifier(quotaNotificationBatchEventsTable+"_event_id_fkey"),
			),
		},
		{
			name:  "state listing index",
			query: fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (updated_at DESC, auth_id DESC, provider DESC, resource DESC, window_key DESC)", s.indexName("quota_alert_state_list_desc_idx"), s.fullTableName(quotaAlertStateTable)),
		},
		{
			name:  "event listing index",
			query: fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (occurred_at DESC, id DESC)", s.indexName("quota_alert_events_list_idx"), s.fullTableName(quotaAlertEventsTable)),
		},
		{
			name:  "notification claim index",
			query: fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (status, claimable_at, created_at, id)", s.indexName("quota_notification_claimable_idx"), s.fullTableName(quotaNotificationBatchesTable)),
		},
		{
			name:  "terminal notification retention index",
			query: fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (updated_at, id) WHERE status IN ('sent', 'failed')", s.indexName("quota_notification_terminal_retention_idx"), s.fullTableName(quotaNotificationBatchesTable)),
		},
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement.query); err != nil {
			return fmt.Errorf("postgres store: create quota alert %s: %w", statement.name, err)
		}
	}
	seedQuery := fmt.Sprintf(`
		INSERT INTO %s (id, enabled, poll_interval_seconds, warning_threshold, revision)
		VALUES ($1, FALSE, $2, $3, 1)
		ON CONFLICT (id) DO NOTHING
	`, s.fullTableName(quotaAlertSettingsTable))
	if _, err := s.db.ExecContext(ctx, seedQuery, quotaAlertSettingsID, int64(quotaalert.DefaultPollInterval/time.Second), float64(quotaalert.DefaultWarningThreshold)); err != nil {
		return fmt.Errorf("postgres store: seed quota alert settings: %w", err)
	}
	return nil
}

// LoadSettings loads the singleton settings and provider overrides from one snapshot.
func (s *PostgresStore) LoadSettings(ctx context.Context) (quotaalert.Settings, error) {
	settings, _, err := s.LoadSettingsWithSecret(ctx)
	return settings, err
}

// LoadSettingsWithSecret loads settings and encrypted Telegram material from one snapshot.
func (s *PostgresStore) LoadSettingsWithSecret(ctx context.Context) (quotaalert.Settings, *quotaalert.EncryptedSecret, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: begin quota alert settings read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := fmt.Sprintf(`
		SELECT enabled, poll_interval_seconds, warning_threshold, notify_recovery,
		       reminder_interval_seconds, telegram_enabled, telegram_chat_id,
		       telegram_secret_version, telegram_secret_key_id,
		       telegram_secret_nonce, telegram_secret_ciphertext, revision
		FROM %s WHERE id = $1
	`, s.fullTableName(quotaAlertSettingsTable))
	var settings quotaalert.Settings
	var pollSeconds, reminderSeconds int64
	var secretVersion sql.NullInt64
	var secretKeyID sql.NullString
	var secretNonce, secretCiphertext []byte
	if err = tx.QueryRowContext(ctx, query, quotaAlertSettingsID).Scan(
		&settings.Enabled,
		&pollSeconds,
		&settings.WarningThreshold,
		&settings.NotifyRecovery,
		&reminderSeconds,
		&settings.Telegram.Enabled,
		&settings.Telegram.ChatID,
		&secretVersion,
		&secretKeyID,
		&secretNonce,
		&secretCiphertext,
		&settings.Revision,
	); err != nil {
		return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: load quota alert settings: %w", err)
	}
	secret, err := quotaAlertEncryptedSecret(secretVersion, secretKeyID, secretNonce, secretCiphertext)
	if err != nil {
		return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: load encrypted quota alert secret: %w", err)
	}
	settings.PollInterval = time.Duration(pollSeconds) * time.Second
	settings.ReminderInterval = time.Duration(reminderSeconds) * time.Second
	settings.Telegram.TokenConfigured = secret != nil

	providerQuery := fmt.Sprintf("SELECT provider, enabled, warning_threshold FROM %s ORDER BY provider", s.fullTableName(quotaAlertProviderSettingsTable))
	rows, err := tx.QueryContext(ctx, providerQuery)
	if err != nil {
		return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: load quota alert provider settings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var override quotaalert.ProviderOverride
		var threshold sql.NullFloat64
		if err = rows.Scan(&override.Provider, &override.Enabled, &threshold); err != nil {
			return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: scan quota alert provider settings: %w", err)
		}
		if threshold.Valid {
			value := quotaalert.Percentage(threshold.Float64)
			override.WarningThreshold = &value
		}
		settings.ProviderOverrides = append(settings.ProviderOverrides, override)
	}
	if err = rows.Err(); err != nil {
		return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: iterate quota alert provider settings: %w", err)
	}
	if err = rows.Close(); err != nil {
		return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: close quota alert provider settings: %w", err)
	}
	if err = settings.Validate(); err != nil {
		return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: invalid stored quota alert settings: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return quotaalert.Settings{}, nil, fmt.Errorf("postgres store: commit quota alert settings read: %w", err)
	}
	return settings, secret, nil
}

// SaveSettings updates settings while preserving the stored Telegram secret.
func (s *PostgresStore) SaveSettings(ctx context.Context, expectedRevision int64, settings quotaalert.Settings) (quotaalert.Settings, error) {
	return s.SaveSettingsWithSecret(ctx, expectedRevision, settings, quotaalert.PreserveSecret(), nil, "")
}

// SaveSettingsWithSecret atomically updates settings, provider overrides, and explicit secret intent.
func (s *PostgresStore) SaveSettingsWithSecret(
	ctx context.Context,
	expectedRevision int64,
	settings quotaalert.Settings,
	update quotaalert.SecretUpdate,
	cipher *quotaalert.SecretCipher,
	purpose string,
) (quotaalert.Settings, error) {
	if expectedRevision <= 0 {
		return quotaalert.Settings{}, fmt.Errorf("postgres store: expected quota alert settings revision must be positive")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return quotaalert.Settings{}, fmt.Errorf("postgres store: begin quota alert settings update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	currentRevision, currentSecret, err := s.lockQuotaAlertSettings(ctx, tx)
	if err != nil {
		return quotaalert.Settings{}, err
	}
	if currentRevision != expectedRevision {
		return quotaalert.Settings{}, fmt.Errorf("postgres store: quota alert settings revision conflict")
	}
	nextSecret, err := update.Apply(currentSecret, cipher, purpose)
	if err != nil {
		return quotaalert.Settings{}, fmt.Errorf("postgres store: apply quota alert secret update: %w", err)
	}
	settings.Telegram.TokenConfigured = nextSecret != nil
	settings.Revision = currentRevision + 1
	if err = settings.Validate(); err != nil {
		return quotaalert.Settings{}, fmt.Errorf("postgres store: validate quota alert settings: %w", err)
	}
	settings.PollInterval = settings.PollInterval.Truncate(time.Second)
	settings.ReminderInterval = settings.ReminderInterval.Truncate(time.Second)
	settings.Telegram.ChatID = strings.TrimSpace(settings.Telegram.ChatID)
	sort.Slice(settings.ProviderOverrides, func(left, right int) bool {
		return settings.ProviderOverrides[left].Provider < settings.ProviderOverrides[right].Provider
	})

	var secretVersion, secretKeyID, secretNonce, secretCiphertext any
	if nextSecret != nil {
		secretVersion = int16(nextSecret.Version())
		secretKeyID = nextSecret.KeyID()
		secretNonce = nextSecret.Nonce()
		secretCiphertext = nextSecret.Ciphertext()
	}
	updateQuery := fmt.Sprintf(`
		UPDATE %s
		SET enabled = $1,
		    poll_interval_seconds = $2,
		    warning_threshold = $3,
		    notify_recovery = $4,
		    reminder_interval_seconds = $5,
		    telegram_enabled = $6,
		    telegram_chat_id = $7,
		    telegram_secret_version = $8,
		    telegram_secret_key_id = $9,
		    telegram_secret_nonce = $10,
		    telegram_secret_ciphertext = $11,
		    revision = $12,
		    updated_at = NOW()
		WHERE id = $13
	`, s.fullTableName(quotaAlertSettingsTable))
	if _, err = tx.ExecContext(
		ctx,
		updateQuery,
		settings.Enabled,
		int64(settings.PollInterval/time.Second),
		float64(settings.WarningThreshold),
		settings.NotifyRecovery,
		int64(settings.ReminderInterval/time.Second),
		settings.Telegram.Enabled,
		strings.TrimSpace(settings.Telegram.ChatID),
		secretVersion,
		secretKeyID,
		secretNonce,
		secretCiphertext,
		settings.Revision,
		quotaAlertSettingsID,
	); err != nil {
		return quotaalert.Settings{}, fmt.Errorf("postgres store: update quota alert settings: %w", err)
	}

	if _, err = tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", s.fullTableName(quotaAlertProviderSettingsTable))); err != nil {
		return quotaalert.Settings{}, fmt.Errorf("postgres store: replace quota alert provider settings: %w", err)
	}
	providerQuery := fmt.Sprintf(`
		INSERT INTO %s (provider, enabled, warning_threshold, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, s.fullTableName(quotaAlertProviderSettingsTable))
	for _, override := range settings.ProviderOverrides {
		var threshold any
		if override.WarningThreshold != nil {
			threshold = float64(*override.WarningThreshold)
		}
		if _, err = tx.ExecContext(ctx, providerQuery, override.Provider, override.Enabled, threshold); err != nil {
			return quotaalert.Settings{}, fmt.Errorf("postgres store: save quota alert provider setting: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return quotaalert.Settings{}, fmt.Errorf("postgres store: commit quota alert settings update: %w", err)
	}
	return settings, nil
}

func (s *PostgresStore) lockQuotaAlertSettings(ctx context.Context, tx *sql.Tx) (int64, *quotaalert.EncryptedSecret, error) {
	query := fmt.Sprintf(`
		SELECT revision, telegram_secret_version, telegram_secret_key_id,
		       telegram_secret_nonce, telegram_secret_ciphertext
		FROM %s WHERE id = $1 FOR UPDATE
	`, s.fullTableName(quotaAlertSettingsTable))
	var revision int64
	var version sql.NullInt64
	var keyID sql.NullString
	var nonce, ciphertext []byte
	if err := tx.QueryRowContext(ctx, query, quotaAlertSettingsID).Scan(&revision, &version, &keyID, &nonce, &ciphertext); err != nil {
		return 0, nil, fmt.Errorf("postgres store: lock quota alert settings: %w", err)
	}
	secret, err := quotaAlertEncryptedSecret(version, keyID, nonce, ciphertext)
	if err != nil {
		return 0, nil, fmt.Errorf("postgres store: load encrypted quota alert secret: %w", err)
	}
	return revision, secret, nil
}

func quotaAlertEncryptedSecret(version sql.NullInt64, keyID sql.NullString, nonce, ciphertext []byte) (*quotaalert.EncryptedSecret, error) {
	if !version.Valid && !keyID.Valid && nonce == nil && ciphertext == nil {
		return nil, nil
	}
	if !version.Valid || !keyID.Valid || nonce == nil || ciphertext == nil || version.Int64 < 0 || version.Int64 > 255 {
		return nil, fmt.Errorf("encrypted quota alert secret columns are incomplete")
	}
	secret, err := quotaalert.NewEncryptedSecret(uint8(version.Int64), keyID.String, nonce, ciphertext)
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

// TryAcquireCollection obtains database-wide ownership for one schema's collection cycle.
func (s *PostgresStore) TryAcquireCollection(ctx context.Context) (quotaalert.CollectionLease, bool, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("postgres store: acquire quota alert collection connection: %w", err)
	}
	key := s.quotaAlertAdvisoryLockKey()
	var acquired bool
	if err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false, fmt.Errorf("postgres store: acquire quota alert collection lock: %w", err)
	}
	if !acquired {
		_ = conn.Close()
		return nil, false, nil
	}
	return &postgresQuotaAlertLease{owner: s, conn: conn, key: key}, true, nil
}

func (s *PostgresStore) quotaAlertAdvisoryLockKey() int64 {
	return s.quotaAlertLockKey(quotaAlertSettingsTable)
}

func (s *PostgresStore) quotaAlertRetentionLockKey() int64 {
	return s.quotaAlertLockKey(quotaNotificationBatchEventsTable)
}

func (s *PostgresStore) quotaAlertLockKey(purpose string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(strings.TrimSpace(s.cfg.Schema)))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(purpose))
	return int64(hash.Sum64())
}

func (s *PostgresStore) lockQuotaAlertRetention(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(
		ctx,
		"SELECT pg_advisory_xact_lock($1)",
		s.quotaAlertRetentionLockKey(),
	); err != nil {
		return fmt.Errorf("postgres store: lock quota alert retention: %w", err)
	}
	return nil
}

type postgresQuotaAlertLease struct {
	mu       sync.Mutex
	owner    *PostgresStore
	conn     *sql.Conn
	key      int64
	released bool
}

func (l *postgresQuotaAlertLease) Release(context.Context) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return nil
	}
	if l.conn == nil {
		l.released = true
		return nil
	}

	cleanupCtx, cancel := context.WithTimeout(context.Background(), quotaAlertLockCleanupTimeout)
	defer cancel()
	var unlocked bool
	if err := l.conn.QueryRowContext(cleanupCtx, "SELECT pg_advisory_unlock($1)", l.key).Scan(&unlocked); err != nil {
		_ = l.conn.Raw(func(any) error { return driver.ErrBadConn })
		_ = l.conn.Close()
		l.conn = nil
		l.released = true
		return fmt.Errorf("postgres store: release quota alert collection lock: %w", err)
	}
	closeErr := l.conn.Close()
	l.conn = nil
	l.released = true
	if !unlocked {
		return fmt.Errorf("postgres store: quota alert collection lock was not held")
	}
	if closeErr != nil {
		return fmt.Errorf("postgres store: close quota alert collection connection: %w", closeErr)
	}
	return nil
}

// LoadStates loads current state for the requested durable identities in bounded chunks.
func (s *PostgresStore) LoadStates(ctx context.Context, identities []quotaalert.StateIdentity) ([]quotaalert.CurrentState, error) {
	if len(identities) == 0 {
		return nil, nil
	}
	if len(identities) > maxLoadStateIdentities {
		return nil, fmt.Errorf(
			"postgres store: quota alert state load exceeds %d identities",
			maxLoadStateIdentities,
		)
	}
	normalized := make([]quotaalert.StateIdentity, 0, len(identities))
	seen := make(map[quotaalert.StateIdentity]struct{}, len(identities))
	for _, identity := range identities {
		identity, err := identity.Normalize()
		if err != nil {
			return nil, fmt.Errorf("postgres store: invalid quota alert identity: %w", err)
		}
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		normalized = append(normalized, identity)
	}

	states := make([]quotaalert.CurrentState, 0, len(normalized))
	for start := 0; start < len(normalized); start += maxLoadStatesBatchSize {
		end := min(start+maxLoadStatesBatchSize, len(normalized))
		chunk, err := s.loadQuotaAlertStateChunk(ctx, normalized[start:end])
		if err != nil {
			return nil, err
		}
		states = append(states, chunk...)
	}
	sort.Slice(states, func(left, right int) bool {
		leftIdentity := states[left].Identity
		rightIdentity := states[right].Identity
		if leftIdentity.AuthID != rightIdentity.AuthID {
			return leftIdentity.AuthID < rightIdentity.AuthID
		}
		if leftIdentity.Provider != rightIdentity.Provider {
			return leftIdentity.Provider < rightIdentity.Provider
		}
		if leftIdentity.Resource != rightIdentity.Resource {
			return leftIdentity.Resource < rightIdentity.Resource
		}
		return leftIdentity.Window < rightIdentity.Window
	})
	return states, nil
}

func (s *PostgresStore) loadQuotaAlertStateChunk(ctx context.Context, identities []quotaalert.StateIdentity) ([]quotaalert.CurrentState, error) {
	args := make([]any, 0, len(identities)*4)
	placeholders := make([]string, 0, len(identities))
	for index, identity := range identities {
		base := index*4 + 1
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d)", base, base+1, base+2, base+3))
		args = append(args, identity.AuthID, identity.Provider, identity.Resource, identity.Window)
	}
	query := fmt.Sprintf(`
		SELECT auth_id, provider, resource, window_key, auth_label, alert_state,
		       collection_health, remaining, reset_at, observed_at, transitioned_at,
		       updated_at, revision
		FROM %s
		WHERE (auth_id, provider, resource, window_key) IN (%s)
		ORDER BY auth_id, provider, resource, window_key
	`, s.fullTableName(quotaAlertStateTable), strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres store: load quota alert states: %w", err)
	}
	defer rows.Close()
	states := make([]quotaalert.CurrentState, 0, len(identities))
	for rows.Next() {
		state, errScan := scanQuotaAlertState(rows)
		if errScan != nil {
			return nil, fmt.Errorf("postgres store: scan quota alert state: %w", errScan)
		}
		states = append(states, state)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate quota alert states: %w", err)
	}
	return states, nil
}

// CommitCollection atomically persists one actively owned collection cycle.
func (s *PostgresStore) CommitCollection(ctx context.Context, lease quotaalert.CollectionLease, commit quotaalert.CollectionCommit) error {
	owned, ok := lease.(*postgresQuotaAlertLease)
	if !ok || owned == nil {
		return fmt.Errorf("postgres store: quota alert collection lease is invalid")
	}
	owned.mu.Lock()
	defer owned.mu.Unlock()
	if owned.released || owned.owner != s || owned.conn == nil || owned.key != s.quotaAlertAdvisoryLockKey() {
		return fmt.Errorf("postgres store: quota alert collection lease is not active")
	}
	itemCount := len(commit.States) + len(commit.RemovedStates) +
		len(commit.Events) + len(commit.Batches)
	if itemCount > maxCollectionCommitItems {
		return fmt.Errorf(
			"postgres store: quota alert collection commit exceeds %d items",
			maxCollectionCommitItems,
		)
	}
	normalizedStates := make([]quotaalert.CurrentState, len(commit.States))
	statesByIdentity := make(map[quotaalert.StateIdentity]quotaalert.CurrentState, len(commit.States))
	for index, state := range commit.States {
		state, err := state.Normalize()
		if err != nil {
			return fmt.Errorf("postgres store: invalid quota alert state: %w", err)
		}
		if _, exists := statesByIdentity[state.Identity]; exists {
			return fmt.Errorf("postgres store: duplicate quota alert state in collection commit")
		}
		normalizedStates[index] = state
		statesByIdentity[state.Identity] = state
	}
	normalizedRemovals := make([]quotaalert.StateIdentity, len(commit.RemovedStates))
	removedIdentities := make(map[quotaalert.StateIdentity]struct{}, len(commit.RemovedStates))
	for index, identity := range commit.RemovedStates {
		identity, err := identity.Normalize()
		if err != nil {
			return fmt.Errorf("postgres store: invalid removed quota alert state identity: %w", err)
		}
		if _, exists := statesByIdentity[identity]; exists {
			return fmt.Errorf("postgres store: quota alert state cannot be updated and removed in one collection commit")
		}
		if _, exists := removedIdentities[identity]; exists {
			return fmt.Errorf("postgres store: duplicate removed quota alert state in collection commit")
		}
		normalizedRemovals[index] = identity
		removedIdentities[identity] = struct{}{}
	}

	normalizedEvents := make([]quotaalert.TransitionEvent, len(commit.Events))
	eventsByID := make(map[string]quotaalert.TransitionEvent, len(commit.Events))
	eventsByIdentity := make(map[quotaalert.StateIdentity]struct{}, len(commit.Events))
	for index, event := range commit.Events {
		event, err := event.Normalize()
		if err != nil {
			return fmt.Errorf("postgres store: invalid quota alert event: %w", err)
		}
		if _, exists := eventsByID[event.ID]; exists {
			return fmt.Errorf("postgres store: duplicate quota alert event %q in collection commit", event.ID)
		}
		if _, exists := eventsByIdentity[event.Identity]; exists {
			return fmt.Errorf("postgres store: duplicate quota alert event identity in collection commit")
		}
		if !event.AcknowledgedAt.IsZero() {
			return fmt.Errorf("postgres store: collection event %q cannot carry acknowledgement state", event.ID)
		}
		normalizedEvents[index] = event
		eventsByID[event.ID] = event
		eventsByIdentity[event.Identity] = struct{}{}
	}
	for _, event := range normalizedEvents {
		state, exists := statesByIdentity[event.Identity]
		if !exists {
			return fmt.Errorf("postgres store: quota alert event %q has no matching collection state", event.ID)
		}
		if err := validateQuotaAlertEventState(event, state); err != nil {
			return fmt.Errorf("postgres store: quota alert event %q does not match collection state: %w", event.ID, err)
		}
	}
	type batchEventAssignment struct {
		batchID  string
		eventID  string
		position int
	}
	assignments := make([]batchEventAssignment, 0, len(normalizedEvents))
	batchedEvents := make(map[string]struct{})
	for _, batch := range commit.Batches {
		for position, event := range batch.Events() {
			committed, exists := eventsByID[event.ID]
			if !exists || committed != event {
				return fmt.Errorf("postgres store: quota notification batch event %q does not match the collection commit", event.ID)
			}
			if _, exists = batchedEvents[event.ID]; exists {
				return fmt.Errorf("postgres store: quota alert event %q appears in multiple notification batches", event.ID)
			}
			batchedEvents[event.ID] = struct{}{}
			assignments = append(assignments, batchEventAssignment{batchID: batch.ID(), eventID: event.ID, position: position})
		}
	}
	if commit.SettingsRevision <= 0 {
		return fmt.Errorf("postgres store: quota alert collection settings revision must be positive")
	}

	tx, err := owned.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres store: begin quota alert collection commit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = s.lockQuotaAlertRetention(ctx, tx); err != nil {
		return err
	}
	if err = s.requireQuotaAlertSettingsRevision(ctx, tx, commit.SettingsRevision); err != nil {
		return err
	}
	if err = s.validateQuotaAlertTransitionHistory(ctx, tx, normalizedEvents, statesByIdentity); err != nil {
		return err
	}
	if err = s.deleteQuotaAlertStates(ctx, tx, normalizedRemovals); err != nil {
		return err
	}
	for _, state := range normalizedStates {
		if err = s.upsertQuotaAlertState(ctx, tx, state); err != nil {
			return err
		}
	}
	for _, event := range normalizedEvents {
		if err = s.insertQuotaAlertEvent(ctx, tx, event); err != nil {
			return err
		}
	}
	for _, batch := range commit.Batches {
		if err = s.insertQuotaNotificationBatch(ctx, tx, batch); err != nil {
			return err
		}
	}
	for _, assignment := range assignments {
		if err = s.insertQuotaNotificationBatchEvent(ctx, tx, assignment.batchID, assignment.eventID, assignment.position); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("postgres store: commit quota alert collection: %w", err)
	}
	return nil
}

func (s *PostgresStore) requireQuotaAlertSettingsRevision(ctx context.Context, tx *sql.Tx, expected int64) error {
	query := fmt.Sprintf("SELECT revision FROM %s WHERE id = $1 FOR SHARE", s.fullTableName(quotaAlertSettingsTable))
	var current int64
	if err := tx.QueryRowContext(ctx, query, quotaAlertSettingsID).Scan(&current); err != nil {
		return fmt.Errorf("postgres store: lock quota alert settings revision: %w", err)
	}
	if current != expected {
		return fmt.Errorf("postgres store: quota alert collection settings revision conflict")
	}
	return nil
}

func (s *PostgresStore) validateQuotaAlertTransitionHistory(
	ctx context.Context,
	tx *sql.Tx,
	events []quotaalert.TransitionEvent,
	states map[quotaalert.StateIdentity]quotaalert.CurrentState,
) error {
	stateQuery := fmt.Sprintf(`
		SELECT auth_id, provider, resource, window_key, auth_label, alert_state,
		       collection_health, remaining, reset_at, observed_at, transitioned_at,
		       updated_at, revision
		FROM %s
		WHERE auth_id = $1 AND provider = $2 AND resource = $3 AND window_key = $4
		FOR UPDATE
	`, s.fullTableName(quotaAlertStateTable))
	eventQuery := fmt.Sprintf(`
		SELECT id, auth_id, provider, resource, window_key, auth_label, kind,
		       from_state, to_state, remaining, reset_at, occurred_at, acknowledged_at
		FROM %s WHERE id = $1
		FOR SHARE
	`, s.fullTableName(quotaAlertEventsTable))
	for _, event := range events {
		previous, err := scanQuotaAlertState(tx.QueryRowContext(
			ctx,
			stateQuery,
			event.Identity.AuthID,
			event.Identity.Provider,
			event.Identity.Resource,
			event.Identity.Window,
		))
		if err == sql.ErrNoRows {
			if event.From != quotaalert.AlertUnknown ||
				(event.Kind != quotaalert.TransitionWarning &&
					event.Kind != quotaalert.TransitionExhausted) {
				return fmt.Errorf(
					"postgres store: quota alert event %q requires persisted transition history",
					event.ID,
				)
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("postgres store: lock previous quota alert state: %w", err)
		}
		next := states[event.Identity]
		if quotaAlertStateContentEqual(previous, next) {
			persisted, eventErr := scanQuotaAlertEvent(tx.QueryRowContext(ctx, eventQuery, event.ID))
			if eventErr == nil && quotaAlertEventContentEqual(persisted, event) {
				continue
			}
			if eventErr != nil && eventErr != sql.ErrNoRows {
				return fmt.Errorf("postgres store: lock existing quota alert event: %w", eventErr)
			}
		}
		if event.From != previous.Alert {
			return fmt.Errorf(
				"postgres store: quota alert event %q starts from %q, want persisted %q",
				event.ID,
				event.From,
				previous.Alert,
			)
		}
		if !event.OccurredAt.After(previous.ObservedAt) {
			return fmt.Errorf("postgres store: quota alert event %q does not follow the persisted observation", event.ID)
		}
	}
	return nil
}

func quotaAlertStateContentEqual(left, right quotaalert.CurrentState) bool {
	left, leftErr := left.Normalize()
	right, rightErr := right.Normalize()
	if leftErr != nil || rightErr != nil {
		return false
	}
	left.Revision = 0
	right.Revision = 0
	return left == right
}

func quotaAlertEventContentEqual(left, right quotaalert.TransitionEvent) bool {
	left.AcknowledgedAt = time.Time{}
	right.AcknowledgedAt = time.Time{}
	left, leftErr := left.Normalize()
	right, rightErr := right.Normalize()
	return leftErr == nil && rightErr == nil && left == right
}

func validateQuotaAlertEventState(event quotaalert.TransitionEvent, state quotaalert.CurrentState) error {
	if state.Health != quotaalert.CollectionReliable {
		return fmt.Errorf("event requires reliable collection state")
	}
	if state.AuthLabel != event.AuthLabel || state.Alert != event.To {
		return fmt.Errorf("event target or auth label differs")
	}
	if state.RemainingKnown != event.RemainingKnown || (state.RemainingKnown && state.Remaining != event.Remaining) {
		return fmt.Errorf("event remaining evidence differs")
	}
	if state.ResetKnown != event.ResetKnown || (state.ResetKnown && !state.ResetAt.Equal(event.ResetAt)) {
		return fmt.Errorf("event reset evidence differs")
	}
	if event.OccurredAt.After(state.UpdatedAt) {
		return fmt.Errorf("event occurs after its collection state")
	}
	if event.Kind != quotaalert.TransitionReminder && !state.TransitionedAt.Equal(event.OccurredAt) {
		return fmt.Errorf("event transition time differs")
	}
	return nil
}

func (s *PostgresStore) deleteQuotaAlertStates(ctx context.Context, tx *sql.Tx, identities []quotaalert.StateIdentity) error {
	for start := 0; start < len(identities); start += maxLoadStatesBatchSize {
		end := min(start+maxLoadStatesBatchSize, len(identities))
		args := make([]any, 0, (end-start)*4)
		placeholders := make([]string, 0, end-start)
		for index, identity := range identities[start:end] {
			base := index*4 + 1
			placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d)", base, base+1, base+2, base+3))
			args = append(args, identity.AuthID, identity.Provider, identity.Resource, identity.Window)
		}
		query := fmt.Sprintf(`
			DELETE FROM %s
			WHERE (auth_id, provider, resource, window_key) IN (%s)
		`, s.fullTableName(quotaAlertStateTable), strings.Join(placeholders, ","))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("postgres store: remove quota alert states: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) upsertQuotaAlertState(ctx context.Context, tx *sql.Tx, state quotaalert.CurrentState) error {
	state, err := state.Normalize()
	if err != nil {
		return fmt.Errorf("postgres store: invalid quota alert state: %w", err)
	}
	var remaining, resetAt any
	if state.RemainingKnown {
		remaining = float64(state.Remaining)
	}
	if state.ResetKnown {
		resetAt = state.ResetAt
	}
	query := fmt.Sprintf(`
		INSERT INTO %s AS current_state (
			auth_id, provider, resource, window_key, auth_label, alert_state,
			collection_health, remaining, reset_at, observed_at, transitioned_at,
			updated_at, revision
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 1)
		ON CONFLICT (auth_id, provider, resource, window_key)
		DO UPDATE SET
			auth_label = EXCLUDED.auth_label,
			alert_state = EXCLUDED.alert_state,
			collection_health = EXCLUDED.collection_health,
			remaining = EXCLUDED.remaining,
			reset_at = EXCLUDED.reset_at,
			observed_at = EXCLUDED.observed_at,
			transitioned_at = EXCLUDED.transitioned_at,
			updated_at = EXCLUDED.updated_at,
			revision = CASE
				WHEN EXCLUDED.observed_at > current_state.observed_at THEN current_state.revision + 1
				ELSE current_state.revision
			END
		WHERE EXCLUDED.observed_at > current_state.observed_at
		   OR (
			EXCLUDED.observed_at = current_state.observed_at
			AND current_state.auth_label = EXCLUDED.auth_label
			AND current_state.alert_state = EXCLUDED.alert_state
			AND current_state.collection_health = EXCLUDED.collection_health
			AND current_state.remaining IS NOT DISTINCT FROM EXCLUDED.remaining
			AND current_state.reset_at IS NOT DISTINCT FROM EXCLUDED.reset_at
			AND current_state.transitioned_at = EXCLUDED.transitioned_at
			AND current_state.updated_at = EXCLUDED.updated_at
		   )
	`, s.fullTableName(quotaAlertStateTable))
	result, err := tx.ExecContext(
		ctx,
		query,
		state.Identity.AuthID,
		state.Identity.Provider,
		state.Identity.Resource,
		state.Identity.Window,
		state.AuthLabel,
		state.Alert,
		state.Health,
		remaining,
		resetAt,
		state.ObservedAt,
		state.TransitionedAt,
		state.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres store: upsert quota alert state: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres store: inspect quota alert state upsert: %w", err)
	}
	if updated == 0 {
		return fmt.Errorf("postgres store: quota alert collection contains a stale or conflicting observation")
	}
	return nil
}

func (s *PostgresStore) insertQuotaAlertEvent(ctx context.Context, tx *sql.Tx, event quotaalert.TransitionEvent) error {
	event, err := event.Normalize()
	if err != nil {
		return fmt.Errorf("postgres store: invalid quota alert event: %w", err)
	}
	var remaining, resetAt, acknowledgedAt any
	if event.RemainingKnown {
		remaining = float64(event.Remaining)
	}
	if event.ResetKnown {
		resetAt = event.ResetAt
	}
	if !event.AcknowledgedAt.IsZero() {
		acknowledgedAt = event.AcknowledgedAt
	}
	query := fmt.Sprintf(`
		WITH retired AS (
			SELECT EXISTS (
				SELECT 1 FROM %s WHERE event_id = $1
			) AND NOT EXISTS (
				SELECT 1 FROM %s WHERE id = $1
			) AS value
		), persisted AS (
			INSERT INTO %s AS existing_event (
				id, auth_id, provider, resource, window_key, auth_label, kind,
				from_state, to_state, remaining, reset_at, occurred_at, acknowledged_at
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			FROM retired
			WHERE NOT retired.value
			ON CONFLICT (id) DO UPDATE SET id = existing_event.id
			WHERE existing_event.auth_id = EXCLUDED.auth_id
			  AND existing_event.provider = EXCLUDED.provider
			  AND existing_event.resource = EXCLUDED.resource
			  AND existing_event.window_key = EXCLUDED.window_key
			  AND existing_event.auth_label = EXCLUDED.auth_label
			  AND existing_event.kind = EXCLUDED.kind
			  AND existing_event.from_state = EXCLUDED.from_state
			  AND existing_event.to_state = EXCLUDED.to_state
			  AND existing_event.remaining IS NOT DISTINCT FROM EXCLUDED.remaining
			  AND existing_event.reset_at IS NOT DISTINCT FROM EXCLUDED.reset_at
			  AND existing_event.occurred_at = EXCLUDED.occurred_at
			RETURNING id
		)
		SELECT CASE
			WHEN EXISTS (SELECT 1 FROM persisted) THEN 1
			WHEN (SELECT value FROM retired) THEN -1
			ELSE 0
		END
	`,
		s.fullTableName(quotaNotificationBatchEventsTable),
		s.fullTableName(quotaAlertEventsTable),
		s.fullTableName(quotaAlertEventsTable),
	)
	var outcome int
	if err = tx.QueryRowContext(
		ctx,
		query,
		event.ID,
		event.Identity.AuthID,
		event.Identity.Provider,
		event.Identity.Resource,
		event.Identity.Window,
		event.AuthLabel,
		event.Kind,
		event.From,
		event.To,
		remaining,
		resetAt,
		event.OccurredAt,
		acknowledgedAt,
	).Scan(&outcome); err != nil {
		return fmt.Errorf("postgres store: insert quota alert event: %w", err)
	}
	switch outcome {
	case 1:
		return nil
	case -1:
		return fmt.Errorf(
			"postgres store: quota alert event %q is retained as a notification assignment tombstone",
			event.ID,
		)
	default:
		return fmt.Errorf("postgres store: quota alert event %q conflicts with existing content", event.ID)
	}
}

func (s *PostgresStore) insertQuotaNotificationBatch(ctx context.Context, tx *sql.Tx, batch quotaalert.NotificationBatch) error {
	if strings.TrimSpace(batch.ID()) == "" || batch.CreatedAt().IsZero() || len(batch.Events()) == 0 {
		return fmt.Errorf("postgres store: quota notification batch is incomplete")
	}
	if err := batch.Provider().Validate(); err != nil {
		return fmt.Errorf("postgres store: invalid quota notification provider: %w", err)
	}
	eventsJSON, err := json.Marshal(batch.Events())
	if err != nil {
		return fmt.Errorf("postgres store: encode quota notification batch: %w", err)
	}
	if len(eventsJSON) > maxNotificationBatchPayloadBytes {
		return fmt.Errorf("postgres store: quota notification batch payload exceeds %d bytes", maxNotificationBatchPayloadBytes)
	}
	query := fmt.Sprintf(`
		WITH retired AS (
			SELECT EXISTS (
				SELECT 1 FROM %s WHERE batch_id = $1
			) AND NOT EXISTS (
				SELECT 1 FROM %s WHERE id = $1
			) AS value
		)
		INSERT INTO %s (
			id, provider, events, available_at, claimable_at,
			created_at, updated_at
		)
		SELECT $1, $2, $3, $4, $4, $4, NOW()
		FROM retired
		WHERE NOT retired.value
		ON CONFLICT (id) DO NOTHING
	`,
		s.fullTableName(quotaNotificationBatchEventsTable),
		s.fullTableName(quotaNotificationBatchesTable),
		s.fullTableName(quotaNotificationBatchesTable),
	)
	if _, err = tx.ExecContext(ctx, query, batch.ID(), batch.Provider(), json.RawMessage(eventsJSON), batch.CreatedAt().UTC()); err != nil {
		return fmt.Errorf("postgres store: insert quota notification batch: %w", err)
	}
	return nil
}

func (s *PostgresStore) insertQuotaNotificationBatchEvent(ctx context.Context, tx *sql.Tx, batchID, eventID string, position int) error {
	query := fmt.Sprintf(`
		INSERT INTO %s AS existing_assignment (batch_id, event_id, position)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id) DO UPDATE SET event_id = existing_assignment.event_id
		WHERE existing_assignment.batch_id = EXCLUDED.batch_id
		  AND existing_assignment.position = EXCLUDED.position
		RETURNING event_id
	`, s.fullTableName(quotaNotificationBatchEventsTable))
	var persistedEventID string
	if err := tx.QueryRowContext(ctx, query, batchID, eventID, position).Scan(&persistedEventID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("postgres store: quota alert event %q is already assigned to another notification batch", eventID)
		}
		return fmt.Errorf("postgres store: assign quota alert event to notification batch: %w", err)
	}
	return nil
}

// ListStates lists current states using a bounded opaque cursor.
func (s *PostgresStore) ListStates(ctx context.Context, page quotaalert.PageRequest) (quotaalert.Page[quotaalert.CurrentState], error) {
	page, err := page.Normalize()
	if err != nil {
		return quotaalert.Page[quotaalert.CurrentState]{}, err
	}
	baseSelect := fmt.Sprintf(`
		SELECT auth_id, provider, resource, window_key, auth_label, alert_state,
		       collection_health, remaining, reset_at, observed_at, transitioned_at,
		       updated_at, revision
		FROM %s
	`, s.fullTableName(quotaAlertStateTable))
	var rows *sql.Rows
	if page.Cursor == "" {
		rows, err = s.db.QueryContext(ctx, baseSelect+" ORDER BY updated_at DESC, auth_id DESC, provider DESC, resource DESC, window_key DESC LIMIT $1", page.Limit+1)
	} else {
		cursor, errDecode := decodeQuotaStateCursor(page.Cursor)
		if errDecode != nil {
			return quotaalert.Page[quotaalert.CurrentState]{}, errDecode
		}
		rows, err = s.db.QueryContext(ctx, baseSelect+`
			WHERE (updated_at, auth_id, provider, resource, window_key) < ($1, $2, $3, $4, $5)
			ORDER BY updated_at DESC, auth_id DESC, provider DESC, resource DESC, window_key DESC
			LIMIT $6
		`, cursor.UpdatedAt, cursor.AuthID, cursor.Provider, cursor.Resource, cursor.Window, page.Limit+1)
	}
	if err != nil {
		return quotaalert.Page[quotaalert.CurrentState]{}, fmt.Errorf("postgres store: list quota alert states: %w", err)
	}
	defer rows.Close()
	items := make([]quotaalert.CurrentState, 0, page.Limit+1)
	for rows.Next() {
		state, errScan := scanQuotaAlertState(rows)
		if errScan != nil {
			return quotaalert.Page[quotaalert.CurrentState]{}, fmt.Errorf("postgres store: scan quota alert state: %w", errScan)
		}
		items = append(items, state)
	}
	if err = rows.Err(); err != nil {
		return quotaalert.Page[quotaalert.CurrentState]{}, fmt.Errorf("postgres store: iterate quota alert states: %w", err)
	}
	result := quotaalert.Page[quotaalert.CurrentState]{Items: items}
	if len(items) > page.Limit {
		last := items[page.Limit-1]
		result.Items = items[:page.Limit]
		result.NextCursor, err = encodeQuotaCursor(quotaStateCursor{
			UpdatedAt: last.UpdatedAt,
			AuthID:    last.Identity.AuthID,
			Provider:  last.Identity.Provider,
			Resource:  last.Identity.Resource,
			Window:    last.Identity.Window,
		})
		if err != nil {
			return quotaalert.Page[quotaalert.CurrentState]{}, err
		}
	}
	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuotaAlertState(scanner rowScanner) (quotaalert.CurrentState, error) {
	var state quotaalert.CurrentState
	var remaining sql.NullFloat64
	var resetAt sql.NullTime
	if err := scanner.Scan(
		&state.Identity.AuthID,
		&state.Identity.Provider,
		&state.Identity.Resource,
		&state.Identity.Window,
		&state.AuthLabel,
		&state.Alert,
		&state.Health,
		&remaining,
		&resetAt,
		&state.ObservedAt,
		&state.TransitionedAt,
		&state.UpdatedAt,
		&state.Revision,
	); err != nil {
		return quotaalert.CurrentState{}, err
	}
	if remaining.Valid {
		state.RemainingKnown = true
		state.Remaining = quotaalert.Percentage(remaining.Float64)
	}
	if resetAt.Valid {
		state.ResetKnown = true
		state.ResetAt = resetAt.Time
	}
	return state, nil
}

// ListEvents lists recent events using a bounded opaque cursor.
func (s *PostgresStore) ListEvents(ctx context.Context, page quotaalert.PageRequest) (quotaalert.Page[quotaalert.TransitionEvent], error) {
	page, err := page.Normalize()
	if err != nil {
		return quotaalert.Page[quotaalert.TransitionEvent]{}, err
	}
	baseSelect := fmt.Sprintf(`
		SELECT id, auth_id, provider, resource, window_key, auth_label, kind,
		       from_state, to_state, remaining, reset_at, occurred_at, acknowledged_at
		FROM %s
	`, s.fullTableName(quotaAlertEventsTable))
	var rows *sql.Rows
	if page.Cursor == "" {
		rows, err = s.db.QueryContext(ctx, baseSelect+" ORDER BY occurred_at DESC, id DESC LIMIT $1", page.Limit+1)
	} else {
		cursor, errDecode := decodeQuotaEventCursor(page.Cursor)
		if errDecode != nil {
			return quotaalert.Page[quotaalert.TransitionEvent]{}, errDecode
		}
		rows, err = s.db.QueryContext(ctx, baseSelect+`
			WHERE (occurred_at, id) < ($1, $2)
			ORDER BY occurred_at DESC, id DESC
			LIMIT $3
		`, cursor.OccurredAt, cursor.ID, page.Limit+1)
	}
	if err != nil {
		return quotaalert.Page[quotaalert.TransitionEvent]{}, fmt.Errorf("postgres store: list quota alert events: %w", err)
	}
	defer rows.Close()
	items := make([]quotaalert.TransitionEvent, 0, page.Limit+1)
	for rows.Next() {
		event, errScan := scanQuotaAlertEvent(rows)
		if errScan != nil {
			return quotaalert.Page[quotaalert.TransitionEvent]{}, fmt.Errorf("postgres store: scan quota alert event: %w", errScan)
		}
		items = append(items, event)
	}
	if err = rows.Err(); err != nil {
		return quotaalert.Page[quotaalert.TransitionEvent]{}, fmt.Errorf("postgres store: iterate quota alert events: %w", err)
	}
	result := quotaalert.Page[quotaalert.TransitionEvent]{Items: items}
	if len(items) > page.Limit {
		last := items[page.Limit-1]
		result.Items = items[:page.Limit]
		result.NextCursor, err = encodeQuotaCursor(quotaEventCursor{OccurredAt: last.OccurredAt, ID: last.ID})
		if err != nil {
			return quotaalert.Page[quotaalert.TransitionEvent]{}, err
		}
	}
	return result, nil
}

// ListEventDeliveryStatuses summarizes Telegram outbox state for transition events.
func (s *PostgresStore) ListEventDeliveryStatuses(ctx context.Context, eventIDs []string) (map[string]quotaalert.EventDeliveryStatus, error) {
	if len(eventIDs) == 0 {
		return map[string]quotaalert.EventDeliveryStatus{}, nil
	}
	if len(eventIDs) > quotaalert.MaxPageSize {
		return nil, fmt.Errorf("postgres store: event delivery status request must not exceed %d events", quotaalert.MaxPageSize)
	}
	placeholders := make([]string, 0, len(eventIDs))
	args := make([]any, 0, len(eventIDs))
	for _, eventID := range eventIDs {
		eventID = strings.TrimSpace(eventID)
		if eventID == "" {
			return nil, fmt.Errorf("postgres store: event delivery status event ID is required")
		}
		args = append(args, eventID)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	query := fmt.Sprintf(`
		SELECT DISTINCT ON (link.event_id)
		       link.event_id, batch.status, batch.failure_code, batch.attempt_count, batch.sent_at
		FROM %s AS link
		JOIN %s AS batch ON batch.id = link.batch_id
		WHERE link.event_id IN (%s)
		ORDER BY link.event_id, batch.created_at DESC, batch.id DESC
	`, s.fullTableName(quotaNotificationBatchEventsTable), s.fullTableName(quotaNotificationBatchesTable), strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list quota alert event delivery statuses: %w", err)
	}
	defer rows.Close()
	statuses := make(map[string]quotaalert.EventDeliveryStatus, len(eventIDs))
	for rows.Next() {
		var eventID string
		var status quotaalert.EventDeliveryStatus
		var failureCode sql.NullString
		var sentAt sql.NullTime
		if err = rows.Scan(&eventID, &status.Status, &failureCode, &status.AttemptCount, &sentAt); err != nil {
			return nil, fmt.Errorf("postgres store: scan quota alert event delivery status: %w", err)
		}
		if failureCode.Valid {
			status.FailureCode = failureCode.String
		}
		if sentAt.Valid {
			status.SentAt = sentAt.Time
		}
		statuses[eventID] = status
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate quota alert event delivery statuses: %w", err)
	}
	return statuses, nil
}

func scanQuotaAlertEvent(scanner rowScanner) (quotaalert.TransitionEvent, error) {
	var event quotaalert.TransitionEvent
	var remaining sql.NullFloat64
	var resetAt, acknowledgedAt sql.NullTime
	if err := scanner.Scan(
		&event.ID,
		&event.Identity.AuthID,
		&event.Identity.Provider,
		&event.Identity.Resource,
		&event.Identity.Window,
		&event.AuthLabel,
		&event.Kind,
		&event.From,
		&event.To,
		&remaining,
		&resetAt,
		&event.OccurredAt,
		&acknowledgedAt,
	); err != nil {
		return quotaalert.TransitionEvent{}, err
	}
	if remaining.Valid {
		event.RemainingKnown = true
		event.Remaining = quotaalert.Percentage(remaining.Float64)
	}
	if resetAt.Valid {
		event.ResetKnown = true
		event.ResetAt = resetAt.Time
	}
	if acknowledgedAt.Valid {
		event.AcknowledgedAt = acknowledgedAt.Time
	}
	return event, nil
}

// AcknowledgeEvent records the first acknowledgement and is idempotent thereafter.
func (s *PostgresStore) AcknowledgeEvent(ctx context.Context, eventID string, acknowledgedAt time.Time) error {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" || acknowledgedAt.IsZero() {
		return fmt.Errorf("postgres store: quota alert acknowledgement is incomplete")
	}
	acknowledgedAt = acknowledgedAt.UTC().Truncate(time.Microsecond)
	query := fmt.Sprintf(`
		WITH target AS (
			SELECT occurred_at FROM %s WHERE id = $2
		), updated AS (
			UPDATE %s
			SET acknowledged_at = COALESCE(acknowledged_at, $1)
			WHERE id = $2 AND $1 >= occurred_at
			RETURNING 1
		)
		SELECT CASE
			WHEN EXISTS (SELECT 1 FROM updated) THEN 1
			WHEN EXISTS (SELECT 1 FROM target) THEN 0
			ELSE -1
		END
	`, s.fullTableName(quotaAlertEventsTable), s.fullTableName(quotaAlertEventsTable))
	var outcome int
	if err := s.db.QueryRowContext(ctx, query, acknowledgedAt, eventID).Scan(&outcome); err != nil {
		return fmt.Errorf("postgres store: acknowledge quota alert event: %w", err)
	}
	switch outcome {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("postgres store: quota alert acknowledgement cannot precede the event")
	default:
		return sql.ErrNoRows
	}
}

// PruneEvents deletes a bounded oldest-first set of events before the cutoff.
func (s *PostgresStore) PruneEvents(ctx context.Context, before time.Time, limit int) (int64, error) {
	if before.IsZero() {
		return 0, fmt.Errorf("postgres store: quota alert event retention cutoff is required")
	}
	if limit < 1 || limit > quotaalert.MaxPageSize {
		return 0, fmt.Errorf("postgres store: quota alert event retention limit must be between 1 and %d", quotaalert.MaxPageSize)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("postgres store: begin quota alert event retention: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = s.lockQuotaAlertRetention(ctx, tx); err != nil {
		return 0, err
	}
	query := fmt.Sprintf(`
		WITH doomed AS (
			SELECT event.id
			FROM %s AS event
			WHERE event.occurred_at < $1
			ORDER BY event.occurred_at, event.id
			LIMIT $2
			FOR UPDATE OF event
		), deleted_events AS (
			DELETE FROM %s AS event
			USING doomed
			WHERE event.id = doomed.id
			RETURNING event.id
		), cleaned_assignments AS (
			DELETE FROM %s AS assignment
			USING deleted_events
			WHERE assignment.event_id = deleted_events.id
			  AND NOT EXISTS (
				SELECT 1 FROM %s AS batch WHERE batch.id = assignment.batch_id
			  )
		)
		SELECT COUNT(*) FROM deleted_events
	`,
		s.fullTableName(quotaAlertEventsTable),
		s.fullTableName(quotaAlertEventsTable),
		s.fullTableName(quotaNotificationBatchEventsTable),
		s.fullTableName(quotaNotificationBatchesTable),
	)
	var deleted int64
	if err := tx.QueryRowContext(ctx, query, before.UTC(), limit).Scan(&deleted); err != nil {
		return 0, fmt.Errorf("postgres store: prune quota alert events: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("postgres store: commit quota alert event retention: %w", err)
	}
	return deleted, nil
}

// PruneNotificationBatches deletes a bounded oldest-first set of terminal batches.
func (s *PostgresStore) PruneNotificationBatches(ctx context.Context, before time.Time, limit int) (int64, error) {
	if before.IsZero() {
		return 0, fmt.Errorf("postgres store: quota notification retention cutoff is required")
	}
	if limit < 1 || limit > quotaalert.MaxPageSize {
		return 0, fmt.Errorf("postgres store: quota notification retention limit must be between 1 and %d", quotaalert.MaxPageSize)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("postgres store: begin quota notification retention: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = s.lockQuotaAlertRetention(ctx, tx); err != nil {
		return 0, err
	}
	query := fmt.Sprintf(`
		WITH doomed AS (
			SELECT id
			FROM %s
			WHERE status IN ('sent', 'failed')
			  AND updated_at < $1
			ORDER BY updated_at, id
			LIMIT $2
			FOR UPDATE
		), deleted_batches AS (
			DELETE FROM %s AS batch
			USING doomed
			WHERE batch.id = doomed.id
			RETURNING batch.id
		), cleaned_assignments AS (
			DELETE FROM %s AS assignment
			USING deleted_batches
			WHERE assignment.batch_id = deleted_batches.id
			  AND NOT EXISTS (
				SELECT 1 FROM %s AS event WHERE event.id = assignment.event_id
			  )
		)
		SELECT COUNT(*) FROM deleted_batches
	`,
		s.fullTableName(quotaNotificationBatchesTable),
		s.fullTableName(quotaNotificationBatchesTable),
		s.fullTableName(quotaNotificationBatchEventsTable),
		s.fullTableName(quotaAlertEventsTable),
	)
	var deleted int64
	if err := tx.QueryRowContext(ctx, query, before.UTC(), limit).Scan(&deleted); err != nil {
		return 0, fmt.Errorf("postgres store: prune quota notification batches: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("postgres store: commit quota notification retention: %w", err)
	}
	return deleted, nil
}

// ClaimNotificationBatches leases pending or expired provider batches without blocking other workers.
func (s *PostgresStore) ClaimNotificationBatches(ctx context.Context, options quotaalert.NotificationClaimOptions) ([]quotaalert.NotificationClaim, error) {
	if options.Limit == 0 {
		options.Limit = defaultNotificationClaimLimit
	}
	if options.Limit < 1 || options.Limit > quotaalert.MaxPageSize {
		return nil, fmt.Errorf("postgres store: notification claim limit must be between 1 and %d", quotaalert.MaxPageSize)
	}
	if options.LeaseDuration < quotaalert.MinNotificationLeaseDuration || options.LeaseDuration > quotaalert.MaxNotificationLeaseDuration {
		return nil, fmt.Errorf(
			"postgres store: notification lease duration must be between %s and %s",
			quotaalert.MinNotificationLeaseDuration,
			quotaalert.MaxNotificationLeaseDuration,
		)
	}
	leaseID := uuid.NewString()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("postgres store: begin quota notification claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	query := fmt.Sprintf(`
		WITH clock AS (
			SELECT clock_timestamp() AS now
		), picked AS (
			SELECT batch.id, clock.now
			FROM %s AS batch
			CROSS JOIN clock
			WHERE batch.status = 'pending'
			  AND batch.claimable_at <= clock.now
			ORDER BY batch.claimable_at, batch.created_at, batch.id
			LIMIT $1
			FOR UPDATE OF batch SKIP LOCKED
		)
		UPDATE %s AS batch
		SET lease_id = $2,
		    lease_until = picked.now + ($3 * INTERVAL '1 microsecond'),
		    claimable_at = picked.now + ($3 * INTERVAL '1 microsecond'),
		    attempt_count = batch.attempt_count + 1,
		    updated_at = picked.now
		FROM picked
		WHERE batch.id = picked.id
		RETURNING batch.id, batch.provider, batch.events, batch.created_at,
		          batch.attempt_count
	`, s.fullTableName(quotaNotificationBatchesTable), s.fullTableName(quotaNotificationBatchesTable))
	rows, err := tx.QueryContext(ctx, query, options.Limit, leaseID, options.LeaseDuration.Microseconds())
	if err != nil {
		return nil, fmt.Errorf("postgres store: claim quota notification batches: %w", err)
	}
	defer rows.Close()
	type claimedRow struct {
		id         string
		provider   quotaalert.Provider
		eventsJSON []byte
		createdAt  time.Time
		attempt    int
	}
	rawClaims := make([]claimedRow, 0, options.Limit)
	for rows.Next() {
		var claimed claimedRow
		if err = rows.Scan(&claimed.id, &claimed.provider, &claimed.eventsJSON, &claimed.createdAt, &claimed.attempt); err != nil {
			return nil, fmt.Errorf("postgres store: scan quota notification claim: %w", err)
		}
		rawClaims = append(rawClaims, claimed)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate quota notification claims: %w", err)
	}
	if err = rows.Close(); err != nil {
		return nil, fmt.Errorf("postgres store: close quota notification claims: %w", err)
	}

	claims := make([]quotaalert.NotificationClaim, 0, len(rawClaims))
	quarantineQuery := fmt.Sprintf(`
		UPDATE %s
		SET status = 'failed', failure_code = 'invalid_payload',
		    lease_id = NULL, lease_until = NULL, updated_at = clock_timestamp()
		WHERE id = $1 AND lease_id = $2 AND status = 'pending'
	`, s.fullTableName(quotaNotificationBatchesTable))
	for _, claimed := range rawClaims {
		var batch quotaalert.NotificationBatch
		valid := len(claimed.eventsJSON) <= maxNotificationBatchPayloadBytes
		if valid {
			var events []quotaalert.TransitionEvent
			if err = json.Unmarshal(claimed.eventsJSON, &events); err == nil {
				batch, err = quotaalert.NewNotificationBatch(claimed.provider, events, claimed.createdAt)
				valid = err == nil && batch.ID() == claimed.id
			} else {
				valid = false
			}
		}
		if !valid {
			result, quarantineErr := tx.ExecContext(ctx, quarantineQuery, claimed.id, leaseID)
			if quarantineErr != nil {
				return nil, fmt.Errorf("postgres store: quarantine invalid quota notification batch: %w", quarantineErr)
			}
			updated, rowsErr := result.RowsAffected()
			if rowsErr != nil {
				return nil, fmt.Errorf("postgres store: inspect invalid quota notification quarantine: %w", rowsErr)
			}
			if updated != 1 {
				return nil, fmt.Errorf("postgres store: invalid quota notification lease is no longer active")
			}
			continue
		}
		claims = append(claims, quotaalert.NotificationClaim{
			Batch:   batch,
			LeaseID: leaseID,
			Attempt: claimed.attempt,
		})
	}
	if len(claims) > 0 {
		refreshArgs := make([]any, 0, len(claims)+2)
		refreshArgs = append(refreshArgs, leaseID, options.LeaseDuration.Microseconds())
		refreshIDs := make([]string, 0, len(claims))
		for index, claim := range claims {
			refreshIDs = append(refreshIDs, fmt.Sprintf("$%d", index+3))
			refreshArgs = append(refreshArgs, claim.Batch.ID())
		}
		refreshQuery := fmt.Sprintf(`
			WITH clock AS (
				SELECT clock_timestamp() AS now
			)
			UPDATE %s AS batch
			SET lease_until = clock.now + ($2 * INTERVAL '1 microsecond'),
			    claimable_at = clock.now + ($2 * INTERVAL '1 microsecond'),
			    updated_at = clock.now
			FROM clock
			WHERE batch.id IN (%s)
			  AND batch.lease_id = $1
			  AND batch.status = 'pending'
			RETURNING batch.id, batch.lease_until
		`, s.fullTableName(quotaNotificationBatchesTable), strings.Join(refreshIDs, ","))
		refreshedRows, refreshErr := tx.QueryContext(
			ctx,
			refreshQuery,
			refreshArgs...,
		)
		if refreshErr != nil {
			return nil, fmt.Errorf("postgres store: refresh quota notification leases: %w", refreshErr)
		}
		refreshed := make(map[string]time.Time, len(claims))
		for refreshedRows.Next() {
			var id string
			var leaseUntil time.Time
			if refreshErr = refreshedRows.Scan(&id, &leaseUntil); refreshErr != nil {
				_ = refreshedRows.Close()
				return nil, fmt.Errorf("postgres store: scan refreshed quota notification lease: %w", refreshErr)
			}
			refreshed[id] = leaseUntil
		}
		if refreshErr = refreshedRows.Err(); refreshErr != nil {
			_ = refreshedRows.Close()
			return nil, fmt.Errorf("postgres store: iterate refreshed quota notification leases: %w", refreshErr)
		}
		if refreshErr = refreshedRows.Close(); refreshErr != nil {
			return nil, fmt.Errorf("postgres store: close refreshed quota notification leases: %w", refreshErr)
		}
		if len(refreshed) != len(claims) {
			return nil, fmt.Errorf("postgres store: quota notification lease set changed before commit")
		}
		for index := range claims {
			leaseUntil, exists := refreshed[claims[index].Batch.ID()]
			if !exists {
				return nil, fmt.Errorf("postgres store: quota notification lease is no longer active")
			}
			claims[index].LeaseUntil = leaseUntil
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("postgres store: commit quota notification claim: %w", err)
	}
	sort.Slice(claims, func(left, right int) bool {
		if claims[left].Batch.CreatedAt().Equal(claims[right].Batch.CreatedAt()) {
			return claims[left].Batch.ID() < claims[right].Batch.ID()
		}
		return claims[left].Batch.CreatedAt().Before(claims[right].Batch.CreatedAt())
	})
	return claims, nil
}

// ResolveNotification marks a matching lease sent, permanently failed, or retryable.
func (s *PostgresStore) ResolveNotification(ctx context.Context, result quotaalert.NotificationResult) error {
	result.BatchID = strings.TrimSpace(result.BatchID)
	result.LeaseID = strings.TrimSpace(result.LeaseID)
	if result.BatchID == "" || result.LeaseID == "" {
		return fmt.Errorf("postgres store: quota notification result is incomplete")
	}
	intentCount := 0
	if !result.SentAt.IsZero() {
		intentCount++
	}
	if result.PermanentFailure {
		intentCount++
	}
	if !result.RetryAt.IsZero() {
		intentCount++
	}
	if intentCount != 1 {
		return fmt.Errorf("postgres store: quota notification result requires exactly one sent, retry, or permanent failure state")
	}
	failureCode := sanitizeQuotaNotificationFailureCode(result.FailureCode)
	var query string
	var args []any
	switch {
	case !result.SentAt.IsZero():
		query = fmt.Sprintf(`
			UPDATE %s
			SET status = 'sent', sent_at = $1, failure_code = NULL,
			    lease_id = NULL, lease_until = NULL, updated_at = $1
			WHERE id = $2 AND lease_id = $3 AND status = 'pending'
		`, s.fullTableName(quotaNotificationBatchesTable))
		args = []any{result.SentAt.UTC(), result.BatchID, result.LeaseID}
	case result.PermanentFailure:
		if failureCode == "" {
			failureCode = "delivery_failed"
		}
		query = fmt.Sprintf(`
			UPDATE %s
			SET status = 'failed', failure_code = $1,
			    lease_id = NULL, lease_until = NULL, updated_at = NOW()
			WHERE id = $2 AND lease_id = $3 AND status = 'pending'
		`, s.fullTableName(quotaNotificationBatchesTable))
		args = []any{failureCode, result.BatchID, result.LeaseID}
	case !result.RetryAt.IsZero():
		query = fmt.Sprintf(`
			UPDATE %s
			SET status = 'pending', available_at = $1, claimable_at = $1,
			    failure_code = $2, lease_id = NULL, lease_until = NULL,
			    updated_at = NOW()
			WHERE id = $3 AND lease_id = $4 AND status = 'pending'
		`, s.fullTableName(quotaNotificationBatchesTable))
		args = []any{result.RetryAt.UTC(), nullableString(failureCode), result.BatchID, result.LeaseID}
	default:
		return fmt.Errorf("postgres store: quota notification result requires sent, retry, or permanent failure state")
	}
	updateResult, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("postgres store: resolve quota notification: %w", err)
	}
	count, err := updateResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres store: inspect quota notification resolution: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("postgres store: quota notification lease is no longer active")
	}
	return nil
}

func sanitizeQuotaNotificationFailureCode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "timeout", "rate_limited", "transport_failure", "temporary_failure", "telegram_rejected", "invalid_destination", "unauthorized", "delivery_failed":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type quotaStateCursor struct {
	UpdatedAt time.Time           `json:"updated_at"`
	AuthID    string              `json:"auth_id"`
	Provider  quotaalert.Provider `json:"provider"`
	Resource  string              `json:"resource"`
	Window    string              `json:"window"`
}

type quotaEventCursor struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         string    `json:"id"`
}

func encodeQuotaCursor(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("postgres store: encode quota alert cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeQuotaStateCursor(value string) (quotaStateCursor, error) {
	var cursor quotaStateCursor
	if err := decodeQuotaCursor(value, &cursor); err != nil {
		return quotaStateCursor{}, err
	}
	if cursor.UpdatedAt.IsZero() || cursor.AuthID == "" || cursor.Provider == "" || cursor.Resource == "" || cursor.Window == "" {
		return quotaStateCursor{}, fmt.Errorf("postgres store: invalid quota alert state cursor")
	}
	return cursor, nil
}

func decodeQuotaEventCursor(value string) (quotaEventCursor, error) {
	var cursor quotaEventCursor
	if err := decodeQuotaCursor(value, &cursor); err != nil {
		return quotaEventCursor{}, err
	}
	if cursor.OccurredAt.IsZero() || cursor.ID == "" {
		return quotaEventCursor{}, fmt.Errorf("postgres store: invalid quota alert event cursor")
	}
	return cursor, nil
}

func decodeQuotaCursor(value string, destination any) error {
	value = strings.TrimSpace(value)
	if len(value) > maxQuotaAlertCursorLength {
		return fmt.Errorf("postgres store: invalid quota alert cursor")
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("postgres store: invalid quota alert cursor")
	}
	if err = json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("postgres store: invalid quota alert cursor")
	}
	return nil
}
