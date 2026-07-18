// Package settings manages typed SQL-backed application settings.
package settings

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

// Key identifies one dotted application setting.
type Key string

const (
	// VerificationMessageKey selects the managed verification message.
	VerificationMessageKey Key = "verification.message.key"
	// VerificationTrapChannelID stores the reconciled trap channel snowflake.
	VerificationTrapChannelID Key = "verification.trap.channel_id"
	// VerificationTrapMessageID stores the reconciled warning message snowflake.
	VerificationTrapMessageID Key = "verification.trap.message_id"
	// VerificationTrapChannelName configures the trap channel name.
	VerificationTrapChannelName Key = "verification.trap.channel_name"
	// VerificationTrapWarning configures the trap warning text.
	VerificationTrapWarning Key = "verification.trap.warning"
)

const (
	// DefaultVerificationMessageKey is the default managed message key.
	DefaultVerificationMessageKey = "verification"
	// DefaultVerificationTrapChannelName is the default trap channel name.
	DefaultVerificationTrapChannelName = "no-escribas"
)

var dottedKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

var (
	// ErrNotFound reports an unknown setting key.
	ErrNotFound = errors.New("setting not found")
	// ErrConflict reports an optimistic concurrency conflict.
	ErrConflict = errors.New("setting revision conflict")
	// ErrInvalid reports an invalid key or value.
	ErrInvalid = errors.New("invalid setting")
)

// Record is one persisted setting.
type Record struct {
	// Key is the dotted setting identifier.
	Key Key `json:"key"`
	// Value is the typed JSON value.
	Value json.RawMessage `json:"value"`
	// Revision is the optimistic concurrency version.
	Revision uint64 `json:"revision"`
	// CreatedAt is the persistence creation time.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is the last persistence update time.
	UpdatedAt time.Time `json:"updatedAt"`
	// Default reports whether Value came from the code default.
	Default bool `json:"default"`
}
