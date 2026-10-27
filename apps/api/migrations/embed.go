// Package migrations embeds the versioned SQL migration files and Atlas checksum file.
package migrations

import "embed"

// FS holds the embedded SQL migration files and atlas.sum checksum file.
//
//go:embed *.sql atlas.sum
var FS embed.FS

