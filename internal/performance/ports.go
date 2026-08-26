package performance

import (
	"context"
	"errors"
)

var (
	// ErrNotFound reports that no performance state has been persisted yet.
	ErrNotFound = errors.New("business performance not found")
	// ErrConflict reports an optimistic persistence conflict.
	ErrConflict = errors.New("business performance conflict")
	// ErrDisabled reports that collection is not enabled.
	ErrDisabled = errors.New("business performance is disabled")
)

// Source reads one current business snapshot.
type Source interface {
	// Fetch reads current business and employee counters.
	Fetch(context.Context, int64) (Snapshot, error)
}

// Repository persists the business performance aggregate.
type Repository interface {
	// Get returns one persisted business aggregate.
	Get(context.Context, int64) (State, error)
	// Save creates or compare-and-swaps one aggregate.
	Save(context.Context, State, uint64) (State, error)
}
