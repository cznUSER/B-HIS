package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDriverEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "local.yaml")
	yaml := "Host:\n  ApiAddr: \"127.0.0.1:9090\"\nAuth:\n  JWTSecret: \"file-secret-not-from-env-32chars\"\nDatabase:\n  Driver: sqlserver\n  DSN: \"sqlserver://<user>:<password>@<host>:1433?database=BHIS\"\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BHIS_DB_DRIVER", "memory")
	t.Setenv("BHIS_JWT_SECRET", "env-secret-must-be-at-least-16")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Driver != "memory" {
		t.Fatalf("driver %s", cfg.Database.Driver)
	}
	if cfg.Auth.JWTSecret != "env-secret-must-be-at-least-16" {
		t.Fatal("env secret should win")
	}
	if strings.Contains(cfg.Database.DSN, "S3cret-value") {
		t.Fatal("unexpected secret")
	}
}

func TestSQLServerPlaceholderRejected(t *testing.T) {
	t.Setenv("BHIS_DB_DRIVER", "sqlserver")
	t.Setenv("BHIS_DB_DSN", "sqlserver://<user>:S3cret-value@<host>:1433?database=BHIS")
	t.Setenv("BHIS_JWT_SECRET", "env-secret-must-be-at-least-16")
	_, err := Load("")
	if err == nil {
		t.Fatal("expected placeholder rejection")
	}
	if strings.Contains(err.Error(), "S3cret-value") {
		t.Fatalf("error leaked dsn secret: %v", err)
	}
}

func TestExampleHasNoRealDSN(t *testing.T) {
	root := repoRoot(t)
	example, err := os.ReadFile(filepath.Join(root, "config.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(example)
	if !strings.Contains(text, "Driver: memory") || !strings.Contains(text, "database=BHIS") {
		t.Fatalf("example missing driver/database placeholder:\n%s", text)
	}
	if !strings.Contains(text, "<user>") || !strings.Contains(text, "<password>") || !strings.Contains(text, "<host>") {
		t.Fatal("example DSN must stay placeholder-only")
	}
	for _, bad := range []string{"sqlserver://sa:", "192.168.", "8888"} {
		if strings.Contains(text, bad) {
			t.Fatalf("example contains %s", bad)
		}
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "BHIS_DB_DRIVER") || !strings.Contains(string(readme), "BHIS_DB_DSN") {
		t.Fatal("readme missing env names")
	}
	sqlb, err := os.ReadFile(filepath.Join(root, "migrations/sqlserver/core/001_identity.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlb)
	if !strings.Contains(sql, "BHIS") || !strings.Contains(sql, "$2a$10$uVUbI161ahIWvB2/rbXUjuWubRAMog3zNNfldqp4mx/M.ZtuD8TgS") {
		t.Fatal("migration must target BHIS and embed the bcrypt hash")
	}
	if strings.Contains(sql, "Admin@123") {
		t.Fatal("sql seed must not contain the demo plaintext password")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
