package discordlinks

import (
	"context"
	"time"
)

// CreateIntentRecord contains normalized intent persistence input.
type CreateIntentRecord struct {
	// Kind identifies whether the attempt links or authenticates.
	Kind IntentKind
	// Subject is the caller-owned stable account identifier.
	Subject string
	// CompletionKey selects a configured return destination.
	CompletionKey string
	// IdempotencyKey identifies one logical creation request.
	IdempotencyKey string
	// RequestHash identifies the canonical creation payload.
	RequestHash string
	// ExpiresAt is the intent expiry time.
	ExpiresAt time.Time
	// Now is the operation timestamp.
	Now time.Time
}

// CompleteLoginIntentRecord contains a Discord identity proven for login.
type CompleteLoginIntentRecord struct {
	// IntentID identifies the started OAuth attempt.
	IntentID string
	// Identity contains the current Discord user profile.
	Identity Identity
	// Scopes contains normalized granted OAuth scopes.
	Scopes []string
	// ResultHash identifies the opaque exchange credential.
	ResultHash string
	// ResultExpiresAt is the result credential expiry time.
	ResultExpiresAt time.Time
	// Now is the operation timestamp.
	Now time.Time
}

// CompleteIntentRecord contains successful Discord identity proof.
type CompleteIntentRecord struct {
	// IntentID identifies the started OAuth attempt.
	IntentID string
	// Identity contains the current Discord user profile.
	Identity Identity
	// Scopes contains normalized granted OAuth scopes.
	Scopes []string
	// ResultHash identifies the opaque exchange credential.
	ResultHash string
	// ResultExpiresAt is the result credential expiry time.
	ResultExpiresAt time.Time
	// Now is the operation timestamp.
	Now time.Time
}

// FailIntentRecord contains a safe OAuth failure outcome.
type FailIntentRecord struct {
	// IntentID identifies the started OAuth attempt.
	IntentID string
	// Status identifies the safe result category.
	Status ResultStatus
	// ErrorCode is the stable caller-facing failure identifier.
	ErrorCode string
	// ResultHash identifies the opaque exchange credential.
	ResultHash string
	// ResultExpiresAt is the result credential expiry time.
	ResultExpiresAt time.Time
	// Now is the operation timestamp.
	Now time.Time
}

// Repository persists intents, results, and account links.
type Repository interface {
	// CreateIntent creates or replays one idempotent OAuth attempt.
	CreateIntent(context.Context, CreateIntentRecord) (Intent, error)
	// StartIntent atomically binds a pending attempt to an OAuth state hash.
	StartIntent(context.Context, string, string, time.Time) (Intent, error)
	// ClaimIntentByState atomically owns one live callback by state hash.
	ClaimIntentByState(context.Context, string, time.Time) (Intent, error)
	// CompleteLinkIntent persists a proven identity and exchangeable link result.
	CompleteLinkIntent(context.Context, CompleteIntentRecord) (ResultStatus, error)
	// CompleteLoginIntent resolves a proven identity to an existing active link.
	CompleteLoginIntent(context.Context, CompleteLoginIntentRecord) (ResultStatus, error)
	// FailIntent persists an exchangeable safe failure result.
	FailIntent(context.Context, FailIntentRecord) error
	// ExchangeResult consumes or idempotently replays one result.
	ExchangeResult(context.Context, string, string, time.Time) (Result, error)
	// LinkBySubject returns the latest link history for one local subject.
	LinkBySubject(context.Context, string) (Link, error)
	// LinkByDiscordUser returns the active link for one Discord user.
	LinkByDiscordUser(context.Context, string) (Link, error)
	// Unlink conditionally removes one active association.
	Unlink(context.Context, string, time.Time) (Link, error)
	// DeleteExpiredIntents removes OAuth artifacts older than the retention boundary.
	DeleteExpiredIntents(context.Context, time.Time) (int64, error)
}

// OAuthGateway performs Discord's browser and server-side OAuth operations.
type OAuthGateway interface {
	// Enabled reports whether Discord OAuth has complete runtime configuration.
	Enabled() bool
	// AuthorizationURL builds Discord's authorization endpoint URL.
	AuthorizationURL(string) string
	// Exchange exchanges one authorization code for a transient user grant.
	Exchange(context.Context, string) (AccessGrant, error)
	// CurrentUser resolves the Discord identity represented by a grant.
	CurrentUser(context.Context, AccessGrant) (Identity, error)
	// Revoke invalidates the transient Discord authorization.
	Revoke(context.Context, AccessGrant) error
}
