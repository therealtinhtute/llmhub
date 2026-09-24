---
id: plan-20260924-pgcache
type: plan
intake_id: intake-20260924-pgcache
lane: normal
status: active
created: 2026-09-24
updated: 2026-09-24
---

# Plan: Postgres remote caching — cut watcher/request-path egress and latency against remote Supabase

## Outcome
- result: llmhub keeps Postgres as the authoritative store while collapsing steady-state DB chatter to one small revision probe per watcher tick and zero per-request queries on the `/v1` hot path — so a remote `PGSTORE_DSN` (Supabase pooler) no longer leaks ~10+ GB/month egress or adds RTTs to every proxied request.
- success_signals:
  - An idle server issues exactly 1 DB query per watcher tick (down from 3–4), with a response payload of ~hundreds of bytes regardless of auth count (measured via counting wrapper or query log).
  - `/v1` requests execute 0 Postgres queries for runtime controls and 0 synchronous usage inserts (verified by a counting-store test or query log during a request).
  - `GET /v0/management/config.yaml` and dashboard auth counts return without pulling full `content`/`config` payloads on every call.
  - Auth add/remove is still detected within one poll interval (2s); runtime-control changes apply within ≤ poll interval + margin.
  - Quota alert collection lease (advisory locks) works end-to-end against the documented Supabase session-mode DSN; the `postgresPingError` hint no longer recommends a mode that breaks it.
  - `TestPostgresStoreIntegration` passes against a real Postgres (jsonb payload normalization discrepancy resolved).
  - `go test ./...`, `make build`, `gofmt -l .`, `git diff --check` clean at gates.

## Authority and Requirements
- authority:
  - `internal/store/postgresstore.go` — `AuthVersion`/`AuthSnapshot` (SELECT all ids every tick :468), `List` selects `content` (:359), `NewPostgresStore` has no pool tuning (:61), `postgresPingError` recommends `pooler.supabase.com:6543` (:96).
  - `sdk/cliproxy/storage_watcher.go` — poll interval 2s (:18); `poll` = `pollConfig`+`pollAuth`+`pollSynthAuths` (:151); `pollSynthAuths`→`synthConfigAuths`→`nativeproviders.HydrateConfig` runs unconditionally every tick (:253-296).
  - `internal/store/native_provider.go:42` — hydration selects full `content` per provider.
  - `internal/api/server.go` — `runtimeControlRequestContext` calls `LoadRuntimeSettings` on every `/v1` + codexDirect request (:184, :503, :532); `CountAuthFiles` used at :1723.
  - `internal/redisqueue/queue.go:88` — `AppendUsage` executes synchronously inside `EnqueueUsage`.
  - `internal/store/postgres_quota_alert.go` — `pg_try_advisory_lock`/`pg_advisory_unlock` session locks (:494, :557) require session-mode pooling; `pg_advisory_xact_lock` (:524) is tx-scoped and pooler-safe.
  - `internal/api/handlers/management/handler.go:335` — bcrypt compare ~75ms on this host per management request when `remote-management.secret-key` hash is the auth path (operator-side fix: `MANAGEMENT_PASSWORD` env; not a code change).
  - Benchmarks measured 2026-09-24 on local Docker postgres:16: CurrentVersion 0.47ms, AuthSnapshot 1.44ms, LoadConfigBytes 0.87ms, AppendUsage 2.46ms, Save(auth) 4.30ms, PopUsage(100) 6.27ms — all ×WAN RTT when remote.
  - `docs/WORKFLOW.md`, `CLAUDE.md` — Postgres-authoritative store is a standing constraint; no alternative store reintroduction.
- rejected_alternatives:
  - LISTEN/NOTIFY push instead of polling — session-scoped, breaks on transaction poolers, adds reconnect complexity; the existing 2s poll is already the invalidation channel and stays.
  - Reintroducing file/SQLite store to avoid remote DB — contradicts the Postgres-authoritative architecture decision.
  - Per-request TTL cache on runtime settings without revision — simpler but drifts up to the TTL even for local writers; revision in the batched probe gives correctness at the same cost.
