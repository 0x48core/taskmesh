// Domain errors: ErrSkipRetry, ErrTaskNotFound, ErrInvalidState, ErrInvalidTask.
package domain

import "errors"

// Callers match these with errors.Is; the core wraps them with details using %w.
var (
	// ErrTaskNotFound: no task has the given ID.
	ErrTaskNotFound = errors.New("taskmesh: task not found")

	// ErrInvalidState: the task's current status does not allow the requested change,
	// e.g. cancelling a task that already succeeded.
	ErrInvalidState = errors.New("taskmesh: invalid task state")

	// ErrInvalidTask: the task or its enqueue options failed validation.
	ErrInvalidTask = errors.New("taskmesh: invalid task")

	// ErrSkipRetry: returned (or wrapped) by a handler to send the task straight to dead
	// instead of retrying, e.g. fmt.Errorf("bad payload: %w", taskmesh.ErrSkipRetry).
	ErrSkipRetry = errors.New("taskmesh: skip retry")
)
