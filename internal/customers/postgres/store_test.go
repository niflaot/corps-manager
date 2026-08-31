package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/customers"
)

func TestStoreSpendingAndPeriodSearchIntegration(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `TRUNCATE frequent_customer_visits,
		frequent_customer_attendants, frequent_customers`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	if _, err := store.Record(ctx, "jane_doe", "111", "Thomas J.", 2500); err != nil {
		t.Fatal(err)
	}
	item, err := store.Record(ctx, "jane_doe", "222", "Astrid S.", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if item.Visits != 2 || item.TotalSpent != 3500 || item.AttendantCount != 2 || len(item.Attendants) != 2 {
		t.Fatalf("customer = %+v", item)
	}
	items, err := store.Search(ctx, customers.Query{Name: "jane", Days: 1, Sort: customers.SortSpend})
	if err != nil || len(items) != 1 || items[0].TotalSpent != 3500 || items[0].Visits != 2 {
		t.Fatalf("Search() = %+v, %v", items, err)
	}
	if err := store.Delete(ctx, "jane_doe"); err != nil {
		t.Fatal(err)
	}
}