- requirements:
  - R1 [accepted]: **Batched revision probe** — `storageWatcher.poll` performs at most one DB round-trip per tick when nothing changed: config version, auth revision (count + max(updated_at) + order-stable hash over ids preserving add/remove detection), runtime-settings revision, and native-provider `max(updated_at)` fetched in a single query. | source: `storage_watcher.go:151`, `postgresstore.go:440-476`
  - R2 [accepted]: **Gated native hydration** — `synthConfigAuths`/`HydrateConfig` runs only when the native-provider revision from the probe changed; skipped ticks cost zero hydration queries. | source: `storage_watcher.go:253-296`, `native_provider.go:42`
  - R3 [accepted]: **Runtime controls cache** — `runtimeControlRequestContext` reads an in-memory settings snapshot invalidated by the watcher probe revision; `/v1` and codexDirect requests issue no Postgres query for settings. Staleness bound: one poll interval. | source: `server.go:175-196`, `postgres_runtime_controls.go:52`
  - R4 [accepted]: **Cached config bytes** — `GET /v0/management/config.yaml` serves watcher-cached raw YAML bytes keyed by config version (watcher already fetches on change); falls back to store on cache miss. | source: `storage_watcher.go:161-190`, `config_basic.go:147`
  - R5 [accepted]: **Count without content** — the `CountAuthFiles` call path uses a `SELECT count(*)`-level query; no `content` column is read for counting. | source: `util.go:97-112`, `postgresstore.go:359`
  - R6 [accepted]: **Batched usage appends** — `EnqueueUsage` buffers and flushes `AppendUsage` by count or interval with a bounded in-memory window; request path performs no synchronous insert. Documented trade-off: up to the flush window of usage rows may be lost on crash. | source: `redisqueue/queue.go:75-93`, `postgresstore.go:477`
  - R7 [accepted]: **Pool limits** — `NewPostgresStore` applies bounded pool settings (`SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxIdleTime`, `SetConnMaxLifetime`) with values recorded in Decisions; env overrides allowed but not required. | source: `postgresstore.go:74-76`
  - R8 [accepted]: **Pooler-mode guidance** — `postgresPingError` recommends Supabase session-mode pooling (port 5432) or direct IPv6, and warns that transaction-mode pooling (`:6543`) breaks session advisory locks used by quota collection. | source: `postgresstore.go:94-107`, `postgres_quota_alert.go:487-560`
  - R9 [accepted]: **Usage payload byte fidelity** — resolve `TestPostgresStoreIntegration` failure (`{"id":1}` vs jsonb-normalized `{"id": 1}`) by either storing payload as text/bytea or updating the contract test to assert semantic JSON equality; integration test passes against real Postgres. | source: `postgresstore.go:164-178`, `postgresstore_integration_test.go:106-117`
  - R10 [accepted]: **Gate** — `go test ./...`, `make build`, `gofmt -l .`, `git diff --check` clean. | source: PROJECT.md

## Non-goals
- Frontend pagination of `/auth-files` (`?page`/`page_size` already exists server-side; wiring the SPA is a separate UI initiative).
- `MANAGEMENT_PASSWORD` vs bcrypt deployment choice — operator config, documented as advisory only.
- LISTEN/NOTIFY, CDC, or any push channel; poll stays.
- Changing usage event schema, retention semantics, or quota-alert behavior.
- Supabase-side infra changes (tier, region) — advisory only.
- Multi-writer coordination beyond what the existing watcher/revision model already provides.

