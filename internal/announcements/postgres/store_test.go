package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/announcements"
)

func TestStoreCooldownLifecycleIntegration(t *testing.T) {
	dsn := os.Getenv("DISCORD_BOT_INTEGRATION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set DISCORD_BOT_INTEGRATION_POSTGRES_DSN after applying Liquibase migrations")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE announcement_cooldowns`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	created, err := store.Acquire(ctx, announcements.OpeningCooldownKey, now, now.Add(time.Minute), "Thomas J.")
	if err != nil || created.Actor != "Thomas J." {
		t.Fatalf("Acquire() = %#v, %v", created, err)
	}
	if _, err := store.Acquire(ctx, announcements.OpeningCooldownKey, now.Add(time.Second),
		now.Add(2*time.Minute), "Other"); !errors.Is(err, announcements.ErrCooldownActive) {
		t.Fatalf("Acquire(active) error = %v", err)
	}
	replaced, err := store.Acquire(ctx, announcements.OpeningCooldownKey, now.Add(time.Minute),
		now.Add(2*time.Minute), "Other")
	if err != nil || replaced.Actor != "Other" {
		t.Fatalf("Acquire(expired) = %#v, %v", replaced, err)
	}
	if err := store.Release(ctx, announcements.OpeningCooldownKey, created.AnnouncedAt); err != nil {
		t.Fatal(err)
	}
	if current, err := store.Get(ctx, announcements.OpeningCooldownKey); err != nil || current.Actor != "Other" {
		t.Fatalf("Get() = %#v, %v", current, err)
	}
	if err := store.Clear(ctx, announcements.OpeningCooldownKey); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, announcements.OpeningCooldownKey); !errors.Is(err, announcements.ErrNotFound) {
		t.Fatalf("Get(cleared) error = %v", err)
	}
}
