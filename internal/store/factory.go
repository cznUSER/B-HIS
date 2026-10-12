package store

import (
	"fmt"
	"io"
	"strings"

	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/redact"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/mssql"
)

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// Open selects the identity store from Database.Driver. Callers outside main should not import drivers.
func Open(cfg config.Config) (port.UserStore, port.RevocationStore, port.AuditStore, io.Closer, error) {
	switch strings.ToLower(cfg.Database.Driver) {
	case "", "memory":
		m := memory.New()
		return m, m, m, closerFunc(func() error { return nil }), nil
	case "sqlserver":
		if strings.TrimSpace(cfg.Database.DSN) == "" {
			return nil, nil, nil, nil, fmt.Errorf("Database.DSN is required when Driver is sqlserver")
		}
		db, err := mssql.Open(cfg.Database.DSN)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("sqlserver open failed: %s", redact.String(err.Error()))
		}
		st := mssql.New(db)
		return st, st, st, db, nil
	default:
		return nil, nil, nil, nil, fmt.Errorf("unsupported Database.Driver %q", cfg.Database.Driver)
	}
}
