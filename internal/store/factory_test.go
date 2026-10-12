package store

import (
	"testing"

	"github.com/cznUSER/B-HIS/internal/config"
)

func TestOpenMemory(t *testing.T) {
	users, _, _, closer, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "memory"}})
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()
	u, err := users.FindByUsername(nil, "admin")
	if err != nil || u.Username != "admin" {
		t.Fatalf("%v %+v", err, u)
	}
}

func TestOpenSQLServerRequiresDSN(t *testing.T) {
	_, _, _, _, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "sqlserver"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
