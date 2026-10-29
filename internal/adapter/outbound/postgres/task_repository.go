// pgx implementation of port.TaskRepository. ALL SQL lives here.
package postgres

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/0x48core/taskmesh/internal/core/domain"
	"github.com/0x48core/taskmesh/internal/core/port"
)

var _ port.TaskRepository = (*TaskRepository)(nil)

// TaskRepository stores tasks in the tasks table.
type TaskRepository struct {
	db DBTX
}

// NewTaskRepository returns a repository that runs its queries on db.
func NewTaskRepository(db DBTX) *TaskRepository {
	return &TaskRepository{db: db}
}

// taskColumns is the column list every query selects or returns, in scanTask order.
const taskColumns = `id, type, payload, status, priority, attempts, max_attempts, run_at,
	idempotency_key, last_error, worker_id, heartbeat_at, created_at, updated_at`

const enqueueSQL = `
INSERT INTO tasks (type, payload, status, priority, max_attempts, run_at, idempotency_key)
VALUES ($1, $2, $3, $4, $5, COALESCE($6, now()), $7)
RETURNING ` + taskColumns

func (r *TaskRepository) Enqueue(ctx context.Context, task domain.Task) (domain.Task, error) {
	row := r.db.QueryRow(ctx, enqueueSQL,
		task.Type,
		task.Payload,
		task.Status,
		task.Priority,
		task.MaxAttempts,
		nullTime(task.RunAt),
		nullString(task.IdempotencyKey),
	)
	stored, err := scanTask(row)
	if err != nil {
		return domain.Task{}, fmt.Errorf("postgres: enqueue %s task: %w", task.Type, err)
	}
	return stored, nil
}

const getSQL = `SELECT ` + taskColumns + ` FROM tasks WHERE id = $1`

func (r *TaskRepository) Get(ctx context.Context, id string) (domain.Task, error) {
	task, err := scanTask(r.db.QueryRow(ctx, getSQL, id))
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Task{}, fmt.Errorf("%w: %s", domain.ErrTaskNotFound, id)
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("postgres: get task %s: %w", id, err)
	}
	return task, nil
}

// claimSQL picks due tasks in claim-index order and marks them running in one statement.
// SKIP LOCKED makes concurrent claimers pass over rows another transaction has locked,
// so no task is handed out twice. MATERIALIZED keeps the pick from being re-run per row;
// picked_id avoids an ambiguous "id" in RETURNING.
const claimSQL = `
WITH picked AS MATERIALIZED (
	SELECT id AS picked_id FROM tasks
	WHERE status = 'pending' AND run_at <= now()
	ORDER BY priority DESC, run_at, id
	LIMIT $2
	FOR UPDATE SKIP LOCKED
)
UPDATE tasks t
SET status = 'running',
	worker_id = $1,
	heartbeat_at = now(),
	attempts = t.attempts + 1,
	updated_at = now()
FROM picked
WHERE t.id = picked.picked_id
RETURNING ` + taskColumns

func (r *TaskRepository) Claim(ctx context.Context, workerID string, limit int) ([]domain.Task, error) {
	if limit < 1 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, claimSQL, workerID, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: claim: %w", err)
	}
	tasks, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Task, error) {
		return scanTask(row)
	})
	if err != nil {
		return nil, fmt.Errorf("postgres: claim: %w", err)
	}

	// RETURNING has no guaranteed order; put the batch back in claim order.
	slices.SortFunc(tasks, func(a, b domain.Task) int {
		return cmp.Or(
			cmp.Compare(b.Priority, a.Priority),
			a.RunAt.Compare(b.RunAt),
			cmp.Compare(a.ID, b.ID),
		)
	})
	return tasks, nil
}

const completeSQL = `
UPDATE tasks
SET status = 'succeeded', updated_at = now()
WHERE id = $1 AND status = 'running' AND worker_id = $2`

func (r *TaskRepository) Complete(ctx context.Context, id, workerID string) error {
	tag, err := r.db.Exec(ctx, completeSQL, id, workerID)
	if err != nil && !isInvalidUUID(err) {
		return fmt.Errorf("postgres: complete task %s: %w", id, err)
	}
	if err == nil && tag.RowsAffected() == 1 {
		return nil
	}

	// Nothing updated: find out why, for a useful error. Rare, so the extra query is fine.
	task, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: task %s is %s for worker %q, cannot complete as %q",
		domain.ErrInvalidState, id, task.Status, task.WorkerID, workerID)
}

// scanTask reads one row selected with taskColumns. NULL columns become zero values.
func scanTask(row pgx.Row) (domain.Task, error) {
	var (
		t                      domain.Task
		idemKey, lastErr, wkID pgtype.Text
		heartbeat              pgtype.Timestamptz
	)
	err := row.Scan(
		&t.ID, &t.Type, &t.Payload, &t.Status, &t.Priority, &t.Attempts, &t.MaxAttempts, &t.RunAt,
		&idemKey, &lastErr, &wkID, &heartbeat, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return domain.Task{}, err
	}
	t.IdempotencyKey = idemKey.String
	t.LastError = lastErr.String
	t.WorkerID = wkID.String
	t.HeartbeatAt = heartbeat.Time
	return t, nil
}

// nullString stores "" as NULL.
func nullString(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// nullTime stores the zero time as NULL.
func nullTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

// isInvalidUUID reports a malformed ID. No task can have one, so callers treat it as not found.
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02" // invalid_text_representation
}