## Approach and Risks
- approach: single batched revision probe drives store-level caches. `PostgresStore` gains one `StorageRevisions(ctx)` query (config version, auth count+max(updated_at)+ordered-id md5, runtime-settings revision, native-provider max(updated_at)); `storageWatcher.poll` calls it once per tick instead of three separate probes. Revision-keyed caches inside `PostgresStore` serve `LoadRuntimeSettings` and `LoadConfigBytes` (invalidated by probe or local write); watcher gates `HydrateConfig` on the (configVersion, nativeMax) pair. Usage writes move to a buffered flush in `redisqueue` backed by a new `AppendUsageBatch` bulk insert. Count path gains `CountAuths` (`SELECT count(*)`) via interface assertion. Pool limits are fixed constants. `postgresPingError` points at session-mode pooler `:5432`; usage payload fidelity resolved by keeping JSONB and fixing the test to semantic JSON equality (payload stays SQL-queryable; no column migration on existing deployments).
- decisions (made at to-plan): staleness bound = one poll interval (2s); usage flush window = 5s or 100 rows, whichever first, flushed on shutdown; pool = MaxOpenConns 20, MaxIdleConns 8, ConnMaxIdleTime 2m, ConnMaxLifetime 10m (Supavisor session pool default is 20); R9 resolution = keep `payload JSONB`, assert semantic equality in test.
- rejected_alternatives:
  - TTL-only caches without revision — drifts for local writers; revision probe costs the same.
  - LISTEN/NOTIFY — session-scoped, dead on transaction poolers.
  - `payload` column JSONB→TEXT migration — loses queryability for zero semantic gain; rejected after confirming consumers parse JSON.
- risks:
  - Staleness: runtime controls/config now apply ≤2s late — mitigated by watcher-driven invalidation; acceptable per lock.
  - Crash loses ≤5s buffered usage rows — accepted in lock; flush also on shutdown hook.
  - Probe `md5(string_agg(id))` scans auth ids each tick — bounded (ids only, indexed PK); at extreme auth counts still << current full-snapshot cost.
  - `sql.Conn`-pinned advisory locks unchanged; only the DSN guidance changes (R8) — verify session-pooler claim against Supabase docs (confirmed: session mode `:5432` supports advisory locks + prepared statements; transaction `:6543` does not).
- recovery: each phase is additive (new methods/caches beside existing paths); rollback = revert phase commits; worst case watchers fall back to current per-poll queries by disabling cache via probe error path (probe failure must degrade to direct queries, never serve stale forever).

## Phases and Verification

### Phase `p1-pool-and-dsn` — story-20260924-pool
- status: checked
- goal: bounded connection pool + corrected Supabase DSN guidance (R7, R8).
- dependencies: none. surfaces: `internal/store/postgresstore.go` (NewPostgresStore, postgresPingError), `internal/store/postgresstore_init_test.go` or new unit test.
- wave 1 tasks:
  - t1: set `db.SetMaxOpenConns(20)`, `SetMaxIdleConns(8)`, `SetConnMaxIdleTime(2*time.Minute)`, `SetConnMaxLifetime(10*time.Minute)` in `NewPostgresStore` before `PingContext`. output: pool limits applied. check: `go test ./internal/store/ -run 'Postgres'` (unit green) + `go build ./...`.
  - t2: rewrite `postgresPingError` Supabase hint: recommend `postgres://postgres.<ref>:<pw>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require` (session mode) and state that transaction mode `:6543` breaks advisory locks used by quota collection. check: `go test ./internal/store/ -run 'Ping'` or grep assertion test; `go build ./...`.
- stop conditions: any change to DSN parsing semantics beyond the hint text → escalate to owner.

### Phase `p2-revision-probe` — story-20260924-probe
- status: checked
- goal: one `StorageRevisions` probe per watcher tick (R1).
- dependencies: none (foundation for p3). surfaces: `internal/store/postgresstore.go` (new method), `sdk/cliproxy/storage_watcher.go` (poll wiring, auth-version comparison), watcher tests.
- wave 2 tasks:
  - t1: add `StorageRevisions` struct + method on `PostgresStore`: single `SELECT` of scalar subqueries — `config_store.version` (key `default`), `auth_store` `count(*)`/`max(updated_at)`/`md5(string_agg(id,',' ORDER BY id))`, `runtime_control_settings.revision` (id 1), `native_provider_resources.max(updated_at)`; all NULL-tolerant for missing rows/tables. check: `LLMHUB_POSTGRES_TEST_DSN=... go test ./internal/store/ -run 'Revisions'` — new integration test asserting fields on empty + populated schema.
  - t2: rewire `storageWatcher.poll` to call `StorageRevisions` once; replace `CurrentVersion`+`AuthVersion` calls; derive auth-change detection from (count, max, hash) triple; keep existing dispatch/diff behavior identical. check: `go test ./sdk/cliproxy/ -run 'Watcher|Storage'` green; add test that a no-change poll issues exactly one query (counting driver or pgx tracer wrapper in test).
  - t3: probe failure handling — on probe error, log warn and skip tick (never leave caches stale-forever: caches hold last-good values; direct-query fallback documented in code path for first boot when revision unknown). check: unit test injecting failing store → poll returns error, no panic.
