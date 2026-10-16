# TaskMesh

A distributed task queue for Go, built on PostgreSQL. No Redis, no Kafka, just your database.

TaskMesh stores tasks as rows in Postgres, so you can enqueue a task in the same
transaction as your business data. Either both commit or neither does, which
removes the dual write problem by design. Multiple workers claim tasks safely
using `FOR UPDATE SKIP LOCKED`, and failed or abandoned tasks are retried
automatically.

## Features

- **Transactional enqueue** with `EnqueueTx`: tasks commit atomically with your data
- **Safe concurrent workers** via `FOR UPDATE SKIP LOCKED`, no task is claimed twice
- **Retries** with exponential backoff and jitter, plus a dead-letter state
- **Crash recovery** through worker heartbeats and a background reaper
- **Idempotent enqueue** using idempotency keys
- **Priorities and delayed tasks**
- **Graceful shutdown** that lets in-flight tasks finish
- **HTTP API and Go client library** sharing one code path
- **Prometheus metrics** for queue depth, throughput, and failures

> Delivery guarantee: at-least-once. Handlers should be idempotent.