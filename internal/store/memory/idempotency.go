package memory

import "time"

type idempotencyRecord struct {
	scope       string
	key         string
	requestHash string
	state       string
	resourceID  string
	expiresAt   time.Time
}

const (
	idempotencyPending   = "PENDING"
	idempotencyCompleted = "COMPLETED"
)

func idempotencyMapKey(scope, key string) string { return scope + "\x00" + key }

func newIdempotencyMap() map[string]idempotencyRecord {
	return make(map[string]idempotencyRecord)
}