- stop conditions: probe query cannot cover a store implementation (non-postgres `RuntimeStorage`) → degrade: probe-capable interface assertion, fall back to legacy 3-call poll; record in Decisions.

### Phase `p4-usage-batch` — story-20260924-usagebatch
- status: checked
- goal: buffered usage appends, no synchronous insert on request path (R6).
- dependencies: none. surfaces: `internal/redisqueue/queue.go`, `internal/store/postgresstore.go` (new `AppendUsageBatch`), shutdown flush hook in `sdk/cliproxy/service.go` or main.
- wave 2 tasks:
  - t1: `AppendUsageBatch(ctx, payloads [][]byte, requestedAts []time.Time)` on PostgresStore — bulk `INSERT ... SELECT * FROM unnest($1::jsonb[], $2::timestamptz[])` preserving (requested_at,id) pop order. check: integration test `LLMHUB_POSTGRES_TEST_DSN=... go test ./internal/store/ -run 'UsageBatch'` asserts order + byte fidelity via PopUsage.
  - t2: `redisqueue` in-memory buffer: `EnqueueUsage` appends to slice; flusher goroutine drains on ≥100 rows or 5s ticker; existing `global.publishToSubscribers`/`enqueue` fallback path preserved; flush on service shutdown (locate existing shutdown path, add hook). check: `go test ./internal/redisqueue/` green incl. new flush-trigger tests (count-triggered and timer-triggered with fake clock or short interval).
- stop conditions: no clean shutdown hook exists → flush-at-Enqueue-cap only + document residual loss window; escalate if usage consumers require synchronous durability (verify `PopOldest` consumers first).

### Phase `p3-store-caches` — story-20260924-caches
- status: checked
- goal: revision-keyed in-memory caches for runtime settings, config bytes, and gated native hydration (R2, R3, R4).
- dependencies: p2 (probe exists). surfaces: `internal/store/postgresstore.go`, `internal/store/postgres_runtime_controls.go`, `sdk/cliproxy/storage_watcher.go`, `internal/api/handlers/management/config_basic.go`.
- wave 3 tasks:
  - t1: `runtimeSettingsCache` in PostgresStore — `LoadRuntimeSettings` returns cached settings keyed by revision; watcher probe invalidates on revision change (store method `ApplyStorageRevisions` or watcher calls invalidate); `SaveRuntimeSettings` updates/invalidates same-process cache. check: unit test — 2nd `LoadRuntimeSettings` = 0 queries (counting wrapper); `go test -race ./internal/store/`.
  - t2: same pattern for `LoadConfigBytes`/`SaveConfig` — cached raw bytes keyed by config version; `GetConfigYAML` unchanged (still calls `LoadConfigBytes`). check: counting test — repeat GET = 0 queries; `go test -race`.
  - t3: gate `synthConfigAuths`→`HydrateConfig` on (configVersion, nativeMaxUpdate) change pair from probe; unchanged → reuse last synthesized map without hydrating. check: watcher unit test counting hydrate invocations across ticks.
  - t4: staleness bound — end-to-end test: write via second store instance → watcher poll → `LoadRuntimeSettings` reflects change (≤1 tick). check: integration test with two stores sharing DSN.
- stop conditions: cache invalidation races that serve pre-write values to the writing process → fix by write-through invalidation before proceeding.

