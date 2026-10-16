package store

import (
	"strings"
	"testing"

	"github.com/cznUSER/B-HIS/internal/config"
)

func TestOpenMemoryRejected(t *testing.T) {
	users, _, _, patients, registers, orders, charges, dispenses, closer, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "memory"}})
	if err == nil || users != nil || patients != nil || registers != nil || orders != nil || charges != nil || dispenses != nil || closer != nil {
		t.Fatalf("memory runtime must be rejected: err=%v", err)
	}
}

func TestOpenDamengReserved(t *testing.T) {
	_, _, _, _, _, _, _, _, _, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "dameng", DSN: "dm://placeholder"}})
	if err == nil || !strings.Contains(err.Error(), "dameng database adapter is not implemented") {
		t.Fatalf("expected explicit Dameng reservation error, got %v", err)
	}
}

func TestOpenSQLServerRequiresDSN(t *testing.T) {
	_, _, _, _, _, _, _, _, _, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "sqlserver"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
