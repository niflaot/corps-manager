// Package notification delivers durable informational verification messages.
package notification

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Kind identifies one verification membership transition.
type Kind string

const (
	// KindVerified reports a newly active verification membership.
	KindVerified Kind = "verified"
	// KindUnverified reports an explicitly removed verification membership.
	KindUnverified Kind = "unverified"
)

// State identifies one durable notification delivery state.
type State string

const (
	// StatePending awaits its first delivery attempt.
	StatePending State = "pending"
	// StateDelivering is leased by one dispatcher.
	StateDelivering State = "delivering"
	// StateRetry awaits its next automatic attempt.
	StateRetry State = "retry"
	// StateDelivered records successful Discord delivery.
	StateDelivered State = "delivered"
	// StateDead is the durable dead-letter queue.
	StateDead State = "dead"
)

var (
	// ErrConflict reports a stale notification lease.
	ErrConflict = errors.New("verification notification conflict")
)

// Event describes an informational membership transition to enqueue.
type Event struct {
	// IdempotencyKey deduplicates repeated publication.
	IdempotencyKey string
	// Kind identifies verification or explicit unverification.
	Kind Kind
	// UserID identifies the Discord DM recipient.
	UserID string
	// GroupID identifies the verification group used by interaction buttons.
	GroupID string
	// GroupKey identifies the localized group name.
	GroupKey string
}

// Delivery is one persisted outbox record.
type Delivery struct {
	// ID identifies the outbox record.
	ID string
	// IdempotencyKey identifies the logical transition and Discord nonce.
	IdempotencyKey string
	// Kind identifies verification or explicit unverification.
	Kind Kind
	// UserID identifies the Discord DM recipient.
	UserID string
	// GroupID identifies the verification group.
	GroupID string
	// GroupKey identifies the localized group name.
	GroupKey string
	// State is the current delivery state.
	State State
	// Attempts is the number of completed delivery attempts.
	Attempts int
}

// ClaimRequest selects due outbox records.
type ClaimRequest struct {
	// Owner identifies the claiming dispatcher.
	Owner string
	// Limit bounds one batch.
	Limit int
	// LeaseDuration controls crash recovery.
	LeaseDuration time.Duration
	// Now is the deterministic claim time.
	Now time.Time
}

// Completion records one successful delivery.
type Completion struct {
	// ID identifies the outbox record.
	ID string
	// Owner must match the active lease.
	Owner string
	// DiscordMessageID identifies the delivered DM.
	DiscordMessageID string
	// DeliveredAt records successful delivery.
	DeliveredAt time.Time
}

// Release records one failed delivery attempt.
type Release struct {
	// ID identifies the outbox record.
	ID string
	// Owner must match the active lease.
	Owner string
	// State selects retry or dead-letter handling.
	State State
	// Error is the sanitized delivery failure.
	Error string
	// NextAttemptAt schedules the retry.
	NextAttemptAt time.Time
}

// NewEvent creates a stable idempotent transition event.
func NewEvent(kind Kind, membershipID, userID, groupID, groupKey string) Event {
	return Event{
		IdempotencyKey: fmt.Sprintf("verification:%s:%s", kind, membershipID),
		Kind:           kind,
		UserID:         userID,
		GroupID:        groupID,
		GroupKey:       groupKey,
	}
}

// valid reports whether an event contains every durable delivery identity.
func (event Event) valid() bool {
	prefix := fmt.Sprintf("verification:%s:", event.Kind)
	return (event.Kind == KindVerified || event.Kind == KindUnverified) &&
		strings.HasPrefix(event.IdempotencyKey, prefix) && len(event.IdempotencyKey) > len(prefix) &&
		event.UserID != "" && event.GroupID != "" && event.GroupKey != ""
}
