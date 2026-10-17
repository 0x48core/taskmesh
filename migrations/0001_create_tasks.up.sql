CREATE TABLE tasks (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    type            text        NOT NULL,
    payload         jsonb       NOT NULL DEFAULT '{}'::jsonb,
    status          text        NOT NULL DEFAULT 'pending',
    priority        integer     NOT NULL DEFAULT 0,
    attempts        integer     NOT NULL DEFAULT 0,
    max_attempts    integer     NOT NULL DEFAULT 5,
    run_at          timestamptz NOT NULL DEFAULT now(),
    idempotency_key text,
    last_error      text,
    worker_id       text,
    heartbeat_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT tasks_type_not_empty CHECK (type <> ''),
    CONSTRAINT tasks_status_valid CHECK (status IN ('pending', 'running', 'succeeded', 'dead', 'cancelled')),
    CONSTRAINT tasks_attempts_valid CHECK (attempts >= 0 AND max_attempts >= 1)
);

-- Claim: pending tasks in pick order. Matches ORDER BY priority DESC, run_at, id.
CREATE INDEX tasks_claim_idx ON tasks (priority DESC, run_at, id) WHERE status = 'pending';

-- Reaper: running tasks by heartbeat age.
CREATE INDEX tasks_reap_idx ON tasks (heartbeat_at) WHERE status = 'running';

-- Idempotency: one task per (type, key); tasks without a key never conflict.
CREATE UNIQUE INDEX tasks_idempotency_idx ON tasks (type, idempotency_key) WHERE idempotency_key IS NOT NULL;