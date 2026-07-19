package discordlinks

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/pixelados-net/discord-bot/platform/clock"
)

const tokenBytes = 32

var completionKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var subjectPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

// Service coordinates durable account-link OAuth workflows.
type Service struct {
	repository Repository
	oauth      OAuthGateway
	clock      clock.Clock
	config     Config
}

// NewService creates the Discord account-link service.
func NewService(repository Repository, oauth OAuthGateway, serviceClock clock.Clock, config Config) *Service {
	return &Service{repository: repository, oauth: oauth, clock: serviceClock, config: config}
}

// CreateIntent creates or replays one caller-bound OAuth attempt.
func (service *Service) CreateIntent(ctx context.Context, request CreateIntent) (Intent, error) {
	request.Subject = strings.TrimSpace(request.Subject)
	request.CompletionKey = strings.TrimSpace(request.CompletionKey)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if !service.oauth.Enabled() {
		return Intent{}, ErrUnavailable
	}
	if !subjectPattern.MatchString(request.Subject) || !completionKeyPattern.MatchString(request.CompletionKey) ||
		!idempotencyKeyPattern.MatchString(request.IdempotencyKey) {
		return Intent{}, fmt.Errorf("%w: invalid subject, completion key, or idempotency key", ErrInvalid)
	}
	return service.create(ctx, IntentKindLink, request.Subject, request.CompletionKey, request.IdempotencyKey)
}

// CreateLoginIntent creates or replays one subject-free Discord login attempt.
func (service *Service) CreateLoginIntent(ctx context.Context, request CreateLoginIntent) (Intent, error) {
	request.CompletionKey = strings.TrimSpace(request.CompletionKey)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if !service.oauth.Enabled() {
		return Intent{}, ErrUnavailable
	}
	if !completionKeyPattern.MatchString(request.CompletionKey) ||
		!idempotencyKeyPattern.MatchString(request.IdempotencyKey) {
		return Intent{}, fmt.Errorf("%w: invalid completion key or idempotency key", ErrInvalid)
	}
	return service.create(ctx, IntentKindLogin, "", request.CompletionKey, request.IdempotencyKey)
}

// Start activates one pending intent and returns Discord's authorization URL.
func (service *Service) Start(ctx context.Context, intentID string) (string, error) {
	if !service.oauth.Enabled() {
		return "", ErrUnavailable
	}
	if _, err := uuid.Parse(intentID); err != nil {
		return "", fmt.Errorf("%w: invalid intent ID", ErrInvalid)
	}
	state, err := randomToken()
	if err != nil {
		return "", err
	}
	if _, err = service.repository.StartIntent(ctx, intentID, tokenHash(state), service.clock.Now()); err != nil {
		return "", err
	}
	return service.oauth.AuthorizationURL(state), nil
}

// Complete validates a Discord callback and creates an exchangeable result.
func (service *Service) Complete(ctx context.Context, callback Callback) (Completion, error) {
	callback.State = strings.TrimSpace(callback.State)
	callback.Code = strings.TrimSpace(callback.Code)
	callback.ProviderError = strings.TrimSpace(callback.ProviderError)
	if callback.State == "" || len(callback.State) > 256 {
		return Completion{}, fmt.Errorf("%w: invalid OAuth state", ErrInvalid)
	}
	intent, err := service.repository.ClaimIntentByState(ctx, tokenHash(callback.State), service.clock.Now())
	if err != nil {
		return Completion{}, err
	}
	resultCode, err := randomToken()
	if err != nil {
		return Completion{}, err
	}
	if callback.ProviderError != "" {
		status := ResultStatusFailed
		code := ErrorCodeAuthorizationFailed
		if callback.ProviderError == "access_denied" {
			status, code = ResultStatusDenied, ErrorCodeAuthorizationDenied
		}
		err = service.fail(ctx, intent.ID, status, code, resultCode)
		return Completion{CompletionKey: intent.CompletionKey, Code: resultCode}, err
	}
	if callback.Code == "" || len(callback.Code) > 512 {
		return Completion{}, fmt.Errorf("%w: missing authorization code", ErrInvalid)
	}
	grant, exchangeErr := service.oauth.Exchange(ctx, callback.Code)
	if exchangeErr != nil {
		err = service.fail(ctx, intent.ID, ResultStatusFailed, ErrorCodeProviderExchangeFailed, resultCode)
		return Completion{CompletionKey: intent.CompletionKey, Code: resultCode}, err
	}
	defer func() { _ = service.oauth.Revoke(ctx, grant) }()
	identity, identityErr := service.oauth.CurrentUser(ctx, grant)
	if identityErr != nil || !validIdentity(identity) {
		err = service.fail(ctx, intent.ID, ResultStatusFailed, ErrorCodeIdentityUnavailable, resultCode)
		return Completion{CompletionKey: intent.CompletionKey, Code: resultCode}, err
	}
	now := service.clock.Now()
	if intent.Kind == IntentKindLogin {
		_, err = service.repository.CompleteLoginIntent(ctx, CompleteLoginIntentRecord{
			IntentID: intent.ID, Identity: identity, Scopes: normalizeScopes(grant.Scope),
			ResultHash: tokenHash(resultCode), ResultExpiresAt: now.Add(service.config.ResultTTL), Now: now,
		})
	} else {
		_, err = service.repository.CompleteLinkIntent(ctx, CompleteIntentRecord{
			IntentID: intent.ID, Identity: identity, Scopes: normalizeScopes(grant.Scope),
			ResultHash: tokenHash(resultCode), ResultExpiresAt: now.Add(service.config.ResultTTL), Now: now,
		})
	}
	return Completion{CompletionKey: intent.CompletionKey, Code: resultCode}, err
}

