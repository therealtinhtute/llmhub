package store

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/quotaalert"
)

func TestPostgresQuotaAlertDDLIdempotenceAndDefaults(t *testing.T) {
	ctx, store, _, schema := newPostgresQuotaAlertTestStore(t)
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema() second call error = %v", err)
	}

	var tableCount int
	if err := store.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_name = ANY($2)
	`, schema, []string{
		quotaAlertSettingsTable,
		quotaAlertProviderSettingsTable,
		quotaAlertStateTable,
		quotaAlertEventsTable,
		quotaNotificationBatchesTable,
		quotaNotificationBatchEventsTable,
	}).Scan(&tableCount); err != nil {
		t.Fatalf("count quota alert tables: %v", err)
	}
	if tableCount != 6 {
		t.Fatalf("quota alert table count = %d, want 6", tableCount)
	}

	if _, err := store.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s SET telegram_chat_id = $1 WHERE id = $2
	`, store.fullTableName(quotaAlertSettingsTable)), strings.Repeat("a", quotaalert.MaxTelegramChatIDLength+1), quotaAlertSettingsID); err == nil {
		t.Fatal("oversized Telegram chat ID bypassed database constraint")
	}
	if _, err := store.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET telegram_secret_version = $1, telegram_secret_key_id = $2,
		    telegram_secret_nonce = $3, telegram_secret_ciphertext = $4
		WHERE id = $5
	`, store.fullTableName(quotaAlertSettingsTable)),
		quotaalert.SecretCipherVersion,
		"key",
		bytes.Repeat([]byte{1}, quotaalert.SecretNonceSize+1),
		bytes.Repeat([]byte{2}, quotaalert.SecretCiphertextOverhead),
		quotaAlertSettingsID,
	); err == nil {
		t.Fatal("oversized Telegram secret nonce bypassed database constraint")
	}
	now := time.Now().UTC()
	if _, err := store.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			auth_id, provider, resource, window_key, auth_label, alert_state,
			collection_health, remaining, observed_at, transitioned_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $9)
	`, store.fullTableName(quotaAlertStateTable)),
		strings.Repeat("a", quotaalert.MaxIdentityFieldLength+1),
		quotaalert.ProviderClaude,
		"messages",
		"five-hour",
		"Account",
		quotaalert.AlertWarning,
		quotaalert.CollectionReliable,
		5,
		now,
	); err == nil {
		t.Fatal("oversized state identity bypassed database constraint")
	}
	if _, err := store.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			id, auth_id, provider, resource, window_key, auth_label, kind,
			from_state, to_state, remaining, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, store.fullTableName(quotaAlertEventsTable)),
		strings.Repeat("e", quotaalert.MaxTransitionEventIDLength+1),
		"auth",
		quotaalert.ProviderClaude,
		"messages",
		"five-hour",
		"Account",
		quotaalert.TransitionWarning,
		quotaalert.AlertHealthy,
		quotaalert.AlertWarning,
		5,
		now,
	); err == nil {
		t.Fatal("oversized event ID bypassed database constraint")
	}
	if _, err := store.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			id, provider, events, available_at, claimable_at, created_at
		)
		VALUES ($1, $2, to_jsonb($3::text), $4, $4, $4)
	`, store.fullTableName(quotaNotificationBatchesTable)),
		strings.Repeat("b", 64),
		quotaalert.ProviderClaude,
		strings.Repeat("x", maxNotificationBatchPayloadBytes+1),
		now,
	); err == nil {
		t.Fatal("oversized notification JSON bypassed database constraint")
	}

	settings, err := store.LoadSettings(ctx)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	if settings.Enabled || settings.PollInterval != quotaalert.DefaultPollInterval || settings.WarningThreshold != quotaalert.DefaultWarningThreshold {
		t.Fatalf("seeded settings = %#v", settings)
	}
	if settings.Revision != 1 || settings.Telegram.TokenConfigured || len(settings.ProviderOverrides) != 0 {
		t.Fatalf("seeded settings metadata = %#v", settings)
	}
}

func TestPostgresQuotaAlertSettingsAndEncryptedSecretRoundTrip(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	settings, err := store.LoadSettings(ctx)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	providerThreshold := quotaalert.Percentage(25)
	settings.Enabled = true
	settings.NotifyRecovery = true
	settings.ReminderInterval = 30 * time.Minute
	settings.ProviderOverrides = []quotaalert.ProviderOverride{
		{Provider: quotaalert.ProviderClaude, Enabled: true, WarningThreshold: &providerThreshold},
		{Provider: quotaalert.ProviderKiro, Enabled: false},
	}
	settings.Telegram.Enabled = true
	settings.Telegram.ChatID = "-1001234567890"
	settings.Telegram.TokenConfigured = true

	const token = "123456789:database-secret-token"
	secretUpdate, err := quotaalert.ReplaceSecret(token)
	if err != nil {
		t.Fatalf("ReplaceSecret() error = %v", err)
	}
	cipher, err := quotaalert.NewSecretCipher("quota-key-1", bytes.Repeat([]byte{7}, quotaalert.SecretKeySize))
	if err != nil {
		t.Fatalf("NewSecretCipher() error = %v", err)
	}
	saved, err := store.SaveSettingsWithSecret(ctx, settings.Revision, settings, secretUpdate, cipher, "telegram-bot-token")
	if err != nil {
		t.Fatalf("SaveSettingsWithSecret() error = %v", err)
	}
	if saved.Revision != 2 || !saved.Telegram.TokenConfigured {
		t.Fatalf("saved settings = %#v", saved)
	}

	var keyID string
	var nonce, ciphertext []byte
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT telegram_secret_key_id, telegram_secret_nonce, telegram_secret_ciphertext
		FROM %s WHERE id = $1
	`, store.fullTableName(quotaAlertSettingsTable)), quotaAlertSettingsID).Scan(&keyID, &nonce, &ciphertext); err != nil {
		t.Fatalf("load raw encrypted secret: %v", err)
	}
	if keyID != "quota-key-1" || len(nonce) != quotaalert.SecretNonceSize || len(ciphertext) == 0 {
		t.Fatalf("stored secret metadata = key %q, nonce %d, ciphertext %d", keyID, len(nonce), len(ciphertext))
	}
	if bytes.Contains(nonce, []byte(token)) || bytes.Contains(ciphertext, []byte(token)) {
		t.Fatal("stored encrypted columns contain plaintext token")
	}
	var plaintextMatches int
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE telegram_secret_ciphertext = $1::bytea
	`, store.fullTableName(quotaAlertSettingsTable)), []byte(token)).Scan(&plaintextMatches); err != nil {
		t.Fatalf("check plaintext storage: %v", err)
	}
	if plaintextMatches != 0 {
		t.Fatal("plaintext token was stored as ciphertext")
	}

	snapshotSettings, storedSecret, err := store.LoadSettingsWithSecret(ctx)
	if err != nil {
		t.Fatalf("LoadSettingsWithSecret() error = %v", err)
	}
	if snapshotSettings.Revision != saved.Revision || snapshotSettings.Telegram.ChatID != saved.Telegram.ChatID || !snapshotSettings.Telegram.TokenConfigured {
		t.Fatalf("atomic settings snapshot = %#v, want revision %d Telegram destination %#v", snapshotSettings, saved.Revision, saved.Telegram)
	}
	decrypted, err := cipher.Decrypt("telegram-bot-token", *storedSecret)
	if err != nil {
		t.Fatalf("SecretCipher.Decrypt() error = %v", err)
	}
	if string(decrypted) != token {
		t.Fatalf("decrypted token = %q", decrypted)
	}

	loaded, err := store.LoadSettings(ctx)
	if err != nil {
		t.Fatalf("LoadSettings() after save error = %v", err)
	}
	if loaded.Revision != 2 || !loaded.Enabled || !loaded.NotifyRecovery || loaded.ReminderInterval != 30*time.Minute {
		t.Fatalf("loaded settings = %#v", loaded)
	}
	if !loaded.Telegram.Enabled || loaded.Telegram.ChatID != settings.Telegram.ChatID || !loaded.Telegram.TokenConfigured {
		t.Fatalf("loaded Telegram settings = %#v", loaded.Telegram)
	}
	if len(loaded.ProviderOverrides) != 2 || loaded.ProviderOverrides[0].Provider != quotaalert.ProviderClaude || loaded.ProviderOverrides[1].Provider != quotaalert.ProviderKiro {
		t.Fatalf("loaded provider overrides = %#v", loaded.ProviderOverrides)
	}

	if _, err = store.SaveSettings(ctx, 1, loaded); err == nil {
		t.Fatal("SaveSettings() error = nil for stale revision")
	}
	loaded.WarningThreshold = 15
	updated, err := store.SaveSettings(ctx, loaded.Revision, loaded)
	if err != nil {
		t.Fatalf("SaveSettings() preserving secret error = %v", err)
	}
	if updated.Revision != 3 || !updated.Telegram.TokenConfigured {
		t.Fatalf("updated settings = %#v", updated)
	}

	updated.Telegram.Enabled = false
	updated.Telegram.TokenConfigured = false
	cleared, err := store.SaveSettingsWithSecret(ctx, updated.Revision, updated, quotaalert.ClearSecret(), nil, "")
	if err != nil {
		t.Fatalf("SaveSettingsWithSecret(clear) error = %v", err)
	}
	if cleared.Revision != 4 || cleared.Telegram.TokenConfigured {
		t.Fatalf("cleared settings = %#v", cleared)
	}
	snapshotSettings, storedSecret, err = store.LoadSettingsWithSecret(ctx)
	if err != nil {
		t.Fatalf("LoadSettingsWithSecret() after clear error = %v", err)
	}
	if snapshotSettings.Revision != cleared.Revision || snapshotSettings.Telegram.TokenConfigured || storedSecret != nil {
		t.Fatalf("cleared atomic settings snapshot = settings %#v, secret %#v", snapshotSettings, storedSecret)
	}
}

func TestPostgresQuotaAlertConcurrentSettingsReadsRemainCoherent(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	settings, err := store.LoadSettings(ctx)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	threshold := quotaalert.Percentage(11)
	settings.Enabled = false
	settings.WarningThreshold = threshold
	settings.ProviderOverrides = []quotaalert.ProviderOverride{{
		Provider:         quotaalert.ProviderClaude,
		Enabled:          settings.Enabled,
		WarningThreshold: &threshold,
	}}
	settings.Telegram = quotaalert.TelegramDestination{Enabled: true, ChatID: "chat-11", TokenConfigured: true}
	cipher, err := quotaalert.NewSecretCipher("settings-snapshot-key", bytes.Repeat([]byte{11}, quotaalert.SecretKeySize))
	if err != nil {
		t.Fatalf("NewSecretCipher() error = %v", err)
	}
	initialSecret, err := quotaalert.ReplaceSecret("token-11")
	if err != nil {
		t.Fatalf("ReplaceSecret(initial) error = %v", err)
	}
	settings, err = store.SaveSettingsWithSecret(ctx, settings.Revision, settings, initialSecret, cipher, "telegram-bot-token")
	if err != nil {
		t.Fatalf("SaveSettingsWithSecret(initial) error = %v", err)
	}

	start := make(chan struct{})
	done := make(chan struct{})
	errors := make(chan error, 1)
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-start
			for {
				select {
				case <-done:
					return
				default:
				}
				loaded, storedSecret, loadErr := store.LoadSettingsWithSecret(ctx)
				if loadErr != nil {
					select {
					case errors <- loadErr:
					default:
					}
					return
				}
				if len(loaded.ProviderOverrides) != 1 || loaded.ProviderOverrides[0].WarningThreshold == nil {
					select {
					case errors <- fmt.Errorf("incomplete provider overrides for revision %d", loaded.Revision):
					default:
					}
					return
				}
				override := loaded.ProviderOverrides[0]
				expectedSuffix := fmt.Sprintf("%.0f", loaded.WarningThreshold)
				if loaded.Enabled != override.Enabled || loaded.WarningThreshold != *override.WarningThreshold || loaded.Telegram.ChatID != "chat-"+expectedSuffix || storedSecret == nil {
					select {
					case errors <- fmt.Errorf("torn settings revision %d: enabled %t/%t, threshold %v/%v, chat %q, secret configured %t", loaded.Revision, loaded.Enabled, override.Enabled, loaded.WarningThreshold, *override.WarningThreshold, loaded.Telegram.ChatID, storedSecret != nil):
					default:
					}
					return
				}
				plaintext, decryptErr := cipher.Decrypt("telegram-bot-token", *storedSecret)
				if decryptErr != nil || string(plaintext) != "token-"+expectedSuffix {
					select {
					case errors <- fmt.Errorf("torn secret revision %d: token %q, error %v", loaded.Revision, plaintext, decryptErr):
					default:
					}
					return
				}
			}
		}()
	}
	close(start)

	for index := range 300 {
		value := quotaalert.Percentage(11)
		enabled := false
		if index%2 == 1 {
			value = 22
			enabled = true
		}
		settings.Enabled = enabled
		settings.WarningThreshold = value
		settings.ProviderOverrides = []quotaalert.ProviderOverride{{
			Provider:         quotaalert.ProviderClaude,
			Enabled:          enabled,
			WarningThreshold: &value,
		}}
		suffix := fmt.Sprintf("%.0f", value)
		settings.Telegram.ChatID = "chat-" + suffix
		secretUpdate, updateErr := quotaalert.ReplaceSecret("token-" + suffix)
		if updateErr != nil {
			close(done)
			readers.Wait()
			t.Fatalf("ReplaceSecret(iteration %d) error = %v", index, updateErr)
		}
		settings, err = store.SaveSettingsWithSecret(ctx, settings.Revision, settings, secretUpdate, cipher, "telegram-bot-token")
		if err != nil {
			close(done)
			readers.Wait()
			t.Fatalf("SaveSettingsWithSecret(iteration %d) error = %v", index, err)
		}
		select {
		case readErr := <-errors:
			close(done)
			readers.Wait()
			t.Fatal(readErr)
		default:
		}
	}
	close(done)
	readers.Wait()
	select {
	case readErr := <-errors:
		t.Fatal(readErr)
	default:
	}
}

func TestPostgresQuotaAlertCollectionRejectsStaleSettingsRevisionWithoutMutation(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	settings, err := store.LoadSettings(ctx)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	state, event := quotaAlertTestStateAndEvent("stale-settings", time.Now().UTC())
	batch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{event}, event.OccurredAt)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection() = acquired %t, error %v", acquired, err)
	}

	settings.WarningThreshold = 11
	if _, err = store.SaveSettings(ctx, settings.Revision, settings); err != nil {
		_ = lease.Release(ctx)
		t.Fatalf("SaveSettings() error = %v", err)
	}
	commit := quotaalert.CollectionCommit{
		SettingsRevision: settings.Revision,
		States:           []quotaalert.CurrentState{state},
		Events:           []quotaalert.TransitionEvent{event},
		Batches:          []quotaalert.NotificationBatch{batch},
	}
	if err = store.CommitCollection(ctx, lease, commit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(stale settings revision) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release() error = %v", err)
	}

	for table, want := range map[string]int{
		quotaAlertStateTable:              0,
		quotaAlertEventsTable:             0,
		quotaNotificationBatchesTable:     0,
		quotaNotificationBatchEventsTable: 0,
	} {
		var count int
		if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(table))).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", table, count, want)
		}
	}
}

func TestPostgresQuotaAlertAtomicCommitDeduplicationPaginationAndAcknowledgement(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Date(2026, time.July, 28, 7, 0, 0, 0, time.UTC)
	firstState, firstEvent := quotaAlertTestStateAndEvent("1", now)
	secondState, secondEvent := quotaAlertTestStateAndEvent("2", now.Add(time.Minute))
	batch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{secondEvent, firstEvent}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	commit := quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{firstState, secondState},
		Events:  []quotaalert.TransitionEvent{firstEvent, secondEvent},
		Batches: []quotaalert.NotificationBatch{batch},
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, commit)
	commitPostgresQuotaAlertTestCollection(t, ctx, store, commit)

	for table, want := range map[string]int{
		quotaAlertStateTable:              2,
		quotaAlertEventsTable:             2,
		quotaNotificationBatchesTable:     1,
		quotaNotificationBatchEventsTable: 2,
	} {
		var count int
		if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(table))).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", table, count, want)
		}
	}

	states, err := store.LoadStates(ctx, []quotaalert.StateIdentity{secondState.Identity, firstState.Identity})
	if err != nil {
		t.Fatalf("LoadStates() error = %v", err)
	}
	if len(states) != 2 || states[0].Revision != 1 || states[1].Revision != 1 {
		t.Fatalf("loaded states = %#v", states)
	}

	statePage, err := store.ListStates(ctx, quotaalert.PageRequest{Limit: 1})
	if err != nil {
		t.Fatalf("ListStates() first page error = %v", err)
	}
	if len(statePage.Items) != 1 || statePage.NextCursor == "" {
		t.Fatalf("first state page = %#v", statePage)
	}
	statePageTwo, err := store.ListStates(ctx, quotaalert.PageRequest{Limit: 1, Cursor: statePage.NextCursor})
	if err != nil {
		t.Fatalf("ListStates() second page error = %v", err)
	}
	if len(statePageTwo.Items) != 1 || statePageTwo.Items[0].Identity == statePage.Items[0].Identity {
		t.Fatalf("second state page = %#v", statePageTwo)
	}

	eventPage, err := store.ListEvents(ctx, quotaalert.PageRequest{Limit: 1})
	if err != nil {
		t.Fatalf("ListEvents() first page error = %v", err)
	}
	if len(eventPage.Items) != 1 || eventPage.NextCursor == "" {
		t.Fatalf("first event page = %#v", eventPage)
	}
	eventPageTwo, err := store.ListEvents(ctx, quotaalert.PageRequest{Limit: 1, Cursor: eventPage.NextCursor})
	if err != nil {
		t.Fatalf("ListEvents() second page error = %v", err)
	}
	if len(eventPageTwo.Items) != 1 || eventPageTwo.Items[0].ID == eventPage.Items[0].ID {
		t.Fatalf("second event page = %#v", eventPageTwo)
	}

	if err = store.AcknowledgeEvent(ctx, firstEvent.ID, firstEvent.OccurredAt.Add(-time.Second)); err == nil {
		t.Fatal("AcknowledgeEvent(before event) error = nil")
	}
	acknowledgedAt := now.Add(10 * time.Minute)
	if err = store.AcknowledgeEvent(ctx, firstEvent.ID, acknowledgedAt); err != nil {
		t.Fatalf("AcknowledgeEvent() first error = %v", err)
	}
	if err = store.AcknowledgeEvent(ctx, firstEvent.ID, acknowledgedAt.Add(time.Minute)); err != nil {
		t.Fatalf("AcknowledgeEvent() second error = %v", err)
	}
	var storedAcknowledgement time.Time
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT acknowledged_at FROM %s WHERE id = $1", store.fullTableName(quotaAlertEventsTable)), firstEvent.ID).Scan(&storedAcknowledgement); err != nil {
		t.Fatalf("load acknowledgement: %v", err)
	}
	if !storedAcknowledgement.Equal(acknowledgedAt) {
		t.Fatalf("acknowledgement = %v, want %v", storedAcknowledgement, acknowledgedAt)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States: []quotaalert.CurrentState{firstState},
		Events: []quotaalert.TransitionEvent{firstEvent},
	})
	var acknowledgementAfterReplay time.Time
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT acknowledged_at FROM %s WHERE id = $1", store.fullTableName(quotaAlertEventsTable)), firstEvent.ID).Scan(&acknowledgementAfterReplay); err != nil {
		t.Fatalf("load acknowledgement after event replay: %v", err)
	}
	if !acknowledgementAfterReplay.Equal(acknowledgedAt) {
		t.Fatalf("acknowledgement after replay = %v, want %v", acknowledgementAfterReplay, acknowledgedAt)
	}
	if err = store.AcknowledgeEvent(ctx, "missing", acknowledgedAt); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("AcknowledgeEvent(missing) error = %v, want sql.ErrNoRows", err)
	}

	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		RemovedStates: []quotaalert.StateIdentity{firstState.Identity},
	})
	states, err = store.LoadStates(ctx, []quotaalert.StateIdentity{firstState.Identity, secondState.Identity})
	if err != nil {
		t.Fatalf("LoadStates() after removal error = %v", err)
	}
	if len(states) != 1 || states[0].Identity != secondState.Identity {
		t.Fatalf("states after removal = %#v, want only %#v", states, secondState.Identity)
	}
}

func TestPostgresQuotaAlertStatePaginationSeeksAcrossEqualTimestamps(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	updatedAt := time.Date(2026, time.July, 28, 11, 0, 0, 0, time.UTC)
	var states []quotaalert.CurrentState
	var events []quotaalert.TransitionEvent
	for _, suffix := range []string{"a", "b", "c"} {
		state, event := quotaAlertTestStateAndEvent("pagination-"+suffix, updatedAt)
		states = append(states, state)
		events = append(events, event)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States: states,
		Events: events,
	})

	var authIDs []string
	cursor := ""
	for {
		page, err := store.ListStates(ctx, quotaalert.PageRequest{
			Cursor: cursor,
			Limit:  1,
		})
		if err != nil {
			t.Fatalf("ListStates(cursor %q) error = %v", cursor, err)
		}
		if len(page.Items) != 1 {
			t.Fatalf("ListStates(cursor %q) items = %#v, want one", cursor, page.Items)
		}
		authIDs = append(authIDs, page.Items[0].Identity.AuthID)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	want := []string{"auth-pagination-c", "auth-pagination-b", "auth-pagination-a"}
	if len(authIDs) != len(want) {
		t.Fatalf("paginated auth IDs = %v, want %v", authIDs, want)
	}
	for index := range want {
		if authIDs[index] != want[index] {
			t.Fatalf("paginated auth IDs = %v, want %v", authIDs, want)
		}
	}
}

func TestPostgresQuotaAlertEventBelongsToOnlyOneNotificationBatch(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)
	firstState, firstEvent := quotaAlertTestStateAndEvent("batch-first", now)
	firstBatch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{firstEvent}, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch(first) error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{firstState},
		Events:  []quotaalert.TransitionEvent{firstEvent},
		Batches: []quotaalert.NotificationBatch{firstBatch},
	})

	secondState, secondEvent := quotaAlertTestStateAndEvent("batch-second", now.Add(time.Minute))
	combinedBatch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{firstEvent, secondEvent}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("NewNotificationBatch(combined) error = %v", err)
	}
	commit := quotaAlertTestCollectionCommit(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{firstState, secondState},
		Events:  []quotaalert.TransitionEvent{firstEvent, secondEvent},
		Batches: []quotaalert.NotificationBatch{combinedBatch},
	})
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection() = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, commit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(event assigned to another batch) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release() error = %v", err)
	}

	for table, want := range map[string]int{
		quotaAlertStateTable:              1,
		quotaAlertEventsTable:             1,
		quotaNotificationBatchesTable:     1,
		quotaNotificationBatchEventsTable: 1,
	} {
		var count int
		if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(table))).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", table, count, want)
		}
	}
	states, err := store.LoadStates(ctx, []quotaalert.StateIdentity{secondState.Identity})
	if err != nil {
		t.Fatalf("LoadStates(second) error = %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("rolled-back second state = %#v", states)
	}

	if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s SET status = 'sent', updated_at = $1 WHERE id = $2
	`, store.fullTableName(quotaNotificationBatchesTable)), now, firstBatch.ID()); err != nil {
		t.Fatalf("mark first notification batch terminal: %v", err)
	}
	deleted, err := store.PruneNotificationBatches(ctx, now.Add(24*time.Hour), quotaalert.MaxPageSize)
	if err != nil {
		t.Fatalf("PruneNotificationBatches() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PruneNotificationBatches() deleted = %d, want 1", deleted)
	}
	for table, want := range map[string]int{
		quotaAlertEventsTable:             1,
		quotaNotificationBatchesTable:     0,
		quotaNotificationBatchEventsTable: 1,
	} {
		var count int
		if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(table))).Scan(&count); err != nil {
			t.Fatalf("count %s after batch prune: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count after batch prune = %d, want %d", table, count, want)
		}
	}

	replayCommit := quotaAlertTestCollectionCommit(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{firstState},
		Events:  []quotaalert.TransitionEvent{firstEvent},
		Batches: []quotaalert.NotificationBatch{firstBatch},
	})
	lease, acquired, err = store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(exact replay after batch prune) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, replayCommit); err != nil {
		_ = lease.Release(ctx)
		t.Fatalf("CommitCollection(exact replay after batch prune) error = %v", err)
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release(exact replay after batch prune) error = %v", err)
	}
	claims, err := store.ClaimNotificationBatches(ctx, quotaalert.NotificationClaimOptions{
		Limit:         1,
		LeaseDuration: time.Minute,
	})
	if err != nil {
		t.Fatalf("ClaimNotificationBatches(after retired replay) error = %v", err)
	}
	if len(claims) != 0 {
		t.Fatalf("claims after retired replay = %#v, want none", claims)
	}

	lease, acquired, err = store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(after batch prune) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, commit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(rebatch after batch prune) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release(after batch prune) error = %v", err)
	}

	deleted, err = store.PruneEvents(ctx, now.Add(24*time.Hour), quotaalert.MaxPageSize)
	if err != nil {
		t.Fatalf("PruneEvents() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PruneEvents() deleted = %d, want 1", deleted)
	}
	for table, want := range map[string]int{
		quotaAlertEventsTable:             0,
		quotaNotificationBatchEventsTable: 0,
	} {
		var count int
		if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(table))).Scan(&count); err != nil {
			t.Fatalf("count %s after both prunes: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count after both prunes = %d, want %d", table, count, want)
		}
	}
}

