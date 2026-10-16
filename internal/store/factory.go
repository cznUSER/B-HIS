package store

import (
	"fmt"
	"io"
	"strings"

	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/redact"
	"github.com/cznUSER/B-HIS/internal/store/dameng"
	"github.com/cznUSER/B-HIS/internal/store/mssql"
)

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// Open selects the persistent store driver from Database.Driver. Memory is
// intentionally not a runtime option; tests may instantiate memory.Store
// directly without routing through this composition root.
func Open(cfg config.Config) (port.UserStore, port.RevocationStore, port.AuditStore, port.PatientStore, port.RegisterStore, port.OrderStore, port.ChargeStore, port.DispenseStore, io.Closer, error) {
	switch strings.ToLower(cfg.Database.Driver) {
	case "sqlserver":
		if strings.TrimSpace(cfg.Database.DSN) == "" {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("Database.DSN is required when Driver is sqlserver")
		}
		set, err := (mssql.Driver{}).Open(cfg.Database.DSN)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("sqlserver open failed: %s", redact.String(err.Error()))
		}
		return flatten(set)
	case "dameng":
		_, err := (dameng.Driver{}).Open(cfg.Database.DSN)
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	default:
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("unsupported Database.Driver %q", cfg.Database.Driver)
	}
}

func flatten(set port.StoreSet) (port.UserStore, port.RevocationStore, port.AuditStore, port.PatientStore, port.RegisterStore, port.OrderStore, port.ChargeStore, port.DispenseStore, io.Closer, error) {
	if set.Users == nil || set.Revocations == nil || set.Audit == nil || set.Patients == nil || set.Registers == nil || set.Orders == nil || set.Charges == nil || set.Dispenses == nil || set.Closer == nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("database driver returned an incomplete store set")
	}
	return set.Users, set.Revocations, set.Audit, set.Patients, set.Registers, set.Orders, set.Charges, set.Dispenses, set.Closer, nil
}
