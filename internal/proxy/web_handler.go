package proxy

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed web/*
var webAssets embed.FS

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	subFS, err := fs.Sub(webAssets, "web")
	if err != nil {
		http.Error(w, "Internal asset error", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Only fallback to index.html for non-API routes if the file doesn't exist (client-side routing)
	if !strings.HasPrefix(path, "v1/") && !strings.HasPrefix(path, "ocgt/") {
		_, err = subFS.Open(path)
		if err != nil {
			r.URL.Path = "/"
		}
	}

	http.FileServer(http.FS(subFS)).ServeHTTP(w, r)
}
