// Registry: task type -> TaskHandler.
package service

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/0x48core/taskmesh/internal/core/port"
)

// ErrHandlerRegistered is returned when a task type already has a handler.
var ErrHandlerRegistered = errors.New("taskmesh: handler already registered")

// Registry maps task types to their handlers. Safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]port.TaskHandler
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]port.TaskHandler)}
}

// Register sets the handler for taskType. Each type can have only one handler;
// registering it again returns ErrHandlerRegistered.
func (r *Registry) Register(taskType string, h port.TaskHandler) error {
	if taskType == "" {
		return errors.New("taskmesh: register handler: task type is empty")
	}
	if isNil(h) {
		return fmt.Errorf("taskmesh: register handler for %q: handler is nil", taskType)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.handlers[taskType]; ok {
		return fmt.Errorf("%w: %q", ErrHandlerRegistered, taskType)
	}
	r.handlers[taskType] = h
	return nil
}

// Handler returns the handler for taskType, if one is registered.
func (r *Registry) Handler(taskType string) (port.TaskHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[taskType]
	return h, ok
}

// Types returns the registered task types in sorted order.
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.handlers))
}

// isNil catches both a nil interface and a nil TaskHandlerFunc wrapped in one.
func isNil(h port.TaskHandler) bool {
	if h == nil {
		return true
	}
	f, ok := h.(port.TaskHandlerFunc)
	return ok && f == nil
}
