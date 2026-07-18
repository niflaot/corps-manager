package settings

import "context"

// Repository persists application settings.
type Repository interface {
	// Get returns one persisted setting.
	Get(context.Context, Key) (Record, error)
	// List returns all persisted settings.
	List(context.Context) ([]Record, error)
	// Set creates or updates a setting at an optional expected revision.
	Set(context.Context, Key, []byte, uint64) (Record, error)
	// Reset deletes one setting at an optional expected revision.
	Reset(context.Context, Key, uint64) error
}
