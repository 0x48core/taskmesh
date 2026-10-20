// Config: poll interval, concurrency, timeouts, thresholds, with defaults.
package service

import (
	"errors"
	"fmt"
	"time"
)

// Defaults applied by Config.WithDefaults to fields left at zero.
const (
	DefaultConcurrency  = 10
	DefaultPollInterval = time.Second
)

// Config tunes the worker. Zero fields take the defaults above.
// Phase 2 adds heartbeat, stale-after, reap interval and timeout settings here.
type Config struct {
	// Concurrency is the most tasks one worker runs at the same time.
	Concurrency int `yaml:"concurrency"`
	// PollInterval is how long the worker sleeps when a claim finds no due tasks.
	PollInterval time.Duration `yaml:"poll_interval"`
}

// WithDefaults returns a copy of c with every zero field set to its default.
func (c Config) WithDefaults() Config {
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConcurrency
	}
	if c.PollInterval == 0 {
		c.PollInterval = DefaultPollInterval
	}
	return c
}

// Validate reports settings that can never work. Call it after WithDefaults.
func (c Config) Validate() error {
	var errs []error
	if c.Concurrency < 1 {
		errs = append(errs, fmt.Errorf("concurrency %d is less than 1", c.Concurrency))
	}
	if c.PollInterval <= 0 {
		errs = append(errs, fmt.Errorf("poll interval %v is not positive", c.PollInterval))
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("service config: %w", err)
	}
	return nil
}
