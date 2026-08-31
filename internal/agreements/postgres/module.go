// Package postgres persists business agreements in PostgreSQL.
package postgres

import (
	"github.com/niflaot/corps-manager/internal/agreements"
	platformpostgres "github.com/niflaot/corps-manager/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL agreement repository.
var Module = fx.Module("agreements-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(agreements.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