func TestPostgresQuotaAlertEventFirstRetentionPreservesAssignmentTombstone(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Date(2026, time.July, 28, 13, 0, 0, 0, time.UTC)
	state, event := quotaAlertTestStateAndEvent("event-first-retention", now)
	batch, err := quotaalert.NewNotificationBatch(
		quotaalert.ProviderClaude,
		[]quotaalert.TransitionEvent{event},
		now,
	)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{state},
		Events:  []quotaalert.TransitionEvent{event},
		Batches: []quotaalert.NotificationBatch{batch},
	})
	if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s SET status = 'sent', updated_at = $1 WHERE id = $2
	`, store.fullTableName(quotaNotificationBatchesTable)), now, batch.ID()); err != nil {
		t.Fatalf("mark notification batch terminal: %v", err)
	}

	deleted, err := store.PruneEvents(
		ctx,
		now.Add(24*time.Hour),
		quotaalert.MaxPageSize,
	)
	if err != nil {
		t.Fatalf("PruneEvents() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PruneEvents() deleted = %d, want 1", deleted)
	}
	for table, want := range map[string]int{
		quotaAlertEventsTable:             0,
		quotaNotificationBatchesTable:     1,
		quotaNotificationBatchEventsTable: 1,
	} {
		var count int
		if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(table))).Scan(&count); err != nil {
			t.Fatalf("count %s after event prune: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count after event prune = %d, want %d", table, count, want)
		}
	}

	recoveryAt := now.Add(time.Minute)
	recoveryState := state
	recoveryState.Alert = quotaalert.AlertHealthy
	recoveryState.Remaining = 50
	recoveryState.ObservedAt = recoveryAt
	recoveryState.TransitionedAt = recoveryAt
	recoveryState.UpdatedAt = recoveryAt
	recoveryEvent := event
	recoveryEvent.Kind = quotaalert.TransitionRecovery
	recoveryEvent.From = quotaalert.AlertWarning
	recoveryEvent.To = quotaalert.AlertHealthy
	recoveryEvent.Remaining = 50
	recoveryEvent.OccurredAt = recoveryAt
	commit := quotaAlertTestCollectionCommit(t, ctx, store, quotaalert.CollectionCommit{
		States: []quotaalert.CurrentState{recoveryState},
		Events: []quotaalert.TransitionEvent{recoveryEvent},
	})
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(reused event ID) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, commit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(reused pruned event ID) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release(reused event ID) error = %v", err)
	}

	states, err := store.LoadStates(ctx, []quotaalert.StateIdentity{state.Identity})
	if err != nil {
		t.Fatalf("LoadStates() error = %v", err)
	}
	if len(states) != 1 || states[0].Alert != quotaalert.AlertWarning || states[0].Revision != 1 {
		t.Fatalf("state after rejected event ID reuse = %#v", states)
	}
	deleted, err = store.PruneNotificationBatches(
		ctx,
		now.Add(24*time.Hour),
		quotaalert.MaxPageSize,
	)
	if err != nil {
		t.Fatalf("PruneNotificationBatches() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PruneNotificationBatches() deleted = %d, want 1", deleted)
	}
	var assignmentCount int
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(quotaNotificationBatchEventsTable))).Scan(&assignmentCount); err != nil {
		t.Fatalf("count assignments after both prunes: %v", err)
	}
	if assignmentCount != 0 {
		t.Fatalf("assignment count after both prunes = %d, want 0", assignmentCount)
	}
}

func TestPostgresQuotaAlertConcurrentOppositeOrderPruningCleansAssignment(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Date(2026, time.July, 28, 13, 30, 0, 0, time.UTC)
	state, event := quotaAlertTestStateAndEvent("concurrent-retention", now)
	batch, err := quotaalert.NewNotificationBatch(
		quotaalert.ProviderClaude,
		[]quotaalert.TransitionEvent{event},
		now,
	)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{state},
		Events:  []quotaalert.TransitionEvent{event},
		Batches: []quotaalert.NotificationBatch{batch},
	})
	if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s SET status = 'sent', updated_at = $1 WHERE id = $2
	`, store.fullTableName(quotaNotificationBatchesTable)), now, batch.ID()); err != nil {
		t.Fatalf("mark notification batch terminal: %v", err)
	}

	functionName := store.fullTableName("quota_alert_retention_delay")
	if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_sleep(0.25);
			RETURN OLD;
		END;
		$$ LANGUAGE plpgsql
	`, functionName)); err != nil {
		t.Fatalf("create retention delay function: %v", err)
	}
	for name, table := range map[string]string{
		"quota_alert_event_retention_delay": quotaAlertEventsTable,
		"quota_alert_batch_retention_delay": quotaNotificationBatchesTable,
	} {
		if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
			CREATE TRIGGER %s
			BEFORE DELETE ON %s
			FOR EACH ROW EXECUTE FUNCTION %s()
		`, quoteIdentifier(name), store.fullTableName(table), functionName)); err != nil {
			t.Fatalf("create %s trigger: %v", name, err)
		}
	}

	type pruneResult struct {
		deleted int64
		err     error
	}
	start := make(chan struct{})
	results := make(chan pruneResult, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		deleted, pruneErr := store.PruneEvents(
			ctx,
			now.Add(24*time.Hour),
			quotaalert.MaxPageSize,
		)
		results <- pruneResult{deleted: deleted, err: pruneErr}
	}()
	go func() {
		defer wait.Done()
		<-start
		deleted, pruneErr := store.PruneNotificationBatches(
			ctx,
			now.Add(24*time.Hour),
			quotaalert.MaxPageSize,
		)
		results <- pruneResult{deleted: deleted, err: pruneErr}
	}()
	close(start)
	wait.Wait()
	close(results)
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent retention error = %v", result.err)
		}
		if result.deleted != 1 {
			t.Fatalf("concurrent retention deleted = %d, want 1", result.deleted)
		}
	}

	for table, want := range map[string]int{
		quotaAlertEventsTable:             0,
		quotaNotificationBatchesTable:     0,
		quotaNotificationBatchEventsTable: 0,
	} {
		var count int
		if err = store.db.QueryRowContext(ctx, fmt.Sprintf(
			"SELECT COUNT(*) FROM %s",
			store.fullTableName(table),
		)).Scan(&count); err != nil {
			t.Fatalf("count %s after concurrent retention: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count after concurrent retention = %d, want %d", table, count, want)
		}
	}
}