### Phase `p5-count-and-fidelity` — story-20260924-countfidelity
- status: checked
- goal: count path without `content` reads + usage payload test fix (R5, R9).
- dependencies: none. surfaces: `internal/store/postgresstore.go` (CountAuths), `internal/util/util.go` (interface assertion), `internal/store/postgresstore_integration_test.go` (semantic JSON assertion).
- wave 1 tasks:
  - t1: add `CountAuths(ctx) (int, error)` = `SELECT count(*) FROM auth_store`; `util.CountAuthFiles` type-asserts optional `Count(context.Context) (int, error)` interface, falls back to `List`. check: `go test ./internal/util/ ./internal/store/`; integration count test asserts no content column read (assert via query log/tracer or code inspection note).
  - t2: fix `TestPostgresStoreIntegration` PopUsage assertions — compare `json.Unmarshal`→`reflect.DeepEqual` (or `assert.JSONEq`) instead of byte equality; add comment noting JSONB normalization is the contract. check: `LLMHUB_POSTGRES_TEST_DSN=... go test ./internal/store/ -run 'TestPostgresStoreIntegration'` passes against real Postgres.
- stop conditions: none.

### Phase `p6-gate` — story-20260924-gate
- status: checked
- goal: full verification + evidence assembly (R10).
- dependencies: p1–p5.
- wave 4 tasks:
  - t1: `go test ./...` green; `make build` green; `gofmt -l .` empty; `git diff --check` clean.
  - t2: evidence summary appended to Validation: query-count assertions (1 probe/tick, 0 request-path queries), race-detector runs, integration DSN results.
- stop conditions: any gate failure → do not mark done; record failing command + output in Progress.

