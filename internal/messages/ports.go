package messages

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound reports an unknown managed message.
	ErrNotFound = errors.New("managed message not found")
	// ErrConflict reports idempotency or optimistic concurrency conflict.
	ErrConflict = errors.New("managed message conflict")
	// ErrForbidden reports missing Discord permission.
	ErrForbidden = errors.New("discord operation forbidden")
	// ErrRateLimited reports an exhausted Discord rate limit.
	ErrRateLimited = errors.New("discord operation rate limited")
	// ErrInvalidRemote reports a payload rejected by Discord.
	ErrInvalidRemote = errors.New("discord rejected the message")
	// ErrAmbiguousCreate reports an unsafe unknown creation result.
	ErrAmbiguousCreate = errors.New("discord message creation outcome is ambiguous")
	// ErrOwnership reports a Discord message owned by another author.
	ErrOwnership = errors.New("discord message is not owned by the configured bot")
	// ErrInvalidDefinition reports invalid desired state or API preconditions.
	ErrInvalidDefinition = errors.New("invalid managed message definition")
	// ErrInvalidAssignment reports a channel outside the configured guild.
	ErrInvalidAssignment = errors.New("discord channel does not belong to the configured guild")
)

// Idempotency identifies one API mutation and its canonical request.
type Idempotency struct {
	// Key is the caller-provided idempotency key.
	Key string
	// Operation identifies the route and logical message.
	Operation string
	// RequestHash is the canonical request digest.
	RequestHash string
}

// MutationResult contains a persisted record and replay metadata.
type MutationResult struct {
	// Record is the resulting managed message.
	Record Record
	// Replay reports whether a prior response was returned.
	Replay bool
}

// ListQuery filters and paginates managed messages.
type ListQuery struct {
	// State filters by reconciliation state.
	State State
	// GuildID filters by guild.
	GuildID string
	// ChannelID filters by channel.
	ChannelID string
	// Limit bounds the page size.
	Limit int
	// Offset is the row offset.
	Offset int
}

// Page contains one managed-message result page.
type Page struct {
	// Items contains ordered records.
	Items []Record `json:"items"`
	// Total is the filtered record count.
	Total int `json:"total"`
	// Limit is the applied page size.
	Limit int `json:"limit"`
	// Offset is the applied row offset.
	Offset int `json:"offset"`
}

// ClaimRequest selects due reconciliation work.
type ClaimRequest struct {
	// Owner identifies this reconciliation worker.
	Owner string
	// Limit bounds claimed records.
	Limit int
	// LeaseDuration controls crash recovery.
	LeaseDuration time.Duration
	// Now is the claim timestamp.
	Now time.Time
}

// Completion atomically records successful reconciliation.
type Completion struct {
	// ID identifies the managed message.
	ID string
	// Owner must match the active lease.
	Owner string
	// Revision prevents stale completion.
	Revision uint64
	// DiscordMessageID is the verified remote message.
	DiscordMessageID string
	// ObservedHash is the verified remote hash.
	ObservedHash string
	// Repaired reports whether Discord was mutated.
	Repaired bool
	// CheckedAt is the observation timestamp.
	CheckedAt time.Time
	// NextCheckAt schedules the next observation.
	NextCheckAt time.Time
}

// Release records failed reconciliation and retry policy.
type Release struct {
	// ID identifies the managed message.
	ID string
	// Owner must match the active lease.
	Owner string
	// Revision prevents stale failure writes.
	Revision uint64
	// State is the resulting drifted or blocked state.
	State State
	// Error is a sanitized failure message.
	Error string
	// CheckedAt is the failed observation timestamp.
	CheckedAt time.Time
	// NextCheckAt schedules retry.
	NextCheckAt time.Time
}

// Repository persists desired state, idempotency, and reconciliation leases.
type Repository interface {
	// Create persists a definition and its idempotent response.
	Create(context.Context, Definition, Idempotency) (MutationResult, error)
	// GetByKey returns one definition by logical key.
	GetByKey(context.Context, string) (Record, error)
	// List returns one filtered page.
	List(context.Context, ListQuery) (Page, error)
	// Replace applies a compare-and-swap desired-state update.
	Replace(context.Context, string, uint64, Definition, Idempotency) (MutationResult, error)
	// Archive stops managing one definition.
	Archive(context.Context, string, uint64, Idempotency) (MutationResult, error)
	// MarkDue schedules immediate reconciliation.
	MarkDue(context.Context, string) error
	// ClaimDue leases a bounded due batch.
	ClaimDue(context.Context, ClaimRequest) ([]Record, error)
	// Complete records a successful leased reconciliation.
	Complete(context.Context, Completion) error
	// Release records a failed leased reconciliation.
	Release(context.Context, Release) error
}

// ObservedMessage is the Discord-controlled observable projection.
type ObservedMessage struct {
	// ID is the Discord message snowflake.
	ID string
	// GuildID is the Discord guild snowflake.
	GuildID string
	// ChannelID is the Discord channel snowflake.
	ChannelID string
	// Payload is the normalized observable body.
	Payload Payload
	// Owned reports whether the configured bot authored it.
	Owned bool
}

// CreateRequest contains an idempotent Discord creation operation.
type CreateRequest struct {
	// ChannelID is the target Discord channel.
	ChannelID string
	// Payload is the desired message body.
	Payload Payload
	// Nonce is the stable Discord enforced nonce.
	Nonce string
}

// ReplaceRequest contains a complete Discord message replacement.
type ReplaceRequest struct {
	// ChannelID is the assigned Discord channel.
	ChannelID string
	// MessageID is the managed Discord message.
	MessageID string
	// Payload is the complete desired body.
	Payload Payload
}

// Gateway reads and mutates messages for the configured Discord bot.
type Gateway interface {
	// ValidateAssignment verifies guild and channel ownership.
	ValidateAssignment(context.Context, string, string) error
	// Get reads a Discord message.
	Get(context.Context, string, string) (ObservedMessage, error)
	// Create sends one idempotent Discord message.
	Create(context.Context, CreateRequest) (ObservedMessage, error)
	// Replace fully edits one Discord message.
	Replace(context.Context, ReplaceRequest) (ObservedMessage, error)
	// Delete removes one Discord message.
	Delete(context.Context, string, string) error
}

// Trigger wakes reconciliation after desired-state changes.
type Trigger interface {
	// Notify schedules a non-blocking wakeup.
	Notify()
	// C exposes reconciliation wakeups.
	C() <-chan struct{}
}
