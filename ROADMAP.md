# TaskMesh Roadmap

Each phase leaves the project building, tested and usable on its own.
Each numbered step is roughly one PR. Tick the boxes as you go.

Phases 0 → 1 → 2 are strictly sequential. Phases 3 and 4 can be done in
either order. Phases 5 and 6 can wait until the core is stable.

---

## Phase 0: Foundations

**Goal:** a local Postgres with the schema applied, and config loading working.

- [x] 1. Dependencies: `go get` pgx v5, testcontainers-go (postgres module), a YAML library, google/wire.
- [x] 2. `services/taskmesh/migrations/0001_create_tasks.up.sql`
  - `tasks` table: `id`, `type`, `payload jsonb`, `status`, `priority`, `attempts`, `max_attempts`,
    `run_at`, `idempotency_key`, `last_error`, `worker_id`, `heartbeat_at`, timestamps.
  - Claim index: `(priority DESC, run_at, id) WHERE status = 'pending'`.
  - Reaper index: `(heartbeat_at) WHERE status = 'running'`.
  - Unique index: `(type, idempotency_key)`.
  - Matching `down.sql`, plus `embed.go` so tests can read the SQL files.
- [x] 3. `docker/docker-compose.dev.yaml` with Postgres only. Makefile targets `up`, `down`, `migrate` (golang-migrate).
- [x] 4. `pkg/env`, `pkg/gconfig`, `conf/*.yaml`: load `config.yaml` and merge `application-<env>.yaml` over it.
- [x] 5. `pkg/db/postgres.go`: create the pool with a ping and timeouts.

**Done when:** `make up && make migrate` gives you a `tasks` table, and a small test loads the config.

---

## Phase 1: Core library MVP (happy path)

**Goal:** enqueue a task and have a worker run it.

- [x] 1. `task.go`, `errors.go`, `config/config.go`
  - `Task` struct and `Status` constants.
  - Enqueue options: priority, run-at time, delay, max attempts.
  - Errors: `ErrTaskNotFound`, `ErrInvalidState`, `ErrSkipRetry`.
  - Config with defaults.
- [x] 2. `internal/store`
  - `Store` interface and its Postgres version of `Enqueue`, `Get`, `Claim`, `Complete`.
  - `Claim` is a single `UPDATE ... WHERE id IN (SELECT ... FOR UPDATE SKIP LOCKED)` statement.
  - `main_test.go` starts Postgres with testcontainers and applies the migrations.
  - Tests check that concurrent claimers never get the same task.
- [ ] 3. `Handler`, `HandlerFunc`, `Registry`
  - ⚠️ Decide where these live first. In `internal/handler`, other projects can't register their
    own handlers. The root package or a top-level `handler/` folder works.
- [ ] 4. `client.go`: `NewClient`, `Enqueue`, `Get`.
- [ ] 5. `worker.go`: the claim loop
  - A semaphore limits how many tasks run at once.
  - Claim in batches to fill free slots; sleep for the poll interval when the queue is empty.
  - Each task runs in its own goroutine; a handler panic is turned into an error.

**Done when:** an integration test enqueues 100 tasks and all of them reach `succeeded`.

---

## Phase 2: Reliability (the README's core promises)

**Goal:** failures, crashes and restarts never lose a task.

- [ ] 1. Retries
  - `backoff.go`: exponential backoff with full jitter; table-test it.
  - `store.Fail` retries the task or moves it to `dead`, based on `attempts >= max_attempts` and `ErrSkipRetry`.
- [ ] 2. `EnqueueTx(ctx, pgx.Tx, ...)`. Test both outcomes: rollback leaves no task, commit creates one.
- [ ] 3. Idempotency keys: `ON CONFLICT DO NOTHING`, then return the existing task.
- [ ] 4. Heartbeats
  - The worker refreshes `heartbeat_at` on every task it's running.
  - `Complete` and `Fail` only apply when `worker_id` still matches, so a worker that lost the task can't overwrite the result.
  - If a heartbeat shows the task was taken over, cancel the handler's context.
