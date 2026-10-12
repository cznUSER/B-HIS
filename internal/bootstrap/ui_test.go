package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/config"
)

func TestEmbeddedUIAndMemoryLogin(t *testing.T) {
	cfg := config.Defaults()
	cfg.Auth.JWTSecret = "test-secret-must-be-long-enough-32b!!"
	cfg.Logging.Dir = t.TempDir()
	cfg.Host.ApiAddr = "127.0.0.1:0"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunWith(ctx, cfg, Options{
			UI: true,
			OnReady: func(addr string) {
				ready <- addr
			},
		})
	}()

	var addr string
	select {
	case addr = <-ready:
	case err := <-errCh:
		t.Fatalf("server: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not become ready")
	}

	client := &http.Client{Timeout: 3 * time.Second}
	page, err := client.Get("http://" + addr + "/login")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(page.Body)
	page.Body.Close()
	if page.StatusCode != http.StatusOK || !strings.Contains(string(body), "医院信息系统") {
		t.Fatalf("login page %d %s", page.StatusCode, body)
	}

	health, err := client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatal(err)
	}
	hb, _ := io.ReadAll(health.Body)
	health.Body.Close()
	if health.StatusCode != http.StatusOK || !strings.Contains(string(hb), `"driver":"memory"`) {
		t.Fatalf("health %d %s", health.StatusCode, hb)
	}

	login, err := client.Post(
		"http://"+addr+"/api/v1/auth/login",
		"application/json",
		strings.NewReader(`{"username":"admin","password":"Admin@123"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	lb, _ := io.ReadAll(login.Body)
	login.Body.Close()
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login %d %s", login.StatusCode, lb)
	}
	var payload struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(lb, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.AccessToken == "" || payload.User.Username != "admin" {
		t.Fatalf("payload %+v", payload)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown timed out")
	}
}
