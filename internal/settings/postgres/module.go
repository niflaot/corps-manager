package postgres

import (
	"github.com/niflaot/corps-manager/internal/settings"
	platformpostgres "github.com/niflaot/corps-manager/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL runtime settings store.
var Module = fx.Module("settings-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(settings.Store))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
