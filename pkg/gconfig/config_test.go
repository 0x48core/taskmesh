package gconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testConfig struct {
	Database struct {
		URL            string        `yaml:"url"`
		MaxConns       int32         `yaml:"max_conns"`
		ConnectTimeout time.Duration `yaml:"connect_timeout"`
	} `yaml:"database"`
	Tags []string `yaml:"tags"`
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_MergesEnvironmentOverBase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", `
database:
  url: postgres://base
  max_conns: 10
  connect_timeout: 5s
tags: [a, b]
`)
	writeFile(t, dir, "application-prod.yaml", `
database:
  max_conns: 50
tags: [c]
`)

	var cfg testConfig
	if err := Load(dir, "prod", &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Database.MaxConns != 50 {
		t.Errorf("max_conns = %d, want 50 (overridden)", cfg.Database.MaxConns)
	}
	if cfg.Database.URL != "postgres://base" {
		t.Errorf("url = %q, want base value kept", cfg.Database.URL)
	}
	if cfg.Database.ConnectTimeout != 5*time.Second {
		t.Errorf("connect_timeout = %v, want 5s", cfg.Database.ConnectTimeout)
	}
	if len(cfg.Tags) != 1 || cfg.Tags[0] != "c" {
		t.Errorf("tags = %v, want [c] (lists are replaced, not merged)", cfg.Tags)
	}
}

func TestLoad_MissingOrEmptyEnvironmentFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "database:\n  max_conns: 10\n")
	writeFile(t, dir, "application-dev.yaml", "# no overrides\n")

	for _, environment := range []string{"dev", "stg"} {
		var cfg testConfig
		if err := Load(dir, environment, &cfg); err != nil {
			t.Fatalf("Load(%q): %v", environment, err)
		}
		if cfg.Database.MaxConns != 10 {
			t.Errorf("Load(%q): max_conns = %d, want 10", environment, cfg.Database.MaxConns)
		}
	}
}

func TestLoad_ExpandsEnvironmentVariables(t *testing.T) {
	t.Setenv("TEST_DATABASE_URL", "postgres://from-env")
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "database:\n  url: ${TEST_DATABASE_URL}\n")

	var cfg testConfig
	if err := Load(dir, "dev", &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Database.URL != "postgres://from-env" {
		t.Errorf("url = %q, want value from environment", cfg.Database.URL)
	}
}

func TestLoad_Errors(t *testing.T) {
	t.Run("missing base file", func(t *testing.T) {
		var cfg testConfig
		if err := Load(t.TempDir(), "dev", &cfg); err == nil {
			t.Fatal("want error for missing config.yaml")
		}
	})
	t.Run("unknown key", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "config.yaml", "database:\n  max_con: 10\n")
		var cfg testConfig
		if err := Load(dir, "dev", &cfg); err == nil {
			t.Fatal("want error for misspelled key")
		}
	})
}
