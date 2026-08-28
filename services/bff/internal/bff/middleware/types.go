package middleware

import (
	"github.com/ZoroNewbie00/kalakriti/pkg/idempotency"
)

// IdempotencyRecord is re-exported from pkg/idempotency for the Store interface.
type IdempotencyRecord = idempotency.Record