// ExchangeResult consumes or replays one private callback result.
func (service *Service) ExchangeResult(ctx context.Context, code string, idempotencyKey string) (Result, error) {
	code, idempotencyKey = strings.TrimSpace(code), strings.TrimSpace(idempotencyKey)
	if code == "" || len(code) > 256 || !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return Result{}, fmt.Errorf("%w: invalid result code or idempotency key", ErrInvalid)
	}
	return service.repository.ExchangeResult(ctx, tokenHash(code), idempotencyKey, service.clock.Now())
}

// LinkBySubject returns the latest association for one caller-owned subject.
func (service *Service) LinkBySubject(ctx context.Context, subject string) (Link, error) {
	if !subjectPattern.MatchString(subject) {
		return Link{}, fmt.Errorf("%w: invalid subject", ErrInvalid)
	}
	return service.repository.LinkBySubject(ctx, subject)
}

// LinkByDiscordUser returns the active association for one Discord identity.
func (service *Service) LinkByDiscordUser(ctx context.Context, userID string) (Link, error) {
	if !snowflakePattern.MatchString(userID) {
		return Link{}, fmt.Errorf("%w: invalid Discord user ID", ErrInvalid)
	}
	return service.repository.LinkByDiscordUser(ctx, userID)
}

// Unlink removes one active association while preserving its history.
func (service *Service) Unlink(ctx context.Context, subject string) (Link, error) {
	if !subjectPattern.MatchString(subject) {
		return Link{}, fmt.Errorf("%w: invalid subject", ErrInvalid)
	}
	return service.repository.Unlink(ctx, subject, service.clock.Now())
}

// Cleanup removes expired OAuth artifacts while retaining durable link history.
func (service *Service) Cleanup(ctx context.Context) error {
	_, err := service.repository.DeleteExpiredIntents(ctx, service.clock.Now().Add(-service.config.ArtifactRetention))
	return err
}

func (service *Service) fail(ctx context.Context, intentID string, status ResultStatus, code string, resultCode string) error {
	now := service.clock.Now()
	return service.repository.FailIntent(ctx, FailIntentRecord{IntentID: intentID, Status: status,
		ErrorCode: code, ResultHash: tokenHash(resultCode), ResultExpiresAt: now.Add(service.config.ResultTTL), Now: now})
}

func (service *Service) create(ctx context.Context, kind IntentKind, subject string, completionKey string,
	idempotencyKey string) (Intent, error) {
	now := service.clock.Now()
	digest := sha256.Sum256([]byte(string(kind) + "\x00" + subject + "\x00" + completionKey))
	return service.repository.CreateIntent(ctx, CreateIntentRecord{
		Kind: kind, Subject: subject, CompletionKey: completionKey, IdempotencyKey: idempotencyKey,
		RequestHash: hex.EncodeToString(digest[:]), ExpiresAt: now.Add(service.config.IntentTTL), Now: now,
	})
}

func randomToken() (string, error) {
	buffer := make([]byte, tokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate secure token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func tokenHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func normalizeScopes(scope string) []string {
	scopes := strings.Fields(scope)
	sort.Strings(scopes)
	return scopes
}

func validIdentity(identity Identity) bool {
	return !identity.Bot && snowflakePattern.MatchString(identity.UserID) &&
		len(identity.Username) >= 1 && len(identity.Username) <= 64 && len(identity.GlobalName) <= 64 &&
		len(identity.AvatarHash) <= 128
}
