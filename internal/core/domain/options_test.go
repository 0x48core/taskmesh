package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewEnqueueOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		want    EnqueueOptions
		wantErr bool
	}{
		{name: "defaults", want: EnqueueOptions{MaxAttempts: DefaultMaxAttempts}},
		{
			name: "all set",
			opts: []Option{WithPriority(-3), WithDelay(time.Second), WithMaxAttempts(1), WithIdempotencyKey("k")},
			want: EnqueueOptions{Priority: -3, Delay: time.Second, MaxAttempts: 1, IdempotencyKey: "k"},
		},
		{name: "run at and delay together", opts: []Option{WithRunAt(time.Now()), WithDelay(time.Second)}, wantErr: true},
		{name: "negative delay", opts: []Option{WithDelay(-time.Second)}, wantErr: true},
		{name: "zero max attempts", opts: []Option{WithMaxAttempts(0)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEnqueueOptions(tt.opts...)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidTask) {
					t.Fatalf("error = %v, want ErrInvalidTask", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewEnqueueOptions: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