## Progress
- `2026-09-24T14:12Z` — phase: `p1-pool-and-dsn` (wave 1) — task: phase-start — task_status: in-progress — run anchor for wave 1 (p1 + p5 parallel)
- `2026-09-24T14:12Z` — phase: `p1-pool-and-dsn` (wave 1) — task: t1 pool limits — task_status: DONE — `internal/store/postgresstore.go` NewPostgresStore: SetMaxOpenConns(20), SetMaxIdleConns(8), SetConnMaxIdleTime(2m), SetConnMaxLifetime(10m) before PingContext; `/usr/local/go/bin/go build ./internal/store/` ok; `go test ./internal/store/ -run Postgres` ok
- `2026-09-24T14:12Z` — phase: `p1-pool-and-dsn` (wave 1) — task: t2 DSN hint — task_status: DONE — postgresPingError now recommends session-mode pooler `:5432` and warns `:6543` transaction mode breaks session advisory locks used by quota collection
- `2026-09-24T14:12Z` — phase: `p5-count-and-fidelity` (wave 1) — task: t1 CountAuths — task_status: DONE — `PostgresStore.CountAuths` = `SELECT count(*)`; `util.CountAuthFiles` type-asserts `CountAuths` interface, falls back to `List`
- `2026-09-24T14:12Z` — phase: `p5-count-and-fidelity` (wave 1) — task: t2 payload fidelity — task_status: DONE — PopUsage assertions switched to `jsonUsageEqual` (Unmarshal+DeepEqual); `TestPostgresStoreIntegration` PASS 1.60s on real postgres:16 container
- `2026-09-24T14:12Z` — wave 1 summary — phases p1, p5 verified and gated `checked`; surfaces: `internal/store/postgresstore.go`, `internal/util/util.go`, `internal/store/postgresstore_integration_test.go`; pre-existing `gofmt -l` flag on `internal/store/objectstore.go` (not in diff, from rebrand commit 691337a6 — left untouched)
- `2026-09-24T14:35Z` — phase: `p2-revision-probe` (wave 2) — task: phase-start — task_status: in-progress — run anchor for wave 2 (p2 + p4 parallel)
- `2026-09-24T14:35Z` — phase: `p2-revision-probe` (wave 2) — task: t1 StorageRevisions — task_status: DONE — single-SELECT probe over config_store/auth_store(count+max+md5(ids))/runtime_control_settings/native_provider_resources, NULL-tolerant; integration test PASS on postgres:16
- `2026-09-24T14:35Z` — phase: `p2-revision-probe` (wave 2) — task: t2 watcher rewire — task_status: DONE — `poll` uses `revisionProber` assertion → 1 probe call/tick (fake test: 3 polls=3 probes, 0 legacy calls); auth add detected via hash change; legacy 3-call path preserved for non-prober stores
- `2026-09-24T14:35Z` — phase: `p2-revision-probe` (wave 2) — task: t3 probe failure — task_status: DONE — probe error propagates → tick skipped with warn (existing ticker path); test `TestStorageWatcherProbeErrorSkipsTick`
- `2026-09-24T14:35Z` — phase: `p4-usage-batch` (wave 2) — task: t1 AppendUsageBatch — task_status: DONE — unnest(text[],timestamptz[]) bulk insert with p::jsonb cast; order + mismatch-length error covered by integration test
- `2026-09-24T14:35Z` — phase: `p4-usage-batch` (wave 2) — task: t2 redisqueue buffer — task_status: DONE — `EnqueueUsage` buffers (cap 10k, drop-oldest), flush on >=100 rows or 5s ticker or `FlushUsage`; batch via `usageBatcher` assertion, per-row fallback; `Service.Shutdown` calls `FlushUsage`
- `2026-09-24T14:35Z` — wave 2 summary — phases p2, p4 verified and gated `checked`; surfaces: postgresstore.go, storage_watcher.go(+test), queue.go(+test), service.go; pre-existing go-git v6 data race in gitstore tests (library-internal, unrelated to diff, left untouched)
- `2026-09-24T15:05Z` — phase: `p3-store-caches` (wave 3) — task: phase-start — task_status: in-progress — run anchor for wave 3
- `2026-09-24T15:05Z` — phase: `p3-store-caches` (wave 3) — task: t1 runtimeSettingsCache — task_status: DONE — rtCache{revision,settings} on PostgresStore; LoadRuntimeSettings serves Clone() while valid; SaveRuntimeSettings write-through post-commit; probe → `ObserveRevisions` invalidates remote revisions
- `2026-09-24T15:05Z` — phase: `p3-store-caches` (wave 3) — task: t2 configCache — task_status: DONE — configCache{version,content} wraps LoadConfigBytes; SaveConfig write-through; `GetConfigYAML` unchanged (still calls LoadConfigBytes)
- `2026-09-24T15:05Z` — phase: `p3-store-caches` (wave 3) — task: t3 gated hydration — task_status: DONE — watcher `probeMode` + hydrated{ConfigVersion,NativeMax} gate HydrateConfig; legacy stores keep per-tick hydration; test counts ListNativeProviderResources (first tick hydrates, unchanged skips, nativeMax bump rehydrates)
- `2026-09-24T15:05Z` — phase: `p3-store-caches` (wave 3) — task: t4 staleness bound — task_status: DONE — `TestPostgresRevisionKeyedCaches`: two stores on shared schema; st2 writes invisible to st1 until ObserveRevisions (≤1 tick bound); local write-through immediate
- `2026-09-24T15:05Z` — wave 3 summary — phase p3 verified and gated `checked`; surfaces: postgresstore.go, postgres_runtime_controls.go, storage_watcher.go(+test), integration test
- `2026-09-24T15:20Z` — phase: `p6-gate` (wave 4) — task: phase-start — task_status: in-progress — final gate run
- `2026-09-24T15:20Z` — phase: `p6-gate` (wave 4) — task: diff-review fix — task_status: DONE — review caught count-trigger flush running inline on request goroutine (1 RTT per 100 reqs, violated 0-request-path-write invariant); changed to non-blocking `usageFlushSignal` chan consumed by flusher loop; test fake store mutex-guarded (race under -race), count-trigger test polls with 2s deadline
- `2026-09-24T15:20Z` — phase: `p6-gate` (wave 4) — task: t1 full gates — task_status: DONE — `go test ./...` zero FAIL; `make build` green (web embed + binary, PATH=/usr/local/go/bin); `git diff --check` clean; `gofmt -l` clean on all touched files (38 pre-existing unformatted files repo-wide, none in diff — includes known `internal/store/objectstore.go`)

