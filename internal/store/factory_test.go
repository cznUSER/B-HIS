package store

import (
	"testing"

	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestOpenMemory(t *testing.T) {
	users, _, _, patients, closer, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "memory"}})
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()
	u, err := users.FindByUsername(nil, seed.DemoUsername)
	if err != nil || u.Username != "system" || u.DisplayName != seed.DemoDisplayName {
		t.Fatalf("%v %+v", err, u)
	}
	if patients == nil {
		t.Fatal("nil patient store")
	}
}

func TestOpenSQLServerRequiresDSN(t *testing.T) {
	_, _, _, _, _, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "sqlserver"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
