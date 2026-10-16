// Package dameng reserves the persistent-driver boundary for a future Dameng
// implementation. It deliberately fails closed until every port is backed by
// Dameng SQL, so selecting it cannot silently fall back to memory or SQL Server.
package dameng

import (
	"errors"
	"fmt"

	"github.com/cznUSER/B-HIS/internal/port"
)

var ErrNotImplemented = errors.New("dameng database adapter is not implemented")

// Driver is the stable switch point for a future DM8 implementation.
type Driver struct{}

func (Driver) Name() string { return "dameng" }

func (Driver) Open(_ string) (port.StoreSet, error) {
	return port.StoreSet{}, fmt.Errorf("%w: implement port.DatabaseDriver before enabling Database.Driver=dameng", ErrNotImplemented)
}
