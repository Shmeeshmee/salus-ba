// Package migrations embeds the goose SQL migrations so every binary and test applies the same files.
package migrations

import "embed"

// FS holds the migration files at its root, in goose's NNNNN_name.sql form.
//
//go:embed *.sql
var FS embed.FS
