// Package settings defines runtime-editable configuration persistence.
package settings

import "context"

// Store persists runtime overrides keyed by dotted setting names.
type Store interface {
	// List returns every setting whose key starts with the prefix.
	List(ctx context.Context, prefix string) (map[string]string, error)
	// SetMany atomically writes all given settings.
	SetMany(ctx context.Context, values map[string]string) error
}
