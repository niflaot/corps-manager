// Package inactivity manages a durable registry of employees dismissed for inactivity.
package inactivity

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound reports an employee absent from the inactivity registry.
	ErrNotFound = errors.New("inactivity dismissal not found")
	// ErrAlreadyExists reports an employee already present in the registry.
	ErrAlreadyExists = errors.New("inactivity dismissal already exists")
	// ErrInvalidName reports a value outside the Nombre_Apellido format.
	ErrInvalidName = errors.New("employee name must use Nombre_Apellido format")
	// ErrDisabled reports that the inactivity registry is disabled.
	ErrDisabled = errors.New("inactivity registry is disabled")
)

// Entry is one employee dismissed for inactivity.
type Entry struct {
	// Name is the preserved Nombre_Apellido display value.
	Name string `json:"name"`
	// AddedBy is the Discord user snowflake or API actor that created the entry.
	AddedBy string `json:"addedBy"`
	// AddedAt is the persistence creation time.
	AddedAt time.Time `json:"addedAt"`
}

// Repository persists inactivity dismissal entries.
type Repository interface {
	// List returns all entries ordered by normalized employee name.
	List(context.Context) ([]Entry, error)
	// Add inserts one unique employee entry.
	Add(context.Context, string, string, string) (Entry, error)
	// Remove deletes one employee by normalized name.
	Remove(context.Context, string) error
}
