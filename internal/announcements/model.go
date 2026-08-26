// Package announcements manages public business-opening announcements.
package announcements

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	// OpeningCooldownKey is the durable key for business-opening throttling.
	OpeningCooldownKey = "business-opening"
	// DefaultAPIActor is used when an API caller omits attribution.
	DefaultAPIActor = "API"
)

var (
	// ErrNotFound reports that no announcement cooldown is stored.
	ErrNotFound = errors.New("announcement cooldown not found")
	// ErrCooldownActive reports an announcement attempted before its cooldown expired.
	ErrCooldownActive = errors.New("announcement cooldown is active")
	// ErrDisabled reports that no announcement channel is configured.
	ErrDisabled = errors.New("business opening announcements are disabled")
	// ErrInvalidActor reports invalid announcement attribution.
	ErrInvalidActor = errors.New("announcement actor is invalid")
)

// State describes one persisted announcement cooldown.
type State struct {
	// Key identifies the announcement operation.
	Key string `json:"key"`
	// Actor identifies who triggered the announcement.
	Actor string `json:"actor"`
	// AnnouncedAt is the time the cooldown was acquired.
	AnnouncedAt time.Time `json:"announcedAt"`
	// AvailableAt is the next allowed announcement time.
	AvailableAt time.Time `json:"availableAt"`
}

// CooldownActiveError exposes the current cooldown state.
type CooldownActiveError struct{ State State }

// Error describes the active cooldown.
func (err *CooldownActiveError) Error() string {
	return fmt.Sprintf("%s until %s", ErrCooldownActive, err.State.AvailableAt.Format(time.RFC3339))
}

// Unwrap supports errors.Is with ErrCooldownActive.
func (*CooldownActiveError) Unwrap() error { return ErrCooldownActive }

// Repository atomically persists announcement cooldowns.
type Repository interface {
	// Acquire creates or replaces an expired cooldown.
	Acquire(context.Context, string, time.Time, time.Time, string) (State, error)
	// Get returns one current cooldown.
	Get(context.Context, string) (State, error)
	// Release removes the exact acquisition after a failed publication.
	Release(context.Context, string, time.Time) error
	// Clear removes a cooldown regardless of its current state.
	Clear(context.Context, string) error
}

// Gateway publishes opening announcements to Discord.
type Gateway interface {
	// SendOpening publishes one attributed public opening announcement.
	SendOpening(context.Context, string, string) error
}
