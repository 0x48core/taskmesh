// Repository tests against real Postgres.
package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0x48core/taskmesh/internal/core/domain"
	"github.com/0x48core/taskmesh/internal/core/port"
	"github.com/0x48core/taskmesh/internal/core/port/porttest"
)

func newRepo(t *testing.T) *TaskRepository {
	pg.Reset(t)
	return NewTaskRepository(pg.Pool)
}

func TestTaskRepository_Contract(t *testing.T) {
	porttest.TaskRepository(t, func(t *testing.T) port.TaskRepository { return newRepo(t) })
}

func TestTaskRepository_MalformedIDIsNotFound(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if _, err := repo.Get(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("Get: error = %v, want ErrTaskNotFound", err)
	}
	if err := repo.Complete(ctx, "not-a-uuid", "worker-1"); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("Complete: error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskRepository_StoresOptionalColumns(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	runAt := time.Now().Add(time.Hour).Truncate(time.Microsecond) // Postgres keeps microseconds

	task, err := domain.NewTask("email", nil, time.Now(),
		domain.WithRunAt(runAt), domain.WithIdempotencyKey("order-42"))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repo.Enqueue(ctx, task)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if !stored.RunAt.Equal(runAt) || stored.IdempotencyKey != "order-42" {
		t.Errorf("stored run at = %v key = %q, want %v and order-42", stored.RunAt, stored.IdempotencyKey, runAt)
	}

	// Empty strings must be stored as NULL, not '', so the partial unique index ignores them.
	var keyIsNull bool
	plain, err := repo.Enqueue(ctx, mustTask(t, "email"))
	if err != nil {
		t.Fatal(err)
	}
	err = pg.Pool.QueryRow(ctx, "SELECT idempotency_key IS NULL FROM tasks WHERE id = $1", plain.ID).Scan(&keyIsNull)
	if err != nil || !keyIsNull {
		t.Errorf("idempotency_key IS NULL = %v, %v; want true", keyIsNull, err)
	}
}

func mustTask(t *testing.T, taskType string) domain.Task {
	t.Helper()
	task, err := domain.NewTask(taskType, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return task
}
