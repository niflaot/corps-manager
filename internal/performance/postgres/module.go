package postgres

import (
	"github.com/pixelados-net/discord-bot/internal/performance"
	platformpostgres "github.com/pixelados-net/discord-bot/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL performance repository.
var Module = fx.Module("performance-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(performance.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
