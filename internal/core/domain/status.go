// Status values: pending, running, succeeded, dead, cancelled.
package domain

import "fmt"

// Status is where a task is in its lifecycle. The values match the tasks.status column.
type Status string

const (
	// StatusPending: waiting to be claimed once run_at has passed.
	StatusPending Status = "pending"
	// StatusRunning: claimed by a worker and executing.
	StatusRunning Status = "running"
	// StatusSucceeded: the handler returned nil. Terminal.
	StatusSucceeded Status = "succeeded"
	// StatusDead: out of attempts, or the handler returned ErrSkipRetry. Terminal until retried by hand.
	StatusDead Status = "dead"
	// StatusCancelled: cancelled before it ran. Terminal until retried by hand.
	StatusCancelled Status = "cancelled"
)

// ParseStatus converts a stored or user-supplied value into a Status.
func ParseStatus(s string) (Status, error) {
	st := Status(s)
	if !st.Valid() {
		return "", fmt.Errorf("%w: unknown status %q", ErrInvalidTask, s)
	}
	return st, nil
}

// Valid reports whether s is one of the defined statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusSucceeded, StatusDead, StatusCancelled:
		return true
	}
	return false
}

// Terminal reports whether no worker will touch the task again on its own.
func (s Status) Terminal() bool {
	return s == StatusSucceeded || s == StatusDead || s == StatusCancelled
}

func (s Status) String() string { return string(s) }

// transitions lists every allowed status change. Anything not listed is ErrInvalidState.
var transitions = map[Status][]Status{
	StatusPending:   {StatusRunning, StatusCancelled},
	StatusRunning:   {StatusSucceeded, StatusPending, StatusDead},
	StatusDead:      {StatusPending},
	StatusCancelled: {StatusPending},
}

// CanTransition reports whether a task may move from one status to another.
//
//   - pending   -> running (claim), cancelled (cancel)
//   - running   -> succeeded (complete), pending (retry later or reaped), dead (out of attempts)
//   - dead      -> pending (manual retry)
//   - cancelled -> pending (manual retry)
func CanTransition(from, to Status) bool {
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