func TestPostgresQuotaAlertCollectionCanonicalizesAndRejectsInvalidEvents(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	localTime := time.Date(2026, time.July, 28, 14, 0, 0, 123, time.FixedZone("ICT", 7*60*60))
	state, event := quotaAlertTestStateAndEvent("canonical", localTime)
	event.ID = " event-canonical "
	event.Identity = quotaalert.StateIdentity{
		AuthID:   " auth-canonical ",
		Provider: " CLAUDE ",
		Resource: " messages ",
		Window:   " five-hour ",
	}
	event.AuthLabel = " Account canonical "
	normalized, err := event.Normalize()
	if err != nil {
		t.Fatalf("TransitionEvent.Normalize() error = %v", err)
	}
	state.Identity = normalized.Identity
	state.AuthLabel = normalized.AuthLabel
	state.Alert = normalized.To
	state.Remaining = normalized.Remaining
	state.RemainingKnown = normalized.RemainingKnown
	state.ResetAt = normalized.ResetAt
	state.ResetKnown = normalized.ResetKnown
	state.ObservedAt = normalized.OccurredAt
	state.TransitionedAt = normalized.OccurredAt
	state.UpdatedAt = normalized.OccurredAt
	batch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{event}, localTime)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{state},
		Events:  []quotaalert.TransitionEvent{event},
		Batches: []quotaalert.NotificationBatch{batch},
	})
	var stored quotaalert.TransitionEvent
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT id, auth_id, provider, resource, window_key, auth_label, kind,
		       from_state, to_state, occurred_at
		FROM %s WHERE id = $1
	`, store.fullTableName(quotaAlertEventsTable)), normalized.ID).Scan(
		&stored.ID,
		&stored.Identity.AuthID,
		&stored.Identity.Provider,
		&stored.Identity.Resource,
		&stored.Identity.Window,
		&stored.AuthLabel,
		&stored.Kind,
		&stored.From,
		&stored.To,
		&stored.OccurredAt,
	); err != nil {
		t.Fatalf("load canonical quota alert event: %v", err)
	}
	if stored.ID != normalized.ID || stored.Identity != normalized.Identity || stored.AuthLabel != normalized.AuthLabel ||
		stored.Kind != normalized.Kind || stored.From != normalized.From || stored.To != normalized.To ||
		!stored.OccurredAt.Equal(normalized.OccurredAt) {
		t.Fatalf("stored event = %#v, want canonical %#v", stored, normalized)
	}
	claims, err := store.ClaimNotificationBatches(ctx, quotaalert.NotificationClaimOptions{Limit: 1, LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("ClaimNotificationBatches() error = %v", err)
	}
	if len(claims) != 1 || len(claims[0].Batch.Events()) != 1 || claims[0].Batch.Events()[0] != normalized {
		t.Fatalf("claimed canonical batch = %#v, want event %#v", claims, normalized)
	}

	invalid := event
	invalid.ID = "event-invalid-transition"
	invalid.Kind = quotaalert.TransitionRecovery
	invalid.From = quotaalert.AlertHealthy
	invalid.To = quotaalert.AlertWarning
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(invalid event) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, quotaalert.CollectionCommit{Events: []quotaalert.TransitionEvent{invalid}}); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(invalid transition) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("Release(invalid event lease) error = %v", err)
	}
	var invalidCount int
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE id = $1",
		store.fullTableName(quotaAlertEventsTable),
	), invalid.ID).Scan(&invalidCount); err != nil {
		t.Fatalf("count invalid quota alert event: %v", err)
	}
	if invalidCount != 0 {
		t.Fatalf("invalid quota alert event count = %d, want 0", invalidCount)
	}

	invalidRemaining := event
	invalidRemaining.ID = "event-invalid-remaining"
	invalidRemaining.Kind = quotaalert.TransitionExhausted
	invalidRemaining.From = quotaalert.AlertWarning
	invalidRemaining.To = quotaalert.AlertExhausted
	invalidRemaining.Remaining = 5
	invalidRemaining.RemainingKnown = true
	lease, acquired, err = store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(invalid remaining event) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, quotaalert.CollectionCommit{Events: []quotaalert.TransitionEvent{invalidRemaining}}); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(exhausted event with positive remaining) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("Release(invalid remaining event lease) error = %v", err)
	}

	lease, acquired, err = store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(batch without event) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, quotaalert.CollectionCommit{Batches: []quotaalert.NotificationBatch{batch}}); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(batch without event) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("Release(batch without event lease) error = %v", err)
	}

	conflictingAt := normalized.OccurredAt.Add(time.Minute)
	conflictingState := state
	conflictingState.ObservedAt = conflictingAt
	conflictingState.UpdatedAt = conflictingAt
	conflicting := event
	conflicting.Kind = quotaalert.TransitionReminder
	conflicting.From = quotaalert.AlertWarning
	conflicting.To = quotaalert.AlertWarning
	conflicting.OccurredAt = conflictingAt
	conflictingBatch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{conflicting}, conflictingAt)
	if err != nil {
		t.Fatalf("NewNotificationBatch(conflicting event) error = %v", err)
	}
	lease, acquired, err = store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(conflicting event) = acquired %t, error %v", acquired, err)
	}
	conflictingCommit := quotaAlertTestCollectionCommit(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{conflictingState},
		Events:  []quotaalert.TransitionEvent{conflicting},
		Batches: []quotaalert.NotificationBatch{conflictingBatch},
	})
	if err = store.CommitCollection(ctx, lease, conflictingCommit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(conflicting event content) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("Release(conflicting event lease) error = %v", err)
	}
	var conflictingBatchCount int
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE id = $1",
		store.fullTableName(quotaNotificationBatchesTable),
	), conflictingBatch.ID()).Scan(&conflictingBatchCount); err != nil {
		t.Fatalf("count conflicting notification batch: %v", err)
	}
	if conflictingBatchCount != 0 {
		t.Fatalf("conflicting notification batch count = %d, want 0", conflictingBatchCount)
	}

	invalidState, _ := quotaAlertTestStateAndEvent("invalid-state", localTime)
	invalidState.Alert = quotaalert.AlertExhausted
	invalidState.Remaining = 50
	lease, acquired, err = store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(invalid state) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, quotaalert.CollectionCommit{States: []quotaalert.CurrentState{invalidState}}); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(invalid state) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("Release(invalid state lease) error = %v", err)
	}
	states, err := store.LoadStates(ctx, []quotaalert.StateIdentity{invalidState.Identity})
	if err != nil {
		t.Fatalf("LoadStates(invalid state) error = %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("invalid state persisted = %#v", states)
	}

	evidencelessState, _ := quotaAlertTestStateAndEvent("evidenceless-state", localTime)
	evidencelessState.Remaining = 0
	evidencelessState.RemainingKnown = false
	lease, acquired, err = store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(evidenceless state) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, quotaalert.CollectionCommit{States: []quotaalert.CurrentState{evidencelessState}}); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(reliable state without evidence) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("Release(evidenceless state lease) error = %v", err)
	}
	states, err = store.LoadStates(ctx, []quotaalert.StateIdentity{evidencelessState.Identity})
	if err != nil {
		t.Fatalf("LoadStates(evidenceless state) error = %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("evidenceless state persisted = %#v", states)
	}
}

func TestPostgresQuotaAlertCollectionEnforcesEventStateCoherence(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	settings, err := store.LoadSettings(ctx)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	now := time.Date(2026, time.July, 28, 15, 0, 0, 0, time.UTC)
	baseState, baseEvent := quotaAlertTestStateAndEvent("coherence", now)

	tests := []struct {
		name   string
		mutate func(*quotaalert.CurrentState, *quotaalert.TransitionEvent)
	}{
		{name: "identity", mutate: func(state *quotaalert.CurrentState, _ *quotaalert.TransitionEvent) {
			state.Identity.Resource = "tokens"
		}},
		{name: "target", mutate: func(state *quotaalert.CurrentState, _ *quotaalert.TransitionEvent) {
			state.Alert = quotaalert.AlertHealthy
		}},
		{name: "auth label", mutate: func(state *quotaalert.CurrentState, _ *quotaalert.TransitionEvent) {
			state.AuthLabel = "Changed account"
		}},
		{name: "remaining evidence", mutate: func(state *quotaalert.CurrentState, _ *quotaalert.TransitionEvent) {
			state.Remaining = 6
		}},
		{name: "reset evidence", mutate: func(state *quotaalert.CurrentState, _ *quotaalert.TransitionEvent) {
			state.ResetAt = state.ResetAt.Add(time.Minute)
		}},
		{name: "unknown collection health", mutate: func(state *quotaalert.CurrentState, _ *quotaalert.TransitionEvent) {
			state.Health = quotaalert.CollectionUnknown
		}},
		{name: "event after state", mutate: func(state *quotaalert.CurrentState, _ *quotaalert.TransitionEvent) {
			state.ObservedAt = now.Add(-time.Minute)
			state.TransitionedAt = now.Add(-time.Minute)
			state.UpdatedAt = now.Add(-time.Minute)
		}},
		{name: "transition time", mutate: func(state *quotaalert.CurrentState, _ *quotaalert.TransitionEvent) {
			state.TransitionedAt = now.Add(-time.Minute)
		}},
		{name: "acknowledgement", mutate: func(_ *quotaalert.CurrentState, event *quotaalert.TransitionEvent) {
			event.AcknowledgedAt = now.Add(time.Minute)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := baseState
			event := baseEvent
			test.mutate(&state, &event)
			if _, normalizeErr := state.Normalize(); normalizeErr != nil {
				t.Fatalf("CurrentState.Normalize() error = %v", normalizeErr)
			}
			if _, normalizeErr := event.Normalize(); normalizeErr != nil {
				t.Fatalf("TransitionEvent.Normalize() error = %v", normalizeErr)
			}
			lease, acquired, acquireErr := store.TryAcquireCollection(ctx)
			if acquireErr != nil || !acquired {
				t.Fatalf("TryAcquireCollection() = acquired %t, error %v", acquired, acquireErr)
			}
			commit := quotaalert.CollectionCommit{
				SettingsRevision: settings.Revision,
				States:           []quotaalert.CurrentState{state},
				Events:           []quotaalert.TransitionEvent{event},
			}
			if commitErr := store.CommitCollection(ctx, lease, commit); commitErr == nil {
				_ = lease.Release(ctx)
				t.Fatal("CommitCollection(incoherent event and state) error = nil")
			}
			if releaseErr := lease.Release(ctx); releaseErr != nil {
				t.Fatalf("CollectionLease.Release() error = %v", releaseErr)
			}
		})
	}

	initialReminderState, initialReminderEvent := quotaAlertTestStateAndEvent("reminder", now)
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		SettingsRevision: settings.Revision,
		States:           []quotaalert.CurrentState{initialReminderState},
		Events:           []quotaalert.TransitionEvent{initialReminderEvent},
	})
	reminderState := initialReminderState
	reminderState.ObservedAt = now.Add(time.Hour)
	reminderState.UpdatedAt = now.Add(time.Hour)
	reminderEvent := initialReminderEvent
	reminderEvent.ID = "event-reminder-follow-up"
	reminderEvent.Kind = quotaalert.TransitionReminder
	reminderEvent.From = quotaalert.AlertWarning
	reminderEvent.To = quotaalert.AlertWarning
	reminderEvent.OccurredAt = now.Add(time.Hour)
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		SettingsRevision: settings.Revision,
		States:           []quotaalert.CurrentState{reminderState},
		Events:           []quotaalert.TransitionEvent{reminderEvent},
	})
}

func TestPostgresQuotaAlertLoadStatesChunksAndRejectsOversizedCursors(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	identities := make([]quotaalert.StateIdentity, 0, maxLoadStateIdentities)
	for index := range maxLoadStateIdentities - 1 {
		identities = append(identities, quotaalert.StateIdentity{
			AuthID:   fmt.Sprintf("bulk-auth-%d", index),
			Provider: quotaalert.ProviderClaude,
			Resource: "messages",
			Window:   "five-hour",
		})
	}
	identities = append(identities, identities[0])
	states, err := store.LoadStates(ctx, identities)
	if err != nil {
		t.Fatalf("LoadStates(large input) error = %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("LoadStates(large input) returned %d states, want 0", len(states))
	}

	oversizedCursor := strings.Repeat("a", maxQuotaAlertCursorLength+1)
	if _, err = store.ListStates(ctx, quotaalert.PageRequest{Cursor: oversizedCursor, Limit: 1}); err == nil {
		t.Fatal("ListStates(oversized cursor) error = nil")
	}
	if _, err = store.ListEvents(ctx, quotaalert.PageRequest{Cursor: oversizedCursor, Limit: 1}); err == nil {
		t.Fatal("ListEvents(oversized cursor) error = nil")
	}
}

func TestPostgresQuotaAlertStateRejectsEqualTimeConflict(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Date(2026, time.July, 28, 16, 0, 0, 0, time.UTC)
	state, _ := quotaAlertTestStateAndEvent("equal-time", now)
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States: []quotaalert.CurrentState{state},
	})

	conflicting := state
	conflicting.Remaining = 6
	commit := quotaAlertTestCollectionCommit(t, ctx, store, quotaalert.CollectionCommit{
		States: []quotaalert.CurrentState{conflicting},
	})
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection() = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, commit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(equal-time conflict) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release() error = %v", err)
	}

	loaded, err := store.LoadStates(ctx, []quotaalert.StateIdentity{state.Identity})
	if err != nil {
		t.Fatalf("LoadStates() error = %v", err)
	}
	if len(loaded) != 1 || loaded[0].Remaining != state.Remaining || loaded[0].Revision != 1 {
		t.Fatalf("state after equal-time conflict = %#v", loaded)
	}

	newer := conflicting
	newer.ObservedAt = now.Add(time.Minute)
	newer.UpdatedAt = now.Add(time.Minute)
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States: []quotaalert.CurrentState{newer},
	})
	loaded, err = store.LoadStates(ctx, []quotaalert.StateIdentity{state.Identity})
	if err != nil {
		t.Fatalf("LoadStates(newer) error = %v", err)
	}
	if len(loaded) != 1 || loaded[0].Remaining != newer.Remaining || loaded[0].Revision != 2 {
		t.Fatalf("newer state = %#v", loaded)
	}
}

func TestPostgresQuotaAlertCollectionRejectsInvalidInitialTransitionHistory(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Date(2026, time.July, 28, 16, 30, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*quotaalert.CurrentState, *quotaalert.TransitionEvent)
	}{
		{
			name: "recovery",
			mutate: func(state *quotaalert.CurrentState, event *quotaalert.TransitionEvent) {
				state.Alert = quotaalert.AlertHealthy
				state.Remaining = 50
				event.Kind = quotaalert.TransitionRecovery
				event.From = quotaalert.AlertWarning
				event.To = quotaalert.AlertHealthy
				event.Remaining = 50
			},
		},
		{
			name: "reminder",
			mutate: func(_ *quotaalert.CurrentState, event *quotaalert.TransitionEvent) {
				event.Kind = quotaalert.TransitionReminder
				event.From = quotaalert.AlertWarning
				event.To = quotaalert.AlertWarning
			},
		},
		{
			name: "warning from healthy",
			mutate: func(_ *quotaalert.CurrentState, event *quotaalert.TransitionEvent) {
				event.From = quotaalert.AlertHealthy
			},
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, event := quotaAlertTestStateAndEvent(
				fmt.Sprintf("initial-history-%d", index),
				now.Add(time.Duration(index)*time.Minute),
			)
			test.mutate(&state, &event)
			commit := quotaAlertTestCollectionCommit(t, ctx, store, quotaalert.CollectionCommit{
				States: []quotaalert.CurrentState{state},
				Events: []quotaalert.TransitionEvent{event},
			})
			lease, acquired, err := store.TryAcquireCollection(ctx)
			if err != nil || !acquired {
				t.Fatalf("TryAcquireCollection() = acquired %t, error %v", acquired, err)
			}
			if err = store.CommitCollection(ctx, lease, commit); err == nil {
				_ = lease.Release(ctx)
				t.Fatal("CommitCollection(invalid initial transition) error = nil")
			}
			if err = lease.Release(ctx); err != nil {
				t.Fatalf("CollectionLease.Release() error = %v", err)
			}
		})
	}
}

func TestPostgresQuotaAlertCollectionValidatesPersistedTransitionHistory(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Date(2026, time.July, 28, 17, 0, 0, 0, time.UTC)
	state, event := quotaAlertTestStateAndEvent("history", now)
	batch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{event}, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch(initial) error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{state},
		Events:  []quotaalert.TransitionEvent{event},
		Batches: []quotaalert.NotificationBatch{batch},
	})

	duplicateState := state
	duplicateState.ObservedAt = now.Add(time.Minute)
	duplicateState.TransitionedAt = now.Add(time.Minute)
	duplicateState.UpdatedAt = now.Add(time.Minute)
	duplicateEvent := event
	duplicateEvent.ID = "event-history-duplicate-warning"
	duplicateEvent.OccurredAt = now.Add(time.Minute)
	duplicateBatch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{duplicateEvent}, duplicateEvent.OccurredAt)
	if err != nil {
		t.Fatalf("NewNotificationBatch(duplicate) error = %v", err)
	}
	duplicateCommit := quotaAlertTestCollectionCommit(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{duplicateState},
		Events:  []quotaalert.TransitionEvent{duplicateEvent},
		Batches: []quotaalert.NotificationBatch{duplicateBatch},
	})
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(duplicate) = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, duplicateCommit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(duplicate warning transition) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release(duplicate) error = %v", err)
	}

	recoveryAt := now.Add(2 * time.Minute)
	recoveryState := state
	recoveryState.Alert = quotaalert.AlertHealthy
	recoveryState.Remaining = 50
	recoveryState.ObservedAt = recoveryAt
	recoveryState.TransitionedAt = recoveryAt
	recoveryState.UpdatedAt = recoveryAt
	recoveryEvent := event
	recoveryEvent.ID = "event-history-recovery"
	recoveryEvent.Kind = quotaalert.TransitionRecovery
	recoveryEvent.From = quotaalert.AlertWarning
	recoveryEvent.To = quotaalert.AlertHealthy
	recoveryEvent.Remaining = 50
	recoveryEvent.OccurredAt = recoveryAt
	recoveryBatch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{recoveryEvent}, recoveryAt)
	if err != nil {
		t.Fatalf("NewNotificationBatch(recovery) error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{recoveryState},
		Events:  []quotaalert.TransitionEvent{recoveryEvent},
		Batches: []quotaalert.NotificationBatch{recoveryBatch},
	})

	states, err := store.LoadStates(ctx, []quotaalert.StateIdentity{state.Identity})
	if err != nil {
		t.Fatalf("LoadStates() error = %v", err)
	}
	if len(states) != 1 || states[0].Alert != quotaalert.AlertHealthy || states[0].Revision != 2 {
		t.Fatalf("state after recovery = %#v", states)
	}
	for table, want := range map[string]int{
		quotaAlertEventsTable:         2,
		quotaNotificationBatchesTable: 2,
	} {
		var count int
		if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(table))).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", table, count, want)
		}
	}
}

func TestPostgresQuotaAlertCollectionRejectsDuplicateEventIdentity(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Date(2026, time.July, 28, 17, 30, 0, 0, time.UTC)
	state, firstEvent := quotaAlertTestStateAndEvent("duplicate-identity", now)
	secondEvent := firstEvent
	secondEvent.ID = "event-duplicate-identity-second"
	commit := quotaAlertTestCollectionCommit(t, ctx, store, quotaalert.CollectionCommit{
		States: []quotaalert.CurrentState{state},
		Events: []quotaalert.TransitionEvent{firstEvent, secondEvent},
	})
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection() = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, commit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(duplicate event identity) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release() error = %v", err)
	}

	states, err := store.LoadStates(ctx, []quotaalert.StateIdentity{state.Identity})
	if err != nil {
		t.Fatalf("LoadStates() error = %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("states after rejected commit = %#v", states)
	}
	var eventCount int
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", store.fullTableName(quotaAlertEventsTable))).Scan(&eventCount); err != nil {
		t.Fatalf("count events after rejected commit: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("event count after rejected commit = %d, want 0", eventCount)
	}
}

func TestPostgresQuotaAlertCollectionCommitIsFencedAndRejectsStaleState(t *testing.T) {
	ctx, first, dsn, schema := newPostgresQuotaAlertTestStore(t)
	second, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore(second) error = %v", err)
	}
	defer second.Close()

	oldAt := time.Date(2026, time.July, 28, 7, 30, 0, 0, time.UTC)
	oldState, oldEvent := quotaAlertTestStateAndEvent("fenced", oldAt)
	oldEvent.ID = "event-fenced-old"
	oldBatch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{oldEvent}, oldAt)
	if err != nil {
		t.Fatalf("NewNotificationBatch(old) error = %v", err)
	}
	oldCommit := quotaAlertTestCollectionCommit(t, ctx, first, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{oldState},
		Events:  []quotaalert.TransitionEvent{oldEvent},
		Batches: []quotaalert.NotificationBatch{oldBatch},
	})

	staleLease, acquired, err := first.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(first) = acquired %t, error %v", acquired, err)
	}
	owned := staleLease.(*postgresQuotaAlertLease)
	var backendPID int
	if err = owned.conn.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&backendPID); err != nil {
		t.Fatalf("load collection backend PID: %v", err)
	}
	var terminated bool
	if err = second.db.QueryRowContext(ctx, "SELECT pg_terminate_backend($1)", backendPID).Scan(&terminated); err != nil {
		t.Fatalf("terminate collection backend: %v", err)
	}
	if !terminated {
		t.Fatal("collection backend was not terminated")
	}

	var currentLease quotaalert.CollectionLease
	for range 20 {
		currentLease, acquired, err = second.TryAcquireCollection(ctx)
		if err == nil && acquired {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(second) = acquired %t, error %v", acquired, err)
	}

	newAt := oldAt.Add(time.Minute)
	newState, newEvent := quotaAlertTestStateAndEvent("fenced", newAt)
	newEvent.ID = "event-fenced-new"
	newBatch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{newEvent}, newAt)
	if err != nil {
		t.Fatalf("NewNotificationBatch(new) error = %v", err)
	}
	newCommit := quotaAlertTestCollectionCommit(t, ctx, second, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{newState},
		Events:  []quotaalert.TransitionEvent{newEvent},
		Batches: []quotaalert.NotificationBatch{newBatch},
	})
	if err = second.CommitCollection(ctx, currentLease, newCommit); err != nil {
		t.Fatalf("CommitCollection(new owner) error = %v", err)
	}
	if err = first.CommitCollection(ctx, staleLease, oldCommit); err == nil {
		t.Fatal("CommitCollection(stale owner) error = nil")
	}
	if err = staleLease.Release(ctx); err == nil {
		t.Fatal("Release(terminated lease) error = nil")
	}
	if err = staleLease.Release(ctx); err != nil {
		t.Fatalf("Release(terminated lease retry) error = %v", err)
	}
	if err = currentLease.Release(ctx); err != nil {
		t.Fatalf("Release(current lease) error = %v", err)
	}

	freshLease, acquired, err := first.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(fresh) = acquired %t, error %v", acquired, err)
	}
	if err = first.CommitCollection(ctx, freshLease, oldCommit); err == nil {
		t.Fatal("CommitCollection(older observation) error = nil")
	}
	if err = freshLease.Release(ctx); err != nil {
		t.Fatalf("Release(fresh lease) error = %v", err)
	}

	states, err := first.LoadStates(ctx, []quotaalert.StateIdentity{newState.Identity})
	if err != nil {
		t.Fatalf("LoadStates() error = %v", err)
	}
	if len(states) != 1 || !states[0].ObservedAt.Equal(newAt) || states[0].Revision != 1 {
		t.Fatalf("fenced state = %#v", states)
	}
	for table, want := range map[string]int{
		quotaAlertEventsTable:         1,
		quotaNotificationBatchesTable: 1,
	} {
		var count int
		if err = first.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", first.fullTableName(table))).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", table, count, want)
		}
	}
}

func TestPostgresQuotaAlertBoundsCollectionAndStateLoadWork(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	state, _ := quotaAlertTestStateAndEvent(
		"work-bounds",
		time.Date(2026, time.July, 28, 19, 0, 0, 0, time.UTC),
	)
	identities := make(
		[]quotaalert.StateIdentity,
		maxLoadStateIdentities+1,
	)
	for index := range identities {
		identities[index] = state.Identity
	}
	if _, err := store.LoadStates(ctx, identities); err == nil ||
		!strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("LoadStates(oversized) error = %v", err)
	}

	states := make([]quotaalert.CurrentState, maxCollectionCommitItems)
	for index := range states {
		states[index] = state
	}
	commit := quotaalert.CollectionCommit{
		States:        states,
		RemovedStates: []quotaalert.StateIdentity{state.Identity},
	}
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection() = acquired %t, error %v", acquired, err)
	}
	if err = store.CommitCollection(ctx, lease, commit); err == nil ||
		!strings.Contains(err.Error(), "exceeds") {
		_ = lease.Release(ctx)
		t.Fatalf("CommitCollection(oversized) error = %v", err)
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release() error = %v", err)
	}
	var stateCount int
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM %s",
		store.fullTableName(quotaAlertStateTable),
	)).Scan(&stateCount); err != nil {
		t.Fatalf("count states after oversized commit: %v", err)
	}
	if stateCount != 0 {
		t.Fatalf("state count after oversized commit = %d, want 0", stateCount)
	}
}

func TestPostgresQuotaAlertBoundedEventRetention(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	oldestAt := time.Date(2026, time.July, 28, 6, 0, 0, 0, time.UTC)
	oldestState, oldest := quotaAlertTestStateAndEvent("retention-oldest", oldestAt)
	secondOldestState, secondOldest := quotaAlertTestStateAndEvent("retention-second", oldestAt.Add(time.Minute))
	newerState, newer := quotaAlertTestStateAndEvent("retention-newer", oldestAt.Add(2*time.Minute))
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States: []quotaalert.CurrentState{oldestState, secondOldestState, newerState},
		Events: []quotaalert.TransitionEvent{oldest, secondOldest, newer},
	})

	if _, err := store.PruneEvents(ctx, time.Time{}, 1); err == nil {
		t.Fatal("PruneEvents() error = nil for zero cutoff")
	}
	for _, limit := range []int{0, quotaalert.MaxPageSize + 1} {
		if _, err := store.PruneEvents(ctx, newer.OccurredAt, limit); err == nil {
			t.Fatalf("PruneEvents() error = nil for limit %d", limit)
		}
	}

	deleted, err := store.PruneEvents(ctx, newer.OccurredAt, 1)
	if err != nil {
		t.Fatalf("PruneEvents() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PruneEvents() deleted = %d, want 1", deleted)
	}

	rows, err := store.db.QueryContext(ctx, fmt.Sprintf(
		"SELECT id FROM %s ORDER BY occurred_at, id",
		store.fullTableName(quotaAlertEventsTable),
	))
	if err != nil {
		t.Fatalf("list retained events: %v", err)
	}
	defer rows.Close()
	var retained []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			t.Fatalf("scan retained event: %v", err)
		}
		retained = append(retained, id)
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("iterate retained events: %v", err)
	}
	want := []string{secondOldest.ID, newer.ID}
	if len(retained) != len(want) || retained[0] != want[0] || retained[1] != want[1] {
		t.Fatalf("retained events = %v, want %v", retained, want)
	}
}

func TestPostgresQuotaAlertBoundedTerminalNotificationRetention(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Now().UTC().Add(-time.Hour)
	var states []quotaalert.CurrentState
	var events []quotaalert.TransitionEvent
	var batches []quotaalert.NotificationBatch
	for index, suffix := range []string{"terminal-oldest", "terminal-second", "terminal-pending"} {
		state, event := quotaAlertTestStateAndEvent(suffix, now.Add(time.Duration(index)*time.Minute))
		batch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{event}, event.OccurredAt)
		if err != nil {
			t.Fatalf("NewNotificationBatch(%s) error = %v", suffix, err)
		}
		states = append(states, state)
		events = append(events, event)
		batches = append(batches, batch)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{States: states, Events: events, Batches: batches})

	updates := []struct {
		batchID   string
		status    string
		updatedAt time.Time
	}{
		{batchID: batches[0].ID(), status: "sent", updatedAt: now},
		{batchID: batches[1].ID(), status: "failed", updatedAt: now.Add(time.Minute)},
		{batchID: batches[2].ID(), status: "pending", updatedAt: now.Add(2 * time.Minute)},
	}
	for _, update := range updates {
		if _, err := store.db.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %s SET status = $1, updated_at = $2 WHERE id = $3
		`, store.fullTableName(quotaNotificationBatchesTable)), update.status, update.updatedAt, update.batchID); err != nil {
			t.Fatalf("set notification batch %s status: %v", update.batchID, err)
		}
	}

	if _, err := store.PruneNotificationBatches(ctx, time.Time{}, 1); err == nil {
		t.Fatal("PruneNotificationBatches() error = nil for zero cutoff")
	}
	for _, limit := range []int{0, quotaalert.MaxPageSize + 1} {
		if _, err := store.PruneNotificationBatches(ctx, time.Now().UTC(), limit); err == nil {
			t.Fatalf("PruneNotificationBatches() error = nil for limit %d", limit)
		}
	}
	deleted, err := store.PruneNotificationBatches(ctx, time.Now().UTC(), 1)
	if err != nil {
		t.Fatalf("PruneNotificationBatches(first) error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PruneNotificationBatches(first) deleted = %d, want 1", deleted)
	}
	deleted, err = store.PruneNotificationBatches(ctx, time.Now().UTC(), quotaalert.MaxPageSize)
	if err != nil {
		t.Fatalf("PruneNotificationBatches(second) error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PruneNotificationBatches(second) deleted = %d, want 1", deleted)
	}
	var retainedID string
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf("SELECT id FROM %s", store.fullTableName(quotaNotificationBatchesTable))).Scan(&retainedID); err != nil {
		t.Fatalf("load retained notification batch: %v", err)
	}
	if retainedID != batches[2].ID() {
		t.Fatalf("retained notification batch = %q, want pending %q", retainedID, batches[2].ID())
	}
}

