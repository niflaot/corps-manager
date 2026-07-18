package postgres

import (
	"github.com/pixelados-net/discord-bot/internal/settings"
	platformpostgres "github.com/pixelados-net/discord-bot/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL settings repository.
var Module = fx.Module("settings-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(settings.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
