// Package migrations exposes the embedded SQL migration files so that
// cmd/migrate.go can run goose against them without needing the migrations
// directory to be present at runtime.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
