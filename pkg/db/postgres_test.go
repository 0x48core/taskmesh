package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewPool_InvalidURL(t *testing.T) {
	_, err := NewPool(context.Background(), Config{URL: "not a url://"})
	if err == nil {
		t.Fatal("want error for invalid url")
	}
}

func TestNewPool_UnreachableServer(t *testing.T) {
	// Port 1 is never a Postgres server; the ping must fail within the timeout.
	cfg := Config{
		URL:            "postgres://u:p@127.0.0.1:1/db?sslmode=disable",
		ConnectTimeout: 500 * time.Millisecond,
	}
	if _, err := NewPool(context.Background(), cfg); err == nil {
		t.Fatal("want error for unreachable server")
	}
}

// TestNewPool_Live runs only when TEST_DATABASE_URL is set, e.g. against `make up`.
func TestNewPool_Live(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := NewPool(context.Background(), Config{URL: url})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	var one int
	if err := pool.QueryRow(context.Background(), "SELECT 1").Scan(&one); err != nil || one != 1 {
		t.Fatalf("SELECT 1 = %d, %v", one, err)
	}
}
