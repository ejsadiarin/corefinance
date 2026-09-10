package migrations

import "embed"

// FS holds the goose migration files so both cmd/migrate and goose-based
// tooling share one schema source of truth.
//
//go:embed *.sql
var FS embed.FS
