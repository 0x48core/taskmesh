package taskmesh_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/0x48core/taskmesh"
)

// This file is in package taskmesh_test, so it only compiles if the public API is
// usable from outside the module's internal packages.

type welcomeEmail struct {
	To string `json:"to"`
}

func Example_registerHandler() {
	registry := taskmesh.NewRegistry()

	err := registry.Register("email.welcome", taskmesh.HandlerFunc(func(ctx context.Context, task taskmesh.Task) error {
		var p welcomeEmail
		if err := json.Unmarshal(task.Payload, &p); err != nil {
			// A payload that can't be decoded will never succeed: don't retry it.
			return fmt.Errorf("decode payload: %w", taskmesh.ErrSkipRetry)
		}
		fmt.Println("sending welcome email to", p.To)
		return nil
	}))
	if err != nil {
		panic(err)
	}

	h, _ := registry.Handler("email.welcome")
	_ = h.Handle(context.Background(), taskmesh.Task{Payload: []byte(`{"to":"ada@example.com"}`)})

	err = h.Handle(context.Background(), taskmesh.Task{Payload: []byte(`not json`)})
	fmt.Println("skip retry:", errors.Is(err, taskmesh.ErrSkipRetry))

	err = registry.Register("email.welcome", h)
	fmt.Println("already registered:", errors.Is(err, taskmesh.ErrHandlerRegistered))
	// Output:
	// sending welcome email to ada@example.com
	// skip retry: true
	// already registered: true
}
