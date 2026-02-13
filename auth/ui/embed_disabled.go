//go:build !embed_ui
// +build !embed_ui

package ui

import (
	"net/http"
)

// FS returns nil when UI is not embedded
func FS() http.FileSystem {
	return nil
}

// Handler returns nil when UI is not embedded
func Handler() http.Handler {
	return nil
}

// Enabled returns false when UI is not embedded
func Enabled() bool {
	return false
}
