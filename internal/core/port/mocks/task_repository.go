// In-memory/mock TaskRepository for core service tests.
package mocks

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/0x48core/taskmesh/internal/core/domain"
	"github.com/0x48core/taskmesh/internal/core/port"
)

var _ port.TaskRepository = (*TaskRepository)(nil)

// TaskRepository is an in-memory port.TaskRepository. It applies status changes through
// the domain methods, so it follows the same rules as the entity. Safe for concurrent use.
type TaskRepository struct {
	// Now is the repository's clock. Defaults to time.Now; set it before first use.
	Now func() time.Time

	mu     sync.Mutex
	tasks  map[string]*domain.Task
	nextID int
}

// NewTaskRepository returns an empty repository using the real clock.
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{Now: time.Now, tasks: make(map[string]*domain.Task)}
}

func (r *TaskRepository) Enqueue(_ context.Context, task domain.Task) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now()
	r.nextID++
	// Same shape as a UUID so callers can't tell the fake from Postgres by ID format.
	task.ID = fmt.Sprintf("00000000-0000-0000-0000-%012d", r.nextID)
	if task.RunAt.IsZero() {
		task.RunAt = now
	}
	task.CreatedAt, task.UpdatedAt = now, now

	r.tasks[task.ID] = clone(task)
	return *clone(task), nil
}

func (r *TaskRepository) Get(_ context.Context, id string) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return domain.Task{}, fmt.Errorf("%w: %s", domain.ErrTaskNotFound, id)
	}
	return *clone(*task), nil
}

func (r *TaskRepository) Claim(_ context.Context, workerID string, limit int) ([]domain.Task, error) {
	if limit < 1 {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now()
	var due []*domain.Task
	for _, task := range r.tasks {
		if task.Due(now) {
			due = append(due, task)
		}
	}
	slices.SortFunc(due, claimOrder)

	claimed := make([]domain.Task, 0, min(limit, len(due)))
	for _, task := range due[:min(limit, len(due))] {
		if err := task.Claim(workerID, now); err != nil {
			return nil, err
		}
		claimed = append(claimed, *clone(*task))
	}
	return claimed, nil
}

func (r *TaskRepository) Complete(_ context.Context, id, workerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrTaskNotFound, id)
	}
	return task.Complete(workerID, r.Now())
}

// claimOrder matches the Postgres claim: priority DESC, run_at, id.
func claimOrder(a, b *domain.Task) int {
	return cmp.Or(
		cmp.Compare(b.Priority, a.Priority),
		a.RunAt.Compare(b.RunAt),
		cmp.Compare(a.ID, b.ID),
	)
}

// clone copies a task so callers never share its payload slice with the store.
func clone(t domain.Task) *domain.Task {
	t.Payload = slices.Clone(t.Payload)
	return &t
}
