package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/agreements"
)

func TestCompanyScopedAgreementsAndDeletion(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE business_agreements, agreement_companies CASCADE"); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	for _, id := range []string{"company-a", "company-b"} {
		if _, err := store.CreateCompany(ctx, agreements.Company{ID: id, Name: id}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateCompany(ctx, agreements.Company{ID: "company-a", Name: "Duplicate"}); !errors.Is(err, agreements.ErrAlreadyExists) {
		t.Fatal(err)
	}
	for _, id := range []string{"company-a", "company-b"} {
		if _, err := store.Create(ctx, agreements.Agreement{CompanyID: id, ID: "shared", Description: "Example agreement", CreatedBy: "123"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Create(ctx, agreements.Agreement{CompanyID: "missing", ID: "shared", Description: "Example agreement", CreatedBy: "123"}); !errors.Is(err, agreements.ErrCompanyNotFound) {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, agreements.Agreement{CompanyID: "company-a", ID: "shared", Description: "Duplicate agreement", CreatedBy: "123"}); !errors.Is(err, agreements.ErrAlreadyExists) {
		t.Fatal(err)
	}
	items, err := store.List(ctx)
	if err != nil || len(items) != 2 || items[0].CompanyName != "company-a" {
		t.Fatalf("items=%v error=%v", items, err)
	}
	if err := store.DeleteCompany(ctx, "company-a"); !errors.Is(err, agreements.ErrCompanyInUse) {
		t.Fatal(err)
	}
	if _, err := store.CreateCompany(ctx, agreements.Company{ID: "empty", Name: "Empty company"}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteCompany(ctx, "empty"); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteCompany(ctx, "empty"); !errors.Is(err, agreements.ErrCompanyNotFound) {
		t.Fatal(err)
	}
}