func TestPostgresQuotaAlertQuarantinesTamperedNotificationPayload(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Now().UTC().Add(-time.Minute)
	state, event := quotaAlertTestStateAndEvent("tampered", now)
	batch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{event}, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{state},
		Events:  []quotaalert.TransitionEvent{event},
		Batches: []quotaalert.NotificationBatch{batch},
	})
	if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET events = jsonb_set(events, '{0,AuthLabel}', to_jsonb('Tampered label'::text))
		WHERE id = $1
	`, store.fullTableName(quotaNotificationBatchesTable)), batch.ID()); err != nil {
		t.Fatalf("tamper notification payload: %v", err)
	}
	claims, err := store.ClaimNotificationBatches(ctx, quotaalert.NotificationClaimOptions{Limit: 1, LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("ClaimNotificationBatches(tampered payload) error = %v", err)
	}
	if len(claims) != 0 {
		t.Fatalf("tampered notification claims = %#v, want none", claims)
	}
	var status, failureCode string
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT status, failure_code FROM %s WHERE id = $1
	`, store.fullTableName(quotaNotificationBatchesTable)), batch.ID()).Scan(&status, &failureCode); err != nil {
		t.Fatalf("load quarantined notification batch: %v", err)
	}
	if status != "failed" || failureCode != "invalid_payload" {
		t.Fatalf("quarantined notification batch = status %q, failure %q", status, failureCode)
	}

	validState, validEvent := quotaAlertTestStateAndEvent("after-tampered", now.Add(time.Second))
	validBatch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{validEvent}, validEvent.OccurredAt)
	if err != nil {
		t.Fatalf("NewNotificationBatch(valid) error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{validState},
		Events:  []quotaalert.TransitionEvent{validEvent},
		Batches: []quotaalert.NotificationBatch{validBatch},
	})
	claims, err = store.ClaimNotificationBatches(ctx, quotaalert.NotificationClaimOptions{Limit: 1, LeaseDuration: time.Minute})
	if err != nil {
		t.Fatalf("ClaimNotificationBatches(after quarantine) error = %v", err)
	}
	if len(claims) != 1 || claims[0].Batch.ID() != validBatch.ID() {
		t.Fatalf("claims after quarantine = %#v, want valid batch %q", claims, validBatch.ID())
	}
}

