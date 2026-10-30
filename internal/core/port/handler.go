// TaskHandler: the port user code implements to process a task type.
package port

import (
	"context"

	"github.com/0x48core/taskmesh/internal/core/domain"
)

// TaskHandler processes tasks of one type. The worker calls Handle once per attempt.
//
// Return nil when the work is done. Any other error schedules a retry, unless it wraps
// domain.ErrSkipRetry, which sends the task straight to dead. ctx is cancelled when the
// worker shuts down or loses the task, so long-running handlers should watch it.
//
// Delivery is at-least-once: the same task can be handled more than once, so handlers
// must be idempotent.
type TaskHandler interface {
	Handle(ctx context.Context, task domain.Task) error
}

// TaskHandlerFunc lets an ordinary function be used as a TaskHandler.
type TaskHandlerFunc func(ctx context.Context, task domain.Task) error

// Handle calls f(ctx, task).
func (f TaskHandlerFunc) Handle(ctx context.Context, task domain.Task) error {
	return f(ctx, task)
}
