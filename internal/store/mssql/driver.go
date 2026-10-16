package mssql

import "github.com/cznUSER/B-HIS/internal/port"

// Driver adapts the SQL Server implementation to the application store
// contract. The composition root is the only caller of this adapter.
type Driver struct{}

func (Driver) Name() string { return "sqlserver" }

func (Driver) Open(dsn string) (port.StoreSet, error) {
	db, err := Open(dsn)
	if err != nil {
		return port.StoreSet{}, err
	}
	st := New(db)
	return port.StoreSet{
		Users: st, Revocations: st, Audit: st, Patients: st,
		Registers: st, Orders: st, Charges: st, Dispenses: st,
		Closer: db,
	}, nil
}