func TestPostgresQuotaAlertAdvisoryLockExclusivity(t *testing.T) {
	ctx, first, dsn, schema := newPostgresQuotaAlertTestStore(t)
	second, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore(second) error = %v", err)
	}
	defer second.Close()

	lease, acquired, err := first.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(owner check) = acquired %t, error %v", acquired, err)
	}
	state, _ := quotaAlertTestStateAndEvent(
		"cross-store-owner",
		time.Date(2026, time.July, 28, 18, 0, 0, 0, time.UTC),
	)
	commit := quotaAlertTestCollectionCommit(t, ctx, second, quotaalert.CollectionCommit{
		States: []quotaalert.CurrentState{state},
	})
	if err = second.CommitCollection(ctx, lease, commit); err == nil {
		_ = lease.Release(ctx)
		t.Fatal("CommitCollection(cross-store lease) error = nil")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release(owner check) error = %v", err)
	}

	type lockResult struct {
		lease    quotaalert.CollectionLease
		acquired bool
		err      error
	}
	start := make(chan struct{})
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	results := make(chan lockResult, 2)
	releaseErrors := make(chan error, 1)
	for _, candidate := range []*PostgresStore{first, second} {
		go func(store *PostgresStore) {
			ready <- struct{}{}
			<-start
			lease, acquired, lockErr := store.TryAcquireCollection(ctx)
			results <- lockResult{lease: lease, acquired: acquired, err: lockErr}
			if acquired {
				<-release
				releaseErrors <- lease.Release(ctx)
			}
		}(candidate)
	}
	<-ready
	<-ready
	close(start)
	firstResult := <-results
	secondResult := <-results
	if firstResult.err != nil || secondResult.err != nil {
		t.Fatalf("concurrent TryAcquireCollection() errors = %v, %v", firstResult.err, secondResult.err)
	}
	acquiredCount := 0
	for _, result := range []lockResult{firstResult, secondResult} {
		if result.acquired {
			acquiredCount++
			if result.lease == nil {
				t.Fatal("TryAcquireCollection() acquired without a lease")
			}
		} else if result.lease != nil {
			t.Fatal("TryAcquireCollection() returned a lease without ownership")
		}
	}
	close(release)
	for range acquiredCount {
		if err = <-releaseErrors; err != nil {
			t.Fatalf("CollectionLease.Release() error = %v", err)
		}
	}
	if acquiredCount != 1 {
		t.Fatalf("concurrent acquired count = %d, want 1", acquiredCount)
	}

	lease, acquired, err = second.TryAcquireCollection(ctx)
	if err != nil {
		t.Fatalf("TryAcquireCollection() after release error = %v", err)
	}
	if !acquired || lease == nil {
		t.Fatal("TryAcquireCollection() after release did not acquire lock")
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("lease Release() error = %v", err)
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("idempotent lease Release() error = %v", err)
	}

	canceledLease, acquired, err := first.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(canceled release) = acquired %t, error %v", acquired, err)
	}
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	if err = canceledLease.Release(canceledCtx); err != nil {
		t.Fatalf("Release(canceled context) error = %v", err)
	}
	reacquired, acquired, err := second.TryAcquireCollection(ctx)
	if err != nil || !acquired {
		t.Fatalf("TryAcquireCollection(after canceled release) = acquired %t, error %v", acquired, err)
	}
	if err = reacquired.Release(ctx); err != nil {
		t.Fatalf("Release(reacquired) error = %v", err)
	}
}

