package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var frontendFS embed.FS

func frontendAssets() http.FileSystem {
	sub, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		// Return empty filesystem if dist doesn't exist properly
		return http.FS(frontendFS)
	}
	return http.FS(sub)
}
