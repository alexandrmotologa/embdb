package server

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distEmbedFS embed.FS

// GetEmbeddedFS returns the filesystem pointing to the built React 19 studio assets.
func GetEmbeddedFS() (fs.FS, error) {
	return fs.Sub(distEmbedFS, "dist")
}
