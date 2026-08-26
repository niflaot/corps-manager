package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/performance"
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
	if _, err := pool.Exec(ctx, `TRUNCATE business_performance_state`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	state := performance.State{BusinessID: 1995, Name: "Benny", Employees: map[string]performance.EmployeeState{},
		PeriodStartedAt: time.Now(), LastCollectedAt: time.Now()}
	created, err := store.Save(ctx, state, 0)
	if err != nil || created.Revision != 1 {
		t.Fatalf("Save(create) = %#v, %v", created, err)
	}
	read, err := store.Get(ctx, 1995)
	if err != nil || read.Name != "Benny" || read.Revision != 1 {
		t.Fatalf("Get() = %#v, %v", read, err)
	}
	read.Bank = 500
	updated, err := store.Save(ctx, read, 1)
	if err != nil || updated.Revision != 2 || updated.Bank != 500 {
		t.Fatalf("Save(update) = %#v, %v", updated, err)
	}
	if _, err := store.Save(ctx, updated, 1); !errors.Is(err, performance.ErrConflict) {
		t.Fatalf("Save(stale) error = %v", err)
	}
}
