package service

import (
	"testing"
	"time"
)

func TestConfig_WithDefaults(t *testing.T) {
	got := Config{}.WithDefaults()
	if got.Concurrency != DefaultConcurrency || got.PollInterval != DefaultPollInterval {
		t.Errorf("WithDefaults() = %+v, want defaults", got)
	}

	set := Config{Concurrency: 3, PollInterval: 50 * time.Millisecond}
	if got := set.WithDefaults(); got != set {
		t.Errorf("WithDefaults() = %+v, want explicit values kept %+v", got, set)
	}
}

func TestConfig_Validate(t *testing.T) {
	if err := (Config{}).WithDefaults().Validate(); err != nil {
		t.Errorf("defaults should be valid: %v", err)
	}
	bad := Config{Concurrency: -1, PollInterval: -time.Second}.WithDefaults()
	if err := bad.Validate(); err == nil {
		t.Error("want error for negative concurrency and poll interval")
	}
}
