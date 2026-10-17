// Package migrations embeds the SQL migrations for tests and tooling.
package migrations

import "embed"

// FS holds every *.sql file in this directory.
//
//go:embed *.sql
var FS embed.FS
