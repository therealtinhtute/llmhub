package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

var cooldownTestDriverID atomic.Uint64

type cooldownTestDriver struct {
	state *cooldownTestState
}

type cooldownTestState struct {
	mu      sync.Mutex
	rows    map[string]cooldownTestRow
	queries []string
}

type cooldownTestRow struct {
	content   []byte
	deleted   bool
	updatedAt time.Time
}

type cooldownTestConn struct {
	state *cooldownTestState
}

type cooldownTestTx struct{}

type cooldownTestRows struct {
	rows  []cooldownTestRow
	index int
}

func (d *cooldownTestDriver) Open(string) (driver.Conn, error) {
	return &cooldownTestConn{state: d.state}, nil
}

func (c *cooldownTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *cooldownTestConn) Close() error              { return nil }
func (c *cooldownTestConn) Begin() (driver.Tx, error) { return &cooldownTestTx{}, nil }

func (c *cooldownTestConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.queries = append(c.state.queries, query)
	if !strings.Contains(query, "INSERT INTO") || (len(args) != 4 && len(args) != 5) {
		return driver.RowsAffected(1), nil
	}
	authID, okAuthID := args[0].Value.(string)
	model, okModel := args[1].Value.(string)
	content, okContent := args[2].Value.([]byte)
	updatedAt, okUpdatedAt := args[3].Value.(time.Time)
	if !okAuthID || !okModel || !okContent || !okUpdatedAt {
		return nil, errors.New("invalid cooldown query arguments")
	}
	key := authID + "\x00" + model
	current, exists := c.state.rows[key]
	if len(args) == 4 {
		if !exists || !current.updatedAt.After(updatedAt) {
			c.state.rows[key] = cooldownTestRow{content: append([]byte(nil), content...), updatedAt: updatedAt}
		}
		return driver.RowsAffected(1), nil
	}
	observedAt, okObservedAt := args[4].Value.(time.Time)
	if !okObservedAt {
		return nil, errors.New("invalid cooldown delete version")
	}
	if !exists || (!current.deleted && !current.updatedAt.After(observedAt)) {
		c.state.rows[key] = cooldownTestRow{content: append([]byte(nil), content...), deleted: true, updatedAt: updatedAt}
	}
	return driver.RowsAffected(1), nil
}

func (c *cooldownTestConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.queries = append(c.state.queries, query)
	rows := make([]cooldownTestRow, 0, len(c.state.rows))
	for _, row := range c.state.rows {
		if !row.deleted {
			row.content = append([]byte(nil), row.content...)
			rows = append(rows, row)
		}
	}
	return &cooldownTestRows{rows: rows}, nil
}

func (*cooldownTestTx) Commit() error         { return nil }
func (*cooldownTestTx) Rollback() error       { return nil }
func (r *cooldownTestRows) Columns() []string { return []string{"content", "updated_at"} }
func (r *cooldownTestRows) Close() error      { return nil }

func (r *cooldownTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	dest[0] = r.rows[r.index].content
	dest[1] = r.rows[r.index].updatedAt
	r.index++
	return nil
}

func TestPostgresStoreCooldownSaveLoadAndClear(t *testing.T) {
	state := &cooldownTestState{rows: make(map[string]cooldownTestRow)}
	store, cooldownStore := newCooldownTestStore(t, state)

	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	if got := store.CooldownStateStore(); got != cooldownStore {
		t.Fatalf("CooldownStateStore() = %T, want configured PostgreSQL store", got)
	}

	nextRetry := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	records := []cliproxyauth.CooldownStateRecord{{
		Provider:       "codex",
		AuthID:         "account-1",
		Model:          "gpt-test",
		Status:         string(cliproxyauth.StatusError),
		NextRetryAfter: nextRetry,
		Reason:         "rate limited",
		UpdatedAt:      nextRetry.Add(-time.Minute),
	}}
	if err := cooldownStore.Save(context.Background(), records); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := cooldownStore.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, records) {
		t.Fatalf("Load() = %#v, want %#v", loaded, records)
	}

	if err = cooldownStore.Save(context.Background(), []cliproxyauth.CooldownStateRecord{{AuthID: "account-2", Model: "gpt-test"}}); err != nil {
		t.Fatalf("Save() with zero UpdatedAt error = %v", err)
	}
	loaded, err = cooldownStore.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() after zero UpdatedAt error = %v", err)
	}
	if len(loaded) != 1 || loaded[0].UpdatedAt.IsZero() {
		t.Fatalf("Load() did not persist a normalized UpdatedAt: %#v", loaded)
	}

	if err = cooldownStore.Save(context.Background(), nil); err != nil {
		t.Fatalf("Save(nil) error = %v", err)
	}
	loaded, err = cooldownStore.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() after Save(nil) error = %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("Load() after Save(nil) returned %d records, want 0", len(loaded))
	}

	state.mu.Lock()
	queries := strings.Join(state.queries, "\n")
	state.mu.Unlock()
	if !strings.Contains(queries, `CREATE TABLE IF NOT EXISTS "cooldown_store"`) {
		t.Fatalf("EnsureSchema() did not create cooldown table; queries:\n%s", queries)
	}
}

