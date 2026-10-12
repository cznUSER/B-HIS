package store

import (
	"context"
	"testing"

	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestOpenMemory(t *testing.T) {
	users, _, _, patients, registers, closer, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "memory"}})
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()
	u, err := users.FindByUsername(nil, seed.DemoUsername)
	if err != nil || u.Username != "system" || u.DisplayName != seed.DemoDisplayName {
		t.Fatalf("%v %+v", err, u)
	}
	if patients == nil || registers == nil {
		t.Fatal("nil patient or register store")
	}
	ctx := context.Background()
	for _, id := range []string{"dept-1", "dept-2"} {
		ok, err := registers.DepartmentExists(ctx, id)
		if err != nil || !ok {
			t.Fatalf("department %s: %v %v", id, ok, err)
		}
	}
	for _, id := range []string{"doc-1", "doc-2"} {
		ok, err := registers.DoctorExists(ctx, id)
		if err != nil || !ok {
			t.Fatalf("doctor %s: %v %v", id, ok, err)
		}
	}
}

func TestOpenSQLServerRequiresDSN(t *testing.T) {
	_, _, _, _, _, _, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "sqlserver"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
