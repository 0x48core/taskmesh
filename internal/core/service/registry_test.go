package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/0x48core/taskmesh/internal/core/domain"
	"github.com/0x48core/taskmesh/internal/core/port"
)

func noop(context.Context, domain.Task) error { return nil }

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistry()
	called := false
	h := port.TaskHandlerFunc(func(context.Context, domain.Task) error {
		called = true
		return nil
	})

	if err := r.Register("email", h); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := r.Handler("email")
	if !ok {
		t.Fatal("Handler(email) not found")
	}
	if err := got.Handle(context.Background(), domain.Task{}); err != nil || !called {
		t.Errorf("Handle() = %v, called = %v; want the registered handler", err, called)
	}

	if _, ok := r.Handler("sms"); ok {
		t.Error("Handler(sms) found, want not registered")
	}
}

func TestRegistry_RegisterErrors(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("email", port.TaskHandlerFunc(noop)); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		taskType string
		handler  port.TaskHandler
		wantIs   error
	}{
		{name: "duplicate type", taskType: "email", handler: port.TaskHandlerFunc(noop), wantIs: ErrHandlerRegistered},
		{name: "empty type", taskType: "", handler: port.TaskHandlerFunc(noop)},
		{name: "nil handler", taskType: "sms", handler: nil},
		{name: "nil handler func", taskType: "sms", handler: port.TaskHandlerFunc(nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Register(tt.taskType, tt.handler)
			if err == nil {
				t.Fatal("Register() succeeded, want error")
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("error = %v, want %v", err, tt.wantIs)
			}
		})
	}

	if got := r.Types(); !slices.Equal(got, []string{"email"}) {
		t.Errorf("Types() = %v, want only [email] after failed registrations", got)
	}
}

func TestRegistry_TypesSorted(t *testing.T) {
	r := NewRegistry()
	for _, typ := range []string{"sms", "email", "push"} {
		if err := r.Register(typ, port.TaskHandlerFunc(noop)); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := r.Types(), []string{"email", "push", "sms"}; !slices.Equal(got, want) {
		t.Errorf("Types() = %v, want %v", got, want)
	}
}

func TestRegistry_ConcurrentUse(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := range 50 {
		typ := fmt.Sprintf("type-%d", i)
		wg.Go(func() {
			if err := r.Register(typ, port.TaskHandlerFunc(noop)); err != nil {
				t.Errorf("Register(%s): %v", typ, err)
			}
		})
		wg.Go(func() {
			r.Handler(typ)
			r.Types()
		})
	}
	wg.Wait()

	if n := len(r.Types()); n != 50 {
		t.Errorf("registered %d types, want 50", n)
	}
}
