// Task entity and its state transitions.
package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Task is one unit of work. It maps one-to-one to a row in the tasks table.
//
// Optional columns use zero values: an empty string or a zero time means NULL.
type Task struct {
	ID             string
	Type           string
	Payload        json.RawMessage
	Status         Status
	Priority       int
	Attempts       int
	MaxAttempts    int
	RunAt          time.Time // zero means "as soon as possible"; the repository uses database time
	IdempotencyKey string
	LastError      string
	WorkerID       string
	HeartbeatAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewTask builds a pending task ready to be stored. It has no ID yet; the repository assigns one.
//
// now is only used to turn WithDelay into a RunAt. An empty payload becomes {}.
func NewTask(taskType string, payload []byte, now time.Time, opts ...Option) (Task, error) {
	if taskType == "" {
		return Task{}, fmt.Errorf("%w: task type is empty", ErrInvalidTask)
	}
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	if !json.Valid(payload) {
		return Task{}, fmt.Errorf("%w: payload is not valid JSON", ErrInvalidTask)
	}

	o, err := NewEnqueueOptions(opts...)
	if err != nil {
		return Task{}, err
	}

	runAt := o.RunAt
	if o.Delay > 0 {
		runAt = now.Add(o.Delay)
	}

	return Task{
		Type:           taskType,
		Payload:        append(json.RawMessage(nil), payload...), // copy: the caller may reuse its slice
		Status:         StatusPending,
		Priority:       o.Priority,
		MaxAttempts:    o.MaxAttempts,
		RunAt:          runAt,
		IdempotencyKey: o.IdempotencyKey,
	}, nil
}

// Due reports whether a pending task may be claimed at now.
func (t Task) Due(now time.Time) bool {
	return t.Status == StatusPending && !t.RunAt.After(now)
}

// Claim moves a pending task to running for workerID and counts the attempt.
func (t *Task) Claim(workerID string, now time.Time) error {
	if workerID == "" {
		return fmt.Errorf("%w: worker id is empty", ErrInvalidTask)
	}
	if err := t.transition(StatusRunning, now); err != nil {
		return err
	}
	t.WorkerID = workerID
	t.HeartbeatAt = now
	t.Attempts++
	return nil
}

// Complete marks a running task as succeeded. Only the worker that owns the task may complete it.
func (t *Task) Complete(workerID string, now time.Time) error {
	if err := t.checkOwner(workerID); err != nil {
		return err
	}
	return t.transition(StatusSucceeded, now)
}

func (t *Task) checkOwner(workerID string) error {
	if t.Status == StatusRunning && t.WorkerID != workerID {
		return fmt.Errorf("%w: task %s is owned by worker %q, not %q", ErrInvalidState, t.ID, t.WorkerID, workerID)
	}
	return nil
}

func (t *Task) transition(to Status, now time.Time) error {
	if !CanTransition(t.Status, to) {
		return fmt.Errorf("%w: task %s cannot move from %s to %s", ErrInvalidState, t.ID, t.Status, to)
	}
	t.Status = to
	t.UpdatedAt = now
	return nil
}
