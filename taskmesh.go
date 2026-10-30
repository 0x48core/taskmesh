// Package taskmesh is a distributed task queue for Go, built on PostgreSQL.
//
// This package is the public API. Its types are aliases for the internal core types,
// so code outside this module can name them, implement Handler and register handlers.
package taskmesh

import (
	"time"

	"github.com/0x48core/taskmesh/internal/core/domain"
	"github.com/0x48core/taskmesh/internal/core/port"
	"github.com/0x48core/taskmesh/internal/core/service"
)

type (
	// Task is one unit of work, as stored in the tasks table.
	Task = domain.Task
	// Status is where a task is in its lifecycle.
	Status = domain.Status
	// Option changes one enqueue setting; see WithPriority and friends.
	Option = domain.Option

	// Handler processes tasks of one type. Return nil on success, an error to retry,
	// or an error wrapping ErrSkipRetry to give up. Handlers must be idempotent.
	Handler = port.TaskHandler
	// HandlerFunc lets an ordinary function be used as a Handler.
	HandlerFunc = port.TaskHandlerFunc
	// Registry maps task types to handlers. Create one with NewRegistry.
	Registry = service.Registry
)

const (
	StatusPending   = domain.StatusPending
	StatusRunning   = domain.StatusRunning
	StatusSucceeded = domain.StatusSucceeded
	StatusDead      = domain.StatusDead
	StatusCancelled = domain.StatusCancelled

	// DefaultMaxAttempts is used when WithMaxAttempts is not given.
	DefaultMaxAttempts = domain.DefaultMaxAttempts
)

// Errors returned by this package. Match them with errors.Is.
var (
	ErrTaskNotFound      = domain.ErrTaskNotFound
	ErrInvalidState      = domain.ErrInvalidState
	ErrInvalidTask       = domain.ErrInvalidTask
	ErrSkipRetry         = domain.ErrSkipRetry
	ErrHandlerRegistered = service.ErrHandlerRegistered
)

// NewRegistry returns an empty handler registry.
func NewRegistry() *Registry { return service.NewRegistry() }

// WithPriority sets the priority. Higher runs first; the default is 0 and negatives are allowed.
func WithPriority(p int) Option { return domain.WithPriority(p) }

// WithRunAt schedules the task for a point in time. Cannot be combined with WithDelay.
func WithRunAt(t time.Time) Option { return domain.WithRunAt(t) }

// WithDelay schedules the task to run d after it is enqueued. Cannot be combined with WithRunAt.
func WithDelay(d time.Duration) Option { return domain.WithDelay(d) }

// WithMaxAttempts sets how many times the task may run before it goes dead. Must be at least 1.
func WithMaxAttempts(n int) Option { return domain.WithMaxAttempts(n) }

// WithIdempotencyKey makes Enqueue return the existing task when one with the same type
// and key already exists, instead of creating a duplicate.
func WithIdempotencyKey(key string) Option { return domain.WithIdempotencyKey(key) }
