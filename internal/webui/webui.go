// Package webui serves the built login SPA from the same process as the API.
// 2026-10-12 Windows 本机演示：/ 为登录页，/api/v1 与 /health 仍走 API。
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:assets
var assets embed.FS

// Mount forwards /api and /health to api and serves the embedded SPA everywhere else.
func Mount(api http.Handler) http.Handler {
	ui := handler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			api.ServeHTTP(w, r)
			return
		}
		ui.ServeHTTP(w, r)
	})
}

func isAPIPath(p string) bool {
	p = path.Clean("/" + p)
	return p == "/health" || p == "/api" || strings.HasPrefix(p, "/api/")
}

func handler() http.Handler {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "ui unavailable", http.StatusInternalServerError)
		})
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if rel != "" {
			if _, err := fs.Stat(sub, rel); err != nil {
				if path.Ext(rel) != "" {
					http.NotFound(w, r)
					return
				}
				serveIndex(w, r, files)
				return
			}
		}
		setCache(w, rel)
		files.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, files http.Handler) {
	u := *r.URL
	u.Path = "/"
	req := r.Clone(r.Context())
	req.URL = &u
	setCache(w, "")
	files.ServeHTTP(w, req)
}

func setCache(w http.ResponseWriter, rel string) {
	ext := path.Ext(rel)
	if ext == "" || ext == ".html" {
		w.Header().Set("Cache-Control", "no-cache")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000")
}
