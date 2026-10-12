package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSettingsRoundTrip(t *testing.T) {
	dsn := os.Getenv("DISCORD_BOT_INTEGRATION_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set DISCORD_BOT_INTEGRATION_POSTGRES_DSN after Liquibase update")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "TRUNCATE bot_settings"); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	if err := store.SetMany(ctx, map[string]string{"security.a": "1", "security.b": "2", "other.c": "3"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetMany(ctx, map[string]string{"security.a": "9"}); err != nil {
		t.Fatal(err)
	}
	values, err := store.List(ctx, "security.")
	if err != nil || len(values) != 2 || values["security.a"] != "9" {
		t.Fatalf("values=%v error=%v", values, err)
	}
}
