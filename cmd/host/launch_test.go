package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cznUSER/B-HIS/internal/config"
)

func TestLoginURL(t *testing.T) {
	if got := loginURL("127.0.0.1:8080"); got != "http://127.0.0.1:8080/login" {
		t.Fatal(got)
	}
	if got := loginURL(":8080"); got != "http://127.0.0.1:8080/login" {
		t.Fatal(got)
	}
}

func TestPlanBrowser(t *testing.T) {
	const page = "http://127.0.0.1:8080/login"
	win := planBrowser("windows", page)
	if !win.UseShellExecute || len(win.Args) != 0 {
		t.Fatal("windows must open the browser via ShellExecute, not a console child")
	}
	if win.CreationFlags&createNoWindow == 0 || win.CreationFlags&createNewConsole != 0 {
		t.Fatalf("fallback flags %#x", win.CreationFlags)
	}
	if len(win.Fallback) != 3 || win.Fallback[0] != "rundll32" || win.Fallback[2] != page {
		t.Fatal(win.Fallback)
	}
	if got := planBrowser("darwin", page).Args; len(got) != 2 || got[0] != "open" || got[1] != page {
		t.Fatal(got)
	}
	if got := planBrowser("linux", page).Args; len(got) != 2 || got[0] != "xdg-open" || got[1] != page {
		t.Fatal(got)
	}
}

func TestBrowserEnabled(t *testing.T) {
	t.Setenv("BHIS_OPEN_BROWSER", "")
	if !browserEnabled(false) || browserEnabled(true) {
		t.Fatal("flag")
	}
	t.Setenv("BHIS_OPEN_BROWSER", "0")
	if browserEnabled(false) {
		t.Fatal("env should disable the browser")
	}
}

func TestSearchConfigOrder(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("config.example.yaml", "example")
	if got := searchConfig([]string{dir}); filepath.Base(got) != "config.example.yaml" {
		t.Fatal(got)
	}
	write("config.yaml", "yaml")
	if got := searchConfig([]string{dir}); filepath.Base(got) != "config.yaml" {
		t.Fatal(got)
	}
	write("config.local.yaml", "local")
	if got := searchConfig([]string{dir}); filepath.Base(got) != "config.local.yaml" {
		t.Fatal(got)
	}
}

func TestLoadDemoUsesPlaceholderWithoutConfig(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv("BHIS_JWT_SECRET", "")
	t.Setenv("BHIS_CONFIG", "")
	t.Setenv("BHIS_DB_DRIVER", "memory")
	t.Setenv("BHIS_DB_DSN", "")
	cfg, err := loadDemo("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.JWTSecret != config.DemoJWTPlaceholder || !cfg.PlaceholderSecret() {
		t.Fatalf("secret %q", cfg.Auth.JWTSecret)
	}
	if cfg.Database.Driver != "memory" {
		t.Fatal(cfg.Database.Driver)
	}
	if cfg.Database.DSN != "" {
		t.Fatal("demo load must not invent a DSN")
	}
}

func TestLoadDemoKeepsEnvSecret(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv("BHIS_JWT_SECRET", "env-secret-must-be-at-least-16")
	t.Setenv("BHIS_DB_DRIVER", "memory")
	t.Setenv("BHIS_DB_DSN", "")
	cfg, err := loadDemo("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.JWTSecret != "env-secret-must-be-at-least-16" {
		t.Fatal(cfg.Auth.JWTSecret)
	}
}

func TestLoadDemoDoesNotHideBadDSN(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv("BHIS_JWT_SECRET", "env-secret-must-be-at-least-16")
	t.Setenv("BHIS_DB_DRIVER", "sqlserver")
	t.Setenv("BHIS_DB_DSN", "sqlserver://<user>:<password>@<host>:1433?database=BHIS")
	_, err := loadDemo("")
	if err == nil {
		t.Fatal("expected placeholder DSN rejection")
	}
	if strings.Contains(err.Error(), "<password>") && strings.Contains(err.Error(), "S3cret") {
		t.Fatal(err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}
