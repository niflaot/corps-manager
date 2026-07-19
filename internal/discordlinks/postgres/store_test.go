package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
)

func TestStoreLinkAndLoginLifecycleIntegration(t *testing.T) {
	dsn := os.Getenv("DISCORD_BOT_INTEGRATION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set DISCORD_BOT_INTEGRATION_POSTGRES_DSN after applying Liquibase migrations")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err = pool.Exec(ctx, `TRUNCATE discord_link_intents, discord_links CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	store := NewStore(pool)
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	linkIntent, err := store.CreateIntent(ctx, discordlinks.CreateIntentRecord{
		Kind: discordlinks.IntentKindLink, Subject: "user:test-001", CompletionKey: "links",
		IdempotencyKey: "link-001", RequestHash: hash("a"), ExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil {
		t.Fatalf("CreateIntent() error = %v", err)
	}
	replayed, err := store.CreateIntent(ctx, discordlinks.CreateIntentRecord{
		Kind: discordlinks.IntentKindLink, Subject: "user:test-001", CompletionKey: "links",
		IdempotencyKey: "link-001", RequestHash: hash("a"), ExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil || replayed.ID != linkIntent.ID {
		t.Fatalf("replayed intent = %#v, error = %v", replayed, err)
	}
	if _, err = store.StartIntent(ctx, linkIntent.ID, hash("state"), now); err != nil {
		t.Fatalf("StartIntent() error = %v", err)
	}
	if _, err = store.ClaimIntentByState(ctx, hash("state"), now); err != nil {
		t.Fatalf("ClaimIntentByState() error = %v", err)
	}
	status, err := store.CompleteLinkIntent(ctx, discordlinks.CompleteIntentRecord{
		IntentID: linkIntent.ID, Identity: discordlinks.Identity{UserID: "123", Username: "tester"},
		Scopes: []string{"identify"}, ResultHash: hash("result"),
		ResultExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil || status != discordlinks.ResultStatusLinked {
		t.Fatalf("CompleteLinkIntent() = %q, error = %v", status, err)
	}
	result, err := store.ExchangeResult(ctx, hash("result"), "consumer-1", now)
	if err != nil || result.Link == nil || result.Subject != "user:test-001" {
		t.Fatalf("ExchangeResult() = %#v, error = %v", result, err)
	}
	if _, err = store.ExchangeResult(ctx, hash("result"), "consumer-2", now); !errors.Is(err, discordlinks.ErrGone) {
		t.Fatalf("different consumer error = %v", err)
	}
	conflictIntent, err := store.CreateIntent(ctx, discordlinks.CreateIntentRecord{
		Kind: discordlinks.IntentKindLink, Subject: "user:test-002", CompletionKey: "links",
		IdempotencyKey: "link-002", RequestHash: hash("c"), ExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil {
		t.Fatalf("CreateIntent(conflict) error = %v", err)
	}
	if _, err = store.StartIntent(ctx, conflictIntent.ID, hash("conflict-state"), now); err != nil {
		t.Fatalf("StartIntent(conflict) error = %v", err)
	}
	if _, err = store.ClaimIntentByState(ctx, hash("conflict-state"), now); err != nil {
		t.Fatalf("ClaimIntentByState(conflict) error = %v", err)
	}
	status, err = store.CompleteLinkIntent(ctx, discordlinks.CompleteIntentRecord{
		IntentID: conflictIntent.ID, Identity: discordlinks.Identity{UserID: "123", Username: "tester"},
		Scopes: []string{"identify"}, ResultHash: hash("conflict-result"),
		ResultExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil || status != discordlinks.ResultStatusConflict {
		t.Fatalf("CompleteLinkIntent(conflict) = %q, error = %v", status, err)
	}
	conflictResult, err := store.ExchangeResult(ctx, hash("conflict-result"), "conflict-consumer", now)
	if err != nil || conflictResult.Status != discordlinks.ResultStatusConflict ||
		conflictResult.ErrorCode != discordlinks.ErrorCodeDiscordUserAlreadyLinked {
		t.Fatalf("conflict result = %#v, error = %v", conflictResult, err)
	}
	loginIntent, err := store.CreateIntent(ctx, discordlinks.CreateIntentRecord{
		Kind: discordlinks.IntentKindLogin, CompletionKey: "login", IdempotencyKey: "login-001",
		RequestHash: hash("b"), ExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil {
		t.Fatalf("CreateIntent(login) error = %v", err)
	}
	if _, err = store.StartIntent(ctx, loginIntent.ID, hash("login-state"), now); err != nil {
		t.Fatalf("StartIntent(login) error = %v", err)
	}
	if _, err = store.ClaimIntentByState(ctx, hash("login-state"), now); err != nil {
		t.Fatalf("ClaimIntentByState(login) error = %v", err)
	}
	status, err = store.CompleteLoginIntent(ctx, discordlinks.CompleteLoginIntentRecord{
		IntentID: loginIntent.ID, Identity: discordlinks.Identity{UserID: "123", Username: "new-name"},
		Scopes: []string{"identify"}, ResultHash: hash("login-result"),
		ResultExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil || status != discordlinks.ResultStatusAuthenticated {
		t.Fatalf("CompleteLoginIntent() = %q, error = %v", status, err)
	}
	loginResult, err := store.ExchangeResult(ctx, hash("login-result"), "login-consumer", now)
	if err != nil || loginResult.Subject != "user:test-001" || loginResult.Link.Username != "new-name" {
		t.Fatalf("login result = %#v, error = %v", loginResult, err)
	}
	unlinked, err := store.Unlink(ctx, "user:test-001", now.Add(time.Second))
	if err != nil || unlinked.Status != discordlinks.LinkStatusUnlinked {
		t.Fatalf("Unlink() = %#v, error = %v", unlinked, err)
	}
	replayedUnlink, err := store.Unlink(ctx, "user:test-001", now.Add(2*time.Second))
	if err != nil || replayedUnlink.ID != unlinked.ID {
		t.Fatalf("replayed Unlink() = %#v, error = %v", replayedUnlink, err)
	}
	if _, err = store.LinkByDiscordUser(ctx, "123"); !errors.Is(err, discordlinks.ErrNotFound) {
		t.Fatalf("LinkByDiscordUser() error = %v", err)
	}
	notLinkedIntent, err := store.CreateIntent(ctx, discordlinks.CreateIntentRecord{
		Kind: discordlinks.IntentKindLogin, CompletionKey: "login", IdempotencyKey: "login-002",
		RequestHash: hash("d"), ExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil {
		t.Fatalf("CreateIntent(not linked) error = %v", err)
	}
	if _, err = store.StartIntent(ctx, notLinkedIntent.ID, hash("not-linked-state"), now); err != nil {
		t.Fatalf("StartIntent(not linked) error = %v", err)
	}
	if _, err = store.ClaimIntentByState(ctx, hash("not-linked-state"), now); err != nil {
		t.Fatalf("ClaimIntentByState(not linked) error = %v", err)
	}
	status, err = store.CompleteLoginIntent(ctx, discordlinks.CompleteLoginIntentRecord{
		IntentID: notLinkedIntent.ID, Identity: discordlinks.Identity{UserID: "123", Username: "tester"},
		Scopes: []string{"identify"}, ResultHash: hash("not-linked-result"),
		ResultExpiresAt: now.Add(time.Minute), Now: now,
	})
	if err != nil || status != discordlinks.ResultStatusNotLinked {
		t.Fatalf("CompleteLoginIntent(not linked) = %q, error = %v", status, err)
	}
}

func hash(value string) string {
	result := value
	for len(result) < 64 {
		result += value
	}
	return result[:64]
}