- [ ] 5. `reaper.go`: tasks whose heartbeat is older than `StaleAfter` go back to `pending`, or to `dead` if out of attempts.
  Concurrent reapers are safe through `SKIP LOCKED`.
- [ ] 6. Graceful shutdown and timeouts
  - On shutdown, stop claiming and let in-flight tasks finish within a time limit, then cancel the rest.
  - Per-task timeout.
- [ ] 7. `Cancel` and `Retry` on the Client.

**Done when** there's a test for each scenario:

- A task that fails twice and then succeeds.
- `ErrSkipRetry` moves the task straight to `dead`.
- Crash recovery, simulated by claiming with a worker ID that never sends heartbeats.
- A shutdown that waits for the running task to finish.

---

## Phase 3: Service

**Goal:** a deployable binary with the HTTP API.

- [ ] 1. `services/taskmesh/config`: service config mapped from `conf/`. Turns the API, worker and reaper
  on independently, so one binary can run as API-only, worker-only or all-in-one.
- [ ] 2. `handler/task`: enqueue, get, cancel, retry — thin wrappers over the Client.
- [ ] 3. `handler/demo`: `noop`, `sleep`, `flaky` handlers for testing and benchmarks.
- [ ] 4. `controller/rest.go` and `helper.go`
  - Routes: `POST /v1/tasks`, `GET /v1/tasks/{id}`, `POST /v1/tasks/{id}/cancel`, `POST /v1/tasks/{id}/retry`, `/healthz`.
  - Error mapping: 400 invalid input, 404 not found, 409 wrong status.
- [ ] 5. `deps/` with wire: providers for the pool, Client, Worker, Reaper. `make wire` generates `wire_gen.go`.
- [ ] 6. `app/app.go`: start the enabled parts with errgroup; shut down on SIGTERM.
- [ ] 7. `cmd/taskmesh/main.go`, `Dockerfile`; add the service to docker-compose.

**Done when:** after `make up`, you can enqueue a task with `curl` and it reaches `succeeded`.

---

## Phase 4: Observability

**Goal:** see queue health at a glance.

- [ ] 1. `internal/metrics`: hand-written Prometheus output (counters, gauges, histograms). Test against a fixed expected output.
- [ ] 2. Metrics to record
  - Tasks enqueued, succeeded, failed, dead, reaped — labelled by type.
  - Task run duration (histogram).
  - Tasks in flight.
  - Queue depth by status, read from Postgres on each scrape.
- [ ] 3. Serve `/metrics`; add Prometheus and Grafana to docker-compose with a provisioned dashboard.

**Done when:** the Grafana dashboard shows queue depth, throughput, failures and p95 duration while load is running.

---

## Phase 5: Examples, benchmarks, docs

- [ ] 1. `examples/basic`: enqueue a task and handle it.
- [ ] 2. `examples/outbox`: insert an order and call `EnqueueTx` in one transaction; show a rollback too.
- [ ] 3. `benchmarks/`
  - k6 script for enqueue throughput through the API.
  - Go benchmark for how fast workers drain the queue.
  - Record real numbers in `RESULTS.md`.
- [ ] 4. README: quick start, configuration reference, task state diagram, at-least-once guarantee.

---

## Phase 6: Enhancements

- [ ] 1. LISTEN/NOTIFY (`internal/notify`)
  - Enqueue sends `pg_notify` in the same transaction.
  - Each worker keeps one listening connection and wakes immediately instead of waiting for the next poll.
  - Polling stays as the fallback.
- [ ] 2. Retention: a cleanup job that deletes old `succeeded` and `cancelled` tasks.
- [ ] 3. Possible extras
  - Named queues with separate worker pools.
  - Cron-style recurring tasks.
  - Dead-letter admin API: list dead tasks and retry them in bulk.
