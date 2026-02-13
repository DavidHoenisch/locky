//go:build embed_ui
// +build embed_ui

package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var uiFS embed.FS

// FS returns the embedded UI filesystem
func FS() http.FileSystem {
	fsys, err := fs.Sub(uiFS, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

// Handler returns an HTTP handler for serving the UI
func Handler() http.Handler {
	return http.FileServer(FS())
}
