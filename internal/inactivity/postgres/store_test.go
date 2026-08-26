package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/inactivity"
)

func TestStoreLifecycleIntegration(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `TRUNCATE inactivity_dismissals`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	created, err := store.Add(ctx, "thomas_jhonson", "Thomas_Jhonson", "123")
	if err != nil || created.Name != "Thomas_Jhonson" || created.AddedBy != "123" {
		t.Fatalf("Add() = %#v, %v", created, err)
	}
	if _, err := store.Add(ctx, "thomas_jhonson", "Thomas_Jhonson", "123"); !errors.Is(err, inactivity.ErrAlreadyExists) {
		t.Fatalf("Add(duplicate) error = %v", err)
	}
	entries, err := store.List(ctx)
	if err != nil || len(entries) != 1 || entries[0].Name != "Thomas_Jhonson" {
		t.Fatalf("List() = %#v, %v", entries, err)
	}
	if err := store.Remove(ctx, "thomas_jhonson"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if err := store.Remove(ctx, "thomas_jhonson"); !errors.Is(err, inactivity.ErrNotFound) {
		t.Fatalf("Remove(missing) error = %v", err)
	}
}
