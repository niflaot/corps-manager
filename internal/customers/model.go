// Package customers manages frequent customers and their Discord attendants.
package customers

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound reports an unknown customer.
	ErrNotFound = errors.New("customer not found")
	// ErrInvalidName reports a customer name outside the accepted format.
	ErrInvalidName = errors.New("invalid customer name")
	// ErrInvalidAttendant reports incomplete Discord attendant identity.
	ErrInvalidAttendant = errors.New("invalid customer attendant")
	// ErrInvalidAmount reports a visit amount outside the accepted range.
	ErrInvalidAmount = errors.New("invalid customer visit amount")
	// ErrInvalidQuery reports invalid customer search filters.
	ErrInvalidQuery = errors.New("invalid customer query")
	// ErrDisabled reports that the customer registry is disabled.
	ErrDisabled = errors.New("customer registry is disabled")
)

// Attendant aggregates visits recorded by one Discord member.
type Attendant struct {
	// DiscordUserID is the stable Discord user snowflake.
	DiscordUserID string `json:"discordUserId"`
	// DisplayName is the latest known server nickname or username.
	DisplayName string `json:"displayName"`
	// Visits is the number of visits recorded by this member.
	Visits int64 `json:"visits"`
	// FirstAttendedAt is the first recorded visit.
	FirstAttendedAt time.Time `json:"firstAttendedAt"`
	// LastAttendedAt is the latest recorded visit.
	LastAttendedAt time.Time `json:"lastAttendedAt"`
}

// Customer contains visit totals and unique attendants.
type Customer struct {
	// Name is the normalized lowercase customer key.
	Name string `json:"name"`
	// Visits is the total number of recorded visits.
	Visits int64 `json:"visits"`
	// TotalSpent is the accumulated customer spending in dollars.
	TotalSpent int64 `json:"totalSpent"`
	// AttendantCount is the number of unique Discord attendants.
	AttendantCount int `json:"attendantCount"`
	// Attendants contains unique Discord users ordered by visit count.
	Attendants []Attendant `json:"attendants,omitempty"`
	// CreatedAt is the first recorded visit.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is the latest recorded visit.
	UpdatedAt time.Time `json:"updatedAt"`
	// LastVisitAt is the latest recorded visit.
	LastVisitAt time.Time `json:"lastVisitAt"`
}

// Sort selects customer search ordering.
type Sort string

const (
	// SortSpend orders by highest spending.
	SortSpend Sort = "spend"
	// SortVisits orders by highest visit count.
	SortVisits Sort = "visits"
	// SortRecent orders by latest visit.
	SortRecent Sort = "recent"
	// SortName orders alphabetically.
	SortName Sort = "name"
)

// Query filters the customer directory.
type Query struct {
	// Name contains a normalized partial-name filter.
	Name string `json:"name,omitempty"`
	// Days limits results to visits in the latest number of days; zero means all time.
	Days int `json:"days,omitempty"`
	// Sort selects result ordering.
	Sort Sort `json:"sort"`
}

// Repository persists customers and their Discord attendants.
type Repository interface {
	// Record atomically records one visit and its attendant.
	Record(context.Context, string, string, string, int64) (Customer, error)
	// List returns all customers ordered by visit count.
	List(context.Context) ([]Customer, error)
	// Search returns customers matching a bounded query.
	Search(context.Context, Query) ([]Customer, error)
	// Get returns one customer and its unique attendants.
	Get(context.Context, string) (Customer, error)
	// Delete removes one customer and its history.
	Delete(context.Context, string) error
}
