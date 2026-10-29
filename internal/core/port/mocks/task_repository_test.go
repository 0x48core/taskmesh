package mocks

import (
	"testing"

	"github.com/0x48core/taskmesh/internal/core/port"
	"github.com/0x48core/taskmesh/internal/core/port/porttest"
)

func TestTaskRepository(t *testing.T) {
	porttest.TaskRepository(t, func(t *testing.T) port.TaskRepository {
		return NewTaskRepository()
	})
}