func TestPostgresCooldownConcurrentSnapshotsRejectStaleDeleteAndResurrection(t *testing.T) {
	state := &cooldownTestState{rows: make(map[string]cooldownTestRow)}
	store, _ := newCooldownTestStore(t, state)
	storeA := &postgresCooldownStateStore{store: store}
	storeB := &postgresCooldownStateStore{store: store}
	staleStore := &postgresCooldownStateStore{store: store}

	for _, cooldownStore := range []*postgresCooldownStateStore{storeA, storeB} {
		if _, err := cooldownStore.Load(context.Background()); err != nil {
			t.Fatalf("initial Load() error = %v", err)
		}
	}
	updatedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	recordA := cliproxyauth.CooldownStateRecord{AuthID: "account-a", Model: "model-a", UpdatedAt: updatedAt}
	recordB := cliproxyauth.CooldownStateRecord{AuthID: "account-b", Model: "model-b", UpdatedAt: updatedAt}
	if err := storeA.Save(context.Background(), []cliproxyauth.CooldownStateRecord{recordA}); err != nil {
		t.Fatalf("storeA.Save() error = %v", err)
	}
	if err := storeB.Save(context.Background(), []cliproxyauth.CooldownStateRecord{recordB}); err != nil {
		t.Fatalf("storeB.Save() error = %v", err)
	}
	staleRecords, err := staleStore.Load(context.Background())
	if err != nil {
		t.Fatalf("staleStore.Load() error = %v", err)
	}
	if len(staleRecords) != 2 {
		t.Fatalf("merged Load() returned %d records, want 2", len(staleRecords))
	}

	newerRecordA := recordA
	newerRecordA.UpdatedAt = updatedAt.Add(time.Hour)
	if err = storeA.Save(context.Background(), []cliproxyauth.CooldownStateRecord{newerRecordA}); err != nil {
		t.Fatalf("storeA.Save(newer) error = %v", err)
	}
	if err = staleStore.Save(context.Background(), []cliproxyauth.CooldownStateRecord{recordB}); err != nil {
		t.Fatalf("staleStore.Save(without newer record) error = %v", err)
	}
	resurrectStore := &postgresCooldownStateStore{store: store}
	activeRecords, err := resurrectStore.Load(context.Background())
	if err != nil {
		t.Fatalf("resurrectStore.Load() error = %v", err)
	}
	if len(activeRecords) != 2 {
		t.Fatalf("Load() after stale delete returned %d records, want 2", len(activeRecords))
	}

	if err = storeA.Save(context.Background(), nil); err != nil {
		t.Fatalf("storeA.Save(nil) error = %v", err)
	}
	if err = resurrectStore.Save(context.Background(), activeRecords); err != nil {
		t.Fatalf("resurrectStore.Save() error = %v", err)
	}
	reader := &postgresCooldownStateStore{store: store}
	loaded, err := reader.Load(context.Background())
	if err != nil {
		t.Fatalf("reader.Load() error = %v", err)
	}
	if len(loaded) != 1 || loaded[0].AuthID != recordB.AuthID {
		t.Fatalf("Load() after stale save = %#v, want only account-b", loaded)
	}
}

func TestPostgresCooldownRejectsDuplicateSnapshotIdentity(t *testing.T) {
	state := &cooldownTestState{rows: make(map[string]cooldownTestRow)}
	_, cooldownStore := newCooldownTestStore(t, state)
	record := cliproxyauth.CooldownStateRecord{AuthID: "account", Model: "model"}
	if err := cooldownStore.Save(context.Background(), []cliproxyauth.CooldownStateRecord{record, record}); err == nil {
		t.Fatal("Save() error = nil for duplicate snapshot identity")
	}
}

func newCooldownTestStore(t *testing.T, state *cooldownTestState) (*PostgresStore, *postgresCooldownStateStore) {
	t.Helper()
	driverName := fmt.Sprintf("llmhub_postgres_cooldown_test_%d", cooldownTestDriverID.Add(1))
	sql.Register(driverName, &cooldownTestDriver{state: state})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("db.Close() error = %v", closeErr)
		}
	})
	store := &PostgresStore{db: db}
	cooldownStore := &postgresCooldownStateStore{store: store}
	store.cooldownStore = cooldownStore
	return store, cooldownStore
}
