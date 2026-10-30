// Transaction support for EnqueueTx (pgx.Tx).
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is what the repository needs to run queries. *pgxpool.Pool, *pgx.Conn and pgx.Tx
// all satisfy it, so the same repository code runs inside or outside a caller's transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
