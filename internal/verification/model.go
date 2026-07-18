// Package verification manages verification groups and memberships.
package verification

import (
	"errors"
	"regexp"
	"time"
)

const (
	// MaximumGroups is Discord's maximum buttons in one action row.
	MaximumGroups = 5
	// JoinCustomIDPrefix namespaces verification join interactions.
	JoinCustomIDPrefix = "verification.join."
	// LeaveCustomIDPrefix namespaces verification removal interactions.
	LeaveCustomIDPrefix = "verification.leave."
)

var groupKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

var (
	// ErrNotFound reports an unknown group or membership.
	ErrNotFound = errors.New("verification record not found")
	// ErrConflict reports a duplicate or stale revision.
	ErrConflict = errors.New("verification conflict")
	// ErrInvalid reports invalid Discord-facing configuration.
	ErrInvalid = errors.New("invalid verification configuration")
)

// Group configures one verification button and assigned Discord role.
type Group struct {
	// ID is the internal immutable UUID.
	ID string `json:"id"`
	// Key is the stable logical identifier.
	Key string `json:"key"`
	// RoleID is the Discord role assigned on verification.
	RoleID string `json:"roleId"`
	// ButtonLabel is the visible button label.
	ButtonLabel string `json:"buttonLabel"`
	// ButtonEmoji is an optional Unicode emoji.
	ButtonEmoji string `json:"buttonEmoji,omitempty"`
	// ButtonStyle is a Discord button style from one through four.
	ButtonStyle int `json:"buttonStyle"`
	// Position controls button ordering from one through five.
	Position int `json:"position"`
	// Enabled controls whether the button is rendered.
	Enabled bool `json:"enabled"`
	// Revision is the optimistic concurrency version.
	Revision uint64 `json:"revision"`
	// CreatedAt is the persistence creation time.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is the last persistence update time.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Membership records one user's active verification group.
type Membership struct {
	// ID is the internal UUID.
	ID string `json:"id"`
	// UserID is the Discord user snowflake.
	UserID string `json:"userId"`
	// GroupID identifies the verification group.
	GroupID string `json:"groupId"`
	// RoleID snapshots the assigned Discord role.
	RoleID string `json:"roleId"`
	// VerifiedAt records when the user accepted verification.
	VerifiedAt time.Time `json:"verifiedAt"`
	// CreatedAt is the persistence creation time.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is the last persistence update time.
	UpdatedAt time.Time `json:"updatedAt"`
}

// MemberState describes the current Discord state needed for reconciliation.
type MemberState struct {
	// Present reports whether the user currently belongs to the configured guild.
	Present bool
	// JoinedAt records the beginning of the user's current guild membership.
	JoinedAt time.Time
	// RoleIDs contains the roles currently assigned to the guild member.
	RoleIDs map[string]struct{}
}

// HasRole reports whether Discord currently assigns a role to the member.
func (state MemberState) HasRole(roleID string) bool {
	_, exists := state.RoleIDs[roleID]
	return exists
}

// Page contains filtered memberships.
type Page struct {
	// Items contains membership records.
	Items []Membership `json:"items"`
	// Total is the matching record count.
	Total int `json:"total"`
}
