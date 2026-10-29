// Test setup: starts Postgres via testcontainers-go and applies migrations.
package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/0x48core/taskmesh/test/setup"
)

// pg is shared by every test in the package. Tests must not use t.Parallel,
// because each one calls pg.Reset.
var pg *setup.Postgres

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	var err error
	pg, err = setup.StartPostgres(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer pg.Terminate()
	return m.Run()
}
