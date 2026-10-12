package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMountKeepsAPIAndServesSPA(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"from":"api"}`))
	})
	h := Mount(api)

	for _, p := range []string{"/health", "/api/v1/auth/login", "/api"} {
		req := httptest.NewRequest(http.MethodPost, p, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"from":"api"`) {
			t.Fatalf("%s status %d body %s", p, rec.Code, rec.Body.String())
		}
	}

	for _, p := range []string{"/", "/login"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status %d", p, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "医院信息系统") {
			t.Fatalf("%s missing app shell: %s", p, body)
		}
		if !strings.Contains(body, `name="bhis-ui" content="api"`) {
			t.Fatalf("%s is not the real-API demo build", p)
		}
		if rec.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("%s cache %q", p, rec.Header().Get("Cache-Control"))
		}
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/favicon.ico", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("favicon status %d", rec.Code)
	}

	bg := httptest.NewRecorder()
	h.ServeHTTP(bg, httptest.NewRequest(http.MethodGet, "/bhis-login-bg.jpg", nil))
	if bg.Code != http.StatusOK {
		t.Fatalf("background status %d", bg.Code)
	}
	if ct := bg.Header().Get("Content-Type"); !strings.Contains(ct, "image/jpeg") {
		t.Fatalf("background content-type %q", ct)
	}
	if bg.Body.Len() < 1024 {
		t.Fatalf("background too small: %d", bg.Body.Len())
	}
}

func TestDemoBundleTalksToAPI(t *testing.T) {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		t.Fatal(err)
	}
	var js strings.Builder
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".js") {
			return err
		}
		b, err := fs.ReadFile(sub, p)
		if err != nil {
			return err
		}
		js.Write(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	text := js.String()
	if !strings.Contains(text, "/auth/login") && !strings.Contains(text, "/api/v1") {
		t.Fatal("demo bundle missing API login path")
	}
	if !strings.Contains(text, "Admin@123") {
		t.Fatal("demo bundle missing documented API demo account hint")
	}
	css, err := fs.ReadFile(sub, findCSS(t, sub))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), "bhis-login-bg.jpg") {
		t.Fatal("demo css missing login background")
	}
}

func findCSS(t *testing.T, sub fs.FS) string {
	t.Helper()
	var name string
	err := fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".css") {
			return err
		}
		name = p
		return fs.SkipAll
	})
	if err != nil && err != fs.SkipAll {
		t.Fatal(err)
	}
	if name == "" {
		t.Fatal("demo bundle missing css")
	}
	return name
}

func TestEmbeddedIndexReadable(t *testing.T) {
	b, err := fs.ReadFile(assets, "assets/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "bhis-ui") {
		t.Fatal(string(b))
	}
}
