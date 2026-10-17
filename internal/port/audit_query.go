package port

import (
	"context"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
)

// UserOperationQuery applies exact module/user filters and a half-open time range.
// CursorOccurredAt/CursorID are both exclusive and define descending order.
type UserOperationQuery struct {
	ModuleID         string
	UserID           string
	Begin            time.Time
	EndExclusive     time.Time
	CursorOccurredAt time.Time
	CursorID         string
	Limit            int
}

// UserOperationQueryStore reads append-only audit rows without changing the
// existing AuditStore.Append contract.
type UserOperationQueryStore interface {
	ListUserOperations(ctx context.Context, query UserOperationQuery) ([]domain.UserOperationLog, error)
}
