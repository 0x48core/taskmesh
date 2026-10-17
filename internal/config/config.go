// Package config is the service configuration mapped from conf/*.yaml.
package config

import (
	"errors"
	"fmt"

	"github.com/0x48core/taskmesh/pkg/db"
	"github.com/0x48core/taskmesh/pkg/gconfig"
)

// Config is the whole service configuration. HTTP, Worker and Reaper sections join in phase 3.
type Config struct {
	Database db.Config `yaml:"database"`
}

// Load reads conf files from dir for the given environment and validates the result.
func Load(dir, environment string) (Config, error) {
	var cfg Config
	if err := gconfig.Load(dir, environment, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.Database.URL == "" {
		return errors.New("database.url is empty (set it in conf/ or via DATABASE_URL)")
	}
	return nil
}
