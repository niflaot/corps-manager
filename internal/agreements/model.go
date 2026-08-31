// Package agreements manages business agreements shown in Discord.
package agreements

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrAlreadyExists reports a duplicate agreement identifier.
	ErrAlreadyExists = errors.New("agreement already exists")
	// ErrInvalidID reports an invalid agreement identifier.
	ErrInvalidID = errors.New("invalid agreement id")
	// ErrInvalidDescription reports an invalid agreement description.
	ErrInvalidDescription = errors.New("invalid agreement description")
	// ErrInvalidImageURL reports an invalid optional image URL.
	ErrInvalidImageURL = errors.New("invalid agreement image URL")
	// ErrDisabled reports that agreements are disabled.
	ErrDisabled = errors.New("agreements are disabled")
)

// Agreement describes one business agreement.
type Agreement struct {
	// ID is the normalized business identifier.
	ID string `json:"id"`
	// Description explains the agreement.
	Description string `json:"description"`
	// ImageURL is an optional HTTPS illustration.
	ImageURL string `json:"imageUrl,omitempty"`
	// CreatedBy is the Discord user snowflake that added it.
	CreatedBy string `json:"createdBy"`
	// CreatedAt is the persistence creation time.
	CreatedAt time.Time `json:"createdAt"`
}

// Repository persists business agreements.
type Repository interface {
	// Create inserts one unique agreement.
	Create(context.Context, Agreement) (Agreement, error)
	// List returns agreements ordered by identifier.
	List(context.Context) ([]Agreement, error)
}
