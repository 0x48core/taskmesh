// Package porttest holds contract tests that every implementation of a port must pass.
package porttest

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/0x48core/taskmesh/internal/core/domain"
	"github.com/0x48core/taskmesh/internal/core/port"
)

// missingID is a well-formed ID that no test ever creates.
const missingID = "00000000-0000-0000-0000-000000000000"

// TaskRepository runs the contract against an implementation. newRepo must return a
// repository with no tasks in it; it is called once per subtest.
func TaskRepository(t *testing.T, newRepo func(t *testing.T) port.TaskRepository) {
	t.Run("EnqueueAndGet", func(t *testing.T) { testEnqueueAndGet(t, newRepo(t)) })
	t.Run("GetMissing", func(t *testing.T) { testGetMissing(t, newRepo(t)) })
	t.Run("ClaimOrderAndLimit", func(t *testing.T) { testClaimOrderAndLimit(t, newRepo(t)) })
	t.Run("ClaimSkipsFutureTasks", func(t *testing.T) { testClaimSkipsFutureTasks(t, newRepo(t)) })
	t.Run("ClaimZeroLimit", func(t *testing.T) { testClaimZeroLimit(t, newRepo(t)) })
	t.Run("ConcurrentClaimers", func(t *testing.T) { testConcurrentClaimers(t, newRepo(t)) })
	t.Run("Complete", func(t *testing.T) { testComplete(t, newRepo(t)) })
}

func enqueue(t *testing.T, repo port.TaskRepository, taskType string, opts ...domain.Option) domain.Task {
	t.Helper()
	task, err := domain.NewTask(taskType, []byte(`{"n":1}`), time.Now(), opts...)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	stored, err := repo.Enqueue(context.Background(), task)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	return stored
}

func testEnqueueAndGet(t *testing.T, repo port.TaskRepository) {
	ctx := context.Background()
	before := time.Now().Add(-time.Second)

	stored := enqueue(t, repo, "email", domain.WithPriority(7), domain.WithMaxAttempts(3))
	if stored.ID == "" {
		t.Fatal("Enqueue returned no ID")
	}
	if stored.RunAt.Before(before) || stored.CreatedAt.Before(before) {
		t.Errorf("run at = %v, created at = %v, want both set to about now", stored.RunAt, stored.CreatedAt)
	}

	got, err := repo.Get(ctx, stored.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Type != "email" || got.Status != domain.StatusPending || got.Priority != 7 ||
		got.MaxAttempts != 3 || got.Attempts != 0 || got.WorkerID != "" || !got.HeartbeatAt.IsZero() {
		t.Errorf("Get returned %+v", got)
	}
	if !jsonEqual(t, got.Payload, []byte(`{"n":1}`)) {
		t.Errorf("payload = %s, want {\"n\":1}", got.Payload)
	}
	if !got.RunAt.Equal(stored.RunAt) {
		t.Errorf("run at = %v, want %v", got.RunAt, stored.RunAt)
	}
}

func testGetMissing(t *testing.T, repo port.TaskRepository) {
	if _, err := repo.Get(context.Background(), missingID); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrTaskNotFound", err)
	}
}

func testClaimOrderAndLimit(t *testing.T, repo port.TaskRepository) {
	ctx := context.Background()
	past := time.Now().Add(-time.Hour)

	low := enqueue(t, repo, "a", domain.WithPriority(0), domain.WithRunAt(past))
	highOld := enqueue(t, repo, "a", domain.WithPriority(5), domain.WithRunAt(past))
	highNew := enqueue(t, repo, "a", domain.WithPriority(5), domain.WithRunAt(past.Add(time.Minute)))

	claimed, err := repo.Claim(ctx, "worker-1", 2)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if len(claimed) != 2 || claimed[0].ID != highOld.ID || claimed[1].ID != highNew.ID {
		t.Fatalf("claimed %v, want [%s %s]", ids(claimed), highOld.ID, highNew.ID)
	}
	for _, c := range claimed {
		if c.Status != domain.StatusRunning || c.WorkerID != "worker-1" || c.Attempts != 1 || c.HeartbeatAt.IsZero() {
			t.Errorf("claimed task %+v not marked running for worker-1", c)
		}
	}

	rest, err := repo.Claim(ctx, "worker-2", 10)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if len(rest) != 1 || rest[0].ID != low.ID {
		t.Fatalf("second claim got %v, want [%s]", ids(rest), low.ID)
	}

	got, err := repo.Get(ctx, highOld.ID)
	if err != nil || got.Status != domain.StatusRunning {
		t.Errorf("stored status = %s, %v; want running", got.Status, err)
	}
}

