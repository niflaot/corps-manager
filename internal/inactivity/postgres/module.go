// Package postgres persists inactivity dismissal entries in PostgreSQL.
package postgres

import (
	"github.com/niflaot/corps-manager/internal/inactivity"
	platformpostgres "github.com/niflaot/corps-manager/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL inactivity registry repository.
var Module = fx.Module("inactivity-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(inactivity.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
