package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelados-net/discord-bot/internal/messages"
)

func TestStoreLifecycleIntegration(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `TRUNCATE message_idempotency, managed_messages`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	store := NewStore(pool)
	definition := messages.Definition{
		Key: "rules", GuildID: "123", ChannelID: "456",
		Payload: messages.Payload{Components: []messages.Component{messages.Component(`{"type":10,"content":"Rules"}`)}, AllowedMentions: messages.AllowedMentions{Parse: []string{}}},
	}
	idempotency := messages.Idempotency{Key: "create-1", Operation: "create:rules", RequestHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	created, err := store.Create(ctx, definition, idempotency)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	replayed, err := store.Create(ctx, definition, idempotency)
	if err != nil || !replayed.Replay || replayed.Record.ID != created.Record.ID {
		t.Fatalf("replay = %#v, error = %v", replayed, err)
	}
	definition.Payload.Components[0] = messages.Component(`{"type":10,"content":"Updated rules"}`)
	replaced, err := store.Replace(ctx, "rules", 1, definition, messages.Idempotency{Key: "replace-1", Operation: "replace:rules", RequestHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil || replaced.Record.Revision != 2 {
		t.Fatalf("Replace() = %#v, error = %v", replaced, err)
	}
	now := time.Now().Add(time.Second)
	claimed, err := store.ClaimDue(ctx, messages.ClaimRequest{Owner: "test", Limit: 10, LeaseDuration: time.Minute, Now: now})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimDue() = %#v, error = %v", claimed, err)
	}
	if err := store.Complete(ctx, messages.Completion{ID: claimed[0].ID, Owner: "test", Revision: 2, DiscordMessageID: "789", ObservedHash: claimed[0].DesiredHash, CheckedAt: now, NextCheckAt: now.Add(time.Minute)}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	stored, err := store.GetByKey(ctx, "rules")
	if err != nil || stored.State != messages.StateHealthy || stored.DiscordMessageID != "789" {
		t.Fatalf("GetByKey() = %#v, error = %v", stored, err)
	}
	conflicting := idempotency
	conflicting.RequestHash = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if _, err := store.Create(ctx, definition, conflicting); !errors.Is(err, messages.ErrConflict) {
		t.Fatalf("conflicting idempotency error = %v", err)
	}
	definition.Payload.AllowedMentions.Users = []string{"123"}
	policyUpdate, err := store.Replace(ctx, "rules", 2, definition, messages.Idempotency{Key: "replace-2", Operation: "replace:rules", RequestHash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"})
	if err != nil || policyUpdate.Record.Revision != 3 || len(policyUpdate.Record.Payload.AllowedMentions.Users) != 1 {
		t.Fatalf("policy Replace() = %#v, error = %v", policyUpdate, err)
	}
	if err := store.MarkDue(ctx, "rules"); err != nil {
		t.Fatalf("MarkDue() error = %v", err)
	}
	claimed, err = store.ClaimDue(ctx, messages.ClaimRequest{Owner: "blocking", Limit: 10, LeaseDuration: time.Minute, Now: time.Now().Add(time.Second)})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("second ClaimDue() = %#v, error = %v", claimed, err)
	}
	if err := store.Complete(ctx, messages.Completion{ID: claimed[0].ID, Owner: "stale", Revision: 3}); !errors.Is(err, messages.ErrConflict) {
		t.Fatalf("stale Complete() error = %v", err)
	}
	if err := store.Release(ctx, messages.Release{ID: claimed[0].ID, Owner: "blocking", Revision: 3, State: messages.StateBlocked, Error: "permission", CheckedAt: time.Now(), NextCheckAt: time.Now()}); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	claimed, err = store.ClaimDue(ctx, messages.ClaimRequest{Owner: "other", Limit: 10, LeaseDuration: time.Minute, Now: time.Now().Add(time.Second)})
	if err != nil || len(claimed) != 0 {
		t.Fatalf("blocked ClaimDue() = %#v, error = %v", claimed, err)
	}
}