func TestPostgresQuotaAlertNotificationClaimExclusivityRetryAndExpiredLeaseRecovery(t *testing.T) {
	ctx, first, dsn, schema := newPostgresQuotaAlertTestStore(t)
	second, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore(second) error = %v", err)
	}
	defer second.Close()

	now := time.Now().UTC().Add(-time.Second)
	state, event := quotaAlertTestStateAndEvent("claim", now)
	batch, err := quotaalert.NewNotificationBatch(quotaalert.ProviderClaude, []quotaalert.TransitionEvent{event}, now)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, first, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{state},
		Events:  []quotaalert.TransitionEvent{event},
		Batches: []quotaalert.NotificationBatch{batch},
	})

	for _, leaseDuration := range []time.Duration{
		quotaalert.MinNotificationLeaseDuration - time.Nanosecond,
		quotaalert.MaxNotificationLeaseDuration + time.Nanosecond,
	} {
		if _, err = first.ClaimNotificationBatches(ctx, quotaalert.NotificationClaimOptions{Limit: 1, LeaseDuration: leaseDuration}); err == nil {
			t.Fatalf("ClaimNotificationBatches(lease duration %s) error = nil", leaseDuration)
		}
	}
	options := quotaalert.NotificationClaimOptions{Limit: 1, LeaseDuration: time.Minute}
	type claimResult struct {
		claims []quotaalert.NotificationClaim
		err    error
	}
	start := make(chan struct{})
	ready := make(chan struct{}, 2)
	results := make(chan claimResult, 2)
	for _, candidate := range []*PostgresStore{first, second} {
		go func(store *PostgresStore) {
			ready <- struct{}{}
			<-start
			claims, claimErr := store.ClaimNotificationBatches(ctx, options)
			results <- claimResult{claims: claims, err: claimErr}
		}(candidate)
	}
	<-ready
	<-ready
	close(start)
	firstResult := <-results
	secondResult := <-results
	if firstResult.err != nil || secondResult.err != nil {
		t.Fatalf("concurrent ClaimNotificationBatches() errors = %v, %v", firstResult.err, secondResult.err)
	}
	allClaims := append(firstResult.claims, secondResult.claims...)
	if len(allClaims) != 1 || allClaims[0].Attempt != 1 {
		t.Fatalf("concurrent claims = %#v, want one first attempt", allClaims)
	}
	initialClaim := allClaims[0]
	var leaseSeconds float64
	if err = first.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT EXTRACT(EPOCH FROM (lease_until - clock_timestamp()))
		FROM %s WHERE id = $1
	`, first.fullTableName(quotaNotificationBatchesTable)), batch.ID()).Scan(&leaseSeconds); err != nil {
		t.Fatalf("load database-clock lease: %v", err)
	}
	if leaseSeconds < 50 || leaseSeconds > 61 {
		t.Fatalf("database-clock lease seconds = %v, want approximately 60", leaseSeconds)
	}

	if _, err = first.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET lease_until = clock_timestamp() - INTERVAL '1 second',
		    claimable_at = clock_timestamp() - INTERVAL '1 second'
		WHERE id = $1
	`, first.fullTableName(quotaNotificationBatchesTable)), batch.ID()); err != nil {
		t.Fatalf("expire notification lease: %v", err)
	}
	expiredClaims, err := second.ClaimNotificationBatches(ctx, options)
	if err != nil {
		t.Fatalf("expired ClaimNotificationBatches() error = %v", err)
	}
	if len(expiredClaims) != 1 || expiredClaims[0].Attempt != 2 || expiredClaims[0].LeaseID == initialClaim.LeaseID {
		t.Fatalf("expired claims = %#v", expiredClaims)
	}

	retryAt := time.Now().UTC().Add(5 * time.Minute)
	for _, invalidResult := range []quotaalert.NotificationResult{
		{SentAt: time.Now().UTC(), PermanentFailure: true},
		{SentAt: time.Now().UTC(), RetryAt: retryAt},
		{PermanentFailure: true, RetryAt: retryAt},
	} {
		invalidResult.BatchID = batch.ID()
		invalidResult.LeaseID = expiredClaims[0].LeaseID
		if err = second.ResolveNotification(ctx, invalidResult); err == nil {
			t.Fatalf("ResolveNotification(conflicting result %#v) error = nil", invalidResult)
		}
	}
	var pendingStatus, pendingLeaseID string
	if err = second.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT status, lease_id FROM %s WHERE id = $1
	`, second.fullTableName(quotaNotificationBatchesTable)), batch.ID()).Scan(&pendingStatus, &pendingLeaseID); err != nil {
		t.Fatalf("load notification after conflicting resolutions: %v", err)
	}
	if pendingStatus != "pending" || pendingLeaseID != expiredClaims[0].LeaseID {
		t.Fatalf("notification after conflicting resolutions = status %q, lease %q", pendingStatus, pendingLeaseID)
	}
	if err = second.ResolveNotification(ctx, quotaalert.NotificationResult{
		BatchID:     batch.ID(),
		LeaseID:     expiredClaims[0].LeaseID,
		RetryAt:     retryAt,
		FailureCode: "temporary_failure",
	}); err != nil {
		t.Fatalf("ResolveNotification(retry) error = %v", err)
	}
	beforeRetry, err := first.ClaimNotificationBatches(ctx, options)
	if err != nil {
		t.Fatalf("ClaimNotificationBatches(before retry) error = %v", err)
	}
	if len(beforeRetry) != 0 {
		t.Fatalf("claims before retry = %#v, want none", beforeRetry)
	}
	if _, err = first.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET available_at = clock_timestamp() - INTERVAL '1 second',
		    claimable_at = clock_timestamp() - INTERVAL '1 second'
		WHERE id = $1
	`, first.fullTableName(quotaNotificationBatchesTable)), batch.ID()); err != nil {
		t.Fatalf("make notification retry available: %v", err)
	}
	retryClaims, err := first.ClaimNotificationBatches(ctx, options)
	if err != nil {
		t.Fatalf("ClaimNotificationBatches(retry) error = %v", err)
	}
	if len(retryClaims) != 1 || retryClaims[0].Attempt != 3 {
		t.Fatalf("retry claims = %#v", retryClaims)
	}
	if err = first.ResolveNotification(ctx, quotaalert.NotificationResult{
		BatchID: batch.ID(),
		LeaseID: retryClaims[0].LeaseID,
		SentAt:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("ResolveNotification(sent) error = %v", err)
	}
	afterSent, err := second.ClaimNotificationBatches(ctx, options)
	if err != nil {
		t.Fatalf("ClaimNotificationBatches(after sent) error = %v", err)
	}
	if len(afterSent) != 0 {
		t.Fatalf("claims after sent = %#v, want none", afterSent)
	}
}

