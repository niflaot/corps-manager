package notification

import "context"

// Repository persists the durable verification notification outbox.
type Repository interface {
	// Enqueue inserts one event unless its idempotency key already exists.
	Enqueue(context.Context, Event) (bool, error)
	// ClaimDue leases one bounded due batch.
	ClaimDue(context.Context, ClaimRequest) ([]Delivery, error)
	// Complete marks one leased delivery successful.
	Complete(context.Context, Completion) error
	// Release schedules retry or moves one leased delivery to the dead-letter state.
	Release(context.Context, Release) error
}

// Sender delivers one informational notification idempotently.
type Sender interface {
	// Send delivers one notification and returns the Discord message ID.
	Send(context.Context, Delivery) (string, error)
}

// Publisher records informational verification transitions.
type Publisher interface {
	// Enqueue durably records one event without affecting the primary operation.
	Enqueue(context.Context, Event)
}
