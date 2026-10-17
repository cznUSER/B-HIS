// Package webui serves the built login SPA and forwards API traffic to the API listener.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

//go:embed all:assets
var assets embed.FS

// Mount forwards /api and /health to the API address and serves the embedded SPA elsewhere.
func Mount(apiBaseURL string) http.Handler {
	ui := handler()
	target, err := url.Parse(apiBaseURL)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "API address is invalid", http.StatusInternalServerError)
		})
	}
	api := httputil.NewSingleHostReverseProxy(target)
	api.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "API unavailable", http.StatusBadGateway)
	}
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