func TestPostgresQuotaAlertClaimRefreshesLeaseAfterValidationDelay(t *testing.T) {
	ctx, store, _, _ := newPostgresQuotaAlertTestStore(t)
	now := time.Now().UTC().Add(-time.Second)
	state, event := quotaAlertTestStateAndEvent("claim-refresh", now)
	batch, err := quotaalert.NewNotificationBatch(
		quotaalert.ProviderClaude,
		[]quotaalert.TransitionEvent{event},
		now,
	)
	if err != nil {
		t.Fatalf("NewNotificationBatch() error = %v", err)
	}
	commitPostgresQuotaAlertTestCollection(t, ctx, store, quotaalert.CollectionCommit{
		States:  []quotaalert.CurrentState{state},
		Events:  []quotaalert.TransitionEvent{event},
		Batches: []quotaalert.NotificationBatch{batch},
	})

	functionName := store.fullTableName("quota_alert_claim_delay")
	if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger AS $$
		BEGIN
			IF OLD.attempt_count = 0 AND NEW.attempt_count = 1 THEN
				PERFORM pg_sleep(1.25);
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`, functionName)); err != nil {
		t.Fatalf("create claim delay function: %v", err)
	}
	if _, err = store.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TRIGGER %s
		BEFORE UPDATE ON %s
		FOR EACH ROW EXECUTE FUNCTION %s()
	`,
		quoteIdentifier("quota_alert_claim_delay_trigger"),
		store.fullTableName(quotaNotificationBatchesTable),
		functionName,
	)); err != nil {
		t.Fatalf("create claim delay trigger: %v", err)
	}

	claims, err := store.ClaimNotificationBatches(ctx, quotaalert.NotificationClaimOptions{
		Limit:         1,
		LeaseDuration: quotaalert.MinNotificationLeaseDuration,
	})
	if err != nil {
		t.Fatalf("ClaimNotificationBatches() error = %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claims = %#v, want one", claims)
	}
	if remaining := time.Until(claims[0].LeaseUntil); remaining < 500*time.Millisecond {
		t.Fatalf("refreshed lease remaining = %s, want at least 500ms", remaining)
	}
	var leaseUntil, claimableAt time.Time
	if err = store.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT lease_until, claimable_at FROM %s WHERE id = $1
	`, store.fullTableName(quotaNotificationBatchesTable)), batch.ID()).Scan(&leaseUntil, &claimableAt); err != nil {
		t.Fatalf("load refreshed claim schedule: %v", err)
	}
	if !leaseUntil.Equal(claims[0].LeaseUntil) || !claimableAt.Equal(leaseUntil) {
		t.Fatalf(
			"claim schedule = lease %s, claimable %s; returned lease %s",
			leaseUntil,
			claimableAt,
			claims[0].LeaseUntil,
		)
	}
}

func quotaAlertTestCollectionCommit(t *testing.T, ctx context.Context, store *PostgresStore, commit quotaalert.CollectionCommit) quotaalert.CollectionCommit {
	t.Helper()
	if commit.SettingsRevision > 0 {
		return commit
	}
	settings, err := store.LoadSettings(ctx)
	if err != nil {
		t.Fatalf("LoadSettings() for collection commit error = %v", err)
	}
	commit.SettingsRevision = settings.Revision
	return commit
}

func commitPostgresQuotaAlertTestCollection(t *testing.T, ctx context.Context, store *PostgresStore, commit quotaalert.CollectionCommit) {
	t.Helper()
	commit = quotaAlertTestCollectionCommit(t, ctx, store, commit)
	lease, acquired, err := store.TryAcquireCollection(ctx)
	if err != nil {
		t.Fatalf("TryAcquireCollection() error = %v", err)
	}
	if !acquired || lease == nil {
		t.Fatal("TryAcquireCollection() did not acquire ownership")
	}
	if err = store.CommitCollection(ctx, lease, commit); err != nil {
		_ = lease.Release(ctx)
		t.Fatalf("CommitCollection() error = %v", err)
	}
	if err = lease.Release(ctx); err != nil {
		t.Fatalf("CollectionLease.Release() error = %v", err)
	}
}

func newPostgresQuotaAlertTestStore(t *testing.T) (context.Context, *PostgresStore, string, string) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LLMHUB_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("LLMHUB_POSTGRES_TEST_DSN is not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("llmhub_quota_alert_test_%d", time.Now().UnixNano())
	store, err := NewPostgresStore(ctx, PostgresStoreConfig{DSN: dsn, Schema: schema})
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
		dropTestSchema(t, dsn, schema)
	})
	if err = store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	return ctx, store, dsn, schema
}

func quotaAlertTestStateAndEvent(suffix string, occurredAt time.Time) (quotaalert.CurrentState, quotaalert.TransitionEvent) {
	identity := quotaalert.StateIdentity{
		AuthID:   "auth-" + suffix,
		Provider: quotaalert.ProviderClaude,
		Resource: "messages",
		Window:   "window-" + suffix,
	}
	state := quotaalert.CurrentState{
		Identity:       identity,
		AuthLabel:      "Account " + suffix,
		Alert:          quotaalert.AlertWarning,
		Health:         quotaalert.CollectionReliable,
		Remaining:      5,
		RemainingKnown: true,
		ResetAt:        occurredAt.Add(time.Hour),
		ResetKnown:     true,
		ObservedAt:     occurredAt,
		TransitionedAt: occurredAt,
		UpdatedAt:      occurredAt,
	}
	event := quotaalert.TransitionEvent{
		ID:             "event-" + suffix,
		Identity:       identity,
		AuthLabel:      state.AuthLabel,
		Kind:           quotaalert.TransitionWarning,
		From:           quotaalert.AlertUnknown,
		To:             quotaalert.AlertWarning,
		Remaining:      state.Remaining,
		RemainingKnown: true,
		ResetAt:        state.ResetAt,
		ResetKnown:     true,
		OccurredAt:     occurredAt,
	}
	return state, event
}
