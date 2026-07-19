package discordlinks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pixelados-net/discord-bot/platform/clock"
)

type serviceRepository struct {
	created       CreateIntentRecord
	intent        Intent
	startedHash   string
	completedLink CompleteIntentRecord
	completedAuth CompleteLoginIntentRecord
	failed        FailIntentRecord
	result        Result
}

func (repository *serviceRepository) CreateIntent(_ context.Context, request CreateIntentRecord) (Intent, error) {
	repository.created = request
	return Intent{ID: "00000000-0000-0000-0000-000000000001", Kind: request.Kind,
		Subject: request.Subject, CompletionKey: request.CompletionKey, ExpiresAt: request.ExpiresAt}, nil
}

func (repository *serviceRepository) StartIntent(_ context.Context, id string, stateHash string, _ time.Time) (Intent, error) {
	repository.startedHash = stateHash
	return Intent{ID: id}, nil
}

func (repository *serviceRepository) ClaimIntentByState(context.Context, string, time.Time) (Intent, error) {
	return repository.intent, nil
}

func (repository *serviceRepository) CompleteLinkIntent(_ context.Context,
	request CompleteIntentRecord) (ResultStatus, error) {
	repository.completedLink = request
	return ResultStatusLinked, nil
}

func (repository *serviceRepository) CompleteLoginIntent(_ context.Context,
	request CompleteLoginIntentRecord) (ResultStatus, error) {
	repository.completedAuth = request
	return ResultStatusAuthenticated, nil
}

func (repository *serviceRepository) FailIntent(_ context.Context, request FailIntentRecord) error {
	repository.failed = request
	return nil
}

func (repository *serviceRepository) ExchangeResult(context.Context, string, string, time.Time) (Result, error) {
	return repository.result, nil
}

func (*serviceRepository) LinkBySubject(context.Context, string) (Link, error) {
	return Link{}, ErrNotFound
}

func (*serviceRepository) LinkByDiscordUser(context.Context, string) (Link, error) {
	return Link{}, ErrNotFound
}

func (*serviceRepository) Unlink(context.Context, string, time.Time) (Link, error) {
	return Link{}, ErrNotFound
}

func (*serviceRepository) DeleteExpiredIntents(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type oauthGateway struct {
	enabled bool
	revoked bool
}

func (gateway *oauthGateway) Enabled() bool { return gateway.enabled }
func (*oauthGateway) AuthorizationURL(state string) string {
	return "https://discord.example/authorize?state=" + state
}
func (*oauthGateway) Exchange(context.Context, string) (AccessGrant, error) {
	return AccessGrant{AccessToken: "secret", TokenType: "Bearer", Scope: "identify"}, nil
}
func (*oauthGateway) CurrentUser(context.Context, AccessGrant) (Identity, error) {
	return Identity{UserID: "123", Username: "tester", GlobalName: "Test User"}, nil
}
func (gateway *oauthGateway) Revoke(context.Context, AccessGrant) error {
	gateway.revoked = true
	return nil
}

func TestServiceCreatesLinkAndLoginIntents(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	repository := &serviceRepository{}
	service := NewService(repository, &oauthGateway{enabled: true}, clock.NewFake(now),
		Config{IntentTTL: 10 * time.Minute, ResultTTL: 5 * time.Minute})
	link, err := service.CreateIntent(context.Background(), CreateIntent{Subject: "user:test-001",
		CompletionKey: "pixelados-links", IdempotencyKey: "link-test-001"})
	if err != nil || link.Kind != IntentKindLink || repository.created.Subject != "user:test-001" ||
		repository.created.RequestHash == "" {
		t.Fatalf("CreateIntent() = %#v, record = %#v, error = %v", link, repository.created, err)
	}
	login, err := service.CreateLoginIntent(context.Background(), CreateLoginIntent{
		CompletionKey: "pixelados-login", IdempotencyKey: "login-test-001"})
	if err != nil || login.Kind != IntentKindLogin || repository.created.Subject != "" {
		t.Fatalf("CreateLoginIntent() = %#v, record = %#v, error = %v", login, repository.created, err)
	}
}

func TestServiceStartsWithHashedState(t *testing.T) {
	repository := &serviceRepository{}
	service := NewService(repository, &oauthGateway{enabled: true}, clock.NewFake(time.Now()), Config{})
	destination, err := service.Start(context.Background(), "00000000-0000-0000-0000-000000000001")
	if err != nil || !strings.HasPrefix(destination, "https://discord.example/authorize?state=") {
		t.Fatalf("Start() = %q, error = %v", destination, err)
	}
	rawState := strings.TrimPrefix(destination, "https://discord.example/authorize?state=")
	if repository.startedHash == "" || repository.startedHash == rawState || repository.startedHash != tokenHash(rawState) {
		t.Fatalf("stored state hash = %q, raw = %q", repository.startedHash, rawState)
	}
	if _, err = service.Start(context.Background(), "not-a-uuid"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid Start() error = %v", err)
	}
}

func TestServiceCompletesLinkAndLoginAndRevokesGrant(t *testing.T) {
	for _, kind := range []IntentKind{IntentKindLink, IntentKindLogin} {
		t.Run(string(kind), func(t *testing.T) {
			repository := &serviceRepository{intent: Intent{ID: "intent", Kind: kind,
				Subject: "user:test-001", CompletionKey: "done"}}
			oauth := &oauthGateway{enabled: true}
			service := NewService(repository, oauth, clock.NewFake(time.Now()),
				Config{ResultTTL: 5 * time.Minute})
			completion, err := service.Complete(context.Background(), Callback{State: "state", Code: "code"})
			if err != nil || completion.CompletionKey != "done" || completion.Code == "" || !oauth.revoked {
				t.Fatalf("Complete() = %#v, revoked = %v, error = %v", completion, oauth.revoked, err)
			}
			if kind == IntentKindLink && repository.completedLink.Identity.UserID != "123" {
				t.Fatalf("link completion = %#v", repository.completedLink)
			}
			if kind == IntentKindLogin && repository.completedAuth.Identity.UserID != "123" {
				t.Fatalf("login completion = %#v", repository.completedAuth)
			}
		})
	}
}

func TestServiceTurnsDeniedCallbackIntoExchangeableResult(t *testing.T) {
	repository := &serviceRepository{intent: Intent{ID: "intent", CompletionKey: "done"}}
	service := NewService(repository, &oauthGateway{enabled: true}, clock.NewFake(time.Now()),
		Config{ResultTTL: time.Minute})
	completion, err := service.Complete(context.Background(), Callback{State: "state", ProviderError: "access_denied"})
	if err != nil || completion.Code == "" || repository.failed.Status != ResultStatusDenied ||
		repository.failed.ErrorCode != ErrorCodeAuthorizationDenied {
		t.Fatalf("Complete() = %#v, failed = %#v, error = %v", completion, repository.failed, err)
	}
}
