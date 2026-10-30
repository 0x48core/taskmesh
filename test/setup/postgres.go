// Package setup starts a throwaway Postgres for adapter and e2e tests.
package setup

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/0x48core/taskmesh/migrations"
	"github.com/0x48core/taskmesh/pkg/db"
)

// Image matches docker-compose and CI.
const Image = "postgres:18-alpine"

// Postgres is a running container with the migrations applied.
type Postgres struct {
	URL  string
	Pool *pgxpool.Pool

	container *tcpostgres.PostgresContainer
}

// StartPostgres starts a container, applies every *.up.sql migration and opens a pool.
// Call Terminate when done, usually from TestMain.
func StartPostgres(ctx context.Context) (*Postgres, error) {
	container, err := tcpostgres.Run(ctx, Image,
		tcpostgres.WithDatabase("taskmesh_test"),
		tcpostgres.WithUsername("taskmesh"),
		tcpostgres.WithPassword("taskmesh"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("setup: start postgres: %w", err)
	}
	pg := &Postgres{container: container}

	pg.URL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		pg.Terminate()
		return nil, fmt.Errorf("setup: connection string: %w", err)
	}
	if err := migrate(ctx, pg.URL); err != nil {
		pg.Terminate()
		return nil, err
	}
	pg.Pool, err = db.NewPool(ctx, db.Config{URL: pg.URL})
	if err != nil {
		pg.Terminate()
		return nil, err
	}
	return pg, nil
}

// Terminate closes the pool and removes the container.
func (pg *Postgres) Terminate() {
	if pg.Pool != nil {
		pg.Pool.Close()
	}
	_ = testcontainers.TerminateContainer(pg.container)
}

// Reset empties every table, so each test starts from a clean database.
func (pg *Postgres) Reset(t *testing.T) {
	t.Helper()
	if _, err := pg.Pool.Exec(context.Background(), "TRUNCATE tasks"); err != nil {
		t.Fatalf("setup: reset: %v", err)
	}
}

// migrate runs the embedded up migrations in file-name order. It uses the simple query
// protocol, which allows several statements per file.
func migrate(ctx context.Context, url string) error {
	names, err := fs.Glob(migrations.FS, "*.up.sql")
	if err != nil {
		return fmt.Errorf("setup: list migrations: %w", err)
	}
	slices.Sort(names)

	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("setup: connect for migrations: %w", err)
	}
	defer conn.Close(ctx)

	for _, name := range names {
		sql, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return fmt.Errorf("setup: read %s: %w", name, err)
		}
		if _, err := conn.PgConn().Exec(ctx, string(sql)).ReadAll(); err != nil {
			return fmt.Errorf("setup: apply %s: %w", strings.TrimSuffix(name, ".up.sql"), err)
		}
	}
	return nil
}
