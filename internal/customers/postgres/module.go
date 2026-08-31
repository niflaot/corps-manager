// Package postgres persists frequent customers in PostgreSQL.
package postgres

import (
	"github.com/niflaot/corps-manager/internal/customers"
	platformpostgres "github.com/niflaot/corps-manager/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL customer repository.
var Module = fx.Module("customers-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(customers.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
