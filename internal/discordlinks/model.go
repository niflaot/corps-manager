// Package discordlinks manages reusable local-subject links to Discord identities.
package discordlinks

import (
	"errors"
	"time"
)

var (
	// ErrConflict reports an occupied subject, Discord identity, or idempotency key.
	ErrConflict = errors.New("discord link conflict")
	// ErrExpired reports an intent or result that passed its validity window.
	ErrExpired = errors.New("discord link artifact expired")
	// ErrGone reports an artifact already consumed by another request.
	ErrGone = errors.New("discord link artifact already consumed")
	// ErrInvalid reports malformed link input.
	ErrInvalid = errors.New("invalid discord link input")
	// ErrNotFound reports an unknown intent, result, or link.
	ErrNotFound = errors.New("discord link record not found")
	// ErrProvider reports a failed Discord OAuth operation.
	ErrProvider = errors.New("discord oauth provider unavailable")
	// ErrUnavailable reports disabled Discord linking.
	ErrUnavailable = errors.New("discord linking unavailable")
)

// IntentStatus identifies the lifecycle of an OAuth link attempt.
type IntentStatus string

// IntentKind identifies the caller workflow using Discord OAuth.
type IntentKind string

const (
	// IntentKindLink binds a known local subject to a Discord identity.
	IntentKindLink IntentKind = "link"
	// IntentKindLogin resolves an existing link without accepting a subject.
	IntentKindLogin IntentKind = "login"
)

const (
	// ErrorCodeAuthorizationDenied reports a user-declined authorization.
	ErrorCodeAuthorizationDenied = "authorization_denied"
	// ErrorCodeAuthorizationFailed reports a provider-returned authorization failure.
	ErrorCodeAuthorizationFailed = "authorization_failed"
	// ErrorCodeProviderExchangeFailed reports a failed authorization-code exchange.
	ErrorCodeProviderExchangeFailed = "provider_exchange_failed"
	// ErrorCodeIdentityUnavailable reports an invalid or unavailable Discord identity.
	ErrorCodeIdentityUnavailable = "identity_unavailable"
	// ErrorCodeSubjectAlreadyLinked reports a local subject occupied by another identity.
	ErrorCodeSubjectAlreadyLinked = "subject_already_linked"
	// ErrorCodeDiscordUserAlreadyLinked reports a Discord identity occupied by another subject.
	ErrorCodeDiscordUserAlreadyLinked = "discord_user_already_linked"
	// ErrorCodeDiscordUserNotLinked reports a login identity without an active association.
	ErrorCodeDiscordUserNotLinked = "discord_user_not_linked"
)

const (
	// IntentStatusPending awaits a browser visit.
	IntentStatusPending IntentStatus = "pending"
	// IntentStatusStarted awaits Discord's callback.
	IntentStatusStarted IntentStatus = "started"
	// IntentStatusProcessing owns one in-flight Discord callback.
	IntentStatusProcessing IntentStatus = "processing"
	// IntentStatusCompleted has a result ready for exchange.
	IntentStatusCompleted IntentStatus = "completed"
)

// ResultStatus identifies the user-facing outcome of a link attempt.
type ResultStatus string

const (
	// ResultStatusLinked reports a persisted active link.
	ResultStatusLinked ResultStatus = "linked"
	// ResultStatusAuthenticated reports an existing link proven for login.
	ResultStatusAuthenticated ResultStatus = "authenticated"
	// ResultStatusNotLinked reports a proven Discord identity without an active link.
	ResultStatusNotLinked ResultStatus = "not_linked"
	// ResultStatusDenied reports that the user declined authorization.
	ResultStatusDenied ResultStatus = "denied"
	// ResultStatusConflict reports an identity already linked elsewhere.
	ResultStatusConflict ResultStatus = "conflict"
	// ResultStatusFailed reports an OAuth or persistence failure.
	ResultStatusFailed ResultStatus = "failed"
)

// LinkStatus identifies whether a durable link remains active.
type LinkStatus string

const (
	// LinkStatusLinked identifies an active Discord identity link.
	LinkStatusLinked LinkStatus = "linked"
	// LinkStatusUnlinked identifies retained link history after removal.
	LinkStatusUnlinked LinkStatus = "unlinked"
)

