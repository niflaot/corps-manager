package postgres

import (
	"github.com/niflaot/corps-manager/internal/performance"
	platformpostgres "github.com/niflaot/corps-manager/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL performance repository.
var Module = fx.Module("performance-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(performance.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