## Decisions
- `2026-09-24` to-plan: staleness bound = one poll interval (2s); usage flush = 5s or 100 rows + shutdown flush; pool = MaxOpenConns 20 / MaxIdleConns 8 / ConnMaxIdleTime 2m / ConnMaxLifetime 10m; R9 = keep `payload JSONB`, fix test to semantic equality (no column migration); Supabase guidance = session-mode pooler `:5432` (advisory locks safe), transaction `:6543` warned against.

## Validation

- `2026-09-24T14:12Z` — phase: `p1-pool-and-dsn` (wave 1) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: pool-limit behavior verified by build+unit tests only, not under live connection churn against a remote Supabase pooler
  - commands:
    - `/usr/local/go/bin/go build ./internal/store/` → ok
    - `/usr/local/go/bin/go test ./internal/store/ -run Postgres -count=1` → ok 0.072s
    - `/usr/local/go/bin/go build ./...` → clean
    - `git diff --check` → clean
  - receipt:
    context_sources: docs/plans/active/postgres-remote-caching.md
    policy: remote-postgres-caching
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: live Supabase session-pooler advisory-lock behavior end-to-end (Supabase docs confirm `:5432` session mode supports advisory locks, `:6543` does not)

- `2026-09-24T14:12Z` — phase: `p5-count-and-fidelity` (wave 1) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: `CountAuths` fast path exercised by unit/build only (no dedicated assertion that `content` is unread beyond code inspection of the single-column SELECT)
  - commands:
    - `docker run --rm -d --name llmhub-bench-pg -e POSTGRES_PASSWORD=bench -p 15432:5432 postgres:16` → container ready via pg_isready
    - `LLMHUB_POSTGRES_TEST_DSN="postgres://postgres:bench@localhost:15432/postgres?sslmode=disable" /usr/local/go/bin/go test ./internal/store/ -run TestPostgresStoreIntegration -v -count=1` → PASS 1.60s (was FAIL on jsonb normalization)
    - `LLMHUB_POSTGRES_TEST_DSN="postgres://postgres:bench@localhost:15432/postgres?sslmode=disable" /usr/local/go/bin/go test ./internal/store/ ./internal/util/ -count=1` → ok store 19.866s, util 0.829s
    - `git diff --check` → clean; `gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/postgres-remote-caching.md
    policy: remote-postgres-caching
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: dashboard `CountAuthFiles` call site (`server.go:1723`) not exercised against a live server

- `2026-09-24T14:35Z` — phase: `p2-revision-probe` (wave 2) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: probe equivalence with production DSN (Supabase pooler) not exercised; md5(string_agg) cost at very large auth counts not profiled
  - commands:
    - `LLMHUB_POSTGRES_TEST_DSN="postgres://postgres:bench@localhost:15432/postgres?sslmode=disable" /usr/local/go/bin/go test ./internal/store/ -run TestPostgresStorageRevisionsAndUsageBatch -v -count=1` → PASS 0.43s (empty-schema zero values, populated revisions, batch order)
    - `/usr/local/go/bin/go test ./sdk/cliproxy/ -run "TestStorageWatcher|TestService" -count=1` → ok 0.420s (3 new probe tests: single-query-per-tick, auth-change detection, probe-error skip)
    - `/usr/local/go/bin/go build ./...` → clean
    - `git diff --check` → clean; `gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/postgres-remote-caching.md
    policy: remote-postgres-caching
    judge: same-session
    judge_model: SWE-2 Max
    retries: 1 (integration test expectation corrected: runtime_control_settings seeds revision=1 at EnsureSchema)
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: watcher probe against live remote Supabase; concurrent multi-instance poll behavior

