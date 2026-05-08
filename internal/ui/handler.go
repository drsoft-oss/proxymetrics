package ui

import (
	"io/fs"
	"net/http"
	"strings"
)

// Handler returns an http.Handler that serves the embedded SPA.
// Non-asset GETs fall back to index.html for client-side routing.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "ui-dist")
	if err != nil {
		return notBuiltHandler{}
	}
	indexBytes, indexErr := fs.ReadFile(sub, "index.html")
	if indexErr != nil {
		return notBuiltHandler{}
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			serveIndex(w, indexBytes)
			return
		}
		if _, err := fs.Stat(sub, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA client-side route — serve index.html with 200 so the router
		// gets a chance to render.
		serveIndex(w, indexBytes)
	})
}

func serveIndex(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(body)
}

type notBuiltHandler struct{}

func (notBuiltHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w,
		"ui not built. Run `make build` or `make dev` and try again.",
		http.StatusNotFound)
}
