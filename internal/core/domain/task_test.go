// State transition tests.
package domain

import (
	"errors"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

func TestNewTask(t *testing.T) {
	runAt := now.Add(time.Hour)

	tests := []struct {
		name        string
		taskType    string
		payload     []byte
		opts        []Option
		wantErr     error
		wantPayload string
		wantRunAt   time.Time
		wantMax     int
	}{
		{name: "defaults", taskType: "email", payload: []byte(`{"to":"a@b.c"}`),
			wantPayload: `{"to":"a@b.c"}`, wantMax: DefaultMaxAttempts},
		{name: "empty payload becomes {}", taskType: "email",
			wantPayload: `{}`, wantMax: DefaultMaxAttempts},
		{name: "delay is turned into run at", taskType: "email", opts: []Option{WithDelay(time.Minute)},
			wantPayload: `{}`, wantRunAt: now.Add(time.Minute), wantMax: DefaultMaxAttempts},
		{name: "run at is kept", taskType: "email", opts: []Option{WithRunAt(runAt), WithMaxAttempts(3)},
			wantPayload: `{}`, wantRunAt: runAt, wantMax: 3},
		{name: "empty type", taskType: "", wantErr: ErrInvalidTask},
		{name: "invalid JSON", taskType: "email", payload: []byte(`{`), wantErr: ErrInvalidTask},
		{name: "invalid option", taskType: "email", opts: []Option{WithMaxAttempts(0)}, wantErr: ErrInvalidTask},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewTask(tt.taskType, tt.payload, now, tt.opts...)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewTask() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if task.Status != StatusPending {
				t.Errorf("status = %s, want pending", task.Status)
			}
			if string(task.Payload) != tt.wantPayload {
				t.Errorf("payload = %s, want %s", task.Payload, tt.wantPayload)
			}
			if !task.RunAt.Equal(tt.wantRunAt) {
				t.Errorf("run at = %v, want %v", task.RunAt, tt.wantRunAt)
			}
			if task.MaxAttempts != tt.wantMax {
				t.Errorf("max attempts = %d, want %d", task.MaxAttempts, tt.wantMax)
			}
		})
	}
}

func TestNewTask_CopiesPayload(t *testing.T) {
	payload := []byte(`{"n":1}`)
	task, err := NewTask("count", payload, now)
	if err != nil {
		t.Fatal(err)
	}
	payload[5] = '2'
	if string(task.Payload) != `{"n":1}` {
		t.Errorf("payload changed with caller's slice: %s", task.Payload)
	}
}

func TestCanTransition(t *testing.T) {
	all := []Status{StatusPending, StatusRunning, StatusSucceeded, StatusDead, StatusCancelled}
	allowed := map[[2]Status]bool{
		{StatusPending, StatusRunning}:   true,
		{StatusPending, StatusCancelled}: true,
		{StatusRunning, StatusSucceeded}: true,
		{StatusRunning, StatusPending}:   true,
		{StatusRunning, StatusDead}:      true,
		{StatusDead, StatusPending}:      true,
		{StatusCancelled, StatusPending}: true,
	}
	// Check every pair, so a new transition can't slip in without updating this test.
	for _, from := range all {
		for _, to := range all {
			want := allowed[[2]Status{from, to}]
			if got := CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestTask_ClaimAndComplete(t *testing.T) {
	task, err := NewTask("email", nil, now)
	if err != nil {
		t.Fatal(err)
	}

	later := now.Add(time.Second)
	if err := task.Claim("worker-1", later); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if task.Status != StatusRunning || task.WorkerID != "worker-1" || task.Attempts != 1 {
		t.Fatalf("after claim: status=%s worker=%q attempts=%d", task.Status, task.WorkerID, task.Attempts)
	}
	if !task.HeartbeatAt.Equal(later) || !task.UpdatedAt.Equal(later) {
		t.Errorf("after claim: heartbeat=%v updated=%v, want %v", task.HeartbeatAt, task.UpdatedAt, later)
	}

	if err := task.Complete("worker-2", later); !errors.Is(err, ErrInvalidState) {
		t.Errorf("Complete by another worker: error = %v, want ErrInvalidState", err)
	}
	if err := task.Complete("worker-1", later); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if task.Status != StatusSucceeded {
		t.Errorf("status = %s, want succeeded", task.Status)
	}

	if err := task.Claim("worker-1", later); !errors.Is(err, ErrInvalidState) {
		t.Errorf("Claim succeeded task: error = %v, want ErrInvalidState", err)
	}
}

func TestTask_ClaimRejectsEmptyWorker(t *testing.T) {
	task, _ := NewTask("email", nil, now)
	if err := task.Claim("", now); !errors.Is(err, ErrInvalidTask) {
		t.Errorf("error = %v, want ErrInvalidTask", err)
	}
	if task.Status != StatusPending {
		t.Errorf("status = %s, want unchanged pending", task.Status)
	}
}

func TestTask_Due(t *testing.T) {
	tests := []struct {
		name string
		task Task
		want bool
	}{
		{"no run at", Task{Status: StatusPending}, true},
		{"run at in the past", Task{Status: StatusPending, RunAt: now.Add(-time.Second)}, true},
		{"run at exactly now", Task{Status: StatusPending, RunAt: now}, true},
		{"run at in the future", Task{Status: StatusPending, RunAt: now.Add(time.Second)}, false},
		{"not pending", Task{Status: StatusRunning}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.Due(now); got != tt.want {
				t.Errorf("Due() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	if _, err := ParseStatus("running"); err != nil {
		t.Errorf("ParseStatus(running): %v", err)
	}
	if _, err := ParseStatus("failed"); !errors.Is(err, ErrInvalidTask) {
		t.Errorf("ParseStatus(failed): error = %v, want ErrInvalidTask", err)
	}
	for _, s := range []Status{StatusSucceeded, StatusDead, StatusCancelled} {
		if !s.Terminal() {
			t.Errorf("%s.Terminal() = false, want true", s)
		}
	}
	for _, s := range []Status{StatusPending, StatusRunning} {
		if s.Terminal() {
			t.Errorf("%s.Terminal() = true, want false", s)
		}
	}
}
