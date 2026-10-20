// Enqueue options: priority, run at, delay, max attempts, idempotency key.
package domain

import (
	"fmt"
	"time"
)

// DefaultMaxAttempts is used when WithMaxAttempts is not given. Matches the column default.
const DefaultMaxAttempts = 5

// EnqueueOptions holds the settings an Enqueue call can change. Build it with Option funcs.
type EnqueueOptions struct {
	Priority       int
	RunAt          time.Time
	Delay          time.Duration
	MaxAttempts    int
	IdempotencyKey string
}

// Option changes one enqueue setting.
type Option func(*EnqueueOptions)

// WithPriority sets the priority. Higher runs first; the default is 0 and negatives are allowed.
func WithPriority(p int) Option {
	return func(o *EnqueueOptions) { o.Priority = p }
}

// WithRunAt schedules the task for a point in time. Cannot be combined with WithDelay.
func WithRunAt(t time.Time) Option {
	return func(o *EnqueueOptions) { o.RunAt = t }
}

// WithDelay schedules the task to run d after it is enqueued. Cannot be combined with WithRunAt.
func WithDelay(d time.Duration) Option {
	return func(o *EnqueueOptions) { o.Delay = d }
}

// WithMaxAttempts sets how many times the task may run before it goes dead. Must be at least 1.
func WithMaxAttempts(n int) Option {
	return func(o *EnqueueOptions) { o.MaxAttempts = n }
}

// WithIdempotencyKey makes Enqueue return the existing task when one with the same type
// and key already exists, instead of creating a duplicate.
func WithIdempotencyKey(key string) Option {
	return func(o *EnqueueOptions) { o.IdempotencyKey = key }
}

// NewEnqueueOptions applies opts over the defaults and validates the result.
func NewEnqueueOptions(opts ...Option) (EnqueueOptions, error) {
	o := EnqueueOptions{MaxAttempts: DefaultMaxAttempts}
	for _, opt := range opts {
		opt(&o)
	}
	if err := o.validate(); err != nil {
		return EnqueueOptions{}, err
	}
	return o, nil
}

func (o EnqueueOptions) validate() error {
	switch {
	case !o.RunAt.IsZero() && o.Delay != 0:
		return fmt.Errorf("%w: WithRunAt and WithDelay cannot be used together", ErrInvalidTask)
	case o.Delay < 0:
		return fmt.Errorf("%w: delay %v is negative", ErrInvalidTask, o.Delay)
	case o.MaxAttempts < 1:
		return fmt.Errorf("%w: max attempts %d is less than 1", ErrInvalidTask, o.MaxAttempts)
	}
	return nil
}