func testClaimSkipsFutureTasks(t *testing.T, repo port.TaskRepository) {
	enqueue(t, repo, "later", domain.WithDelay(time.Hour))

	claimed, err := repo.Claim(context.Background(), "worker-1", 10)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if len(claimed) != 0 {
		t.Errorf("claimed %v, want nothing before run_at", ids(claimed))
	}
}

func testClaimZeroLimit(t *testing.T, repo port.TaskRepository) {
	enqueue(t, repo, "a")

	claimed, err := repo.Claim(context.Background(), "worker-1", 0)
	if err != nil || len(claimed) != 0 {
		t.Errorf("Claim(limit 0) = %v, %v; want nothing", ids(claimed), err)
	}
}

func testConcurrentClaimers(t *testing.T, repo port.TaskRepository) {
	const tasks, workers, batch = 200, 8, 7
	for range tasks {
		enqueue(t, repo, "work")
	}

	var (
		mu   sync.Mutex
		seen = make(map[string]string, tasks) // task ID -> worker that claimed it
		wg   sync.WaitGroup
	)
	for w := range workers {
		workerID := "worker-" + string(rune('A'+w))
		wg.Go(func() {
			for {
				claimed, err := repo.Claim(context.Background(), workerID, batch)
				if err != nil {
					t.Errorf("%s: Claim: %v", workerID, err)
					return
				}
				if len(claimed) == 0 {
					return
				}
				mu.Lock()
				for _, c := range claimed {
					if prev, dup := seen[c.ID]; dup {
						t.Errorf("task %s claimed by both %s and %s", c.ID, prev, workerID)
					}
					seen[c.ID] = workerID
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	if len(seen) != tasks {
		t.Errorf("claimed %d distinct tasks, want %d", len(seen), tasks)
	}
}

func testComplete(t *testing.T, repo port.TaskRepository) {
	ctx := context.Background()
	pending := enqueue(t, repo, "a", domain.WithPriority(-1))
	enqueue(t, repo, "a")

	claimed, err := repo.Claim(ctx, "worker-1", 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("Claim = %v, %v", ids(claimed), err)
	}
	id := claimed[0].ID

	if err := repo.Complete(ctx, id, "worker-2"); !errors.Is(err, domain.ErrInvalidState) {
		t.Errorf("Complete by another worker: error = %v, want ErrInvalidState", err)
	}
	if err := repo.Complete(ctx, id, "worker-1"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	got, err := repo.Get(ctx, id)
	if err != nil || got.Status != domain.StatusSucceeded {
		t.Errorf("status = %s, %v; want succeeded", got.Status, err)
	}

	if err := repo.Complete(ctx, id, "worker-1"); !errors.Is(err, domain.ErrInvalidState) {
		t.Errorf("Complete twice: error = %v, want ErrInvalidState", err)
	}
	if err := repo.Complete(ctx, pending.ID, "worker-1"); !errors.Is(err, domain.ErrInvalidState) {
		t.Errorf("Complete pending task: error = %v, want ErrInvalidState", err)
	}
	if err := repo.Complete(ctx, missingID, "worker-1"); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("Complete missing task: error = %v, want ErrTaskNotFound", err)
	}
}

// jsonEqual compares JSON by value: Postgres jsonb reformats what it stores.
func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		t.Fatalf("unmarshal %s: %v", a, err)
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
	return reflect.DeepEqual(va, vb)
}

func ids(tasks []domain.Task) []string {
	out := make([]string, len(tasks))
	for i, task := range tasks {
		out[i] = task.ID
	}
	return out
}