// CreateIntent requests a browser-based Discord link operation.
type CreateIntent struct {
	// Subject is the caller-owned stable local account identifier.
	Subject string `json:"subject"`
	// CompletionKey selects one server-configured return destination.
	CompletionKey string `json:"completionKey"`
	// IdempotencyKey makes creation safe to retry.
	IdempotencyKey string `json:"-"`
}

// CreateLoginIntent requests Discord authentication without accepting a local subject.
type CreateLoginIntent struct {
	// CompletionKey selects one server-configured return destination.
	CompletionKey string `json:"completionKey"`
	// IdempotencyKey makes creation safe to retry.
	IdempotencyKey string `json:"-"`
}

// Intent is one durable OAuth link attempt.
type Intent struct {
	// ID is the internal immutable UUID and public start token.
	ID string `json:"intentId"`
	// Kind identifies whether the attempt links or authenticates.
	Kind IntentKind `json:"kind"`
	// Subject is the caller-owned local account identifier.
	Subject string `json:"subject,omitempty"`
	// CompletionKey selects the registered return destination.
	CompletionKey string `json:"completionKey"`
	// Status is the current attempt lifecycle state.
	Status IntentStatus `json:"status"`
	// ExpiresAt is the final time at which OAuth may start.
	ExpiresAt time.Time `json:"expiresAt"`
	// CreatedAt is the persistence creation time.
	CreatedAt time.Time `json:"createdAt"`
}

// Callback contains the query values returned by Discord.
type Callback struct {
	// State is the opaque anti-forgery value returned by Discord.
	State string
	// Code is the successful authorization code.
	Code string
	// ProviderError is Discord's stable OAuth error identifier.
	ProviderError string
}

// Completion redirects the browser to a registered caller destination.
type Completion struct {
	// CompletionKey selects the registered return destination.
	CompletionKey string
	// Code is the opaque, short-lived result exchange credential.
	Code string
}

// AccessGrant contains the transient Discord OAuth access grant.
type AccessGrant struct {
	// AccessToken authenticates one Discord user request.
	AccessToken string
	// TokenType is the OAuth token scheme.
	TokenType string
	// Scope contains Discord's space-delimited granted scopes.
	Scope string
}

// Identity is the Discord profile proven by OAuth.
type Identity struct {
	// UserID is the immutable Discord user snowflake.
	UserID string
	// Username is the current Discord username.
	Username string
	// GlobalName is the optional Discord display name.
	GlobalName string
	// AvatarHash is the optional Discord CDN avatar hash.
	AvatarHash string
	// Bot reports whether the identity belongs to an automated account.
	Bot bool
}

// Link records one historical association with a Discord identity.
type Link struct {
	// ID is the internal immutable link UUID.
	ID string `json:"id"`
	// Subject is the caller-owned local account identifier.
	Subject string `json:"subject"`
	// DiscordUserID is the immutable Discord user snowflake.
	DiscordUserID string `json:"discordUserId"`
	// Username is the latest captured Discord username.
	Username string `json:"username"`
	// GlobalName is the latest captured Discord display name.
	GlobalName string `json:"globalName,omitempty"`
	// AvatarHash is the latest captured Discord avatar hash.
	AvatarHash string `json:"avatarHash,omitempty"`
	// AvatarURL is the derived Discord CDN avatar URL.
	AvatarURL string `json:"avatarUrl,omitempty"`
	// Scopes lists the scopes granted during linking.
	Scopes []string `json:"scopes"`
	// Status identifies whether the link remains active.
	Status LinkStatus `json:"status"`
	// LinkedAt records when the association became active.
	LinkedAt time.Time `json:"linkedAt"`
	// UnlinkedAt records when the association was removed.
	UnlinkedAt *time.Time `json:"unlinkedAt,omitempty"`
	// CreatedAt is the persistence creation time.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is the last persistence update time.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Result is the private, exchangeable outcome of one OAuth attempt.
type Result struct {
	// Status is the user-facing link outcome.
	Status ResultStatus `json:"status"`
	// Subject is the caller-owned local account identifier.
	Subject string `json:"subject,omitempty"`
	// ErrorCode is a stable safe failure identifier.
	ErrorCode string `json:"errorCode,omitempty"`
	// Link contains the active association after success.
	Link *Link `json:"link,omitempty"`
}