- `2026-09-24T14:35Z` — phase: `p4-usage-batch` (wave 2) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: flush-on-shutdown wiring verified by code inspection + unit tests, not a live SIGTERM run; PopOldest lag (≤ flush window) not measured end-to-end
  - commands:
    - `/usr/local/go/bin/go test -race -count=1 ./internal/redisqueue/` → ok 1.249s (buffer, count-trigger, error-requeue tests)
    - `LLMHUB_POSTGRES_TEST_DSN="postgres://postgres:bench@localhost:15432/postgres?sslmode=disable" /usr/local/go/bin/go test -race -count=1 ./internal/store/ -run "Postgres|Usage|Revision"` → ok 19.496s
    - `/usr/local/go/bin/go build ./...` → clean
    - `git diff --check` → clean; `gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/postgres-remote-caching.md
    policy: remote-postgres-caching
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: live shutdown flush on production deploy; usage-consumer tolerance of ≤5s pop lag

- `2026-09-24T15:05Z` — phase: `p3-store-caches` (wave 3) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: cache behaviour verified store-level; end-to-end "watcher poll → middleware LoadRuntimeSettings" wiring asserted by code path (poll calls ObserveRevisions on every probe tick), not a live service run; stores without watcher (init-db-from-env) serve stale cache for process lifetime — acceptable, short-lived
  - commands:
    - `LLMHUB_POSTGRES_TEST_DSN=... /usr/local/go/bin/go test ./internal/store/ -run "TestPostgresRevisionKeyedCaches|TestPostgresStorageRevisions" -v` → PASS 1.29s
    - `LLMHUB_POSTGRES_TEST_DSN=... /usr/local/go/bin/go test -race -count=1 ./internal/store/ -run "Postgres|RevisionKeyed|RuntimeSettings|RuntimeControl"` → ok 30.52s
    - `/usr/local/go/bin/go test -race -count=1 ./sdk/cliproxy/ -run "TestStorageWatcher|TestService"` → ok 1.61s (incl. new `TestStorageWatcherProbeGatesHydration`)
    - `/usr/local/go/bin/go build ./...` → clean; `gofmt -l` touched files → clean (after fixing test file alignment); `git diff --check` → clean
  - receipt:
    context_sources: docs/plans/active/postgres-remote-caching.md
    policy: remote-postgres-caching
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: /v1 request path running against live Supabase with caches warm

- `2026-09-24T15:20Z` — phase: `p6-gate` (wave 4) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: live Supabase egress/latency deltas not measured (no prod DSN in env); `gofmt -l .` repo-wide is non-empty due to 38 pre-existing unformatted files outside this diff — invariant scoped to touched files; go-git v6 race in gitstore tests remains (library-internal, pre-existing, unrelated)
  - commands:
    - `/usr/local/go/bin/go test ./...` → zero FAIL lines (full suite incl. cached + fresh runs)
    - `PATH=/usr/local/go/bin:$PATH make build` → green (bun web build 3.85s, embed copy, `go build -o llmhub`)
    - `/usr/local/go/bin/go test -race -count=1 ./internal/redisqueue/` → ok 1.29s (post signal-flush fix)
    - `git diff --check` → clean; `gofmt -l` on 10 touched files → empty
  - receipt:
    context_sources: docs/plans/active/postgres-remote-caching.md
    policy: remote-postgres-caching
    judge: same-session
    judge_model: SWE-2 Max
    retries: 1 (request-path sync flush found in diff review → signal-channel redesign → race fix in test fake)
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: production Supabase egress reduction; advisory-lock behaviour on live session pooler

## Current State and Next Action
- active_phase: none — all 6 phases `checked`
- lifecycle_status: complete
- latest anchors: wave 4 Progress + Validation entries `2026-09-24T15:20Z` (APPROVED); plan validated end-to-end
- blockers: none
- open items: live Supabase verification (egress metrics, session-pooler advisory locks) remains unproven; pre-existing go-git v6 race + repo-wide gofmt skew untouched
- exact_next_action: move plan to docs/plans/completed/; deploy + observe Supabase egress dashboard
