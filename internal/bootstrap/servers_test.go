package bootstrap

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/webui"
)

func TestServeServersRunsAndStopsSeparateAPIAndWebListeners(t *testing.T) {
	apiListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	webListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = apiListener.Close()
		t.Fatal(err)
	}
	apiAddr := apiListener.Addr().String()
	webAddr := webListener.Addr().String()
	apiURL, err := localHTTPURL(apiAddr)
	if err != nil {
		t.Fatal(err)
	}

	log, err := logging.New(t.TempDir(), "error")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan string, 1)
	done := make(chan error, 1)
	bindings := []serverBinding{
		{server: newHTTPServer(apiAddr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" && !strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			_, _ = io.WriteString(w, "api")
		})), listener: apiListener},
		{server: newHTTPServer(webAddr, webui.Mount(apiURL)), listener: webListener},
	}
	go func() {
		done <- serveServers(ctx, bindings, webAddr, func(addr string) { ready <- addr }, log)
	}()

	select {
	case addr := <-ready:
		if addr != webAddr {
			t.Fatalf("ready address %q, want %q", addr, webAddr)
		}
	case err := <-done:
		t.Fatalf("listeners failed before ready: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("listeners did not become ready")
	}

	for url, want := range map[string]string{
		"http://" + apiAddr + "/health":      "api",
		"http://" + webAddr + "/api/v1/test": "api",
	} {
		response, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK || string(body) != want {
			t.Fatalf("GET %s status=%d body=%q err=%v", url, response.StatusCode, body, readErr)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("listeners did not stop after context cancellation")
	}
}
