// Driven ports: TaskRepository, Notifier, MetricsRecorder.
package port

import (
	"context"

	"github.com/0x48core/taskmesh/internal/core/domain"
)

// TaskRepository stores tasks. The Postgres adapter is the real implementation;
// mocks.TaskRepository is an in-memory one for core tests. Both must pass porttest.TaskRepository.
//
// Phase 2 adds Fail, Heartbeat, ReapStale, Cancel and Retry. Notifier and MetricsRecorder
// join in phases 6 and 4.
type TaskRepository interface {
	// Enqueue stores a new pending task and returns it with ID and timestamps filled in.
	// A zero RunAt is stored as the repository's current time.
	Enqueue(ctx context.Context, task domain.Task) (domain.Task, error)

	// Get returns the task with the given ID, or domain.ErrTaskNotFound.
	Get(ctx context.Context, id string) (domain.Task, error)

	// Claim moves up to limit due pending tasks to running for workerID and returns them,
	// highest priority first, then oldest run_at. Concurrent callers never receive the same task.
	// A limit below 1 claims nothing.
	Claim(ctx context.Context, workerID string, limit int) ([]domain.Task, error)

	// Complete marks a running task owned by workerID as succeeded. It returns
	// domain.ErrTaskNotFound for an unknown ID and domain.ErrInvalidState when the task
	// is not running or belongs to another worker.
	Complete(ctx context.Context, id, workerID string) error
}
