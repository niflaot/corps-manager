package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelados-net/discord-bot/internal/verification/notification"
)

func TestStoreOutboxLifecycleIntegration(t *testing.T) {
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
	if _, err = pool.Exec(ctx, `TRUNCATE verification_notification_outbox`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	store := NewStore(pool)
	event := notification.NewEvent(notification.KindVerified, "membership", "123", "group", "member")
	inserted, err := store.Enqueue(ctx, event)
	if err != nil || !inserted {
		t.Fatalf("Enqueue() = %t, %v", inserted, err)
	}
	inserted, err = store.Enqueue(ctx, event)
	if err != nil || inserted {
		t.Fatalf("duplicate Enqueue() = %t, %v", inserted, err)
	}
	now := time.Now().Add(time.Second)
	deliveries, err := store.ClaimDue(ctx, notification.ClaimRequest{
		Owner: "worker", Limit: 10, LeaseDuration: time.Minute, Now: now,
	})
	if err != nil || len(deliveries) != 1 {
		t.Fatalf("ClaimDue() = %#v, %v", deliveries, err)
	}
	if err = store.Release(ctx, notification.Release{
		ID: deliveries[0].ID, Owner: "worker", State: notification.StateRetry,
		Error: "temporary", NextAttemptAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	deliveries, err = store.ClaimDue(ctx, notification.ClaimRequest{
		Owner: "worker", Limit: 10, LeaseDuration: time.Minute, Now: now.Add(2 * time.Minute),
	})
	if err != nil || len(deliveries) != 1 || deliveries[0].Attempts != 1 {
		t.Fatalf("retry ClaimDue() = %#v, %v", deliveries, err)
	}
	if err = store.Complete(ctx, notification.Completion{
		ID: deliveries[0].ID, Owner: "worker", DiscordMessageID: "456", DeliveredAt: now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	deliveries, err = store.ClaimDue(ctx, notification.ClaimRequest{
		Owner: "worker", Limit: 10, LeaseDuration: time.Minute, Now: now.Add(3 * time.Minute),
	})
	if err != nil || len(deliveries) != 0 {
		t.Fatalf("delivered ClaimDue() = %#v, %v", deliveries, err)
	}
}
