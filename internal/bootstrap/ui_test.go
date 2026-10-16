package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/cznUSER/B-HIS/internal/config"
)

func TestEmbeddedUIRequiresPersistentDatabase(t *testing.T) {
	cfg := config.Defaults()
	cfg.Auth.JWTSecret = "test-secret-must-be-long-enough-32b!!"
	cfg.Logging.Dir = t.TempDir()
	cfg.Host.ApiAddr = "127.0.0.1:0"
	if err := RunWith(context.Background(), cfg, Options{UI: true}); err == nil || !strings.Contains(err.Error(), "Database.DSN is required") {
		t.Fatalf("expected persistent database startup error, got %v", err)
	}
}
