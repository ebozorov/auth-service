// Package web holds the built admin UI (web/dist), embedded into the binary.
// Build it first with `make web-build`.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the contents of dist/ (index.html, assets/, ...).
func FS() fs.FS {
	sub, _ := fs.Sub(dist, "dist")
	return sub
}
