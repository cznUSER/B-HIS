// Package port defines store interfaces. app and domain must not import a SQL driver.
package port

import (
	"context"
	"errors"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
)

// ErrNotFound is returned when a user does not exist (or is soft-deleted).
var ErrNotFound = errors.New("not found")

// UserStore loads identity records. Phase1 default is SQL Server; memory is the local demo.
type UserStore interface {
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
}

// RevocationStore is the JWT jti blacklist. Entries live at least until token exp.
type RevocationStore interface {
	Revoke(ctx context.Context, jti, userID string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

// AuditStore appends LOGIN_SUCCESS / LOGIN_FAILURE / LOGOUT. Implementations must not persist passwords.
type AuditStore interface {
	Append(ctx context.Context, event domain.AuditEvent) error
}
