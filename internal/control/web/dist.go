package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:embedded
var embeddedDist embed.FS

// DistFileSystem returns an http.FileSystem serving the compiled web assets.
func DistFileSystem() (http.FileSystem, error) {
	sub, err := fs.Sub(embeddedDist, "embedded")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// EmbeddedDistFS returns the io/fs.FS rooted at the compiled web dist directory.
func EmbeddedDistFS() (fs.FS, error) {
	return fs.Sub(embeddedDist, "embedded")
}
